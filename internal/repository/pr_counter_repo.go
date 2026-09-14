package repository

import (
	"context"

	"rms-backend/internal/db"
)

type PRCounterRepo struct{ db db.Querier }

func NewPRCounterRepo(db db.Querier) *PRCounterRepo { return &PRCounterRepo{db: db} }

func (r *PRCounterRepo) NextSequence(ctx context.Context, key string) (int, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Admin_Counter (counter_key, counter_last_seq)
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE counter_last_seq = LAST_INSERT_ID(counter_last_seq + 1)`,
		key)
	if err != nil {
		return 0, err
	}

	var seq int
	if err := r.db.QueryRowContext(ctx, `SELECT LAST_INSERT_ID()`).Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}