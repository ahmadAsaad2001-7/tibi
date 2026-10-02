package notify

import (
	"context"
	"log/slog"

	"tibi/internal/platform/ws"
	"tibi/internal/queue/getsessionqueue"
)

func Fanout(ctx context.Context, hub ws.Hub, snap *getsessionqueue.Service, sessionID, patientUserID int64, events ...ws.Event) {
	if hub == nil {
		return
	}
	for _, e := range events {
		hub.SendToQueue(sessionID, e)
		if patientUserID != 0 {
			hub.SendToUser(patientUserID, e)
		}
	}
	if snap == nil {
		return
	}
	out, err := snap.Snapshot(ctx, sessionID)
	if err != nil {
		slog.Warn("queue_snapshot_failed", "session_id", sessionID, "err", err)
		return
	}
	hub.SendToQueue(sessionID, ws.Event{Type: "QueueSnapshot", Payload: out})
}
