package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Category struct {
	dbPool *pgxpool.Pool
}

func (p *Category) List(state model.CategoryState) ([]*model.Category, error) {
	sqlStr := `SELECT id, front_id, name, COALESCE(describe, ''), author_id, approved, COALESCE(approval_comment, ''), created_at FROM categories`
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

func (p *Category) Item(frontId string, userId int) (*model.Category, error) {
	fmt.Println("category front id:", frontId)
	fmt.Println("user id:", userId)

	var item model.Category
	var userState model.CategoryUserState

	sqlStr := `SELECT
c.id,
c.front_id,
c.name,
COALESCE(c.describe, ''),
c.author_id,
c.approved,
COALESCE(c.approval_comment, ''),
c.created_at,
(
  SELECT EXISTS (
    SELECT 1 FROM category_subs cs WHERE cs.category_id = c.id AND cs.user_id = $2
  )
) AS subscribed
FROM categories c
WHERE c.front_id = $1`

	// fmt.Println("category item sql:", sqlStr)

	err := p.dbPool.QueryRow(context.Background(), sqlStr,
		frontId,
		userId,
	).Scan(
		&item.Id,
		&item.FrontId,
		&item.Name,
		&item.Describe,
		&item.AuthorId,
		&item.Approved,
		&item.ApprovalComment,
		&item.CreatedAt,

		&userState.Subscribed,
	)
	if err != nil {
		return nil, err
	}
	item.UserState = &userState

	return &item, nil
}

func (p *Category) Approval(frontId string, pass bool, comment string) error {
	return nil
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

func (c *Category) Subscribe(frontId string, loginedUserId int) error {
	err, subscribed := c.subscribeCheck(frontId, loginedUserId)
	if err != nil {
		return err
	}

	sqlStr := `INSERT INTO category_subs (category_id, user_id) VALUES
(
  (
    SELECT c.id FROM categories c WHERE c.front_id = $1
  ), $2
)`

	if subscribed {
		sqlStr = `DELETE FROM category_subs
WHERE category_id = (
  SELECT c.id FROM categories c WHERE c.front_id = $1
) AND user_id = $2`
	}

	// fmt.Println("subscribe category sql:", sqlStr)

	_, err = c.dbPool.Exec(context.Background(), sqlStr, frontId, loginedUserId)
	if err != nil {
		return err
	}

	return nil
}

func (c *Category) subscribeCheck(frontId string, userId int) (error, bool) {
	var count int
	err := c.dbPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*)
FROM category_subs
LEFT JOIN categories c ON c.front_id = $1
WHERE category_id = c.id AND user_id = $2`,
		frontId,
		userId,
	).Scan(&count)
	if err != nil {
		return err, false
	}

	return nil, count > 0
}

func (c *Category) Notify(sourceCateogryFrontId string, senderUserId, contentArticleId int) error {
	sqlStr := `
INSERT INTO messages (sender_id, reciever_id, source_category_id, content_id, type)
SELECT $1, cs.user_id, c.id, $3, 'category' FROM category_subs cs
LEFT JOIN categories c ON c.front_id = $2
WHERE cs.category_id = c.id AND cs.user_id != $1
`
	_, err := c.dbPool.Exec(context.Background(), sqlStr, senderUserId, sourceCateogryFrontId, contentArticleId)

	if err != nil {
		return err
	}

	return nil
}
