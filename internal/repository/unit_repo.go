package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
)

type UnitRepo struct{ db *sql.DB }

func NewUnitRepo(db *sql.DB) *UnitRepo { return &UnitRepo{db: db} }

func (r *UnitRepo) ListSelect(ctx context.Context) ([]models.Unit, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT unit_id, unit_title, unit_create_date, unit_modify_date FROM T_Unit ORDER BY unit_title ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Unit
	for rows.Next() {
		var v models.Unit
		if err := rows.Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *UnitRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Unit, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Unit`).Scan(&total); err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		`SELECT unit_id, unit_title, unit_create_date, unit_modify_date FROM T_Unit
		 ORDER BY unit_create_date DESC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []models.Unit
	for rows.Next() {
		var v models.Unit
		if err := rows.Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *UnitRepo) GetByID(ctx context.Context, id int) (*models.Unit, error) {
	var v models.Unit
	err := r.db.QueryRowContext(ctx,
		`SELECT unit_id, unit_title, unit_create_date, unit_modify_date FROM T_Unit WHERE unit_id = ?`, id).
		Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *UnitRepo) Create(ctx context.Context, title string) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO T_Unit (unit_title, unit_create_date) VALUES (?, NOW())`, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *UnitRepo) Update(ctx context.Context, id int, title string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Unit SET unit_title = ? WHERE unit_id = ?`, title, id)
	return err
}

func (r *UnitRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Unit WHERE unit_id = ?`, id)
	return err
}
