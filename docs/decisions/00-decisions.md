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
