package auth

import "context"

type ctxKey struct{}

type AuthUser struct {
	UserID int64
	Role   string
}

func WithUser(ctx context.Context, u AuthUser) context.Context {
	return context.WithValue(ctx, ctxKey{}, u)
}

func UserFromContext(ctx context.Context) (AuthUser, bool) {
	u, ok := ctx.Value(ctxKey{}).(AuthUser)
	return u, ok
}
