package repository

import (
	"context"
	"database/sql"
	"errors"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
	// "rms-backend/internal/service"
)

type PRRepo struct{ db db.Querier }

func NewPRRepo(db db.Querier) *PRRepo { return &PRRepo{db: db} }

const prDetailSelect = `
	SELECT pr.pr_id, pr.pr_ref_admin, pr.pr_ref_responsible, pr.pr_rfp_no,
	       pr.pr_description_item, pr.pr_subclient, pr.pr_requested_amount, pr.pr_qout_no,
	       pr.pr_po_amount, pr.pr_hpp, pr.pr_target_invoice_date, pr.pr_status, pr.pr_priority,
	       pr.pr_priority_ref_admin, pr.pr_priority_date, pr.pr_signature_ref,
	       pr.pr_ref_previous_pr,
	       pr.pr_create_date, pr.pr_modify_date,
	       COALESCE(ad.admin_name, ''), COALESCE(rp.responsible_name, ''), COALESCE(sg.signature_file, ''),
	       COALESCE(prev.pr_rfp_no, '')
	FROM T_Purchase_Request pr
	LEFT JOIN T_Admin ad ON pr.pr_ref_admin = ad.admin_id
	LEFT JOIN T_Responsible rp ON pr.pr_ref_responsible = rp.responsible_id
	LEFT JOIN T_Admin_Signature sg ON pr.pr_signature_ref = sg.signature_id
	LEFT JOIN T_Purchase_Request prev ON pr.pr_ref_previous_pr = prev.pr_id
`

func scanPR(row interface {
	Scan(dest ...interface{}) error
}) (*models.PurchaseRequest, error) {
	var v models.PurchaseRequest
	err := row.Scan(&v.ID, &v.RefAdmin, &v.RefResponsible, &v.RfpNo,
		&v.DescriptionItem, &v.SubClient, &v.RequestedAmount, &v.QoutNo,
		&v.PoAmount, &v.Hpp, &v.TargetInvoiceDate, &v.Status, &v.Priority,
		&v.PriorityRefAdmin, &v.PriorityDate, &v.SignatureRef,
		&v.RefPreviousPR,
		&v.CreateDate, &v.ModifyDate,
		&v.AdminName, &v.ResponsibleName, &v.SignatureFile,
		&v.PreviousRfpNo)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRRepo) GetByID(ctx context.Context, id int) (*models.PurchaseRequest, error) {
	row := r.db.QueryRowContext(ctx, prDetailSelect+" WHERE pr.pr_id = ?", id)
	return scanPR(row)
}

func (r *PRRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.PurchaseRequest, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Purchase_Request`).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		prDetailSelect+" ORDER BY pr.pr_create_date DESC LIMIT ? OFFSET ?", perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.PurchaseRequest
	for rows.Next() {
		v, err := scanPR(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, *v)
	}
	return out, total, rows.Err()
}

func (r *PRRepo) ListByStatus(ctx context.Context, statuses []string) ([]models.PurchaseRequest, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	placeholders := ""
	args := make([]interface{}, 0, len(statuses))
	for i, s := range statuses {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, s)
	}

	query := prDetailSelect + " WHERE pr.pr_status IN (" + placeholders + ") ORDER BY pr.pr_create_date ASC"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PurchaseRequest
	for rows.Next() {
		v, err := scanPR(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	return out, rows.Err()
}

type PRPaymentDraft struct {
	Stage           string
	Amount          float64
	Type            string
	Bank            string
	BankAccountNo   string
	BankAccountName string
	PriorityDate    string
}

type PRInput struct {
	RefAdmin          string
	RefResponsible    int
	RfpNo             string
	DescriptionItem   string
	SubClient         string
	RequestedAmount   float64
	PoAmount          float64
	Hpp               float64
	QoutNo            string
	TargetInvoiceDate string
	SignatureRef      int
	RefPreviousPR     *int             // BARU
	Payments          []PRPaymentDraft // BARU
}

func (r *PRRepo) Create(ctx context.Context, in PRInput) (int64, error) {
	var subClient, qoutNo, targetInvoiceDate, signatureRef, refPreviousPR interface{}
	if in.SubClient != "" {
		subClient = in.SubClient
	}
	if in.QoutNo != "" {
		qoutNo = in.QoutNo
	}
	if in.TargetInvoiceDate != "" {
		targetInvoiceDate = in.TargetInvoiceDate
	}
	if in.SignatureRef != 0 {
		signatureRef = in.SignatureRef
	}
	if in.RefPreviousPR != nil {
		refPreviousPR = *in.RefPreviousPR
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO T_Purchase_Request (pr_ref_admin, pr_ref_responsible, pr_rfp_no,
		 pr_description_item, pr_subclient, pr_requested_amount, pr_po_amount, pr_hpp,
		 pr_qout_no, pr_signature_ref, pr_target_invoice_date, pr_ref_previous_pr,
		 pr_status, pr_create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', NOW())`,
		in.RefAdmin, in.RefResponsible, in.RfpNo,
		in.DescriptionItem, subClient, in.RequestedAmount, in.PoAmount, in.Hpp,
		qoutNo, signatureRef, targetInvoiceDate, refPreviousPR)
	if err != nil {
		return 0, err
	}
	prID, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	if len(in.Payments) > 0 {
		// status 'draft' - BUKAN 'pending' - supaya tidak ikut muncul di antrean
		// finance sebelum PR ini benar-benar disetujui (lihat ActivateDraftPayments).
		stmt, err := tx.PrepareContext(ctx, `
			INSERT INTO T_Pr_Payment (payment_ref_pr, payment_ref_admin_input, payment_stage,
			 payment_amount, payment_type, payment_bank, payment_bank_account_no,
			 payment_bank_account_name, payment_priority_date, payment_status, payment_create_date)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', NOW())`)
		if err != nil {
			return 0, err
		}
		defer stmt.Close()

		for _, p := range in.Payments {
			var bank, bankNo, bankName, priorityDate interface{}
			if p.Bank != "" {
				bank = p.Bank
			}
			if p.BankAccountNo != "" {
				bankNo = p.BankAccountNo
			}
			if p.BankAccountName != "" {
				bankName = p.BankAccountName
			}
			if p.PriorityDate != "" {
				priorityDate = p.PriorityDate
			}
			if _, err := stmt.ExecContext(ctx, prID, in.RefAdmin, p.Stage, p.Amount, p.Type,
				bank, bankNo, bankName, priorityDate); err != nil {
				return 0, err
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return prID, nil
}

func (r *PRRepo) UpdateSignature(ctx context.Context, id int, signatureRef int) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_signature_ref = ? WHERE pr_id = ?`, signatureRef, id)
	return err
}

func (r *PRDocumentRepo) ExistsByPRAndType(ctx context.Context, prID int, docType string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM T_Pr_Document WHERE document_ref_pr = ? AND document_type = ? LIMIT 1`,
		prID, docType).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *PRRepo) Update(ctx context.Context, id int, in PRInput) error {
	var subClient, qoutNo, targetInvoiceDate, refPreviousPR interface{}
	if in.SubClient != "" {
		subClient = in.SubClient
	}
	if in.QoutNo != "" {
		qoutNo = in.QoutNo
	}
	if in.TargetInvoiceDate != "" {
		targetInvoiceDate = in.TargetInvoiceDate
	}
	if in.RefPreviousPR != nil {
		refPreviousPR = *in.RefPreviousPR
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE T_Purchase_Request SET pr_ref_responsible = ?, pr_description_item = ?,
		 pr_subclient = ?, pr_requested_amount = ?, pr_qout_no = ?, pr_target_invoice_date = ?,
		 pr_ref_previous_pr = ?
		WHERE pr_id = ?`,
		in.RefResponsible, in.DescriptionItem, subClient,
		in.RequestedAmount, qoutNo, targetInvoiceDate, refPreviousPR, id)
	return err
}

const maxPRChainDepth = 20

func (r *PRRepo) GetReferenceChain(ctx context.Context, prID int) ([]models.PurchaseRequest, error) {
	var chain []models.PurchaseRequest
	seen := map[int]bool{}
	currentID := prID
	for i := 0; i < maxPRChainDepth; i++ {
		if currentID == 0 || seen[currentID] {
			break
		}
		seen[currentID] = true
		pr, err := r.GetByID(ctx, currentID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				break
			}
			return nil, err
		}
		chain = append(chain, *pr)
		if pr.RefPreviousPR == nil {
			break
		}
		currentID = *pr.RefPreviousPR
	}
	return chain, nil
}

func (r *PRRepo) UpdateStatus(ctx context.Context, id int, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Purchase_Request SET pr_status = ? WHERE pr_id = ?`, status, id)
	return err
}

func (r *PRRepo) UpdatePriority(ctx context.Context, id int, priority, setByAdminID string) error {
	var p interface{}
	if priority != "" {
		p = priority
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_priority = ?, pr_priority_ref_admin = ?, pr_priority_date = NOW() WHERE pr_id = ?`,
		p, setByAdminID, id)
	return err
}

func (r *PRRepo) UpdateAmounts(ctx context.Context, id int, poAmount, hpp float64) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_po_amount = ?, pr_hpp = ? WHERE pr_id = ?`, poAmount, hpp, id)
	return err
}

type PRHistoryRepo struct{ db db.Querier }

func NewPRHistoryRepo(db db.Querier) *PRHistoryRepo { return &PRHistoryRepo{db: db} }

func (r *PRHistoryRepo) Insert(ctx context.Context, prID int, adminID string, fromStatus *string, toStatus, notes string) (int64, error) {
	var notesArg interface{}
	if notes != "" {
		notesArg = notes
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Status_History (history_ref_admin, history_ref_pr, history_from_status,
		 history_to_status, history_notes, history_create_date)
		VALUES (?, ?, ?, ?, ?, NOW())`,
		adminID, prID, fromStatus, toStatus, notesArg)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRHistoryRepo) ListByPR(ctx context.Context, prID int) ([]models.PRStatusHistory, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT h.history_id, h.history_ref_admin, h.history_ref_pr, h.history_from_status,
		       h.history_to_status, h.history_notes, h.history_create_date, COALESCE(ad.admin_name, '')
		FROM T_Pr_Status_History h
		LEFT JOIN T_Admin ad ON h.history_ref_admin = ad.admin_id
		WHERE h.history_ref_pr = ?
		ORDER BY h.history_create_date ASC`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRStatusHistory
	for rows.Next() {
		var v models.PRStatusHistory
		if err := rows.Scan(&v.ID, &v.RefAdmin, &v.RefPR, &v.FromStatus, &v.ToStatus,
			&v.Notes, &v.CreateDate, &v.AdminName); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// --- PR Document ---

type PRDocumentRepo struct{ db db.Querier }

func NewPRDocumentRepo(db db.Querier) *PRDocumentRepo { return &PRDocumentRepo{db: db} }

func (r *PRDocumentRepo) Insert(ctx context.Context, prID int, docType, fileName, filePath, adminID string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Document (document_type, document_file_name, document_file_path,
		 document_ref_admin, document_ref_pr, document_create_date)
		VALUES (?, ?, ?, ?, ?, NOW())`, docType, fileName, filePath, adminID, prID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRDocumentRepo) ListByPR(ctx context.Context, prID int) ([]models.PRDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT d.document_id, d.document_type, d.document_file_name, d.document_file_path,
		       d.document_ref_admin, d.document_ref_pr, d.document_create_date, COALESCE(ad.admin_name, '')
		FROM T_Pr_Document d
		LEFT JOIN T_Admin ad ON d.document_ref_admin = ad.admin_id
		WHERE d.document_ref_pr = ?
		ORDER BY d.document_create_date ASC`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRDocument
	for rows.Next() {
		var v models.PRDocument
		if err := rows.Scan(&v.ID, &v.Type, &v.FileName, &v.FilePath, &v.RefAdmin, &v.RefPR,
			&v.CreateDate, &v.AdminName); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PRDocumentRepo) GetByID(ctx context.Context, id int) (*models.PRDocument, error) {
	var v models.PRDocument
	err := r.db.QueryRowContext(ctx, `
		SELECT document_id, document_type, document_file_name, document_file_path,
		       document_ref_admin, document_ref_pr, document_create_date
		FROM T_Pr_Document WHERE document_id = ?`, id).
		Scan(&v.ID, &v.Type, &v.FileName, &v.FilePath, &v.RefAdmin, &v.RefPR, &v.CreateDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRDocumentRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Pr_Document WHERE document_id = ?`, id)
	return err
}

// --- PR Comment (BR-REV-04: naratif, tidak memicu perubahan status) ---

type PRCommentRepo struct{ db db.Querier }

func NewPRCommentRepo(db db.Querier) *PRCommentRepo { return &PRCommentRepo{db: db} }

func (r *PRCommentRepo) Insert(ctx context.Context, prID int, adminID, text, commentType string) (int64, error) {
	if commentType == "" {
		commentType = "general"
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Pr_Comment (comment_ref_admin, comment_ref_pr, comment_text, comment_type, comment_create_date)
		VALUES (?, ?, ?, ?, NOW())`, adminID, prID, text, commentType)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *PRCommentRepo) ListByPR(ctx context.Context, prID int) ([]models.PRComment, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT c.comment_id, c.comment_ref_admin, c.comment_ref_pr, c.comment_text, c.comment_type,
		       c.comment_create_date, COALESCE(ad.admin_name, '')
		FROM T_Pr_Comment c
		LEFT JOIN T_Admin ad ON c.comment_ref_admin = ad.admin_id
		WHERE c.comment_ref_pr = ?
		ORDER BY c.comment_create_date ASC`, prID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PRComment
	for rows.Next() {
		var v models.PRComment
		if err := rows.Scan(&v.ID, &v.RefAdmin, &v.RefPR, &v.Text, &v.Type,
			&v.CreateDate, &v.AdminName); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PRApprovalRepo) ListPendingByApprover(ctx context.Context, adminID string) ([]models.PRApproval, error) {
	rows, err := r.db.QueryContext(ctx, prApprovalSelect+`
		WHERE a.approval_ref_admin = ? AND a.approval_status = 'pending'
		  AND a.approval_round = (
		        SELECT MAX(approval_round) FROM T_Pr_Approval WHERE approval_ref_pr = a.approval_ref_pr
		  )
		  AND (a.approval_level = 1 OR EXISTS (
		        SELECT 1 FROM T_Pr_Approval prev
		        WHERE prev.approval_ref_pr = a.approval_ref_pr
		          AND prev.approval_round = a.approval_round
		          AND prev.approval_level = a.approval_level - 1
		          AND prev.approval_status = 'approved'
		  ))
		ORDER BY a.approval_create_date ASC`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPRApprovalRows(rows)
}

func (r *PRRepo) IsAdminRelated(ctx context.Context, prID int, adminID string) (bool, error) {
	var dummy int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1 FROM (
			SELECT 1 FROM T_Purchase_Request WHERE pr_id = ? AND pr_ref_admin = ?
			UNION
			SELECT 1 FROM T_Pr_Approval WHERE approval_ref_pr = ? AND approval_ref_admin = ?
			UNION
			SELECT 1 FROM T_Pr_Payment WHERE payment_ref_pr = ?
			  AND (payment_ref_admin_input = ? OR payment_ref_admin_paid = ?)
		) AS related LIMIT 1`,
		prID, adminID, prID, adminID, prID, adminID, adminID).Scan(&dummy)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}