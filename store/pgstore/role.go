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

	sqlStr := `
SELECT r.id, r.name, r.front_id, r.created_at, COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON rp.permission_id = p.id
ORDER BY r.created_at DESC`

	rows, err := r.dbPool.Query(
		context.Background(),
		sqlStr,
	)

	if err != nil {
		return nil, err
	}

	var list []*model.Role
	listMap := make(map[int]*model.Role)

	for rows.Next() {
		var item model.Role
		var pItem model.Permission

		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.FrontId,
			&item.CreatedAt,
			&pItem.Id,
			&pItem.Name,
			&pItem.FrontId,
			&pItem.Module,
			&pItem.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		// list = append(list, &item)
		if v, ok := listMap[item.Id]; ok {
			if pItem.Id != 0 {
				if v.Permissions == nil {
					v.Permissions = []*model.Permission{&pItem}
				} else {
					v.Permissions = append(v.Permissions, &pItem)
				}
			}
		} else {
			if pItem.Id != 0 {
				item.Permissions = []*model.Permission{&pItem}
			}
			listMap[item.Id] = &item
			list = append(list, &item)
		}
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
		// fmt.Println("create role sql string: ", sqlStr)
		// fmt.Println("create role args: ", args)

		_, err := r.dbPool.Exec(context.Background(), sqlStr, args...)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (r *Role) Update(id int, name string, permissions []int) (int, error) {
	_, err := r.dbPool.Exec(context.Background(), "UPDATE roles SET name = $1 WHERE id = $2",
		name,
		id,
	)

	if err != nil {
		return 0, err
	}

	_, err = r.dbPool.Exec(context.Background(), "DELETE FROM role_permissions WHERE role_id = $1",
		id,
	)

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

		_, err = r.dbPool.Exec(context.Background(), sqlStr,
			args...,
		)

		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (r *Role) Item(id int) (*model.Role, error) {
	sqlStr := `
SELECT r.id, r.name, r.front_id, r.created_at, COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
FROM roles r
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON rp.permission_id = p.id
WHERE r.id = $1`

	rows, err := r.dbPool.Query(context.Background(), sqlStr,
		id,
	)
	if err != nil {
		return nil, err
	}

	var item model.Role
	for rows.Next() {
		var pItem model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.FrontId,
			&item.CreatedAt,
			&pItem.Id,
			&pItem.Name,
			&pItem.FrontId,
			&pItem.Module,
			&pItem.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if pItem.Id != 0 {
			if item.Permissions == nil {
				item.Permissions = []*model.Permission{&pItem}
			} else {
				item.Permissions = append(item.Permissions, &pItem)
			}
		}
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
