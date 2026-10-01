package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"tibi/internal/admin/castvote"
	"tibi/internal/admin/expirevotes"
	"tibi/internal/admin/listpendingdoctors"
	"tibi/internal/admin/listvotes"
	"tibi/internal/admin/proposevote"

	"tibi/internal/consultations/book"
	"tibi/internal/consultations/cancel"
	"tibi/internal/consultations/getbyid"
	"tibi/internal/consultations/listmy"

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

	// ⚠️ New Doctor Posts imports
	"tibi/internal/doctorposts/createpost"
	"tibi/internal/doctorposts/deletepost"
	"tibi/internal/doctorposts/getpost"
	"tibi/internal/doctorposts/listbydoctor"
	"tibi/internal/doctorposts/listfeed"
	"tibi/internal/doctorposts/publishpost"
	"tibi/internal/doctorposts/unpublishpost"
	"tibi/internal/doctorposts/updatepost"

	identityapi "tibi/internal/identity/api"
	identitycontracts "tibi/internal/identity/contracts"
	"tibi/internal/identity/login"
	"tibi/internal/identity/logout"
	"tibi/internal/identity/me"
	"tibi/internal/identity/refresh"
	"tibi/internal/identity/register"

	patientscreate "tibi/internal/patients/createprofile"

	paymentsapi "tibi/internal/payments/api"
	"tibi/internal/payments/createpending"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/config"
	"tibi/internal/platform/database"
	"tibi/internal/platform/httpx"
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

	logLevel := slog.LevelInfo
	if cfg.IsDevelopment() {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: logLevel})))

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Platform
	hasher := auth.NewPasswordHasher()
	issuer := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTAccessTTL)
	refreshStore := auth.NewRefreshStore(db, cfg.JWTRefreshTTL)

	// Core APIs
	identityAPI := identityapi.New(db)

	// Doctors
	createSessionSvc := createsession.NewService(db)
	createProfileSvc := patientscreate.NewService(db)
	doctorsAPI := doctorsapi.New(db, createSessionSvc)

	// Payments
	createPendingSvc := createpending.NewService(db)
	paymentsAPI := paymentsapi.New(createPendingSvc)

	// Consultations
	bookSvc := book.NewService(db, doctorsAPI, paymentsAPI)
	cancelSvc := cancel.NewService(db)
	listMySvc := listmy.NewService(db)
	getByIDSvc := getbyid.NewService(db)

	// Doctor Profile services
	specialtiesSvc := listspecialties.NewService(db)
	updateProfileSvc := updateprofile.NewService(db)
	submitVerifSvc := submitforverification.NewService(db)
	searchSvc := search.NewService(db)
	getDoctorSvc := getdoctor.NewService(db)

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

	// ⚠️ New: Doctor Posts services
	createPostSvc := createpost.NewService(db, doctorsAPI)
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
	meSvc := me.NewService(db)

	r := chi.NewRouter()
	r.Use(httpx.Trace)
	r.Use(httpx.Recover)
	r.Use(httpx.RequestLog)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

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
		})

		r.Get("/specialties", listspecialties.NewHandler(specialtiesSvc).ServeHTTP)

		// Public doctor search and get
		r.Get("/doctors", search.NewHandler(searchSvc).ServeHTTP)
		r.Get("/doctors/{id}", getdoctor.NewHandler(getDoctorSvc).ServeHTTP)

		// ⚠️ New: Public doctor posts feed
		r.Get("/doctors/{doctorId}/posts", listbydoctor.NewHandler(listByDoctorSvc).ServeHTTP)

		r.Route("/profile", func(r chi.Router) {
			r.Group(func(r chi.Router) {
				r.Use(auth.RequireAuth(issuer))
				r.Use(auth.RequireRole(contracts.RoleDoctor, contracts.RolePendingDoctor))
				r.Patch("/doctor", updateprofile.NewHandler(updateProfileSvc).ServeHTTP)
				r.Post("/doctor/submit-for-verification", submitforverification.NewHandler(submitVerifSvc).ServeHTTP)
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

		// ⚠️ New: Doctor Posts routes
		r.Route("/doctor-posts", func(r chi.Router) {
			// Public read access
			r.Get("/", listfeed.NewHandler(listFeedSvc).ServeHTTP)
			r.Get("/{id}", getpost.NewHandler(getPostSvc).ServeHTTP)

			// Protected write access (Doctors only)
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
			})

			r.Get("/", listmy.NewHandler(listMySvc).ServeHTTP)
			r.Get("/{id}", getbyid.NewHandler(getByIDSvc).ServeHTTP)
			r.Post("/{id}/cancel", cancel.NewHandler(cancelSvc).ServeHTTP)
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
