package repository

import (
	"context"
	"database/sql"
	"errors"

	"github.com/go-sql-driver/mysql"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
)

var (
	ErrCancelRequestAlreadyPending = errors.New("sudah ada request pembatalan yang menunggu review untuk purchase request ini")
	ErrCancelRequestNotPending     = errors.New("request pembatalan ini sudah direview")
	ErrPRNotCancellable            = errors.New("status purchase request sudah tidak memungkinkan pembatalan")
	ErrPRHasPaidPayment            = errors.New("purchase request memiliki pembayaran yang sudah lunas, tidak dapat dibatalkan")
)

type PRCancelRepo struct{ db db.Querier }

func NewPRCancelRepo(db db.Querier) *PRCancelRepo { return &PRCancelRepo{db: db} }

type PRCancelRequestView struct {
	models.PRCancelRequest
	RfpNo           string
	DescriptionItem string
	RequestedAmount float64
	PRStatus        string
}

const prCancelSelect = `
	SELECT c.cancel_id, c.cancel_ref_pr, c.cancel_ref_admin, c.cancel_reason, c.cancel_document_name,
	       c.cancel_document_path, c.cancel_status, c.cancel_ref_admin_review, c.cancel_review_notes,
	       c.cancel_review_date, c.cancel_create_date,
	       COALESCE(rq.admin_name, ''), COALESCE(rv.admin_name, ''),
	       pr.pr_rfp_no, pr.pr_description_item, pr.pr_requested_amount, pr.pr_status
	FROM T_Pr_Cancel_Request c
	JOIN T_Purchase_Request pr ON pr.pr_id = c.cancel_ref_pr
	LEFT JOIN T_Admin rq ON c.cancel_ref_admin = rq.admin_id
	LEFT JOIN T_Admin rv ON c.cancel_ref_admin_review = rv.admin_id
`

func scanPRCancel(row interface {
	Scan(dest ...interface{}) error
}) (*PRCancelRequestView, error) {
	var v PRCancelRequestView
	err := row.Scan(&v.ID, &v.RefPR, &v.RefAdmin, &v.Reason, &v.DocumentName, &v.DocumentPath, &v.Status,
		&v.RefAdminReview, &v.ReviewNotes, &v.ReviewDate, &v.CreateDate,
		&v.RequesterName, &v.ReviewerName,
		&v.RfpNo, &v.DescriptionItem, &v.RequestedAmount, &v.PRStatus)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRCancelRepo) GetByID(ctx context.Context, id int) (*PRCancelRequestView, error) {
	return scanPRCancel(r.db.QueryRowContext(ctx, prCancelSelect+" WHERE c.cancel_id = ?", id))
}

func (r *PRCancelRepo) HasPendingByPR(ctx context.Context, prID int) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM T_Pr_Cancel_Request WHERE cancel_ref_pr = ? AND cancel_status = 'pending' LIMIT 1`,
		prID).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *PRCancelRepo) Insert(ctx context.Context, prID int, adminID, reason, docName, docPath string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Cancel_Request (cancel_ref_pr, cancel_ref_admin, cancel_reason,
		 cancel_document_name, cancel_document_path, cancel_status, cancel_create_date)
		VALUES (?, ?, ?, ?, ?, 'pending', NOW())`, prID, adminID, reason, docName, docPath)
	if err != nil {
		var me *mysql.MySQLError
		if errors.As(err, &me) && me.Number == 1062 { // uq_cancel_one_pending
			return 0, ErrCancelRequestAlreadyPending
		}
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRCancelRepo) ListByPR(ctx context.Context, prID int) ([]PRCancelRequestView, error) {
	rows, err := r.db.QueryContext(ctx,
		prCancelSelect+" WHERE c.cancel_ref_pr = ? ORDER BY c.cancel_create_date DESC", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PRCancelRequestView
	for rows.Next() {
		v, err := scanPRCancel(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// ListPaged: antrean review finance. Pending selalu di atas (terlama dulu), sisanya terbaru dulu.
func (r *PRCancelRepo) ListPaged(ctx context.Context, status string, page, perPage int) ([]PRCancelRequestView, int, error) {
	where := ""
	var args []interface{}
	if status != "" {
		where = " WHERE c.cancel_status = ?"
		args = append(args, status)
	}

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Pr_Cancel_Request c`+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	listArgs := append(append([]interface{}{}, args...), perPage, (page-1)*perPage)
	rows, err := r.db.QueryContext(ctx, prCancelSelect+where+`
		ORDER BY (c.cancel_status = 'pending') DESC,
		         CASE WHEN c.cancel_status = 'pending' THEN c.cancel_create_date END ASC,
		         c.cancel_create_date DESC
		LIMIT ? OFFSET ?`, listArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []PRCancelRequestView
	for rows.Next() {
		v, err := scanPRCancel(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

// Reject atomik: hanya berhasil bila request masih pending.
func (r *PRCancelRepo) Reject(ctx context.Context, cancelID int, financeID, notes string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE T_Pr_Cancel_Request
		SET cancel_status = 'rejected', cancel_ref_admin_review = ?, cancel_review_notes = ?, cancel_review_date = NOW()
		WHERE cancel_id = ? AND cancel_status = 'pending'`, financeID, notes, cancelID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrCancelRequestNotPending
	}
	return nil
}

// Approve: menyetujui request DAN membatalkan PR dalam SATU transaksi (request -> approved,
// payment draft/pending -> cancelled, PR -> cancelled, catat riwayat status). Gagal di
// tengah jalan = tidak ada yang berubah.
func (r *PRCancelRepo) Approve(ctx context.Context, cancelID int, financeID, reviewNotes string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var prID int
	var reqStatus, reason string
	err = tx.QueryRowContext(ctx,
		`SELECT cancel_ref_pr, cancel_status, cancel_reason FROM T_Pr_Cancel_Request WHERE cancel_id = ? FOR UPDATE`,
		cancelID).Scan(&prID, &reqStatus, &reason)
	if err != nil {
		return err // sql.ErrNoRows diteruskan apa adanya
	}
	if reqStatus != "pending" {
		return ErrCancelRequestNotPending
	}

	var prStatus string
	if err := tx.QueryRowContext(ctx,
		`SELECT pr_status FROM T_Purchase_Request WHERE pr_id = ? FOR UPDATE`, prID).Scan(&prStatus); err != nil {
		return err
	}
	if prStatus != "submitted" && prStatus != "approved" {
		return ErrPRNotCancellable
	}

	var paid int
	if err := tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Pr_Payment WHERE payment_ref_pr = ? AND payment_status = 'paid'`, prID).Scan(&paid); err != nil {
		return err
	}
	if paid > 0 {
		return ErrPRHasPaidPayment
	}

	var notesArg interface{}
	if reviewNotes != "" {
		notesArg = reviewNotes
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE T_Pr_Cancel_Request
		SET cancel_status = 'approved', cancel_ref_admin_review = ?, cancel_review_notes = ?, cancel_review_date = NOW()
		WHERE cancel_id = ?`, financeID, notesArg, cancelID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE T_Pr_Payment SET payment_status = 'cancelled'
		WHERE payment_ref_pr = ? AND payment_status IN ('draft', 'pending')`, prID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_status = 'cancelled' WHERE pr_id = ?`, prID); err != nil {
		return err
	}

	note := "Request pembatalan disetujui finance. Alasan requester: " + reason
	if reviewNotes != "" {
		note += " | Catatan finance: " + reviewNotes
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO T_Pr_Status_History (history_ref_admin, history_ref_pr, history_from_status,
		 history_to_status, history_notes, history_create_date)
		VALUES (?, ?, ?, 'cancelled', ?, NOW())`, financeID, prID, prStatus, note); err != nil {
		return err
	}

	return tx.Commit()
}