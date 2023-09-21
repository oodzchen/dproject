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
SELECT r.id, r.name, r.front_id, r.created_at, r.is_default, COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
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

	defer rows.Close()

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
			&item.IsDefault,
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

func (r *Role) CreateWithFrontId(frontId, name string, permissionFrontIds []string) (int, error) {
	var id int
	err := r.dbPool.QueryRow(context.Background(), "INSERT INTO roles (front_id, name) VALUES ($1, $2) RETURNING (id)",
		frontId,
		name,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	if len(permissionFrontIds) > 0 {
		sqlStrHead := `INSERT INTO role_permissions (role_id, permission_id) `
		var strArr []string
		var args []any
		var argCount = 1
		for _, pFrontId := range permissionFrontIds {
			strArr = append(strArr, fmt.Sprintf("SELECT $%d::int, id FROM permissions WHERE front_id = $%d", argCount, argCount+1))
			args = append(args, id, pFrontId)
			argCount += 2
		}
		sqlStr := sqlStrHead + strings.Join(strArr, " UNION ALL ")
		fmt.Println("create role sql string: ", sqlStr)
		fmt.Println("create role args: ", args)

		_, err := r.dbPool.Exec(context.Background(), sqlStr, args...)
		if err != nil {
			return 0, err
		}
	}

	return id, nil
}

func (r *Role) CreateManyWithFrontId(list []*model.Role) error {
	sqlStrHead := `INSERT INTO roles (front_id, name, is_default) VALUES `
	sqlStrTail := ` RETURNING (id)`

	var strArr []string
	var args []any
	var argCount = 1

	for _, item := range list {
		strArr = append(strArr, fmt.Sprintf("($%d, $%d, true)", argCount, argCount+1))
		args = append(args, item.FrontId, item.Name)
		argCount += 2
	}

	sqlStr := sqlStrHead + strings.Join(strArr, ", ") + sqlStrTail

	// fmt.Println("create roles sqlStr: ", sqlStr)
	// fmt.Println("create roles args: ", args)

	rows, err := r.dbPool.Query(context.Background(),
		sqlStr,
		args...,
	)

	defer rows.Close()

	var roleIdList []int
	for rows.Next() {
		var roleId int
		err := rows.Scan(&roleId)
		if err != nil {
			return err
		}

		roleIdList = append(roleIdList, roleId)
	}

	// fmt.Println("list length: ", len(list))
	// fmt.Println("roleIdList length: ", len(roleIdList))

	for roleIdx, item := range list {
		if item.Permissions == nil || len(item.Permissions) == 0 {
			continue
		}

		strArr = []string{}
		args = []any{}
		argCount = 1

		sqlStrHead := fmt.Sprintf("INSERT INTO role_permissions (role_id, permission_id) SELECT $%d, p.id FROM permissions p WHERE p.front_id IN (", argCount)
		sqlStrTail := ");\n"
		args = append(args, roleIdList[roleIdx])
		argCount += 1

		var pArr []string
		for _, pItem := range item.Permissions {
			pArr = append(pArr, fmt.Sprintf("$%d", argCount))
			args = append(args, pItem.FrontId)
			argCount += 1

		}
		// strArr = append(strArr, sqlStrHead+strings.Join(pArr, ", ")+sqlStrTail)
		sqlStr = sqlStrHead + strings.Join(pArr, ", ") + sqlStrTail

		// fmt.Println("create many role permissions sql: ", sqlStr)
		// fmt.Println("create many role permissions args: ", args)
		// fmt.Println("create many role permissions args len: ", len(args))

		_, err = r.dbPool.Exec(context.Background(), sqlStr, args...)

		if err != nil {
			// fmt.Println("error: ", err)
			return err
		}
	}
	return nil
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

func (r *Role) UpdateWithFrontId(id int, name string, permissionFrontIds []string) (int, error) {
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

	if len(permissionFrontIds) > 0 {
		sqlStrHead := `INSERT INTO role_permissions (role_id, permission_id) `
		var strArr []string
		var args []any
		var argCount = 1

		for _, pFrontId := range permissionFrontIds {
			// strArr = append(strArr, fmt.Sprintf("($%d, $%d)", argCount, argCount+1))
			strArr = append(strArr, fmt.Sprintf("SELECT $%d::int, id FROM permissions WHERE front_id = $%d", argCount, argCount+1))
			args = append(args, id, pFrontId)
			argCount += 2
		}

		sqlStr := sqlStrHead + strings.Join(strArr, " UNION ALL ")

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
SELECT r.id, r.name, r.front_id, r.created_at, r.is_default, COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
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

	defer rows.Close()

	var item model.Role
	for rows.Next() {
		var pItem model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.FrontId,
			&item.CreatedAt,
			&item.IsDefault,
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
