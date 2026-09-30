package auth

import (
	"net/http"
	"strings"

	"tibi/internal/platform/httpx"
)

func RequireAuth(issuer *TokenIssuer) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			token := strings.TrimPrefix(header, "Bearer ")
			if token == "" || token == header {
				httpx.Error(w, r, httpx.Unauthenticated("missing bearer token"))
				return
			}
			claims, err := issuer.Parse(token)
			if err != nil {
				httpx.Error(w, r, httpx.Unauthenticated("invalid or expired token"))
				return
			}
			ctx := WithUser(r.Context(), AuthUser{UserID: claims.UserID, Role: claims.Role})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole returns 403 when the caller's role is not in allowed.
// Use with constant strings, never literals in handlers.
func RequireRole(allowed ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, ok := UserFromContext(r.Context())
			if !ok {
				httpx.Error(w, r, httpx.Unauthenticated("not authenticated"))
				return
			}
			for _, a := range allowed {
				if u.Role == a {
					next.ServeHTTP(w, r)
					return
				}
			}
			httpx.Error(w, r, httpx.Forbidden("role not permitted"))
		})
	}
}
