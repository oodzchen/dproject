package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Article struct {
	dbPool *pgxpool.Pool
}

func (a *Article) List() ([]*model.Article, error) {
	sqlStr := `select
		p.id,
		p.title,
		u.name as author_name,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at
	from posts p
	left join users u
	on p.author_id = u.id
	where p.reply_to = 0 and p.deleted = false;`
	// rows, err := rs.DBConn.Query(context.Background(), sqlStr)
	rows, err := a.dbPool.Query(context.Background(), sqlStr)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return nil, err
	}

	defer rows.Close()

	var list []*model.Article
	for rows.Next() {
		var item model.Article
		err := rows.Scan(
			&item.Id,
			&item.Title,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
		)
		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return nil, err
		}

		item.FormatTimeStr()
		list = append(list, &item)
	}

	return list, nil
}

func (a *Article) Create(item *model.Article) (int, error) {
	sqlStr := fmt.Sprintf(
		"insert into posts (title, author_id, content) values ('%s', %d, '%s') returning (id)",
		item.Title,
		item.AuthorId,
		item.Content)

	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (a *Article) Update(item *model.Article) (int, error) {
	sqlStr := fmt.Sprintf(
		"update posts set title = '%s', author_id = %d, content = '%s', updated_at = current_timestamp where id = %d returning (id)",
		item.Title,
		item.AuthorId,
		item.Content,
		item.Id,
	)

	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (a *Article) Item(id int) (*model.Article, error) {
	var item model.Article
	sqlStr := fmt.Sprintf(`select
		p.id,
		p.title,
		u.name as author_name,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at
	from posts p
	left join users u
	on p.author_id = u.id
	where p.id = %d`, id)
	err := a.dbPool.QueryRow(context.Background(), sqlStr).Scan(
		&item.Id,
		&item.Title,
		&item.AuthorName,
		&item.AuthorId,
		&item.Content,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	item.FormatTimeStr()
	// fmt.Printf("item: %v\n", item)
	return &item, nil
}

func (a *Article) Delete(id int) error {
	//...
	sqlStr := fmt.Sprintf("update posts set deleted = true where id = %d returning (id)", id)
	// fmt.Printf("sqlStr: %v\n", sqlStr)

	err := a.dbPool.QueryRow(context.Background(), sqlStr).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}
