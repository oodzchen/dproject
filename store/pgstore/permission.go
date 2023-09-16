package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Permission struct {
	dbPool *pgxpool.Pool
}

func (p *Permission) List(page, pageSize int) ([]*model.Permission, error) {
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPage
	}

	rows, err := p.dbPool.Query(
		context.Background(),
		`SELECT id, name, front_id, created_at FROM permissions ORDER BY created_at`,
		pageSize*(page-1),
		pageSize,
	)

	if err != nil {
		return nil, err
	}

	var list []*model.Permission
	for rows.Next() {
		var item model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.FrontId,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		list = append(list, &item)
	}

	return list, nil
}

func (p *Permission) Create(frontId, name string) (int, error) {
	var id int
	err := p.dbPool.QueryRow(context.Background(), "INSERT INTO permissions (front_id, name) VALUES ($1, $2) RETURNING (id)",
		frontId,
		name,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Permission) Update(name string) (int, error) {
	var id int
	err := p.dbPool.QueryRow(context.Background(), "UPDATE permissions SET name = $1 WHERE id = $2 RETURNING (id)",
		name,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Permission) Item(id int) (*model.Permission, error) {
	var item model.Permission
	err := p.dbPool.QueryRow(context.Background(), "SELECT id, front_id, name, created_at FROM permissions id = $1",
		id,
	).Scan(
		&item.Id,
		&item.FrontId,
		&item.Name,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}
