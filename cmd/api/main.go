package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"tibi/internal/admin/castvote"
	"tibi/internal/admin/expirevotes"
	"tibi/internal/admin/listpendingdoctors"
	"tibi/internal/admin/listvotes"
	"tibi/internal/admin/proposevote"

	"tibi/internal/clinical/addattachment"
	"tibi/internal/clinical/deleteattachment"
	"tibi/internal/clinical/getprescription"
	"tibi/internal/clinical/getrecord"
	clinicalscope "tibi/internal/clinical/scope"
	"tibi/internal/clinical/upsertprescription"
	"tibi/internal/clinical/upsertrecord"

	consultationsapi "tibi/internal/consultations/api"
	"tibi/internal/consultations/book"
	"tibi/internal/consultations/cancel"
	"tibi/internal/consultations/checkininfo"
	"tibi/internal/consultations/clinicalcontext"
	"tibi/internal/consultations/getbyid"
	"tibi/internal/consultations/listmy"
	"tibi/internal/consultations/markcompleted"
	"tibi/internal/consultations/markconfirmed"
	consultationswschecker "tibi/internal/consultations/wschecker"

	communicationapi "tibi/internal/communication/api"
	"tibi/internal/communication/createnotification"
	"tibi/internal/communication/listmessages"
	"tibi/internal/communication/listnotifications"
	"tibi/internal/communication/markallread"
	"tibi/internal/communication/markread"
	"tibi/internal/communication/notifymessage"
	"tibi/internal/communication/sendmessage"
	"tibi/internal/communication/unreadcount"

	"tibi/internal/doctors/addexception"
	doctorsapi "tibi/internal/doctors/api"
	"tibi/internal/doctors/contracts"
	"tibi/internal/doctors/createsession"
	"tibi/internal/doctors/deleteexception"
	"tibi/internal/doctors/getdoctor"
	"tibi/internal/doctors/getschedule"
	"tibi/internal/doctors/listavailabilitydays"
	"tibi/internal/doctors/listslots"
	"tibi/internal/doctors/listspecialties"
	"tibi/internal/doctors/putschedule"
	"tibi/internal/doctors/search"
	"tibi/internal/doctors/submitforverification"
	"tibi/internal/doctors/updateprofile"

	"tibi/internal/doctorposts/createpost"
	"tibi/internal/doctorposts/deletepost"
	"tibi/internal/doctorposts/getpost"
	"tibi/internal/doctorposts/listbydoctor"
	"tibi/internal/doctorposts/listfeed"
	"tibi/internal/doctorposts/publishpost"
	"tibi/internal/doctorposts/unpublishpost"
	"tibi/internal/doctorposts/updatepost"

	"tibi/internal/files/accesschecker"
	filesapi "tibi/internal/files/api"
	filedelete "tibi/internal/files/delete"
	"tibi/internal/files/get"
	"tibi/internal/files/getmetadata"
	"tibi/internal/files/upload"

	identityapi "tibi/internal/identity/api"
	identitycontracts "tibi/internal/identity/contracts"
	"tibi/internal/identity/login"
	"tibi/internal/identity/logout"
	"tibi/internal/identity/me"
	"tibi/internal/identity/refresh"
	"tibi/internal/identity/register"
	identityscope "tibi/internal/identity/scope" // ✅ ADDED (verify exact path)
	"tibi/internal/identity/setprofileimage"     // ✅ ADDED

	patientscreate "tibi/internal/patients/createprofile"

	paymentsapi "tibi/internal/payments/api"
	"tibi/internal/payments/createpending"
	"tibi/internal/payments/kashier"
	"tibi/internal/payments/kashier/webhook"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/config"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
	"tibi/internal/platform/ws"
	"tibi/internal/storage"

	"tibi/internal/queue/call"
	"tibi/internal/queue/checkin"
	"tibi/internal/queue/closesession"
	"tibi/internal/queue/complete"
	"tibi/internal/queue/eta"
	"tibi/internal/queue/getpatientqueue"
	"tibi/internal/queue/getsessionqueue"
	"tibi/internal/queue/markcancelled"
	"tibi/internal/queue/noshow"
	"tibi/internal/queue/skip"
	"tibi/internal/queue/start"
	queuewschecker "tibi/internal/queue/wschecker"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.IsProduction() && cfg.JWTSecret == "replace-me-with-32-bytes-of-random" {
		return errors.New("JWT_SECRET not set in production")
	}
	if cfg.IsProduction() && strings.HasPrefix(cfg.KashierAPIKey, "sk_test_") {
		return errors.New("Kashier test key set in production")
	}

	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))
	if cfg.IsProduction() {
		for _, o := range cfg.WSOriginPatterns {
			if o == "*" {
				slog.Warn("ws_origin_wildcard_in_production")
			}
		}
	}

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Platform
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTAccessTTL)
	refreshStore := auth.NewRefreshStore(db, cfg.JWTRefreshTTL)
	kashierClient := kashier.NewClient(cfg.KashierAPIKey, cfg.KashierAPIURL)

	consultationChecker := consultationswschecker.NewService(db)
	queueChecker := queuewschecker.NewService(db)
	hub := ws.NewHub(combinedChecker{c: consultationChecker, q: queueChecker}, slog.Default())

	// Platform storage
	var st storage.Storage
	switch cfg.StorageDriver {
	case "local":
		st, err = storage.NewLocalFS(cfg.StorageLocalRoot, cfg.StorageLocalBaseURL, cfg.StorageLocalSignKey)
	case "s3":
		st, err = storage.NewS3(storage.S3Config{
			Endpoint: cfg.S3Endpoint, AccessKey: cfg.S3AccessKey, SecretKey: cfg.S3SecretKey,
			Bucket: cfg.S3Bucket, UseSSL: cfg.S3UseSSL, Region: cfg.S3Region, PublicURL: cfg.S3PublicURL,
		})
	default:
		return errors.New("unknown storage driver")
	}
	if err != nil {
		return err
	}

	// Files module — access checker with per-scope registry.
	fileAuthz := accesschecker.New(db)
	fileUploadSvc := upload.NewService(db, st)
	fileGetSvc := get.NewService(db, st)
	fileMetaSvc := getmetadata.NewService(db)
	fileDeleteSvc := filedelete.NewService(db, st)
	filesAPI := filesapi.New(fileMetaSvc, st)

	// Register scope rules BEFORE building routes.
	fileAuthz.Register("MedicalAttachment", clinicalscope.New(db))
	fileAuthz.Register("ProfileImage", identityscope.New()) // ✅ ADDED

	// Consultations — new clinicalcontext service
	clinicalCtxSvc := clinicalcontext.NewService(db)

	// Core APIs
	identityAPI := identityapi.New(db)

	// Doctors
	createSessionSvc := createsession.NewService(db)
	createProfileSvc := patientscreate.NewService(db)
	doctorsAPI := doctorsapi.New(db, createSessionSvc)

	// Payments
	createPendingSvc := createpending.NewService(db, kashierClient)
	paymentsAPI := paymentsapi.New(createPendingSvc)

	// Communication — messages + notifications
	createNotifSvc := createnotification.NewService(db)
	notifyMsgSvc := notifymessage.NewService(createNotifSvc)
	notificationsAPI := communicationapi.New(createNotifSvc, notifyMsgSvc)

	sendMessageSvc := sendmessage.NewService(db, hub, notificationsAPI)
	listMessagesSvc := listmessages.NewService(db)
	listNotificationsSvc := listnotifications.NewService(db)
	markReadSvc := markread.NewService(db)
	markAllReadSvc := markallread.NewService(db)
	unreadCountSvc := unreadcount.NewService(db)

	// Consultations
	bookSvc := book.NewService(db, doctorsAPI, paymentsAPI, notificationsAPI)
	listMySvc := listmy.NewService(db)
	getByIDSvc := getbyid.NewService(db)

	markConfirmedSvc := markconfirmed.NewService(db, hub, notificationsAPI)
	checkinInfoSvc := checkininfo.NewService(db)
	markCompletedSvc := markcompleted.NewService(db)
	consultationsAPI := consultationsapi.New(markConfirmedSvc, checkinInfoSvc, markCompletedSvc, clinicalCtxSvc)
	webhookSvc := webhook.NewService(db, cfg.KashierWebhookSecret, consultationsAPI)

	// Clinical module
	upsertRecordSvc := upsertrecord.NewService(db, consultationsAPI)
	getRecordSvc := getrecord.NewService(db, filesAPI)
	addAttachSvc := addattachment.NewService(db, consultationsAPI, filesAPI)
	delAttachSvc := deleteattachment.NewService(db)
	upsertPrescSvc := upsertprescription.NewService(db, consultationsAPI)
	getPrescSvc := getprescription.NewService(db)

	// Queue & Other Services
	etaSvc := eta.NewService(db)
	sessionQueueSvc := getsessionqueue.NewService(db)
	markCancelSvc := markcancelled.NewService(db, hub, sessionQueueSvc)
	cancelSvc := cancel.NewService(db, markCancelSvc)
	checkinSvc := checkin.NewService(db, consultationsAPI, etaSvc, hub, sessionQueueSvc)
	callSvc := call.NewService(db, hub, sessionQueueSvc, notificationsAPI)
	startSvc := start.NewService(db, hub, sessionQueueSvc)
	completeSvc := complete.NewService(db, consultationsAPI, hub, sessionQueueSvc)
	skipSvc := skip.NewService(db, hub, sessionQueueSvc)
	noshowSvc := noshow.NewService(db, hub, sessionQueueSvc)
	closeSessionSvc := closesession.NewService(db, hub, sessionQueueSvc)
	patientQueueSvc := getpatientqueue.NewService(db, etaSvc)

	// Doctor Profile services
	specialtiesSvc := listspecialties.NewService(db)
	updateProfileSvc := updateprofile.NewService(db)
	submitVerifSvc := submitforverification.NewService(db)

	searchSvc := search.NewService(db, filesAPI)
	getDoctorSvc := getdoctor.NewService(db, filesAPI)

	// Schedule & Availability services
	putScheduleSvc := putschedule.NewService(db)
	getScheduleSvc := getschedule.NewService(db)
	addExceptionSvc := addexception.NewService(db)
	delExceptionSvc := deleteexception.NewService(db)
	availDaysSvc := listavailabilitydays.NewService(db)
	slotsSvc := listslots.NewService(db)

	// Admin module services
	proposeVoteSvc := proposevote.NewService(db)
	castVoteSvc := castvote.NewService(db, doctorsAPI, identityAPI)
	listVotesSvc := listvotes.NewService(db)
	listPendingSvc := listpendingdoctors.NewService(db)
	expireWorker := expirevotes.NewService(db)

	// Doctor Posts services
	createPostSvc := createpost.NewService(db, doctorsAPI, filesAPI) // ✅ أضفنا filesAPI هنا
	updatePostSvc := updatepost.NewService(db)
	deletePostSvc := deletepost.NewService(db)
	publishPostSvc := publishpost.NewService(db)
	unpublishPostSvc := unpublishpost.NewService(db)
	listFeedSvc := listfeed.NewService(db)
	getPostSvc := getpost.NewService(db)
	listByDoctorSvc := listbydoctor.NewService(db)

	// Start background worker for expiring votes
	go expireWorker.Run(ctx)

	// Identity services
	registerSvc := register.NewService(db, hasher, issuer, refreshStore, createProfileSvc, doctorsAPI)
	loginSvc := login.NewService(db, hasher, issuer, refreshStore)
	refreshSvc := refresh.NewService(db, issuer, refreshStore)
	logoutSvc := logout.NewService(refreshStore)

	// ✅ UPDATED: Injected filesAPI
	meSvc := me.NewService(db, filesAPI)

	// ✅ ADDED: Profile Image service
	setProfileImageSvc := setprofileimage.NewService(db, filesAPI)

	r := chi.NewRouter()
	r.Use(httpx.Trace)
	r.Use(httpx.Recover)
	r.Use(httpx.RequestLog)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	r.Get("/hubs/consultations", ws.NewHandler(hub, issuer, cfg.WSOriginPatterns).ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Post("/register", register.NewHandler(registerSvc).ServeHTTP)
			r.Post("/login", login.NewHandler(loginSvc).ServeHTTP)
			r.Post("/refresh", refresh.NewHandler(refreshSvc).ServeHTTP)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Get("/me", me.NewHandler(meSvc).ServeHTTP)
				r.Post("/logout", func(w http.ResponseWriter, r *http.Request) {
					var cmd logout.Command
					if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
						httpx.Error(w, r, httpx.BadRequest("malformed json"))
						return
					}
					if err := logoutSvc.Execute(r.Context(), cmd); err != nil {
						httpx.Error(w, r, err)
						return
					}
					w.WriteHeader(http.StatusNoContent)
				})
			})
			r.Post("/webhooks/kashier", webhook.NewHandler(webhookSvc).ServeHTTP)
		})

		r.Get("/specialties", listspecialties.NewHandler(specialtiesSvc).ServeHTTP)

		// Public doctor search and get
		r.Get("/doctors", search.NewHandler(searchSvc).ServeHTTP)
		r.Get("/doctors/{id}", getdoctor.NewHandler(getDoctorSvc).ServeHTTP)

		// Public doctor posts feed
		r.Get("/doctors/{doctorId}/posts", listbydoctor.NewHandler(listByDoctorSvc).ServeHTTP)

		r.Route("/profile", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Use(auth.RequireRole(contracts.RoleDoctor, contracts.RolePendingDoctor))
				r.Patch("/doctor", updateprofile.NewHandler(updateProfileSvc).ServeHTTP)
				r.Post("/doctor/submit-for-verification", submitforverification.NewHandler(submitVerifSvc).ServeHTTP)
			})

			// ✅ ADDED: Profile image update route (available to any authenticated user)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Patch("/image", setprofileimage.NewHandler(setProfileImageSvc).ServeHTTP)
			})
		})

		// Schedule & Availability routes
		r.Get("/doctor-availability/{doctorProfileId}/days", listavailabilitydays.NewHandler(availDaysSvc).ServeHTTP)
		r.Get("/doctor-availability/{doctorProfileId}/hours", listslots.NewHandler(slotsSvc).ServeHTTP)

		r.Route("/doctor-schedule", func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))
			r.Use(auth.RequireRole(contracts.RoleDoctor, contracts.RolePendingDoctor))
			r.Get("/", getschedule.NewHandler(getScheduleSvc).ServeHTTP)
			r.Put("/", putschedule.NewHandler(putScheduleSvc).ServeHTTP)
			r.Post("/exceptions", addexception.NewHandler(addExceptionSvc).ServeHTTP)
			r.Delete("/exceptions/{id}", deleteexception.NewHandler(delExceptionSvc).ServeHTTP)
		})

		// Doctor Posts routes
		r.Route("/doctor-posts", func(r chi.Router) {
			r.Get("/", listfeed.NewHandler(listFeedSvc).ServeHTTP)
			r.Get("/{id}", getpost.NewHandler(getPostSvc).ServeHTTP)

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Use(auth.RequireRole(contracts.RoleDoctor))
				r.Post("/", createpost.NewHandler(createPostSvc).ServeHTTP)
				r.Patch("/{id}", updatepost.NewHandler(updatePostSvc).ServeHTTP)
				r.Delete("/{id}", deletepost.NewHandler(deletePostSvc).ServeHTTP)
				r.Post("/{id}/publish", publishpost.NewHandler(publishPostSvc).ServeHTTP)
				r.Post("/{id}/unpublish", unpublishpost.NewHandler(unpublishPostSvc).ServeHTTP)
			})
		})

		// Consultations routes
		r.Route("/consultations", func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))

			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(identitycontracts.RolePatient))
				r.Post("/", book.NewHandler(bookSvc).ServeHTTP)
				r.Post("/{id}/check-in", checkin.NewHandler(checkinSvc).ServeHTTP)
			})

			r.Get("/", listmy.NewHandler(listMySvc).ServeHTTP)
			r.Get("/{id}", getbyid.NewHandler(getByIDSvc).ServeHTTP)
			r.Get("/{id}/queue", getpatientqueue.NewHandler(patientQueueSvc).ServeHTTP)
			r.Post("/{id}/cancel", cancel.NewHandler(cancelSvc).ServeHTTP)

			// Clinical write routes (doctor only)
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireRole(identitycontracts.RoleDoctor))
				r.Put("/{id}/medical-record", upsertrecord.NewHandler(upsertRecordSvc).ServeHTTP)
				r.Post("/{id}/medical-record/attachments", addattachment.NewHandler(addAttachSvc).ServeHTTP)
				r.Delete("/{id}/medical-record/attachments/{attachmentId}", deleteattachment.NewHandler(delAttachSvc).ServeHTTP)
				r.Put("/{id}/prescription", upsertprescription.NewHandler(upsertPrescSvc).ServeHTTP)
			})

			// Read paths open to both doctor and patient; ownership enforced in SQL.
			r.Get("/{id}/medical-record", getrecord.NewHandler(getRecordSvc).ServeHTTP)
			r.Get("/{id}/prescription", getprescription.NewHandler(getPrescSvc).ServeHTTP)
		})

		// Messages and notifications — any authenticated participant.
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))

			r.Get("/consultations/{id}/messages", listmessages.NewHandler(listMessagesSvc).ServeHTTP)
			r.Post("/consultations/{id}/messages", sendmessage.NewHandler(sendMessageSvc).ServeHTTP)

			r.Get("/notifications", listnotifications.NewHandler(listNotificationsSvc).ServeHTTP)
			r.Post("/notifications/{id}/read", markread.NewHandler(markReadSvc).ServeHTTP)
			r.Post("/notifications/read-all", markallread.NewHandler(markAllReadSvc).ServeHTTP)
			r.Get("/notifications/unread-count", unreadcount.NewHandler(unreadCountSvc).ServeHTTP)
		})

		// Files routes
		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))
			r.Post("/files", upload.NewHandler(fileUploadSvc).ServeHTTP)
			r.Get("/files/{id}", get.NewHandler(fileGetSvc, fileAuthz).ServeHTTP)
			r.Get("/files/{id}/metadata", getmetadata.NewHandler(fileMetaSvc, fileAuthz).ServeHTTP)
			r.Delete("/files/{id}", filedelete.NewHandler(fileDeleteSvc, fileAuthz).ServeHTTP)
		})

		// Dev-only static serving for LocalFS driver. Disabled in production.
		if cfg.StorageDriver == "local" && !cfg.IsProduction() {
			local, _ := st.(*storage.LocalFS)
			r.Get("/files/{key...}", devFileHandler(local))
		}

		r.Group(func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))
			r.Use(auth.RequireRole(identitycontracts.RoleDoctor, identitycontracts.RoleAdmin))
			r.Get("/clinic-sessions/{id}/queue", getsessionqueue.NewHandler(sessionQueueSvc).ServeHTTP)
			r.Post("/clinic-sessions/{id}/close", closesession.NewHandler(closeSessionSvc).ServeHTTP)
			r.Post("/queue/{entryId}/call", call.NewHandler(callSvc).ServeHTTP)
			r.Post("/queue/{entryId}/start", start.NewHandler(startSvc).ServeHTTP)
			r.Post("/queue/{entryId}/complete", complete.NewHandler(completeSvc).ServeHTTP)
			r.Post("/queue/{entryId}/skip", skip.NewHandler(skipSvc).ServeHTTP)
			r.Post("/queue/{entryId}/no-show", noshow.NewHandler(noshowSvc).ServeHTTP)
		})

		// Admin module routes
		r.Route("/admin", func(r chi.Router) {
			r.Use(auth.RequireAuth(issuer))
			r.Use(auth.RequireRole(identitycontracts.RoleAdmin))

			proposeH := proposevote.NewHandler(proposeVoteSvc)
			r.Post("/doctors/{userId}/propose-verification", proposeH.ServeVerify)
			r.Post("/doctors/{userId}/propose-unverification", proposeH.ServeUnverify)

			r.Get("/votes", listvotes.NewHandler(listVotesSvc).ServeHTTP)
			r.Post("/votes/{voteId}/cast", castvote.NewHandler(castVoteSvc).ServeHTTP)
			r.Get("/doctors/pending", listpendingdoctors.NewHandler(listPendingSvc).ServeHTTP)
		})
	})

	srv := &http.Server{
		Addr:              ":" + cfg.HTTPPort,
		Handler:           r,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("http_listening", "port", cfg.HTTPPort, "env", cfg.Env)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		slog.Info("shutdown_initiated")
	case err := <-errCh:
		return err
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

func devFileHandler(local *storage.LocalFS) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if local == nil {
			httpx.Error(w, r, httpx.NotFound("file not found"))
			return
		}
		key := chi.URLParam(r, "key")
		expires, err := strconv.ParseInt(r.URL.Query().Get("expires"), 10, 64)
		if err != nil || key == "" {
			httpx.Error(w, r, httpx.Unauthenticated("invalid signature"))
			return
		}
		if !local.VerifySignature(key, expires, r.URL.Query().Get("sig")) {
			httpx.Error(w, r, httpx.Unauthenticated("invalid signature"))
			return
		}
		http.ServeFile(w, r, local.AbsPath(key))
	}
}
