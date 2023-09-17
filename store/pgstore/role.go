package pgstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Role struct {
	dbPool *pgxpool.Pool
}

func (r *Role) List(page, pageSize int) ([]*model.Role, error) {
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPage
	}

	rows, err := r.dbPool.Query(
		context.Background(),
		`SELECT id, name, front_id, created_at FROM roles ORDER BY created_at DESC`,
	)

	if err != nil {
		return nil, err
	}

	var list []*model.Role
	for rows.Next() {
		var item model.Role
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

func (r *Role) Create(frontId, name string, permissions []int) (int, error) {
	var id int
	err := r.dbPool.QueryRow(context.Background(), "INSERT INTO roles (front_id, name) VALUES ($1, $2) RETURNING (id)",
		frontId,
		name,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if len(permissions) > 0 {
		sqlStrHead := `INSERT INTO role_permissions (role_id, permission_id) VALUES `
		var strArr []string
		var args []any
		var argCount = 1
		for _, pId := range permissions {
			strArr = append(strArr, fmt.Sprintf("($%d, $%d)", argCount, argCount+1))
			args = append(args, id, pId)
			argCount += 2
		}
		sqlStr := sqlStrHead + strings.Join(strArr, ", ")
		fmt.Println("create role sql string: ", sqlStr)
		fmt.Println("create role args: ", args)

		_, err := r.dbPool.Exec(context.Background(), sqlStr, args...)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (r *Role) Update(name string) (int, error) {
	var id int
	err := r.dbPool.QueryRow(context.Background(), "UPDATE roles SET name = $1 WHERE id = $2 RETURNING (id)",
		name,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (r *Role) Item(id int) (*model.Role, error) {
	var item model.Role
	err := r.dbPool.QueryRow(context.Background(), "SELECT id, front_id, name, created_at FROM roles id = $1",
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

func (r *Role) Delete(id int) error {
	err := r.dbPool.QueryRow(context.Background(), "UPDATE roles SET deleted = true WHERE id = $1").Scan(nil)
	if err != nil {
		return err
	}
	return nil
}
