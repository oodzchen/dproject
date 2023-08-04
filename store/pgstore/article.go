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
	var id int
	err := a.dbPool.QueryRow(context.Background(), "insert into posts (title, author_id, content, reply_to) values ($1, $2, $3, $4) returning (id)",
		item.Title,
		item.AuthorId,
		item.Content,
		item.ReplyTo,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (a *Article) Update(item *model.Article) (int, error) {
	sqlStr := `update posts set title = $1, author_id = $2, content = $3, updated_at = current_timestamp where id = $4 returning (id)`
	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr,
		item.Title,
		item.AuthorId,
		item.Content,
		item.Id).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (a *Article) Item(id int) (*model.Article, error) {
	var item model.Article
	sqlStr := `select
		p.id,
		p.title,
		u.name as author_name,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at,
		p.deleted,
		p.reply_to,
		p2.title as reply_to_title
	from posts p
	left join users u on p.author_id = u.id
	left join posts p2 on p.reply_to = p2.id
	where p.id = $1`
	err := a.dbPool.QueryRow(context.Background(), sqlStr, id).Scan(
		&item.Id,
		&item.Title,
		&item.AuthorName,
		&item.AuthorId,
		&item.Content,
		&item.CreatedAt,
		&item.UpdatedAt,
		&item.Deleted,
		&item.ReplyTo,
		&item.NullReplyToTitle,
	)
	if err != nil {
		return nil, err
	}

	item.FormatNullValues()
	item.FormatTimeStr()
	return &item, nil
}

func (a *Article) GetReplies(id int) ([]*model.Article, error) {
	sqlStr := `with recursive recurPosts as (select
		p.id,
		p.title,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at,
		p.deleted,
		p.reply_to,
		1 as recur_depth
	from posts p where p.id = $1 and p.deleted = false
	union all
	select
		p.id,
		p.title,
		p.author_id,
		p.content,
		p.created_at,
		p.updated_at,
		p.deleted,
		p.reply_to,
		rp.recur_depth + 1
	from posts p
	join recurPosts rp on p.reply_to = rp.id
	where rp.recur_depth < 10 and p.deleted = false
)
select
	rp.id,
	rp.title,
	u.name as author_name,
	rp.author_id,
	rp.content,
	rp.created_at,
	rp.updated_at,
	rp.deleted,
	rp.reply_to,
	p2.title as reply_to_title
from recurPosts rp
left join posts p2 on rp.reply_to = p2.id
join users u on rp.author_id = u.id`

	rows, err := a.dbPool.Query(context.Background(), sqlStr, id)

	var list []*model.Article
	for rows.Next() {
		var item model.Article
		err = rows.Scan(
			&item.Id,
			&item.Title,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Deleted,
			&item.ReplyTo,
			&item.NullReplyToTitle,
		)

		if err != nil {
			return nil, err
		}

		// fmt.Printf("row item: %+v\n", &item)

		item.FormatNullValues()
		item.FormatTimeStr()
		list = append(list, &item)
	}

	return list, nil
}

func (a *Article) Delete(id int) error {
	err := a.dbPool.QueryRow(context.Background(), "update posts set deleted = true where id = $1 returning (id)", id).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}
