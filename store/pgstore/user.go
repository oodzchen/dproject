package pgstore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	dbPool *pgxpool.Pool
}

func (u *User) List(page, pageSize int, oldest bool) ([]*model.User, error) {
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPage
	}

	// 	sqlStr := `SELECT u.id, u.username, u.email, u.created_at, COALESCE(u.introduction, ''),
	// COALESCE(r.name, '') as role_name, COALESCE(r.front_id, '') AS role_front_id,
	// COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
	// FROM users u
	// LEFT JOIN user_roles ur ON ur.user_id = u.id
	// LEFT JOIN roles r ON ur.role_id = r.id
	// LEFT JOIN role_permissions rp ON rp.role_id = r.id
	// LEFT JOIN permissions p ON p.id = rp.permission_id
	// INNER JOIN (
	//     SELECT id FROM users
	//     ORDER BY
	//         CASE WHEN $3 = true THEN created_at END ASC,
	//         CASE WHEN $3 = false THEN created_at END DESC
	//     OFFSET $1 LIMIT $2
	// ) AS u1 ON u1.id = u.id;`

	sqlStr := `SELECT 
    u.id, u.username, u.email, u.created_at, 
    COALESCE(u.introduction, ''),
    COALESCE(r.name, '') as role_name, 
    COALESCE(r.front_id, '') AS role_front_id,
    COALESCE(p.id, 0) AS p_id, 
    COALESCE(p.name, '') AS p_name, 
    COALESCE(p.front_id, '') AS p_front_id, 
    COALESCE(p.module, 'user') AS p_module, 
    COALESCE(p.created_at, NOW()) AS p_created_at
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON ur.role_id = r.id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE u.id IN ( SELECT id FROM users OFFSET $1 LIMIT $2)`

	if oldest {
		sqlStr += ` ORDER BY u.created_at`
	} else {
		sqlStr += ` ORDER BY u.created_at DESC`
	}

	fmt.Println("user list sqlStr", sqlStr)

	rows, err := u.dbPool.Query(
		context.Background(),
		sqlStr,
		pageSize*(page-1),
		pageSize,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.User
	listMap := make(map[int]*model.User)

	for rows.Next() {
		var item model.User
		var pItem model.Permission
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.Email,
			&item.RegisteredAt,
			&item.Introduction,
			&item.RoleName,
			&item.RoleFrontId,
			&pItem.Id,
			&pItem.Name,
			&pItem.FrontId,
			&pItem.Module,
			&pItem.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		if v, ok := listMap[item.Id]; ok {
			if pItem.Id > 0 {
				if v.Permissions != nil {
					v.Permissions = append(v.Permissions, &pItem)
				} else {
					v.Permissions = []*model.Permission{&pItem}
				}
			}
		} else {
			if pItem.Id > 0 {
				item.Permissions = []*model.Permission{&pItem}
			}

			listMap[item.Id] = &item
			item.FormatTimeStr()
			list = append(list, &item)
		}
	}

	return list, nil
}

func (u *User) Count() (int, error) {
	var count int
	err := u.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM users;`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (u *User) Create(email, password, name string, roleFrontId string) (int, error) {
	// fmt.Printf("user.create item: %+v\n", item)
	var id int
	err := u.dbPool.QueryRow(context.Background(), "INSERT INTO users (email, password, username) VALUES ($1, $2, $3) RETURNING (id)",
		email,
		password,
		name).Scan(&id)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), "INSERT INTO user_roles (user_id, role_id) SELECT $1, id FROM roles WHERE front_id = $2", id, roleFrontId)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func validUserUpdateField(key string) bool {
	allowedFields := []string{"Introduction", "Banned"}
	for _, field := range allowedFields {
		if key == field {
			return true
		}
	}
	return false
}

func (u *User) Update(item *model.User, fieldNames []string) (int, error) {
	for _, field := range fieldNames {
		if !validUserUpdateField(field) {
			return 0, errors.New(fmt.Sprintf("'%s' is not allowed to update", field))
		}
	}

	var updateStr []string
	var updateVals []any
	itemVal := reflect.ValueOf(*item)

	dbFieldNameMap := map[string]string{
		"Introduction": "introduction",
		"Banned":       "banned",
	}
	for idx, field := range fieldNames {
		updateStr = append(updateStr, fmt.Sprintf("%s = $%d", dbFieldNameMap[field], idx+1))
		updateVals = append(updateVals, itemVal.FieldByName(field))
	}

	sqlStr := "UPDATE users SET " + strings.Join(updateStr, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING(id)", len(updateStr)+1)
	updateVals = append(updateVals, item.Id)

	fmt.Println("update sql string: ", sqlStr)
	fmt.Println("update vals: ", updateVals)

	var id int
	err := u.dbPool.QueryRow(context.Background(), sqlStr, updateVals...).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) Item(id int) (*model.User, error) {
	// fmt.Println("userId: ", id)

	sqlStr := `SELECT u.id, u.username, u.email, u.created_at, u.super_admin, COALESCE(u.introduction, '') as introduction,
COALESCE(r.name, '') as role_name, COALESCE(r.front_id, '') AS role_front_id,
COALESCE(p.id, 0) AS p_id, COALESCE(p.name, '') AS p_name, COALESCE(p.front_id, '') AS p_front_id, COALESCE(p.module, 'user') AS p_module, COALESCE(p.created_at, NOW()) AS p_created_at
FROM users u
LEFT JOIN user_roles ur ON ur.user_id = u.id
LEFT JOIN roles r ON ur.role_id = r.id
LEFT JOIN role_permissions rp ON rp.role_id = r.id
LEFT JOIN permissions p ON p.id = rp.permission_id
WHERE u.id = $1`

	rows, err := u.dbPool.Query(context.Background(), sqlStr, id)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var item model.User
	for rows.Next() {
		var uItem model.User
		var pItem model.Permission

		rows.Scan(
			&uItem.Id,
			&uItem.Name,
			&uItem.Email,
			&uItem.RegisteredAt,
			&uItem.Super,
			&uItem.Introduction,
			&uItem.RoleName,
			&uItem.RoleFrontId,
			&pItem.Id,
			&pItem.Name,
			&pItem.FrontId,
			&pItem.Module,
			&pItem.CreatedAt,
		)

		if item.Id < 1 {
			item = uItem
		}

		// fmt.Println("inscan:", item.Id)

		if pItem.Id != 0 {
			if item.Permissions != nil {
				item.Permissions = append(item.Permissions, &pItem)
			} else {
				item.Permissions = []*model.Permission{&pItem}
			}
		}
	}

	if item.Id < 1 {
		return nil, model.ErrUserNotExist
	}

	// fmt.Println("user data: ", item)

	item.FormatTimeStr()
	item.FormatNullVals()

	return &item, nil
}

func (u *User) Delete(id int) error {
	err := u.dbPool.QueryRow(context.Background(), "UPDATE users SET deleted = true WHERE id = $1", id).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Ban(id int) error {
	err := u.dbPool.QueryRow(context.Background(), "UPDATE users SET banned = true WHERE id = $1", id).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Login(username string, pwd string) (int, error) {
	var id int
	var hasedPwd string
	var isEmail = false

	if regexp.MustCompile(`@`).Match([]byte(username)) {
		isEmail = true
	}

	sqlStr := `SELECT id, password FROM users `
	if isEmail {
		sqlStr += "WHERE email = $1"
	} else {
		sqlStr += "WHERE username = $1"
	}

	err := u.dbPool.QueryRow(context.Background(), sqlStr, username).Scan(&id, &hasedPwd)
	if err != nil {
		return 0, err
	}

	// fmt.Printf("query result: password: %s\n", hasedPwd)
	// fmt.Printf("query result: id: %d\n", id)

	err = bcrypt.CompareHashAndPassword([]byte(hasedPwd), []byte(pwd))

	if err != nil {
		return 0, err
	}

	// fmt.Println("pass! user id: ", id)

	return id, nil
}

func (u *User) GetPosts(userId int, listType string) ([]*model.Article, error) {
	sqlStrHead := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u.username AS author_name,
p.depth,
p3.title AS root_article_title
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
WHERE p.author_id = $1 AND p.deleted = false`
	sqlStrTail := ` ORDER BY p.created_at DESC`

	switch listType {
	case "article":
		sqlStrHead += ` AND p.reply_to = 0`
	case "reply":
		sqlStrHead += ` AND p.reply_to != 0`
	default:
	}

	sqlStr := sqlStrHead + sqlStrTail

	rows, err := u.dbPool.Query(context.Background(), sqlStr, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyTo,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
		)

		if err != nil {
			fmt.Printf("query user's posts error: %v", err)
			return nil, err
		}

		item.FormatNullValues()
		item.FormatTimeStr()

		posts = append(posts, &item)
	}

	return posts, nil
}

func (u *User) GetSavedPosts(userId int) ([]*model.Article, error) {
	sqlStr := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u.username AS author_name,
p.depth,
p3.title AS root_article_title
FROM post_saves ps
LEFT JOIN posts p ON p.id = ps.post_id
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
WHERE ps.user_id = $1 AND p.deleted = false
ORDER BY ps.created_at DESC`
	rows, err := u.dbPool.Query(context.Background(), sqlStr, userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var posts []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.NullTitle,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyTo,
			&item.AuthorId,
			&item.AuthorName,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
		)

		if err != nil {
			fmt.Printf("query user's posts error: %v", err)
			return nil, err
		}

		item.FormatNullValues()
		item.FormatTimeStr()

		posts = append(posts, &item)
	}

	return posts, nil
}

func (u *User) SetRole(userId int, roleFrontId string) (int, error) {
	var roleId int
	err := u.dbPool.QueryRow(context.Background(), `SELECT id FROM roles WHERE front_id = $1`, roleFrontId).Scan(&roleId)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), `DELETE FROM user_roles WHERE user_id = $1`, userId)
	if err != nil {
		return 0, err
	}

	_, err = u.dbPool.Exec(context.Background(), `INSERT INTO user_roles (user_id, role_id) VALUES ($1, $2)`, userId, roleId)
	if err != nil {
		return 0, err
	}

	return userId, nil
}

func (u *User) SetRoleManyWithFrontId(list []*model.User) error {
	sqlStr := `INSERT INTO user_roles (user_id, role_id) `
	var args []any
	var argCount = 1

	for idx, item := range list {
		if item.Id < 1 || item.RoleFrontId == "" {
			continue
		}

		if idx == 0 {
			sqlStr += fmt.Sprintf("\nSELECT $%d::int, r.id FROM roles r WHERE r.front_id = $%d \n", argCount, argCount+1)
		} else {
			sqlStr += fmt.Sprintf("UNION ALL SELECT $%d::int, r.id FROM roles r WHERE r.front_id = $%d \n", argCount, argCount+1)
		}
		args = append(args, item.Id, item.RoleFrontId)
		argCount += 2
	}

	// fmt.Println("set many roles sql string: \n", sqlStr)
	// fmt.Println("set many roles args: ", args)
	// fmt.Println("set many roles args length: ", len(args))

	_, err := u.dbPool.Exec(context.Background(), sqlStr, args...)
	if err != nil {
		return err
	}

	return nil
}
