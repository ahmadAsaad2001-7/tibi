package ws

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
)

const (
	sendBuffer   = 64
	writeTimeout = 10 * time.Second
	pingInterval = 30 * time.Second
)

type Client struct {
	h      *hub
	conn   *websocket.Conn
	userID int64
	role   string
	send   chan Event
	groups map[string]struct{}
}

func (c *Client) enqueue(e Event) {
	select {
	case c.send <- e:
	default:
		// Buffer full: drop the client. It will reconnect and get fresh state.
		if c.conn != nil {
			_ = c.conn.Close(websocket.StatusPolicyViolation, "send buffer full")
		}
	}
}

func (c *Client) readPump(ctx context.Context) {
	defer func() {
		c.h.unregister(c)
		if c.conn != nil {
			_ = c.conn.CloseNow()
		}
	}()

	c.conn.SetReadLimit(32 * 1024)

	for {
		_, data, err := c.conn.Read(ctx)
		if err != nil {
			return
		}
		var msg ClientMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			c.enqueue(Event{Type: "Error", Payload: map[string]string{"code": "malformed"}})
			continue
		}
		c.handle(ctx, msg)
	}
}

func (c *Client) handle(ctx context.Context, msg ClientMessage) {
	if c.h.checker == nil {
		c.enqueue(Event{Type: "Error", Payload: map[string]string{"code": "not_member"}})
		return
	}
	switch msg.Action {
	case ActionJoinConsultation:
		ok, err := c.h.checker.IsConsultationMember(ctx, c.userID, msg.ConsultationID)
		if err != nil || !ok {
			c.enqueue(Event{Type: "Error", Payload: map[string]string{"code": "not_member"}})
			return
		}
		c.h.join(c, consultationGroup(msg.ConsultationID))

	case ActionLeaveConsultation:
		c.h.leave(c, consultationGroup(msg.ConsultationID))

	case ActionJoinQueue:
		ok, err := c.h.checker.IsQueueMember(ctx, c.userID, msg.ClinicSessionID)
		if err != nil || !ok {
			c.enqueue(Event{Type: "Error", Payload: map[string]string{"code": "not_member"}})
			return
		}
		c.h.join(c, queueGroup(msg.ClinicSessionID))

	case ActionLeaveQueue:
		c.h.leave(c, queueGroup(msg.ClinicSessionID))

	case ActionWebRTCSignal:
		other, err := c.h.checker.OtherConsultationMember(ctx, c.userID, msg.ConsultationID)
		if err != nil || other == 0 {
			c.enqueue(Event{Type: "Error", Payload: map[string]string{"code": "no_peer"}})
			return
		}
		c.h.SendToUser(other, Event{
			Type: "WebRTCSignal",
			Payload: map[string]any{
				"from_user_id":    c.userID,
				"consultation_id": msg.ConsultationID,
				"signal":          msg.Signal,
			},
		})

	case ActionPing:
		// Liveness is handled by server-initiated pings in writePump.
	}
}

func (c *Client) writePump(ctx context.Context) {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		if c.conn != nil {
			_ = c.conn.CloseNow()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-c.send:
			if !ok {
				return
			}
			writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
			b, err := json.Marshal(e)
			if err != nil {
				cancel()
				return
			}
			err = c.conn.Write(writeCtx, websocket.MessageText, b)
			cancel()
			if err != nil {
				return
			}
		case <-ticker.C:
			pctx, cancel := context.WithTimeout(ctx, writeTimeout)
			err := c.conn.Ping(pctx)
			cancel()
			if err != nil {
				return
			}
		}
	}
}
