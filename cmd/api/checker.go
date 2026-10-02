package main

import (
	"context"

	consultationswschecker "tibi/internal/consultations/wschecker"
	queuewschecker "tibi/internal/queue/wschecker"
)

type combinedChecker struct {
	c *consultationswschecker.Service
	q *queuewschecker.Service
}

func (c combinedChecker) IsConsultationMember(ctx context.Context, userID, consultationID int64) (bool, error) {
	return c.c.IsConsultationMember(ctx, userID, consultationID)
}

func (c combinedChecker) IsQueueMember(ctx context.Context, userID, clinicSessionID int64) (bool, error) {
	return c.q.IsQueueMember(ctx, userID, clinicSessionID)
}

func (c combinedChecker) OtherConsultationMember(ctx context.Context, userID, consultationID int64) (int64, error) {
	return c.c.OtherConsultationMember(ctx, userID, consultationID)
}
