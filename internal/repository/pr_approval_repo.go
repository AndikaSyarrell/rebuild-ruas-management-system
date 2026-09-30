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
		&v.Round, &v.Notes, &v.CreateDate, &v.SignatureRef, &v.SignatureFile, &v.AdminName, &v.DecidedDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

const prApprovalSelect = `
	SELECT a.approval_id, a.approval_ref_admin, a.approval_ref_pr, a.approval_level,
	       a.approval_type, a.approval_status, a.approval_round, a.approval_notes,
	       a.approval_create_date, a.approval_signature_ref, COALESCE(sg.signature_file, ''),
	       COALESCE(ad.admin_name, ''), a.approval_decided_date
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

type ApprovalSeed struct {
	Level    int
	Type     string
	RefAdmin *string // nil = level general (director/finance)
}

func (r *PRApprovalRepo) CreateRound(ctx context.Context, prID, round int, seeds []ApprovalSeed) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO T_Pr_Approval (approval_ref_admin, approval_ref_pr, approval_level, approval_type,
		 approval_status, approval_round, approval_create_date)
		VALUES (?, ?, ?, ?, 'pending', ?, NOW())`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, sd := range seeds {
		if _, err := stmt.ExecContext(ctx, sd.RefAdmin, prID, sd.Level, sd.Type, round); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *PRApprovalRepo) Decide(ctx context.Context, approvalID int, adminID, status, notes string, signatureRef *int) (bool, error) {
	var notesArg interface{}
	if notes != "" {
		notesArg = notes
	}
	res, err := r.db.ExecContext(ctx, `
		UPDATE T_Pr_Approval
		SET approval_status = ?, approval_ref_admin = ?, approval_notes = ?, approval_signature_ref = ?, approval_decided_date = NOW()
		WHERE approval_id = ? AND approval_status = 'pending'`,
		status, adminID, notesArg, signatureRef, approvalID)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

type DecidedApproval struct {
	models.PRApproval
	RfpNo           string
	DescriptionItem string
	RequestedAmount float64
	PRStatus        string
	RequesterName   string
}

func (r *PRApprovalRepo) ListDecidedByApprover(ctx context.Context, adminID, decision string, page, perPage int) ([]DecidedApproval, int, error) {
	conds := "a.approval_ref_admin = ? AND a.approval_status <> 'pending'"
	args := []interface{}{adminID}
	if decision != "" {
		conds += " AND a.approval_status = ?"
		args = append(args, decision)
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Pr_Approval a WHERE `+conds, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]interface{}{}, args...), perPage, (page-1)*perPage)
	rows, err := r.db.QueryContext(ctx, `
		SELECT a.approval_id, a.approval_ref_admin, a.approval_ref_pr, a.approval_level, a.approval_type,
		       a.approval_status, a.approval_round, a.approval_notes, a.approval_create_date,
		       a.approval_signature_ref, COALESCE(sg.signature_file, ''), COALESCE(ad.admin_name, ''),
		       a.approval_decided_date,
		       pr.pr_rfp_no, pr.pr_description_item, pr.pr_requested_amount, pr.pr_status,
		       COALESCE(rq.admin_name, '')
		FROM T_Pr_Approval a
		JOIN T_Purchase_Request pr ON pr.pr_id = a.approval_ref_pr
		LEFT JOIN T_Admin ad ON a.approval_ref_admin = ad.admin_id
		LEFT JOIN T_Admin rq ON pr.pr_ref_admin = rq.admin_id
		LEFT JOIN T_Admin_Signature sg ON a.approval_signature_ref = sg.signature_id
		WHERE `+conds+`
		ORDER BY COALESCE(a.approval_decided_date, a.approval_create_date) DESC, a.approval_id DESC
		LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []DecidedApproval
	for rows.Next() {
		var v DecidedApproval
		if err := rows.Scan(&v.ID, &v.RefAdmin, &v.RefPR, &v.Level, &v.Type, &v.Status, &v.Round,
			&v.Notes, &v.CreateDate, &v.SignatureRef, &v.SignatureFile, &v.AdminName, &v.DecidedDate,
			&v.RfpNo, &v.DescriptionItem, &v.RequestedAmount, &v.PRStatus, &v.RequesterName); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}