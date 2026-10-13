package retention

import (
	"context"
	"log/slog"
	"time"

	"tibi/internal/platform/database"
)

type Config struct {
	ClinicalYears int
	BatchSize     int
	Interval      time.Duration
}

type Worker struct {
	db  *database.DB
	cfg Config
	log *slog.Logger
}

func NewWorker(db *database.DB, cfg Config, log *slog.Logger) *Worker {
	return &Worker{db: db, cfg: cfg, log: log}
}

// RunOnce performs two passes:
//  1. Soft-delete clinical rows whose consultation completed more than
//     ClinicalYears ago.
//  2. Hard-delete clinical rows soft-deleted more than 1 year ago.
//
// Both passes run in batches. SDK/SD58 placeholder policy — the 7-year
// value is a legal decision, configurable via RETENTION_CLINICAL_YEARS.
func (w *Worker) RunOnce(ctx context.Context) error {
	softCutoff := time.Now().AddDate(-w.cfg.ClinicalYears, 0, 0)
	hardCutoff := time.Now().AddDate(-1, 0, 0)

	// Postgres does not support LIMIT directly on UPDATE. The batched form
	// limits the subquery and updates exactly those rows (R32 corrected).
	soft := []struct {
		name string
		sql  string
	}{
		{
			name: "medical_records",
			sql: `UPDATE clinical_medical_records
			      SET deleted_at = now()
			      WHERE id IN (
			          SELECT r.id
			          FROM clinical_medical_records r
			          JOIN consultations_consultations c ON c.id = r.consultation_id
			          WHERE r.deleted_at IS NULL
			            AND c.status = 'Completed'
			            AND c.updated_at < $1
			          LIMIT $2
			      )`,
		},
		{
			name: "prescriptions",
			sql: `UPDATE clinical_prescriptions
			      SET deleted_at = now()
			      WHERE id IN (
			          SELECT p.id
			          FROM clinical_prescriptions p
			          JOIN consultations_consultations c ON c.id = p.consultation_id
			          WHERE p.deleted_at IS NULL
			            AND c.status = 'Completed'
			            AND c.updated_at < $1
			          LIMIT $2
			      )`,
		},
	}
	for _, q := range soft {
		n, err := w.db.Pool.Exec(ctx, q.sql, softCutoff, w.cfg.BatchSize)
		if err != nil {
			return err
		}
		if n.RowsAffected() > 0 {
			w.log.Info("retention soft-delete", "table", q.name, "rows", n.RowsAffected())
		}
	}

	// Hard delete: children first (FK), then parents.
	hard := []string{
		`DELETE FROM clinical_medical_attachments
		 WHERE medical_record_id IN (
		     SELECT id FROM clinical_medical_records
		     WHERE deleted_at IS NOT NULL AND deleted_at < $1
		     LIMIT $2)`,
		`DELETE FROM clinical_prescribed_medications
		 WHERE prescription_id IN (
		     SELECT id FROM clinical_prescriptions
		     WHERE deleted_at IS NOT NULL AND deleted_at < $1
		     LIMIT $2)`,
		`DELETE FROM clinical_medical_records
		 WHERE id IN (SELECT id FROM clinical_medical_records
		              WHERE deleted_at IS NOT NULL AND deleted_at < $1
		              LIMIT $2)`,
		`DELETE FROM clinical_prescriptions
		 WHERE id IN (SELECT id FROM clinical_prescriptions
		              WHERE deleted_at IS NOT NULL AND deleted_at < $1
		              LIMIT $2)`,
	}
	for i, sql := range hard {
		n, err := w.db.Pool.Exec(ctx, sql, hardCutoff, w.cfg.BatchSize)
		if err != nil {
			return err
		}
		if n.RowsAffected() > 0 {
			w.log.Info("retention hard-delete", "step", i, "rows", n.RowsAffected())
		}
	}
	return nil
}
