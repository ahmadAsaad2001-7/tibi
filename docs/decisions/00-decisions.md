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
