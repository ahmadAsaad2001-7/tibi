# Architecture decisions

## R10 — ClinicSession has no status; runtime state lives on QueueWindow

Date: slice 9.
Reason: slice 6 shipped doctors_clinic_sessions with only structural columns
        (id, doctor, date, start/end). Slice 9 needs runtime session state
        (open/closed) and per-session counters (next_queue_number). Putting
        either on the Doctors table would make Queue write to another
        module's row on the hot path.
Change: doctors_clinic_sessions stays immutable. queue_windows (Queue
        module) owns status, next_queue_number, current_queue_entry_id.
        One QueueWindow per ClinicSession, created lazily on first
        check-in.
Direction: consistent with D-003. Doctors owns existence; Queue owns
           runtime.

## R15 — Files access checker becomes a registry, not a single rule

Date: slice 12.
Reason: slice 11 shipped accesschecker.Default which permits only the
        uploader. That is correct for the standalone file endpoint but
        wrong once a file is embedded in a resource (medical record,
        post attachment, profile image).
Change: accesschecker now holds a per-scope registry of ScopeRule
        implementations. Modules that own a scope (Clinical, later
        Content, Identity) implement ScopeRule and register it at
        startup via Register(). Files never learns what a "medical
        record" is; the owner teaches the checker how to authorize
        reads of files it owns. Uploader is always allowed.
Direction: inversion of control — Files stays unaware of Clinical and
           every future adopter. No import direction from Files to any
           module.

## R16 — Clinical writes require consultation InProgress or Completed

Date: slice 12.
Reason: a doctor cannot write notes for a consultation they have not
        started. Once a consultation is Completed the record stays
        writable (addendum); once it is Cancelled or NoShow, no record
        may be created or updated.
Change: every clinical write upsertrecord, addattachment, and
        upsertprescription first calls consultations.ClinicalContext to
        verify the caller is the treating doctor AND the consultation
        status is InProgress or Completed. Otherwise it returns 422
        unprocessable.
Direction: Consultations remains the authority on consultation state;
           Clinical never re-implements status logic.

## R24 — identity_users.email_verified_at column added

Date: slice 15.
Reason: email verification requires a persistent, per-user flag. Nothing
        gated on it yet (SD35), but /auth/me surfaces email_verified and the
        column exists for a future login gate.
Change: identity_users.email_verified_at TIMESTAMPTZ added. Existing users
        get NULL; new registrations start NULL; the verification flow sets it.
        A partial index (email_verified_at IS NULL AND deleted_at IS NULL)
        supports "who still needs a nudge" queries at no write cost.
        The confirm flow sets it idempotently via `AND email_verified_at
        IS NULL`.
Direction: SD35 — email verification is not required to log in yet. Gating is
           a future product decision that reads this column.

## R25 — Password reset is the second flow that invalidates refresh tokens

Date: slice 15.
Reason: a reset is a security event (SD34). Slice 1's logout revoked one
        token; a reset must terminate every active session, otherwise a stolen
        old session survives the password change.
Change: confirmpasswordreset runs RevokeAllRefreshTokens inside the same
        transaction as ConsumePasswordReset + UpdateUserPassword. Uses the
        existing identity_refresh_tokens.revoked_at column.
Direction: refresh-token invalidation is expressed via revoked_at everywhere;
           no hard delete.

## R26 — ratelimit.Limiter signature changed

Date: slice 16.
Reason: slice 14's Allow(key string) bool cannot serve a persistent
        implementation. A Postgres-backed limiter needs a context and can
        error.
Change: new signature Allow(ctx, key) (bool, error). All five call sites
        updated. Errors fail open (SD36): log at WARN and allow.
        ratelimit.AllowOrFailOpen is the shared helper. Memory and Postgres
        implementations live side by side, selected by RATE_LIMIT_DRIVER.
Direction: R22 still holds for Memory; Postgres is the multi-instance answer.
           Limit behavior otherwise unchanged (SD38 fixed window, Go-computed
           bucket).

## R27 — database.Connect installs a query tracer

Date: slice 16.
Reason: observability (SD39). One slog call per query was judged cheaper than
        a feature flag.
Change: cfg.ConnConfig.Tracer = NewTracer(log). Every query logs at DEBUG
        with op, duration, and trace ID; slow queries (>100ms) log at WARN.
        The tracer also feeds metrics (SD40). Trace ID is read from a shared
        platform/trace package so httpx and database never import each other.
Direction: tracer is always on, no opt-out. Prometheus is served on /metrics.
