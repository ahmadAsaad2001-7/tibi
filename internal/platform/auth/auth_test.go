package auth

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestPasswordHashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher()
	hash, err := hasher.Hash("correct-horse-battery")
	if err != nil {
		t.Fatal(err)
	}
	ok, err := hasher.Verify("correct-horse-battery", hash)
	if err != nil || !ok {
		t.Fatalf("verify matching password: ok=%v err=%v", ok, err)
	}
	ok, err = hasher.Verify("wrong-password", hash)
	if err != nil || ok {
		t.Fatalf("verify wrong password: ok=%v err=%v", ok, err)
	}
}

func TestIssueAndParseAccessToken(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", 15*time.Minute)
	if issuer.ExpiresIn() != 900 {
		t.Fatalf("expires_in = %d", issuer.ExpiresIn())
	}
	token, err := issuer.IssueAccess(42, "PendingDoctor")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := issuer.Parse(token)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.Role != "PendingDoctor" {
		t.Fatalf("claims = %+v", claims)
	}

	if _, err := NewTokenIssuer("other-secret", time.Minute).Parse(token); err == nil {
		t.Fatal("token signed with a different secret was accepted")
	}
	expired, err := NewTokenIssuer("test-secret", -time.Minute).IssueAccess(1, "Patient")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuer.Parse(expired); err == nil {
		t.Fatal("expired token was accepted")
	}
	if _, err := issuer.Parse("not-a-jwt"); err == nil {
		t.Fatal("garbage token was accepted")
	}
}

func TestRequireAuthAndRole(t *testing.T) {
	issuer := NewTokenIssuer("test-secret", time.Minute)
	token, err := issuer.IssueAccess(7, "Admin")
	if err != nil {
		t.Fatal(err)
	}
	protected := RequireAuth(issuer)(RequireRole("Admin")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, ok := UserFromContext(r.Context())
		if !ok || u.UserID != 7 || u.Role != "Admin" {
			t.Fatalf("user = %+v ok=%v", u, ok)
		}
		w.WriteHeader(http.StatusNoContent)
	})))

	okReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/votes", nil)
	okReq.Header.Set("Authorization", "Bearer "+token)
	okRec := httptest.NewRecorder()
	protected.ServeHTTP(okRec, okReq)
	if okRec.Code != http.StatusNoContent {
		t.Fatalf("authorized status = %d body=%s", okRec.Code, okRec.Body.String())
	}

	missing := httptest.NewRecorder()
	protected.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/api/v1/admin/votes", nil))
	assertCode(t, missing, http.StatusUnauthorized, "unauthenticated")

	bad := httptest.NewRecorder()
	badReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/votes", nil)
	badReq.Header.Set("Authorization", "Bearer nope")
	protected.ServeHTTP(bad, badReq)
	assertCode(t, bad, http.StatusUnauthorized, "unauthenticated")

	patientToken, err := issuer.IssueAccess(8, "Patient")
	if err != nil {
		t.Fatal(err)
	}
	forbidden := httptest.NewRecorder()
	forbiddenReq := httptest.NewRequest(http.MethodGet, "/api/v1/admin/votes", nil)
	forbiddenReq.Header.Set("Authorization", "Bearer "+patientToken)
	protected.ServeHTTP(forbidden, forbiddenReq)
	assertCode(t, forbidden, http.StatusForbidden, "forbidden")

	roleOnly := RequireRole("Doctor")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	anon := httptest.NewRecorder()
	roleOnly.ServeHTTP(anon, httptest.NewRequest(http.MethodGet, "/profile/doctor", nil))
	assertCode(t, anon, http.StatusUnauthorized, "unauthenticated")
}

func assertCode(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()
	if rec.Code != status {
		t.Fatalf("status = %d, want %d", rec.Code, status)
	}
	var body struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error.Code != code {
		t.Fatalf("code = %s, want %s", body.Error.Code, code)
	}
}
