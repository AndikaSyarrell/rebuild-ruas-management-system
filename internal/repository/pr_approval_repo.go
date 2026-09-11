package repository

import (
	"context"
	"database/sql"
	"rms-backend/internal/db"

	"rms-backend/internal/models"
)

type PRApprovalRepo struct{ db db.Querier }

func NewPRApprovalRepo(db db.Querier) *PRApprovalRepo { return &PRApprovalRepo{db: db} }

func scanPRApproval(row interface {
	Scan(dest ...interface{}) error
}) (*models.PRApproval, error) {
	var v models.PRApproval
	err := row.Scan(&v.ID, &v.RefAdmin, &v.RefPR, &v.Level, &v.Type, &v.Status,
		&v.Round, &v.Notes, &v.CreateDate, &v.SignatureRef, &v.SignatureFile, &v.AdminName)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

const prApprovalSelect = `
	SELECT a.approval_id, a.approval_ref_admin, a.approval_ref_pr, a.approval_level,
	       a.approval_type, a.approval_status, a.approval_round, a.approval_notes,
	       a.approval_create_date, a.approval_signature_ref, COALESCE(sg.signature_file, ''),
	       COALESCE(ad.admin_name, '')
	FROM T_Pr_Approval a
	LEFT JOIN T_Admin ad ON a.approval_ref_admin = ad.admin_id
	LEFT JOIN T_Admin_Signature sg ON a.approval_signature_ref = sg.signature_id
`

func (r *PRApprovalRepo) GetByID(ctx context.Context, id int) (*models.PRApproval, error) {
	row := r.db.QueryRowContext(ctx, prApprovalSelect+" WHERE a.approval_id = ?", id)
	return scanPRApproval(row)
}

// ListByPR mengembalikan SELURUH baris approval lintas round (riwayat penuh,
// BR-APR-05), terurut round terbaru dulu lalu level ASC.
func (r *PRApprovalRepo) ListByPR(ctx context.Context, prID int) ([]models.PRApproval, error) {
	rows, err := r.db.QueryContext(ctx,
		prApprovalSelect+" WHERE a.approval_ref_pr = ? ORDER BY a.approval_round DESC, a.approval_level ASC", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPRApprovalRows(rows)
}

// ListByPRRound mengembalikan baris approval untuk satu round tertentu saja -
// dipakai service untuk mengevaluasi keputusan round yang sedang berjalan
// (mis. cek BR-APR-03: apakah ada level yang rejected pada round aktif).
func (r *PRApprovalRepo) ListByPRRound(ctx context.Context, prID, round int) ([]models.PRApproval, error) {
	rows, err := r.db.QueryContext(ctx,
		prApprovalSelect+" WHERE a.approval_ref_pr = ? AND a.approval_round = ? ORDER BY a.approval_level ASC",
		prID, round)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPRApprovalRows(rows)
}

func scanPRApprovalRows(rows *sql.Rows) ([]models.PRApproval, error) {
	var out []models.PRApproval
	for rows.Next() {
		v, err := scanPRApproval(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// GetByPRLevelRound mengambil satu baris approval spesifik - dipakai untuk
// cek sequential (BR-APR-02): level 2 baru relevan setelah baris level 1
// pada round yang sama berstatus approved.
func (r *PRApprovalRepo) GetByPRLevelRound(ctx context.Context, prID, level, round int) (*models.PRApproval, error) {
	row := r.db.QueryRowContext(ctx,
		prApprovalSelect+" WHERE a.approval_ref_pr = ? AND a.approval_level = ? AND a.approval_round = ? LIMIT 1",
		prID, level, round)
	return scanPRApproval(row)
}

// GetLatestRound mengembalikan nomor round tertinggi yang sudah ada untuk
// sebuah PR. Mengembalikan 0 jika belum ada baris approval sama sekali -
// pemanggil (service) yang menentukan round pertama = 1.
func (r *PRApprovalRepo) GetLatestRound(ctx context.Context, prID int) (int, error) {
	var round sql.NullInt64
	err := r.db.QueryRowContext(ctx,
		`SELECT MAX(approval_round) FROM T_Pr_Approval WHERE approval_ref_pr = ?`, prID).Scan(&round)
	if err != nil {
		return 0, err
	}
	if !round.Valid {
		return 0, nil
	}
	return int(round.Int64), nil
}

// Create menyisipkan satu baris approval baru. Dipakai baik untuk
// pembentukan baris approval level pertama kali (round=1) maupun saat
// resubmit setelah revisi (round bertambah, BR-APR-05) - pemanggil (service)
// yang menentukan nilai round-nya, repo tidak menghitung sendiri.
func (r *PRApprovalRepo) Create(ctx context.Context, refPR int, refAdmin string, level int, approvalType string, round int) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Approval (approval_ref_admin, approval_ref_pr, approval_level, approval_type,
		 approval_status, approval_round, approval_create_date)
		VALUES (?, ?, ?, ?, 'pending', ?, NOW())`,
		refAdmin, refPR, level, approvalType, round)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRApprovalRepo) UpdateStatus(ctx context.Context, approvalID int, status, notes string, signatureRef *int) error {
	var notesArg interface{}
	if notes != "" {
		notesArg = notes
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Pr_Approval SET approval_status = ?, approval_notes = ?, approval_signature_ref = ? WHERE approval_id = ?`,
		status, notesArg, signatureRef, approvalID)
	return err
}