package repository

import (
	"context"
	// "database/sql"

	"rms-backend/internal/models"
	"rms-backend/internal/db"
)

type DocumentRepo struct{ db db.Querier }

func NewDocumentRepo(db db.Querier) *DocumentRepo { return &DocumentRepo{db: db} }

func (r *DocumentRepo) ListByPO(ctx context.Context, poID string) ([]models.PODocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT document_id, document_ref_po, document_title, document_file, document_create_date, document_modify_date
		FROM T_Document WHERE document_ref_po = ? ORDER BY document_create_date ASC`, poID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.PODocument
	for rows.Next() {
		var v models.PODocument
		if err := rows.Scan(&v.ID, &v.RefPO, &v.Title, &v.File, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *DocumentRepo) GetByID(ctx context.Context, id int) (*models.PODocument, error) {
	var v models.PODocument
	err := r.db.QueryRowContext(ctx, `
		SELECT document_id, document_ref_po, document_title, document_file, document_create_date, document_modify_date
		FROM T_Document WHERE document_id = ?`, id).
		Scan(&v.ID, &v.RefPO, &v.Title, &v.File, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *DocumentRepo) Insert(ctx context.Context, poID, title, filePath string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `
		INSERT INTO T_Document (document_ref_po, document_title, document_file, document_create_date)
		VALUES (?, ?, ?, NOW())`, poID, title, filePath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *DocumentRepo) CountByPO(ctx context.Context, poID string) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Document WHERE document_ref_po = ?`, poID).Scan(&count)
	return count, err
}

func (r *DocumentRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Document WHERE document_id = ?`, id)
	return err
}
