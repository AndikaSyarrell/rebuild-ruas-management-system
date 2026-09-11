package repository

import (
	"context"

	"rms-backend/internal/db"
)

// PRCounterRepo mengelola nomor urut rfp_no per-admin secara atomik.
// Menggunakan idiom MySQL `INSERT ... ON DUPLICATE KEY UPDATE ... LAST_INSERT_ID(expr)`
// supaya increment aman terhadap concurrent request tanpa perlu SELECT ... FOR UPDATE terpisah.
type PRCounterRepo struct{ db db.Querier }

func NewPRCounterRepo(db db.Querier) *PRCounterRepo { return &PRCounterRepo{db: db} }

// NextSequence mengembalikan nomor urut BERIKUTNYA untuk admin tsb (dimulai dari 1).
func (r *PRCounterRepo) NextSequence(ctx context.Context, adminID string) (int, error) {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Admin_Counter (counter_ref_admin, counter_last_seq)
		VALUES (?, 1)
		ON DUPLICATE KEY UPDATE counter_last_seq = LAST_INSERT_ID(counter_last_seq + 1)`,
		adminID)
	if err != nil {
		return 0, err
	}

	var seq int
	if err := r.db.QueryRowContext(ctx, `SELECT LAST_INSERT_ID()`).Scan(&seq); err != nil {
		return 0, err
	}
	return seq, nil
}