# Tibi Frontend — Medica Next.js App (`tibifront/`)

> Arabic-first (RTL) medical platform UI: Patient, Doctor, Receptionist, Admin, and PendingDoctor portals with EHR timelines, prescriptions, live queue, and admin analytics. Standalone prototype UI with its own Prisma/SQLite API layer — see §9 for how it relates to the Go backend.

Related docs: [`../README.md`](../../README.md) · Backend companion [`backend.md`](backend.md) · work log [`../../tibifront/worklog.md`](../../tibifront/worklog.md)

---
## 1. Quick start

```bash
cd tibifront
bun install            # or npm install (bun.lock committed; node_modules present)
bun run db:push        # prisma db push --accept-data-loss (SQLite, DATABASE_URL in .env)
bun run db:generate    # regenerate Prisma client
bun run db:migrate     # prisma migrate dev (schema migrations)
# seed the demo dataset (10 specialties, 7 doctors, 4 patients, 19 consultations, records, reviews):
bunx prisma db seed    # see prisma/seed.ts (668 lines) — or bun prisma/seed.ts
bun run dev            # next dev -p 3000 (logs to dev.log)
bun run build && bun run start   # standalone server (.next/standalone/server.js)
bunx eslint .          # lint (eslint.config.mjs + eslint-config-next)
```

Key URLs: app `http://localhost:3000`, Caddy demo proxy `:81` → `:3000` (`tibifront/Caddyfile`). Layout is `ar`/`rtl` with Cairo font, light/dark themes (`next-themes`), toasts (`sonner` + shadcn `toaster`).

Demo accounts (LoginScreen quick-login): patient, doctor, receptionist, admin — passwords live in `prisma/seed.ts`. Registration creates `Patient` or `PendingDoctor` (awaits admin verification).

## 2. Tech stack

| Layer | Choice |
|---|---|
| Framework | Next.js 16.1.1 (App Router, `output: standalone`, Turbopack root pinned, `reactStrictMode: false`, `ignoreBuildErrors: true`) |
| UI | React 19, Tailwind v4 (`@tailwindcss/postcss`), shadcn/ui (full `components/ui/*` set, `components.json`), Radix primitives, `framer-motion`, `lucide-react`, `embla-carousel`, `vaul`, `cmdk`, `sonner` |
| Data/forms | `@tanstack/react-query` + `react-table`, `react-hook-form` + `zod` + `@hookform/resolvers`, `zustand` (+persist) session, `date-fns` + `date-fns-jalali`, `recharts`, `@mdxeditor/editor`, `react-markdown` + syntax highlighter |
| i18n/UX | `next-intl` (Arabic-first UI), `next-auth` (installed; active flow is the custom token store, not next-auth), `input-otp`, `react-day-picker`, `react-resizable-panels`, `@dnd-kit/*` |
| Backend-for-frontend | Next.js Route Handlers (`src/app/api/**`) + Prisma 6 + SQLite (`prisma/schema.prisma`, `db/custom.db`) |
| Tooling | TypeScript 5 (bundler resolution, `@/*` → `src/*`), ESLint 9 + `eslint-config-next`, `Caddyfile`, `bun.lock` |

## 3. Repository layout (`tibifront/`)

```
src/app/
  page.tsx             boot gate: restores medica-token/user-id, GET /api/auth/me, LoginScreen or AppShell
  layout.tsx           ar/rtl, Cairo + Geist Mono, ThemeProvider, Toaster + Sonner
  globals.css          emerald/teal oklch theme, RTL, ECG/urgent-pulse/shimmer/glass utilities
  api/                 ~45 route handlers (auth, doctors, consultations, schedules, records, prescriptions,
                       family, lab-results, health-goals, reviews, referrals, reception/*, admin/*, dashboard, ...)
src/components/
  app-shell.tsx        topbar + role sidebar + DashboardRouter (the app frame)
  auth/login-screen.tsx  split hero + login/register + 4 demo quick-logins
  dashboards/          patient/doctor/reception/admin/pending dashboards + admin-analytics + dashboard-helpers
  doctor/              consultation workspace, prescription editor, queue, schedule, records, patient search
  patient/             home, doctor directory + booking modal, consultations (list/timeline), records timeline,
                       prescriptions, family manager, self-medical-info, vitals charts, lab/health-goals UI
  reception/           home, queue management, big-screen tracker, patient quick card
  admin/               home, verification queue, users, specialties, stats, financial, audit, health, settings
  shared/              StatusBadge, UrgentBadge, RatingStars, MedicalAvatar, StatCard, PageHeader, EmptyState,
                       SectionCard, dialogs (referral/reschedule/review), prescription print, video-call modal
  ui/                  full shadcn set (button, dialog, table, tabs, calendar, chart, sidebar, ...)
src/lib/               api-client (fetch + ar formatters), session-store (zustand persist), types (roles,
                       statuses, STATUS_CONFIG, ROLE_LABELS), auth (token→user, MRN/token gen), token-store,
                       db (Prisma singleton), specialty-icons, utils
src/hooks/             use-mobile, use-toast
prisma/                schema.prisma (SQLite), seed.ts (demo dataset)
examples/websocket/    frontend.tsx + server.ts reference sketches
```

## 4. Auth and session

Login (`POST /api/auth/login`, email-or-phone plus plaintext password — demo-grade, see §9) mints `tok_*` into the in-memory `token-store` (`globalThis`, survives HMR) and returns `{user, token}`. The client keeps `medica-token` + `medica-user-id` in localStorage (`lib/api-client.ts`); every `apiFetch` sends `x-user-id` (userId preferred — resilient to server restarts — else token), falling back to `Authorization: Bearer` or `?token=`. `getUserFromRequest` resolves token→userId (raw cuid accepted when the store was wiped), loads user with patient/doctor profiles, strips `password`. Session lives in zustand `persist` (`medica-session`); `page.tsx` revalidates via `/api/auth/me` on boot. `requireRole(user, roles)` guards handlers. Register (`POST /api/auth/register`) creates `Patient` (with MRN `PT-YYYY-XXXXX` + PatientProfile) or `PendingDoctor` (+ DoctorProfile unverified); Admin/Receptionist accounts come from seed only.
Session lives in zustand `persist` (`medica-session`); `page.tsx` revalidates via `/api/auth/me` on boot. `requireRole(user, roles)` guards handlers. Register creates `Patient` (MRN `PT-YYYY-XXXXX` + profile) or `PendingDoctor` (unverified profile); Admin/Receptionist come from seed only.

## 5. BFF API routes (`src/app/api/**`, JSON, errors as `{error}`)

Auth: `POST auth/login`, `POST auth/register`, `GET auth/me`. Doctors: `GET doctors` (specialty/fee/sort/search), `GET doctors/[id]`, `GET specialties`. Consultations: `GET|POST consultations` (role-scoped; booking assigns dailyQueueNumber/slotNumber, urgent starts Waiting, notifies doctor), `GET consultations/[id]`, `PATCH consultations/[id]/status`. Schedules: `GET|POST doctor-schedule`, `PATCH|DELETE doctor-schedule/[id]`. EHR: `GET|POST medical-records`, `GET|PATCH medical-records/[id]`, `GET|POST medical-attachments`, `GET|POST lab-results`, `GET|POST health-goals` (+`[id]` patch/delete), `GET patient-medical-info`, `GET|PATCH profile/patient-profile` and `profile/doctor-profile`. Prescriptions: `GET|POST prescriptions` (medicationsJson, drafts in localStorage, print view). Social: `GET|POST family-members`, `reviews`, `referrals`. Reception: `GET reception/queue`, `POST reception/register-patient`, `GET reception/patient-card` (privacy-limited).
Reception: `GET reception/queue`, `POST reception/register-patient`, `GET reception/patient-card` (privacy-limited). Admin: `GET admin/analytics`, `GET admin/doctors` (verify), `GET admin/patients`, `GET admin/users`, `GET|POST admin/specialties`, `PATCH admin/specialty-requests/[id]`, `GET admin/financial-report`, `GET admin/audit-log`, `GET admin/system-health`. Misc: `GET dashboard` (role stats), `GET notifications` + `PATCH notifications/[id]`, `GET route` (probe).

## 6. UI tour (Arabic RTL, dark-mode ready)

**AppShell** (`components/app-shell.tsx`): sticky topbar (logo, search, notifications dropdown + unread count, theme toggle, user menu) + role sidebar (`NAV_BY_ROLE`: Patient 8, Doctor 7, Receptionist 3, Admin 9, Pending 1) + mobile Sheet drawer + animated transitions; `DashboardRouter` switches the five dashboards.

**Patient**: home hero (MRN/blood/insurance badges, stats, quick actions, upcoming, allergy alerts); doctor directory (specialty/fee/sort/search cards + ratings); booking modal (date/time/type/urgent/reason/insurance); consultations (upcoming/past/all + list/timeline); records timeline (vitals/diagnosis/allergies + charts + labs + health goals); prescriptions (dosage cards + print); family manager; notifications; profile.

**Doctor**: home (hero, stats, current-patient card with allergy alerts, start/video buttons, queue, clinic info); queue page (urgent-first, call/start/finish); ConsultationWorkspace tabs (diagnosis + vitals/complaint/notes with localStorage autosave, history timeline, prescription editor with med datalists + save/send/print); records browse; recent-patients shortcuts; patient search; weekly schedule (7-day + breaks); clinic profile.

**Reception**: home (hero, 5 stats, upcoming, quick actions); queue management (doctor filter, auto-refresh, queue numbers, urgent pulse, 7-status legend + footer stats, confirm/wait/admit/cancel, privacy-limited patient card); big-screen tracker (large counters); register-patient form.

**Admin**: home + verification queue + users/patients tables + specialties + specialty-requests + stats/analytics (recharts) + financial + audit + system health + settings. PendingDoctor sees a waiting screen.

**Shared**: StatusBadge, UrgentBadge (pulsing), RatingStars, MedicalAvatar, StatCard, PageHeader, EmptyState, SectionCard, skeletons/shimmer, specialty icon mapper, referral/reschedule/review dialogs, video-call modal, prescription print.
**Shared**: StatusBadge, UrgentBadge (pulsing), RatingStars, MedicalAvatar, StatCard, PageHeader, EmptyState, SectionCard, skeletons/shimmer, specialty icon mapper, referral/reschedule/review dialogs, video-call modal, prescription print.

## 7. Data model (Prisma/SQLite)

`User` (email/phone UNIQUE, role Patient/Doctor/PendingDoctor/Receptionist/Admin) → `PatientProfile` (MRN UNIQUE, dob/gender/blood/vitals/insurance/emergency/address) → `FamilyMember[]`; → `DoctorProfile` (license UNIQUE, clinic/fee/experience/bio/education/languages, verified, ratingAvg/Count) → `DoctorWeeklySchedule[]` (dayOfWeek 0-6, start/end, breaks, slotDuration, isWorking) + `DoctorSpecialty[]` → `Specialty` (name/nameAr UNIQUE, icon, color). `Consultation` (patient/doctor, scheduledAt, duration, 7-state status, isUrgent, dailyQueueNumber/slotNumber/estimatedWait, type InPerson/Video/Chat) → `MedicalRecord` (complaint/diagnosis/allergies/meds/conditions/notes, vitals BP/pulse/temp/weight/O2/resp/glucose, attachmentsJson) → `LabResult[]` (category/value/unit/refs/abnormal); → `Prescription` (1:1 per consultation, notes/status, medicationsJson) → `PrescribedMedication`. Plus `Review` (UNIQUE doctor+author), `SpecialtyRequest`, `Notification`. Seed builds the demo graph with times relative to NOW (past/present/future + urgent).

## 8. Config, build, deploy

`next.config.ts` (standalone, Turbopack root pinned, no strict mode), `tsconfig.json` (`@/*`), Tailwind v4 + `components.json` (shadcn), `eslint.config.mjs`. Scripts: `dev` (`next dev -p 3000 | tee dev.log`), `build` (copies static + public into standalone), `start` (bun standalone server), `lint`, `db:push/generate/migrate/reset`. `prisma/` holds schema + seed; `db/custom.db` is local SQLite (`DATABASE_URL` points at it). `Caddyfile` maps `:81` to `:3000` (+ `XTransformPort` passthrough). No CI for `tibifront/` (backend `ci.yml` is Go-only); verification is eslint + manual browser passes in `worklog.md`.

## 9. Prototype caveats and backend relation

Self-contained demo BFF, NOT wired to the Go API: plaintext passwords, in-memory tokens (die on restart except via userId fallback), no JWT/Argon2/refresh rotation, no Kashier/Minio/Mailhog/WS-hub/retention, no module boundaries; roles include `Receptionist` (absent from Go) while Go has votes, suspensions, rate limiting the BFF lacks. Treat `tibifront/` as the UX reference and the Go backend (`docs/backend.md`) as the production API; a future slice would point `apiFetch` at `/api/v1` with JWT and align Prisma roles with migrations `0001..0017`.