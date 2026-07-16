package repository

import (
	"context"
	"database/sql"

	"rms-backend/internal/models"
)

type PpnRepo struct{ db *sql.DB }

func NewPpnRepo(db *sql.DB) *PpnRepo { return &PpnRepo{db: db} }

// List mengembalikan seluruh baris PPN (biasanya hanya beberapa - histori tarif).
func (r *PpnRepo) List(ctx context.Context) ([]models.Ppn, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT ppn_id, ppn_value, ppn_create_date, ppn_modify_date FROM T_Ppn ORDER BY ppn_create_date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Ppn
	for rows.Next() {
		var v models.Ppn
		if err := rows.Scan(&v.ID, &v.Value, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *PpnRepo) GetCurrent(ctx context.Context) (*models.Ppn, error) {
	var v models.Ppn
	err := r.db.QueryRowContext(ctx,
		`SELECT ppn_id, ppn_value, ppn_create_date, ppn_modify_date FROM T_Ppn ORDER BY ppn_id DESC LIMIT 1`).
		Scan(&v.ID, &v.Value, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PpnRepo) GetByID(ctx context.Context, id int) (*models.Ppn, error) {
	var v models.Ppn
	err := r.db.QueryRowContext(ctx,
		`SELECT ppn_id, ppn_value, ppn_create_date, ppn_modify_date FROM T_Ppn WHERE ppn_id = ?`, id).
		Scan(&v.ID, &v.Value, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *PpnRepo) Update(ctx context.Context, id int, value float64) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Ppn SET ppn_value = ? WHERE ppn_id = ?`, value, id)
	return err
}

func (r *PpnRepo) Create(ctx context.Context, value float64) (int64, error) {
	res, err := r.db.ExecContext(ctx, `INSERT INTO T_Ppn (ppn_value, ppn_create_date) VALUES (?, NOW())`, value)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
