package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	dbPool *pgxpool.Pool
}

func (u *User) List() ([]*model.User, error) {
	rows, err := u.dbPool.Query(context.Background(), "SELECT id, name, email, created_at FROM users ORDER BY created_at desc")

	if err != nil {
		return nil, err
	}

	var list []*model.User
	for rows.Next() {
		var item model.User
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.Email,
			&item.RegisteredAt,
		)

		if err != nil {
			return nil, err
		}

		item.FormatTimeStr()

		list = append(list, &item)
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

func (u *User) Create(item *model.User) (int, error) {
	// fmt.Printf("user.create item: %+v\n", item)
	var id int
	err := u.dbPool.QueryRow(context.Background(), "INSERT INTO users (email, password, name) VALUES ($1, $2, $3) RETURNING (id)",
		item.Email,
		item.Password,
		item.Name).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (u *User) Update(item *model.User) (int, error) {
	var id int
	err := u.dbPool.QueryRow(context.Background(), "UPDATE users SET introduction = $1, password = $2 WHERE id = $3",
		item.Introduction,
		item.Password,
		item.Id).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) Item(id int) (*model.User, error) {
	var item model.User
	err := u.dbPool.QueryRow(context.Background(), "SELECT id, email, name, created_at FROM users WHERE id = $1", id).Scan(
		&item.Id,
		&item.Email,
		&item.Name,
		&item.RegisteredAt)
	if err != nil {
		return nil, err
	}

	item.FormatTimeStr()

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

func (u *User) Login(email string, pwd string) (int, error) {
	var id int
	var hasedPwd string
	err := u.dbPool.QueryRow(context.Background(), "SELECT id, password FROM users WHERE email = $1", email).Scan(&id, &hasedPwd)
	if err != nil {
		return 0, err
	}

	// fmt.Printf("query result: password: %s\n", hasedPwd)
	// fmt.Printf("query result: id: %d\n", id)

	err = bcrypt.CompareHashAndPassword([]byte(hasedPwd), []byte(pwd))

	if err != nil {
		return 0, err
	}

	fmt.Printf("pass!\n")

	return id, nil
}

func (u *User) GetPosts(userId int) ([]*model.Article, error) {
	sqlStr := `
SELECT
p.id,
p.title,
p.content,
p.created_at,
p.updated_at,
p.reply_to,
p.author_id,
u.name AS author_name,
p.depth,
p3.title AS root_article_title
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p3 ON p.root_article_id = p3.id
WHERE p.author_id = $1 AND p.deleted = false
ORDER BY p.created_at DESC`
	rows, err := u.dbPool.Query(context.Background(), sqlStr, userId)
	if err != nil {
		return nil, err
	}

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
