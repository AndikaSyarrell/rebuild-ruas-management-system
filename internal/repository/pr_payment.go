package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
)

// PRPaymentRepo mengakses T_Pr_Payment. Larangan admin yang sama merangkap
// pencatat (RefAdminInput) dan pengonfirmasi (RefAdminPaid) untuk baris yang
// sama (BR-PAY-01) adalah tanggung jawab service layer - repo ini murni
// baca/tulis, tidak membandingkan kedua admin tsb sendiri.
type PRPaymentRepo struct{ db db.Querier }

func NewPRPaymentRepo(db db.Querier) *PRPaymentRepo { return &PRPaymentRepo{db: db} }

const prPaymentSelect = `
	SELECT p.payment_id, p.payment_ref_pr, p.payment_ref_admin_input, p.payment_stage,
	       p.payment_amount, p.payment_type, p.payment_bank, p.payment_bank_account_no,
	       p.payment_bank_account_name, p.payment_priority_date, p.payment_status,
	       p.payment_paid_date, p.payment_ref_admin_paid, p.payment_create_date, p.payment_modify_date,
	       COALESCE(adi.admin_name, ''), COALESCE(adp.admin_name, '')
	FROM T_Pr_Payment p
	LEFT JOIN T_Admin adi ON p.payment_ref_admin_input = adi.admin_id
	LEFT JOIN T_Admin adp ON p.payment_ref_admin_paid = adp.admin_id
`

func scanPRPayment(row interface {
	Scan(dest ...interface{}) error
}) (*models.PRPayment, error) {
	var v models.PRPayment
	err := row.Scan(&v.ID, &v.RefPR, &v.RefAdminInput, &v.Stage, &v.Amount, &v.Type,
		&v.Bank, &v.BankAccountNo, &v.BankAccountName, &v.PriorityDate, &v.Status,
		&v.PaidDate, &v.RefAdminPaid, &v.CreateDate, &v.ModifyDate,
		&v.AdminInputName, &v.AdminPaidName)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRPaymentRepo) GetByID(ctx context.Context, id int) (*models.PRPayment, error) {
	row := r.db.QueryRowContext(ctx, prPaymentSelect+" WHERE p.payment_id = ?", id)
	return scanPRPayment(row)
}

func (r *PRPaymentRepo) ListByPR(ctx context.Context, prID int) ([]models.PRPayment, error) {
	rows, err := r.db.QueryContext(ctx,
		prPaymentSelect+" WHERE p.payment_ref_pr = ? ORDER BY p.payment_create_date ASC", prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRPayment
	for rows.Next() {
		v, err := scanPRPayment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

// PRPaymentInput adalah payload pembuatan satu tahap pembayaran baru.
type PRPaymentInput struct {
	RefPR           int
	RefAdminInput   string
	Stage           string
	Amount          float64
	Type            string
	Bank            string
	BankAccountNo   string
	BankAccountName string
	PriorityDate    string
}

func (r *PRPaymentRepo) Create(ctx context.Context, in PRPaymentInput) (int64, error) {
	var bank, bankAccNo, bankAccName, priorityDate interface{}
	if in.Bank != "" {
		bank = in.Bank
	}
	if in.BankAccountNo != "" {
		bankAccNo = in.BankAccountNo
	}
	if in.BankAccountName != "" {
		bankAccName = in.BankAccountName
	}
	if in.PriorityDate != "" {
		priorityDate = in.PriorityDate
	}

	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Payment (payment_ref_pr, payment_ref_admin_input, payment_stage, payment_amount,
		 payment_type, payment_bank, payment_bank_account_no, payment_bank_account_name,
		 payment_priority_date, payment_status, payment_create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'pending', NOW())`,
		in.RefPR, in.RefAdminInput, in.Stage, in.Amount, in.Type,
		bank, bankAccNo, bankAccName, priorityDate)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// MarkPaid menandai satu baris pembayaran sudah dibayar - dipanggil SETELAH
// service memvalidasi RefAdminPaid != RefAdminInput baris tsb (BR-PAY-01).
func (r *PRPaymentRepo) MarkPaid(ctx context.Context, paymentID int, adminPaidID string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE T_Pr_Payment SET payment_status = 'paid', payment_paid_date = NOW(), payment_ref_admin_paid = ?
		WHERE payment_id = ?`, adminPaidID, paymentID)
	return err
}

// UpdateStatus dipakai untuk transisi status selain "paid" (mis. 'cancelled').
func (r *PRPaymentRepo) UpdateStatus(ctx context.Context, paymentID int, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Pr_Payment SET payment_status = ? WHERE payment_id = ?`, status, paymentID)
	return err
}

// CountByStatus mengembalikan agregasi status pembayaran milik satu PR -
// dipakai service untuk mengevaluasi BR-PR-02/BR-PAY-02 ("semua payment
// berstatus paid -> PR completed"). Map hanya berisi status yang benar-benar
// muncul; pemanggil yang menyimpulkan "semua paid" dari total baris.
func (r *PRPaymentRepo) CountByStatus(ctx context.Context, prID int) (map[string]int, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT payment_status, COUNT(*) FROM T_Pr_Payment WHERE payment_ref_pr = ? GROUP BY payment_status`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var status string
		var count int
		if err := rows.Scan(&status, &count); err != nil {
			return nil, err
		}
		out[status] = count
	}
	return out, rows.Err()
}

func (r *PRPaymentRepo) ActivateDraftPayments(ctx context.Context, prID int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Pr_Payment SET payment_status = 'pending' WHERE payment_ref_pr = ? AND payment_status = 'draft'`,
		prID)
	return err
}

type PaymentExportFilter struct {
	StartDate     string // yyyy-mm-dd, berdasarkan payment_create_date
	EndDate       string
	ResponsibleID int
	Status        string // pending | paid | cancelled | draft, "" = semua
	AdminID       string // requester (pr_ref_admin) - filter "user"
}

type PaymentExportRow struct {
	PaymentID         int
	PRID              int
	RfpNo             string
	RequesterName     string
	DescriptionItem   string
	ResponsibleName   string
	CoaCode           string
	SubClient         string
	Amount            float64
	Bank              string
	BankAccountName   string
	BankAccountNo     string
	PoAmount          float64
	Hpp               float64
	PoNo              string
	TargetInvoiceDate *time.Time
	PaymentStatus     string
	PaymentCreateDate time.Time
}

func (r *PRPaymentRepo) ListForExport(ctx context.Context, f PaymentExportFilter) ([]PaymentExportRow, error) {
	conds := []string{"1=1"}
	var args []interface{}

	if f.StartDate != "" && f.EndDate != "" {
		conds = append(conds, "DATE(p.payment_create_date) BETWEEN ? AND ?")
		args = append(args, f.StartDate, f.EndDate)
	}
	if f.ResponsibleID != 0 {
		conds = append(conds, "pr.pr_ref_responsible = ?")
		args = append(args, f.ResponsibleID)
	}
	if f.Status != "" {
		conds = append(conds, "p.payment_status = ?")
		args = append(args, f.Status)
	}
	if f.AdminID != "" {
		conds = append(conds, "pr.pr_ref_admin = ?")
		args = append(args, f.AdminID)
	}

	query := `
		SELECT p.payment_id, pr.pr_id, pr.pr_rfp_no,
		       COALESCE(ad.admin_name, ''), pr.pr_description_item,
		       COALESCE(rp.responsible_name, ''), COALESCE(rp.responsible_coa_code, ''),
		       COALESCE(pr.pr_subclient, ''), p.payment_amount,
		       COALESCE(p.payment_bank, ''), COALESCE(p.payment_bank_account_name, ''),
		       COALESCE(p.payment_bank_account_no, ''),
		       pr.pr_po_amount, pr.pr_hpp, COALESCE(pr.pr_po_no, ''),
		       pr.pr_target_invoice_date, p.payment_status, p.payment_create_date
		FROM T_Pr_Payment p
		JOIN T_Purchase_Request pr ON p.payment_ref_pr = pr.pr_id
		LEFT JOIN T_Responsible rp ON pr.pr_ref_responsible = rp.responsible_id
		LEFT JOIN T_Admin ad ON pr.pr_ref_admin = ad.admin_id
		WHERE ` + strings.Join(conds, " AND ") + `
		ORDER BY p.payment_create_date ASC, p.payment_id ASC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []PaymentExportRow
	for rows.Next() {
		var v PaymentExportRow
		if err := rows.Scan(&v.PaymentID, &v.PRID, &v.RfpNo,
			&v.RequesterName, &v.DescriptionItem, &v.ResponsibleName, &v.CoaCode,
			&v.SubClient, &v.Amount, &v.Bank, &v.BankAccountName, &v.BankAccountNo,
			&v.PoAmount, &v.Hpp, &v.PoNo, &v.TargetInvoiceDate, &v.PaymentStatus,
			&v.PaymentCreateDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PRPaymentRepo) SumPaidBeforeInChain(ctx context.Context, prIDs []int, before time.Time, excludePaymentID int) (float64, error) {
	if len(prIDs) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(prIDs))
	args := make([]interface{}, 0, len(prIDs)+3)
	for i, id := range prIDs {
		placeholders[i] = "?"
		args = append(args, id)
	}
	query := fmt.Sprintf(`
		SELECT COALESCE(SUM(payment_amount), 0) FROM T_Pr_Payment
		WHERE payment_ref_pr IN (%s) AND payment_status = 'paid'
		  AND payment_id != ?
		  AND (payment_create_date < ? OR (payment_create_date = ? AND payment_id < ?))`,
		strings.Join(placeholders, ","))
	args = append(args, excludePaymentID, before, before, excludePaymentID)

	var total float64
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&total)
	return total, err
}