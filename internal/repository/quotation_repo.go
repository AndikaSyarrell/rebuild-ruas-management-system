package repository

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
)

type PRQuotationRepo struct{ db db.Querier }

func NewPRQuotationRepo(db db.Querier) *PRQuotationRepo { return &PRQuotationRepo{db: db} }

func NormalizeQuotationNo(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

var ErrQuotationLinkConflict = errors.New("status link quotation berubah saat diproses")

const prQuotationSelect = `
	SELECT quotation_id, quotation_no, quotation_ref_po, quotation_link_ref_admin, quotation_link_date,
	       quotation_revoke_ref_admin, quotation_revoke_date, quotation_create_date
	FROM T_Pr_Quotation`

func scanPRQuotation(row interface {
	Scan(dest ...interface{}) error
}) (*models.PRQuotation, error) {
	var v models.PRQuotation
	err := row.Scan(&v.ID, &v.No, &v.RefPO, &v.LinkRefAdmin, &v.LinkDate,
		&v.RevokeRefAdmin, &v.RevokeDate, &v.CreateDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRQuotationRepo) GetByNo(ctx context.Context, no string) (*models.PRQuotation, error) {
	row := r.db.QueryRowContext(ctx, prQuotationSelect+" WHERE quotation_no = ? LIMIT 1", no)
	return scanPRQuotation(row)
}

func (r *PRQuotationRepo) GetByID(ctx context.Context, id int) (*models.PRQuotation, error) {
	row := r.db.QueryRowContext(ctx, prQuotationSelect+" WHERE quotation_id = ?", id)
	return scanPRQuotation(row)
}

func (r *PRQuotationRepo) Create(ctx context.Context, no string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Pr_Quotation (quotation_no, quotation_create_date) VALUES (?, NOW())`, no)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRQuotationRepo) ListByPO(ctx context.Context, poID string) ([]models.PRQuotation, error) {
	rows, err := r.db.QueryContext(ctx,
		prQuotationSelect+` WHERE quotation_ref_po = ? ORDER BY quotation_link_date ASC`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRQuotation
	for rows.Next() {
		v, err := scanPRQuotation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

func (r *PRQuotationRepo) ListCandidates(ctx context.Context, keyword string) ([]models.PRQuotation, error) {
	rows, err := r.db.QueryContext(ctx,
		prQuotationSelect+` WHERE quotation_ref_po IS NULL AND (? = '' OR quotation_no LIKE CONCAT('%', ?, '%'))
		 ORDER BY quotation_create_date DESC limit 50`, keyword, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRQuotation
	for rows.Next() {
		v, err := scanPRQuotation(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

type QuotationCandidate struct {
	Quotation models.PRQuotation
	Members   []QuotationMember
}

type QuotationMember struct {
	PRID   int
	RfpNo  string
	Status string
}

func (r *PRQuotationRepo) MemberPRs(ctx context.Context, quotationID int) ([]QuotationMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT pr_id, pr_rfp_no, pr_status FROM T_Purchase_Request
		 WHERE pr_ref_quotation = ? ORDER BY pr_create_date ASC`, quotationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QuotationMember
	for rows.Next() {
		var m QuotationMember
		if err := rows.Scan(&m.PRID, &m.RfpNo, &m.Status); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *PRQuotationRepo) HasCompletedMember(ctx context.Context, quotationID int) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM T_Purchase_Request WHERE pr_ref_quotation = ? AND pr_status = 'completed' LIMIT 1`,
		quotationID).Scan(&dummy)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *PRQuotationRepo) LinkToPO(ctx context.Context, quotationID int, poID, poOrderNum, adminID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE T_Pr_Quotation
		SET quotation_ref_po = ?, quotation_link_ref_admin = ?, quotation_link_date = NOW(),
		    quotation_revoke_ref_admin = NULL, quotation_revoke_date = NULL
		WHERE quotation_id = ? AND quotation_ref_po IS NULL`, poID, adminID, quotationID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrQuotationLinkConflict // keburu di-link request lain
	}

	// Baris ini WAJIB jalan sebelum Commit — ini yang tadi ke-skip:
	if _, err := tx.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_po_no = ? WHERE pr_ref_quotation = ?`, poOrderNum, quotationID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PRQuotationRepo) Unlink(ctx context.Context, quotationID int, adminID string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		UPDATE T_Pr_Quotation
		SET quotation_ref_po = NULL, quotation_revoke_ref_admin = ?, quotation_revoke_date = NOW()
		WHERE quotation_id = ?`, adminID, quotationID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_po_no = NULL WHERE pr_ref_quotation = ?`, quotationID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *PRQuotationRepo) UnlinkWithAudit(ctx context.Context, quotationID int, expectedPOID, adminID, note string) error{
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `
		Insert Into T_Pr_Comment (comment_ref_admin, comment_ref_pr, comment_text, comment_type, comment_create_date) 
		Select ?, pr_id, ?, 'po_unlink', NOW()
		From T_Purchase_Request Where pr_ref_quotation = ?`, adminID, note, quotationID); err != nil {
		return err
	}

	res, err := tx.ExecContext(ctx, `
		Update T_Pr_Quotation
		Set quotation_ref_po = NULL, quotation_revoke_ref_admin = ?, quotation_revoke_date = NOW()
		Where quotation_id = ? AND quotation_ref_po = ?`, adminID, quotationID, expectedPOID)
	if err != nil {
		return err
	}

	if n, err := res.RowsAffected(); err != nil {
		return err
	} else if n == 0 {
		return ErrQuotationLinkConflict
	}

	if _, err := tx.ExecContext(ctx, `
		Update T_Purchase_Request Set pr_po_no = Null Where pr_ref_quotation = ?`, quotationID); err != nil {
			return err
		}
	return tx.Commit()
}

func (r *PRQuotationRepo) ListQuotationIDsByPO(ctx context.Context, poID string) ([]int, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT quotation_id FROM T_Pr_Quotation WHERE quotation_ref_po = ?`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *PRQuotationRepo) SearchByNo(ctx context.Context, keyword string, limit int) ([]models.PRQuotation, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT q.quotation_id, q.quotation_no, q.quotation_ref_po, q.quotation_link_ref_admin,
		       q.quotation_link_date, q.quotation_revoke_ref_admin, q.quotation_revoke_date,
		       q.quotation_create_date, COALESCE(po.po_order_num, '')
		FROM T_Pr_Quotation q
		LEFT JOIN T_Po po ON q.quotation_ref_po = po.po_id
		WHERE (? = '' OR q.quotation_no LIKE CONCAT('%', ?, '%'))
		ORDER BY q.quotation_create_date DESC
		LIMIT ?`, keyword, keyword, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRQuotation
	for rows.Next() {
		var v models.PRQuotation
		if err := rows.Scan(&v.ID, &v.No, &v.RefPO, &v.LinkRefAdmin, &v.LinkDate,
			&v.RevokeRefAdmin, &v.RevokeDate, &v.CreateDate, &v.POOrderNum); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// QuotationResponsible: responsible (CoA) yang dipakai anggota sebuah grup.
type QuotationResponsible struct {
	ID      int
	Name    string
	CoaCode string
}

func (r *PRQuotationRepo) queryResponsibles(ctx context.Context, query string, args ...interface{}) ([]QuotationResponsible, error) {
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []QuotationResponsible
	for rows.Next() {
		var v QuotationResponsible
		if err := rows.Scan(&v.ID, &v.Name, &v.CoaCode); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// ResponsiblesByQuotation: responsible distinct anggota grup. PR cancelled/rejected
// tidak dihitung; excludePRID = PR yang sedang diedit (0 saat create).
func (r *PRQuotationRepo) ResponsiblesByQuotation(ctx context.Context, quotationID, excludePRID int) ([]QuotationResponsible, error) {
	return r.queryResponsibles(ctx, `
		SELECT DISTINCT rp.responsible_id, rp.responsible_name, rp.responsible_coa_code
		FROM T_Purchase_Request pr
		JOIN T_Responsible rp ON pr.pr_ref_responsible = rp.responsible_id
		WHERE pr.pr_ref_quotation = ? AND pr.pr_id <> ?
		  AND pr.pr_status NOT IN ('cancelled', 'rejected')
		ORDER BY rp.responsible_id`, quotationID, excludePRID)
}

// ResponsiblesByPO: gabungan responsible dari SEMUA grup yang ter-link ke PO tsb.
func (r *PRQuotationRepo) ResponsiblesByPO(ctx context.Context, poID string, excludePRID int) ([]QuotationResponsible, error) {
	return r.queryResponsibles(ctx, `
		SELECT DISTINCT rp.responsible_id, rp.responsible_name, rp.responsible_coa_code
		FROM T_Purchase_Request pr
		JOIN T_Pr_Quotation q ON pr.pr_ref_quotation = q.quotation_id
		JOIN T_Responsible rp ON pr.pr_ref_responsible = rp.responsible_id
		WHERE q.quotation_ref_po = ? AND pr.pr_id <> ?
		  AND pr.pr_status NOT IN ('cancelled', 'rejected')
		ORDER BY rp.responsible_id`, poID, excludePRID)
}