package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
)

type DivisionRepo struct{ db *sql.DB }

func NewDivisionRepo(db *sql.DB) *DivisionRepo { return &DivisionRepo{db: db} }

func (r *DivisionRepo) ListSelect(ctx context.Context) ([]models.Division, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT division_id, division_title, created_at, updated_at FROM T_Division ORDER BY division_title ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Division
	for rows.Next() {
		var v models.Division
		if err := rows.Scan(&v.ID, &v.Title, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *DivisionRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Division, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Division`).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		`SELECT division_id, division_title, created_at, updated_at FROM T_Division
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Division
	for rows.Next() {
		var v models.Division
		if err := rows.Scan(&v.ID, &v.Title, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *DivisionRepo) GetByID(ctx context.Context, id int) (*models.Division, error) {
	var v models.Division
	err := r.db.QueryRowContext(ctx,
		`SELECT division_id, division_title, created_at, updated_at FROM T_Division WHERE division_id = ?`, id).
		Scan(&v.ID, &v.Title, &v.CreatedAt, &v.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *DivisionRepo) Create(ctx context.Context, title string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO T_Division (division_title) VALUES (?)`, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *DivisionRepo) Update(ctx context.Context, id int, title string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Division SET division_title = ? WHERE division_id = ?`, title, id)
	return err
}

func (r *DivisionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Division WHERE division_id = ?`, id)
	return err
}
