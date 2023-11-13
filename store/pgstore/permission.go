package pgstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Permission struct {
	dbPool *pgxpool.Pool
}

func (p *Permission) List(page, pageSize int, module string) ([]*model.Permission, error) {
	// fmt.Println("permission list args:", page, pageSize, module)
	// fmt.Println("pool:", p.dbPool)
	// if page < 1 {
	// 	page = DefaultPage
	// }

	// if pageSize < 1 {
	// 	pageSize = DefaultPage
	// }

	sqlStrHead := `SELECT id, name, front_id, created_at, module FROM permissions`
	sqlStrTail := ` ORDER BY created_at DESC`
	args := []any{}

	sqlStr := sqlStrHead + sqlStrTail

	if module != "all" {
		sqlStr = sqlStrHead + ` WHERE module = $1` + sqlStrTail
		args = append(args, module)
	}

	rows, err := p.dbPool.Query(
		context.Background(),
		sqlStr,
		args...,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.Permission
	for rows.Next() {
		var item model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.FrontId,
			&item.CreatedAt,
			&item.Module,
		)

		if err != nil {
			return nil, err
		}

		list = append(list, &item)
	}

	return list, nil
}

func (p *Permission) Create(module, frontId, name string) (int, error) {
	var id int
	err := p.dbPool.QueryRow(context.Background(), "INSERT INTO permissions (front_id, name, module) VALUES ($1, $2, $3) RETURNING (id)",
		frontId,
		name,
		module,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (p *Permission) CreateMany(list []*model.Permission) error {
	sqlStrHead := `INSERT INTO permissions (front_id, name, module) VALUES `
	var strArr []string
	var args []any
	var argCount = 1

	for _, item := range list {
		strArr = append(strArr, fmt.Sprintf("($%d, $%d, $%d)", argCount, argCount+1, argCount+2))
		args = append(args, item.FrontId, item.Name, item.Module)
		argCount += 3
	}

	sqlStr := sqlStrHead + strings.Join(strArr, ", ")

	// fmt.Println("sqlStr: ", sqlStr)

	_, err := p.dbPool.Exec(context.Background(),
		sqlStr,
		args...,
	)

	if err != nil {
		return err
	}
	return nil
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

func (p *Permission) Clear() error {
	_, err := p.dbPool.Exec(context.Background(), `DELETE FROM role_permissions`)
	if err != nil {
		return err
	}

	_, err = p.dbPool.Exec(context.Background(), `DELETE FROM permissions`)
	if err != nil {
		return err
	}

	return nil
}
