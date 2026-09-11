package repository

import (
	"context"

	"rms-backend/internal/db"
	"rms-backend/internal/models"
)

type AdminSignatureRepo struct{ db db.Querier }

func NewAdminSignatureRepo(db db.Querier) *AdminSignatureRepo { return &AdminSignatureRepo{db: db} }

// GetLatestByAdmin mengembalikan tanda tangan TERBARU milik seorang admin -
// dipakai PRApprovalService untuk mensyaratkan admin punya tanda tangan
// terdaftar sebelum bisa melakukan approve. Mengembalikan sql.ErrNoRows kalau
// admin belum pernah mengunggah tanda tangan sama sekali.
func (r *AdminSignatureRepo) GetLatestByAdmin(ctx context.Context, adminID string) (*models.AdminSignature, error) {
	var v models.AdminSignature
	err := r.db.QueryRowContext(ctx, `
		SELECT signature_id, signature_ref_admin, signature_name_pic, signature_file,
		       signature_create_date, signature_modify_date
		FROM T_Admin_Signature
		WHERE signature_ref_admin = ?
		ORDER BY signature_create_date DESC
		LIMIT 1`, adminID).
		Scan(&v.ID, &v.RefAdmin, &v.NamePic, &v.File, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AdminSignatureRepo) GetByID(ctx context.Context, id int) (*models.AdminSignature, error) {
	var v models.AdminSignature
	err := r.db.QueryRowContext(ctx, `
		SELECT signature_id, signature_ref_admin, signature_name_pic, signature_file,
		       signature_create_date, signature_modify_date
		FROM T_Admin_Signature WHERE signature_id = ?`, id).
		Scan(&v.ID, &v.RefAdmin, &v.NamePic, &v.File, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *AdminSignatureRepo) ListByAdmin(ctx context.Context, adminID string) ([]models.AdminSignature, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT signature_id, signature_ref_admin, signature_name_pic, signature_file,
		       signature_create_date, signature_modify_date
		FROM T_Admin_Signature
		WHERE signature_ref_admin = ?
		ORDER BY signature_create_date DESC`, adminID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.AdminSignature
	for rows.Next() {
		var v models.AdminSignature
		if err := rows.Scan(&v.ID, &v.RefAdmin, &v.NamePic, &v.File, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *AdminSignatureRepo) Create(ctx context.Context, adminID, namePic, filePath string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Admin_Signature (signature_ref_admin, signature_name_pic, signature_file, signature_create_date)
		VALUES (?, ?, ?, NOW())`, adminID, namePic, filePath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *AdminSignatureRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Admin_Signature WHERE signature_id = ?`, id)
	return err
}