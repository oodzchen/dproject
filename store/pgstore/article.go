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
	where p.reply_to = 0 and p.deleted = false
	order by p.created_at desc`
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
	sqlStr := `insert into posts (title, author_id, content, reply_to, depth, root_article_id) values ($1, $2, $3, $4, $5, $6) returning (id);`
	err := a.dbPool.QueryRow(context.Background(), sqlStr,
		item.Title,
		item.AuthorId,
		item.Content,
		item.ReplyTo,
		item.ReplyDepth,
		item.ReplyRootArticleId,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (a *Article) Update(item *model.Article) (int, error) {
	sqlStr := `update posts set title = $1, content = $2, updated_at = current_timestamp where id = $3 returning (id)`
	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr,
		item.Title,
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
		p.depth,
		p.root_article_id,
		p2.title as reply_to_title,
		p3.title as root_article_title
	from posts p
	left join users u on p.author_id = u.id
	left join posts p2 on p.reply_to = p2.id
	left join posts p3 on p.root_article_id = p3.id
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
		&item.ReplyDepth,
		&item.ReplyRootArticleId,
		&item.NullReplyToTitle,
		&item.NullReplyRootArticleTitle,
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
		id,
		title,
		author_id,
		content,
		created_at,
		updated_at,
		deleted,
		reply_to,
		depth,
		root_article_id,
		1 as recur_depth
	from posts where id = $1
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
		p.depth,
		p.root_article_id,
		rp.recur_depth + 1
	from posts p
	join recurPosts rp on p.reply_to = rp.id
	where rp.recur_depth < 11
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
	rp.depth,
	rp.root_article_id,
	p2.title as reply_to_title
from recurPosts rp
left join posts p2 on rp.reply_to = p2.id
join users u on rp.author_id = u.id
order by rp.created_at`

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
			&item.ReplyDepth,
			&item.ReplyRootArticleId,
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
