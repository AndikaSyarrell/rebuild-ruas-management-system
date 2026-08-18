package repository

import (
	"context"
	// "database/sql"

	"rms-backend/internal/models"
	"rms-backend/internal/db"
)

type RegionRepo struct{ db db.Querier }

func NewRegionRepo(db db.Querier) *RegionRepo { return &RegionRepo{db: db} }

func (r *RegionRepo) List(ctx context.Context, keyword string) ([]models.Region, error) {
	query := `SELECT region_id, region_title, region_create_date, region_modify_date FROM T_Region
			  WHERE (? = '' OR region_title LIKE CONCAT('%', ?, '%'))
			  ORDER BY region_title ASC, region_create_date DESC`
	rows, err := r.db.QueryContext(ctx, query, keyword, keyword)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Region
	for rows.Next() {
		var v models.Region
		if err := rows.Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *RegionRepo) ListPaged(ctx context.Context, page, perPage int) ([]models.Region, int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM T_Region`).Scan(&total); err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * perPage
	rows, err := r.db.QueryContext(ctx,
		`SELECT region_id, region_title, region_create_date, region_modify_date FROM T_Region
		 ORDER BY region_create_date DESC LIMIT ? OFFSET ?`, perPage, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var out []models.Region
	for rows.Next() {
		var v models.Region
		if err := rows.Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate); err != nil {
			return nil, 0, err
		}
		out = append(out, v)
	}
	return out, total, rows.Err()
}

func (r *RegionRepo) GetByID(ctx context.Context, id int) (*models.Region, error) {
	var v models.Region
	err := r.db.QueryRowContext(ctx,
		`SELECT region_id, region_title, region_create_date, region_modify_date FROM T_Region WHERE region_id = ?`, id).
		Scan(&v.ID, &v.Title, &v.CreateDate, &v.ModifyDate)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

func (r *RegionRepo) Create(ctx context.Context, title string) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO T_Region (region_title, region_create_date) VALUES (?, NOW())`, title)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *RegionRepo) Update(ctx context.Context, id int, title string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE T_Region SET region_title = ? WHERE region_id = ?`, title, id)
	return err
}

func (r *RegionRepo) Delete(ctx context.Context, id int) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM T_Region WHERE region_id = ?`, id)
	return err
}
