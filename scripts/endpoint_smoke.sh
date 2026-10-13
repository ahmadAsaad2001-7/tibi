#!/bin/bash
# Endpoint smoke test — slice 18 verification.
BASE="http://localhost:8080"
PASS=0; FAIL=0
STAMP=$(date +%s)
PATIENT_EMAIL="epatient${STAMP}@example.com"
PATIENT_PASS="correct-horse-battery-staple"
PATIENT_PHONE="+20100$(date +%H%M%S)"
ADMIN_EMAIL="admin@tibi.test"
ADMIN_PASS='AdminPass123!'

check() { # check <label> <expected_code> <actual_code> [body]
  local label="$1" want="$2" got="$3" body="${4:-}"
  if [ "$got" = "$want" ]; then
    PASS=$((PASS+1)); printf 'PASS  %-58s %s\n' "$label" "$got"
  else
    FAIL=$((FAIL+1)); printf 'FAIL  %-58s want=%s got=%s body=%s\n' "$label" "$want" "$got" "$(echo "$body" | head -c 200)"
  fi
}

req() { # req METHOD PATH [TOKEN] [DATA] -> sets CODE and BODY
  local method="$1" path="$2" token="${3:-}" data="${4:-}"
  local args=(-s -m 10 -o /tmp/ep_body -w '%{http_code}' -X "$method" "$BASE$path")
  [ -n "$token" ] && args+=(-H "Authorization: Bearer $token")
  [ -n "$data" ] && args+=(-H 'Content-Type: application/json' -d "$data")
  CODE=$(curl "${args[@]}")
  BODY=$(cat /tmp/ep_body)
}

echo "=== 1. Health & observability ==="
req GET /healthz;  check "GET /healthz" 200 "$CODE" "$BODY"
req GET /readyz;   check "GET /readyz (db up)" 200 "$CODE" "$BODY"
req GET /health;   check "GET /health" 200 "$CODE" "$BODY"
req GET /metrics;  check "GET /metrics (prometheus)" 200 "$CODE" "$BODY"
req GET /docs;     check "GET /docs (scalar)" 200 "$CODE" "$BODY"
req GET /openapi.yaml; check "GET /openapi.yaml" 200 "$CODE" "$BODY"

echo "=== 2. Public reads ==="
req GET "/api/v1/specialties"; check "GET /api/v1/specialties" 200 "$CODE" "$BODY"
SPEC_COUNT=$(echo "$BODY" | jq -r '(.items // .) | length' 2>/dev/null || echo "?")
echo "      specialties returned: $SPEC_COUNT"
req GET "/api/v1/doctors"; check "GET /api/v1/doctors" 200 "$CODE" "$BODY"
req GET "/api/v1/doctor-posts/"; check "GET /api/v1/doctor-posts/" 200 "$CODE" "$BODY"

echo "=== 3. Auth: register / login / me / refresh / logout ==="
req POST /api/v1/auth/register "" "{\"email\":\"$PATIENT_EMAIL\",\"password\":\"$PATIENT_PASS\",\"role\":\"Patient\",\"full_name\":\"Endpoint Tester\",\"phone_number\":\"$PATIENT_PHONE\"}"
check "POST /api/v1/auth/register (new patient)" 201 "$CODE" "$BODY"
req POST /api/v1/auth/register "" "{\"email\":\"$PATIENT_EMAIL\",\"password\":\"$PATIENT_PASS\",\"role\":\"Patient\",\"full_name\":\"Endpoint Tester\",\"phone_number\":\"$PATIENT_PHONE\"}"
check "POST /api/v1/auth/register (duplicate email)" 409 "$CODE" "$BODY"
req POST /api/v1/auth/register "" '{"email":"bad-email","password":"x","role":"Hacker","full_name":"x","phone_number":"nope"}'
check "POST /api/v1/auth/register (invalid payload)" 400 "$CODE" "$BODY"

req POST /api/v1/auth/login "" "{\"email\":\"$PATIENT_EMAIL\",\"password\":\"$PATIENT_PASS\"}"
check "POST /api/v1/auth/login (ok)" 200 "$CODE" "$BODY"
PAT_TOKEN=$(echo "$BODY" | jq -r '.access_token // .accessToken // empty' 2>/dev/null)
REFRESH=$(echo "$BODY" | jq -r '.refresh_token // .refreshToken // empty' 2>/dev/null)
[ -n "$PAT_TOKEN" ] && echo "      patient token acquired" || echo "      WARNING: no patient token (body: $(echo "$BODY"|head -c 150))"

req POST /api/v1/auth/login "" "{\"email\":\"$PATIENT_EMAIL\",\"password\":\"wrong-password-123\"}"
check "POST /api/v1/auth/login (wrong password)" 401 "$CODE" "$BODY"
req POST /api/v1/auth/login "" '{"email":"nobody@example.com","password":"wrong-password-123"}'
check "POST /api/v1/auth/login (missing user)" 401 "$CODE" "$BODY"

req GET /api/v1/auth/me "$PAT_TOKEN"
check "GET /api/v1/auth/me (with token)" 200 "$CODE" "$BODY"
req GET /api/v1/auth/me ""
check "GET /api/v1/auth/me (no token)" 401 "$CODE" "$BODY"
req GET /api/v1/auth/me "not-a-real-token"
check "GET /api/v1/auth/me (garbage token)" 401 "$CODE" "$BODY"

if [ -n "$REFRESH" ]; then
  req POST /api/v1/auth/refresh "" "{\"refresh_token\":\"$REFRESH\"}"
  check "POST /api/v1/auth/refresh" 200 "$CODE" "$BODY"
  NEW_TOKEN=$(echo "$BODY" | jq -r '.access_token // .accessToken // empty' 2>/dev/null)
  [ -n "$NEW_TOKEN" ] && PAT_TOKEN="$NEW_TOKEN"
else
  echo "SKIP  refresh (no refresh_token in login response)"
fi

echo "=== 4. Admin (seeded via --seed) ==="
req POST /api/v1/auth/login "" "{\"email\":\"$ADMIN_EMAIL\",\"password\":\"$ADMIN_PASS\"}"
check "POST /api/v1/auth/login (seeded admin)" 200 "$CODE" "$BODY"
ADM_TOKEN=$(echo "$BODY" | jq -r '.access_token // .accessToken // empty' 2>/dev/null)

req GET /api/v1/admin/votes "$ADM_TOKEN"
check "GET /api/v1/admin/votes (admin)" 200 "$CODE" "$BODY"
req GET /api/v1/admin/doctors/pending "$ADM_TOKEN"
check "GET /api/v1/admin/doctors/pending (admin)" 200 "$CODE" "$BODY"
req GET /api/v1/admin/votes "$PAT_TOKEN"
check "GET /api/v1/admin/votes (patient -> 403)" 403 "$CODE" "$BODY"
req GET /api/v1/admin/votes ""
check "GET /api/v1/admin/votes (no token -> 401)" 401 "$CODE" "$BODY"

echo "=== 5. Role boundaries ==="
req POST /api/v1/consultations/ "$ADM_TOKEN" '{}'
check "POST /api/v1/consultations/ (admin role -> 403)" 403 "$CODE" "$BODY"
req POST /api/v1/doctor-posts/ "$PAT_TOKEN" '{}'
check "POST /api/v1/doctor-posts/ (patient role -> 403)" 403 "$CODE" "$BODY"
req PUT /api/v1/consultations/1/medical-record "$PAT_TOKEN" '{}'
check "PUT medical-record (patient role -> 403)" 403 "$CODE" "$BODY"
req GET /api/v1/doctor-schedule/ "$PAT_TOKEN"
check "GET /api/v1/doctor-schedule/ (patient role -> 403)" 403 "$CODE" "$BODY"

echo "=== 6. Notifications & consultations (patient) ==="
req GET /api/v1/notifications "$PAT_TOKEN"
check "GET /api/v1/notifications" 200 "$CODE" "$BODY"
req GET /api/v1/notifications/unread-count "$PAT_TOKEN"
check "GET /api/v1/notifications/unread-count" 200 "$CODE" "$BODY"
req GET /api/v1/consultations/ "$PAT_TOKEN"
check "GET /api/v1/consultations/ (patient)" 200 "$CODE" "$BODY"
req GET /api/v1/consultations/999999 "$PAT_TOKEN"
check "GET /api/v1/consultations/999999 (missing -> 404)" 404 "$CODE" "$BODY"

echo "=== 7. Password reset & email verification ==="
req POST /api/v1/auth/forgot-password "" "{\"email\":\"$PATIENT_EMAIL\"}"
check "POST /api/v1/auth/forgot-password" 200 "$CODE" "$BODY"
req POST /api/v1/auth/forgot-password "" '{"email":"nobody@example.com"}'
check "POST /api/v1/auth/forgot-password (unknown -> generic)" 200 "$CODE" "$BODY"
req POST /api/v1/auth/verify-email/request "$PAT_TOKEN" ""
check "POST /api/v1/auth/verify-email/request" 200 "$CODE" "$BODY"

echo "=== 8. Kashier webhook (signature enforced) ==="
req POST /api/v1/auth/webhooks/kashier "" '{"event":"payment.paid"}'
check "POST webhook/kashier (no signature)" 401 "$CODE" "$BODY"

echo "=== 9. Logout ==="
req POST /api/v1/auth/logout "$PAT_TOKEN" "{\"refresh_token\":\"${REFRESH:-}\"}"
check "POST /api/v1/auth/logout" 204 "$CODE" "$BODY"
req POST /api/v1/auth/refresh "" "{\"refresh_token\":\"${REFRESH:-}\"}"
check "POST /api/v1/auth/refresh (revoked -> 401)" 401 "$CODE" "$BODY"

echo ""
echo "================ RESULT: PASS=$PASS FAIL=$FAIL ================"
[ "$FAIL" -eq 0 ]