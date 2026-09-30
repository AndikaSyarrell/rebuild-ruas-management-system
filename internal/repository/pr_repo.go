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
	       pr.pr_ref_previous_pr, pr.pr_po_no, pr.pr_ref_quotation,
	       pr.pr_create_date, pr.pr_modify_date,
	       COALESCE(ad.admin_name, ''), COALESCE(rp.responsible_name, ''), COALESCE(sg.signature_file, ''),
	       COALESCE(prev.pr_rfp_no, ''), COALESCE(q.quotation_no, '')
	FROM T_Purchase_Request pr
	LEFT JOIN T_Admin ad ON pr.pr_ref_admin = ad.admin_id
	LEFT JOIN T_Responsible rp ON pr.pr_ref_responsible = rp.responsible_id
	LEFT JOIN T_Admin_Signature sg ON pr.pr_signature_ref = sg.signature_id
	LEFT JOIN T_Purchase_Request prev ON pr.pr_ref_previous_pr = prev.pr_id
	LEFT JOIN T_Pr_Quotation q ON pr.pr_ref_quotation = q.quotation_id
`

func scanPR(row interface{ Scan(dest ...interface{}) error }) (*models.PurchaseRequest, error) {
	var v models.PurchaseRequest
	err := row.Scan(&v.ID, &v.RefAdmin, &v.RefResponsible, &v.RfpNo,
		&v.DescriptionItem, &v.SubClient, &v.RequestedAmount, &v.QoutNo,
		&v.PoAmount, &v.Hpp, &v.TargetInvoiceDate, &v.Status, &v.Priority,
		&v.PriorityRefAdmin, &v.PriorityDate, &v.SignatureRef,
		&v.RefPreviousPR, &v.PoNo, &v.RefQuotation,
		&v.CreateDate, &v.ModifyDate,
		&v.AdminName, &v.ResponsibleName, &v.SignatureFile, &v.PreviousRfpNo, &v.QuotationNo)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PRRepo) GetByID(ctx context.Context, id int) (*models.PurchaseRequest, error) {
	row := r.db.QueryRowContext(ctx, prDetailSelect+" WHERE pr.pr_id = ?", id)
	return scanPR(row)
}

func (r *PRRepo) ListPaged(ctx context.Context, page, perPage int, adminID string) ([]models.PurchaseRequest, int, error) {
	where := " WHERE " + prVisibleClause

	var total int
	if err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM T_Purchase_Request pr`+where, prViewerArgs(adminID)...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args := append(prViewerArgs(adminID), perPage, (page-1)*perPage)
	rows, err := r.db.QueryContext(ctx,
		prDetailSelect+where+" ORDER BY pr.pr_create_date DESC LIMIT ? OFFSET ?", args...)
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

func (r *PRRepo) ListByStatus(ctx context.Context, statuses []string, adminID string) ([]models.PurchaseRequest, error) {
	if len(statuses) == 0 {
		return nil, nil
	}
	placeholders := ""
	args := make([]interface{}, 0, len(statuses)+prViewerArgCount)
	for i, s := range statuses {
		if i > 0 {
			placeholders += ","
		}
		placeholders += "?"
		args = append(args, s)
	}
	args = append(args, prViewerArgs(adminID)...)

	query := prDetailSelect + " WHERE pr.pr_status IN (" + placeholders + ") AND " + prVisibleClause + " ORDER BY pr.pr_create_date ASC"
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
	RefPreviousPR     *int
	Payments          []PRPaymentDraft
	RefQuotation      *int            
	Priority 		  string
	PoNo 			  *string
}

func (r *PRRepo) Create(ctx context.Context, in PRInput) (int64, error) {
	var subClient, qoutNo, targetInvoiceDate, signatureRef, refPreviousPR, refQuotation, poNo interface{}
	if in.SubClient != "" {
		subClient = in.SubClient
	}
	if in.PoNo != nil && *in.PoNo != "" {
		poNo = *in.PoNo
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
	if in.RefQuotation != nil {
		refQuotation = *in.RefQuotation
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO T_Purchase_Request (pr_ref_admin, pr_ref_responsible, pr_rfp_no,
		 pr_description_item, pr_subclient, pr_requested_amount, pr_po_amount, pr_hpp,
		 pr_qout_no, pr_signature_ref, pr_target_invoice_date, pr_ref_previous_pr, pr_ref_quotation,
		 pr_priority, pr_po_no, pr_status, pr_create_date)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'draft', NOW())`,
		in.RefAdmin, in.RefResponsible, in.RfpNo,
		in.DescriptionItem, subClient, in.RequestedAmount, in.PoAmount, in.Hpp,
		qoutNo, signatureRef, targetInvoiceDate, refPreviousPR, refQuotation,
		in.Priority, poNo)
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
	var subClient, qoutNo, targetInvoiceDate, refPreviousPR, refQuotation, poNoVal interface{}
	poNoSet := false
	if in.PoNo != nil {
		poNoSet = true
		if *in.PoNo != "" {
			poNoVal = *in.PoNo
		}
	}
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
	if in.RefQuotation != nil {
		refQuotation = *in.RefQuotation
	}

	_, err := r.db.ExecContext(ctx, `
		UPDATE T_Purchase_Request SET pr_ref_responsible = ?, pr_description_item = ?,
		 pr_subclient = ?, pr_requested_amount = ?, pr_qout_no = ?, pr_target_invoice_date = ?,
		 pr_ref_previous_pr = ?, pr_ref_quotation = ?, pr_po_no = IF(?, ?, pr_po_no)
		WHERE pr_id = ?`,
		in.RefResponsible, in.DescriptionItem, subClient,
		in.RequestedAmount, qoutNo, targetInvoiceDate, refPreviousPR, refQuotation,
		poNoSet, poNoVal, id)
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

func (r *PRRepo) UpdateAmounts(ctx context.Context, id int, poAmount, hpp float64, poNo string) error {
	if poNo == "" {
		_, err := r.db.ExecContext(ctx,
			`UPDATE T_Purchase_Request SET pr_po_amount = ?, pr_hpp = ? WHERE pr_id = ?`,
			poAmount, hpp, id)
		return err
	}
	_, err := r.db.ExecContext(ctx,
		`UPDATE T_Purchase_Request SET pr_po_amount = ?, pr_hpp = ?, pr_po_no = ? WHERE pr_id = ?`,
		poAmount, hpp, poNo, id)
	return err
}

func (r *PRRepo) SyncPoNo(ctx context.Context, id int, poNo string) error {
	var arg interface{}
	if poNo != "" {
		arg = poNo
	}
	_, err := r.db.ExecContext(ctx, `UPDATE T_Purchase_Request SET pr_po_no = ? WHERE pr_id = ?`, arg, id)
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
		JOIN T_Purchase_Request pr ON pr.pr_id = a.approval_ref_pr
		WHERE a.approval_status = 'pending'
		  AND pr.pr_status = 'submitted'
		  AND a.approval_round = (
		        SELECT MAX(approval_round) FROM T_Pr_Approval WHERE approval_ref_pr = a.approval_ref_pr)
		  AND (a.approval_level = 1 OR EXISTS (
		        SELECT 1 FROM T_Pr_Approval prev
		        WHERE prev.approval_ref_pr = a.approval_ref_pr
		          AND prev.approval_round = a.approval_round
		          AND prev.approval_level = a.approval_level - 1
		          AND prev.approval_status = 'approved'))
		  AND (
		        (a.approval_level = 1 AND a.approval_ref_admin = ?)
		     OR (a.approval_level > 1 AND pr.pr_ref_admin <> ?
		         AND (a.approval_ref_admin = ?
		              OR (a.approval_ref_admin IS NULL AND EXISTS (
		                    SELECT 1 FROM T_Admin ad
		                    JOIN T_Role_Access ra ON ra.role_access_ref_role = ad.admin_ref_role
		                    JOIN T_Access ac ON ac.access_id = ra.role_access_ref_access
		                    WHERE ad.admin_id = ? AND ad.admin_active = 'active'
		                      AND ac.access_slug = a.approval_type))))
		  )
		ORDER BY a.approval_create_date ASC`, adminID, adminID, adminID, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPRApprovalRows(rows)
}

func (r *PRRepo) IsAdminRelated(ctx context.Context, prID int, adminID string) (bool, error) {
	var dummy int
	args := append([]interface{}{prID}, prViewerArgs(adminID)...)
	err := r.db.QueryRowContext(ctx,
		`SELECT 1 FROM T_Purchase_Request pr WHERE pr.pr_id = ? AND `+prVisibleClause+` LIMIT 1`,
		args...).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// slugExistsPrefix: cek apakah aktor (placeholder ?) aktif dan punya slug; slug ditutup pemanggil.
const slugExistsPrefix = `EXISTS (
		SELECT 1 FROM T_Admin vad
		JOIN T_Role_Access vra ON vra.role_access_ref_role = vad.admin_ref_role
		JOIN T_Access vac ON vac.access_id = vra.role_access_ref_access
		WHERE vad.admin_id = ? AND vad.admin_active = 'active' AND vac.access_slug = `

// prVisibleClause: satu-satunya sumber kebenaran "siapa boleh melihat PR ini".
// Butuh prViewerArgCount placeholder, semuanya diisi admin_id aktor (lihat prViewerArgs).
const prVisibleClause = `(
	pr.pr_ref_admin = ?
	OR EXISTS (SELECT 1 FROM T_Pr_Approval va
	           WHERE va.approval_ref_pr = pr.pr_id AND va.approval_ref_admin = ?)
	OR EXISTS (SELECT 1 FROM T_Pr_Payment vp
	           WHERE vp.payment_ref_pr = pr.pr_id
	             AND (vp.payment_ref_admin_input = ? OR vp.payment_ref_admin_paid = ?))
	OR (` + slugExistsPrefix + `'director')
	    AND EXISTS (SELECT 1 FROM T_Pr_Approval v1
	                WHERE v1.approval_ref_pr = pr.pr_id AND v1.approval_level = 1 AND v1.approval_status = 'approved'))
	OR (` + slugExistsPrefix + `'finance')
	    AND EXISTS (SELECT 1 FROM T_Pr_Approval v2
	                WHERE v2.approval_ref_pr = pr.pr_id AND v2.approval_level = 2 AND v2.approval_status = 'approved'))
		OR (` + slugExistsPrefix + `'finance')
	    AND EXISTS (SELECT 1 FROM T_Pr_Cancel_Request vc WHERE vc.cancel_ref_pr = pr.pr_id))
)`

const prViewerArgCount = 7 

func prViewerArgs(adminID string) []interface{} {
	out := make([]interface{}, prViewerArgCount)
	for i := range out {
		out[i] = adminID
	}
	return out
}