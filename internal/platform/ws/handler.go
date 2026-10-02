package ws

import (
	"net/http"

	"github.com/coder/websocket"

	"tibi/internal/platform/auth"
	"tibi/internal/platform/httpx"
)

type Handler struct {
	hub     *hub
	issuer  *auth.TokenIssuer
	origins []string
}

func NewHandler(h Hub, issuer *auth.TokenIssuer, origins []string) *Handler {
	hh, ok := h.(*hub)
	if !ok {
		return &Handler{hub: nil, issuer: issuer, origins: origins}
	}
	return &Handler{hub: hh, issuer: issuer, origins: origins}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.hub == nil {
		httpx.Error(w, r, &httpx.APIError{Status: http.StatusServiceUnavailable, Code: "unavailable", Message: "websocket disabled"})
		return
	}

	token := r.URL.Query().Get("token")
	if token == "" {
		httpx.Error(w, r, httpx.Unauthenticated("missing token"))
		return
	}
	claims, err := h.issuer.Parse(token)
	if err != nil {
		httpx.Error(w, r, httpx.Unauthenticated("invalid token"))
		return
	}

	origins := h.origins
	if len(origins) == 0 {
		origins = []string{"*"}
	}
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: origins,
	})
	if err != nil {
		return
	}

	ctx := r.Context()
	client := &Client{
		h:      h.hub,
		conn:   conn,
		userID: claims.UserID,
		role:   claims.Role,
		send:   make(chan Event, sendBuffer),
		groups: map[string]struct{}{},
	}
	h.hub.register(client)
	h.hub.join(client, userGroup(client.userID))

	go client.writePump(ctx)
	client.readPump(ctx)
}
