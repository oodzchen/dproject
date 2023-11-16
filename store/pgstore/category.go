package pgstore

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Category struct {
	dbPool *pgxpool.Pool
}

func (p *Category) List(page, pageSize int, state model.CategoryState) ([]*model.Category, error) {
	sqlStr := `SELECT id, front_id, name, describe, author_id, approved, approval_comment, created_at FROM categories`
	switch state {
	case model.CategoryStateApproved:
		sqlStr += " WHERE approved = true "
	case model.CategoryStateUnapproved:
		sqlStr += " WHERE approved = false "
	}

	sqlStr += ` ORDER BY created_at DESC`

	rows, err := p.dbPool.Query(
		context.Background(),
		sqlStr,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.Category
	for rows.Next() {
		var item model.Category
		err := rows.Scan(
			&item.Id,
			&item.FrontId,
			&item.Name,
			&item.Describe,
			&item.AuthorId,
			&item.Approved,
			&item.ApprovalComment,
			&item.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		list = append(list, &item)
	}

	return list, nil
}

func (p *Category) Create(frontId, name, describe string, authorId int) (int, error) {
	var id int
	err := p.dbPool.QueryRow(context.Background(), "INSERT INTO categories (front_id, name, describe, author_id) VALUES ($1, $2, $3, $4) RETURNING (id)",
		frontId,
		name,
		describe,
		authorId,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Category) Update(frontId, name, describe string) (int, error) {
	var id int
	err := p.dbPool.QueryRow(context.Background(), "UPDATE categories SET name = $1, describe = $2 WHERE front_id = $3 RETURNING (id)",
		name,
		describe,
		frontId,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Category) Item(frontId string) (*model.Category, error) {
	var item model.Category
	err := p.dbPool.QueryRow(context.Background(), "SELECT id, front_id, name, describe, author_id, approved, approval_comment, created_at FROM categories front_id = $1",
		frontId,
	).Scan(
		&item.Id,
		&item.FrontId,
		&item.Name,
		&item.Describe,
		&item.AuthorId,
		&item.Approved,
		&item.ApprovalComment,
		&item.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (p *Category) Delete(frontId string) error {
	_, err := p.dbPool.Exec(context.Background(), "UPDATE categories SET deleted = true WHERE front_id = $1 RETURNING (id)",
		frontId,
	)
	if err != nil {
		return err
	}
	return nil
}
