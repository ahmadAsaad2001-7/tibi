package proposevote

import (
	"context"
	"testing"

	"tibi/internal/admin/adminvote"
	"tibi/internal/platform/httpx"
)

func TestExecuteRejectsBeforeDatabase(t *testing.T) {
	svc := &Service{}

	_, err := svc.Execute(context.Background(), Input{Action: "Nope", AdminUserID: 1, TargetUserID: 2})
	assertCode(t, err, "validation_failed")

	_, err = svc.Execute(context.Background(), Input{Action: adminvote.ActionVerifyDoctor, AdminUserID: 5, TargetUserID: 5})
	assertCode(t, err, "validation_failed")
}

func TestCheckTargetEligibility(t *testing.T) {
	svc := &Service{}
	cases := []struct {
		action adminvote.ActionType
		role   string
		code   string
	}{
		{adminvote.ActionVerifyDoctor, "PendingDoctor", ""},
		{adminvote.ActionVerifyDoctor, "Doctor", ""},
		{adminvote.ActionVerifyDoctor, "Patient", "unprocessable"},
		{adminvote.ActionUnverifyDoctor, "Doctor", ""},
		{adminvote.ActionUnverifyDoctor, "PendingDoctor", "unprocessable"},
		{adminvote.ActionBanUser, "Patient", "unprocessable"},
	}
	for _, tc := range cases {
		err := svc.checkTargetEligibility(tc.action, tc.role)
		if tc.code == "" {
			if err != nil {
				t.Fatalf("%s/%s: unexpected err %v", tc.action, tc.role, err)
			}
			continue
		}
		assertCode(t, err, tc.code)
	}
}

func assertCode(t *testing.T, err error, code string) {
	t.Helper()
	he := httpx.As(err)
	if he.Code != code {
		t.Fatalf("code = %s, want %s (err=%v)", he.Code, code, err)
	}
}
