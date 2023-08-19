package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

type Article struct {
	dbPool *pgxpool.Pool
}

const defaultPage = 1
const defaultPageSize = 10

func (a *Article) List(page int, pageSize int) ([]*model.Article, error) {
	sqlStr := `
SELECT tp.id, tp.title, u.name as author_name, tp.author_id, tp.content, tp.created_at, tp.updated_at, (
	WITH RECURSIVE replies AS (
	   SELECT id
	   FROM posts
	   WHERE reply_to = tp.id AND deleted = false
	   UNION ALL
	   SELECT p.id
	   FROM posts p
	   INNER JOIN replies pr
	   ON p.reply_to = pr.id
	   WHERE p.deleted = false
	)
	SELECT COUNT(*)
	FROM replies
) AS total_reply_count
FROM posts tp
LEFT JOIN users u
ON u.id = tp.author_id
WHERE tp.reply_to = 0 AND tp.deleted = false
ORDER BY tp.created_at DESC
OFFSET $1
LIMIT $2;`

	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPage
	}

	// fmt.Println("page", page)
	// fmt.Println("pageSize", pageSize)

	rows, err := a.dbPool.Query(context.Background(), sqlStr, pageSize*(page-1), pageSize)

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
			&item.TotalReplyCount,
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

func (a *Article) Count() (int, error) {
	var count int
	err := a.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM posts WHERE reply_to = 0 AND deleted = false;`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (a *Article) Create(item *model.Article) (int, error) {
	var id int
	sqlStr := `
INSERT INTO posts (title, author_id, content, reply_to, root_article_id, depth)
VALUES (
    $1,
    $2,
    $3,
    $4,
    (
        CASE WHEN $4 = 0 THEN 0
	     WHEN (SELECT p.reply_to FROM posts p WHERE $4 = p.id) = 0 THEN $4
             ELSE (SELECT p.root_article_id FROM posts p WHERE $4 = p.id)
        END
    ),
    (
        CASE WHEN $4 = 0 THEN 0
             WHEN (SELECT p.reply_to FROM posts p WHERE $4 = p.id) = 0 THEN 1
             ELSE (SELECT p.depth + 1 FROM posts p WHERE $4 = p.id)
        END
    )
)
RETURNING (id);`
	err := a.dbPool.QueryRow(context.Background(), sqlStr,
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
	sqlStr := `UPDATE posts SET title = $1, content = $2, updated_at = current_timestamp WHERE id = $3 RETURNING (id)`
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
	sqlStr := `
SELECT p.id, p.title, u.name AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title as root_article_title, (
	WITH RECURSIVE replies AS (
	   SELECT id
	   FROM posts
	   WHERE reply_to = p.id AND deleted = false
	   UNION ALL
	   SELECT p1.id
	   FROM posts p1
	   INNER JOIN replies pr
	   ON p1.reply_to = pr.id
	   WHERE p1.deleted = false
	)
	SELECT COUNT(*)
	FROM replies
) AS total_reply_count
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
WHERE p.id = $1;`
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
		&item.NullReplyRootArticleTitle,
		&item.TotalReplyCount,
	)
	if err != nil {
		return nil, err
	}

	item.FormatNullValues()
	item.FormatTimeStr()
	return &item, nil
}

func (a *Article) ItemTree(id int) ([]*model.Article, error) {
	sqlStr := `
WITH RECURSIVE articleTree AS (
     SELECT id, title, author_id, content, created_at, updated_at, deleted, reply_to, depth, 0 AS cur_depth, root_article_id
     FROM posts
     WHERE id = $1
     UNION ALL
     SELECT p.id, p.title, p.author_id, p.content, p.created_at,p.updated_at, p.deleted, p.reply_to, p.depth, ar.cur_depth + 1, p.root_article_id
     FROM posts p
     JOIN articleTree ar
     ON p.reply_to = ar.id
     WHERE ar.cur_depth < $2
)
SELECT ar.id, ar.title, u.name as author_name, ar.author_id, ar.content, ar.created_at, ar.updated_at, ar.deleted, ar.reply_to, ar.depth, ar.root_article_id, p2.title as root_article_title, (
	WITH RECURSIVE replies AS (
	   SELECT id
	   FROM posts
	   WHERE reply_to = ar.id AND deleted = false
	   UNION ALL
	   SELECT p.id
	   FROM posts p
	   INNER JOIN replies pr
	   ON p.reply_to = pr.id
	   WHERE p.deleted = false
	)
	SELECT COUNT(*)
	FROM replies
) AS total_reply_count
FROM articleTree ar
JOIN users u ON ar.author_id = u.id
LEFT JOIN posts p2 ON ar.root_article_id = p2.id
ORDER BY ar.created_at;`

	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize())
	if err != nil {
		return nil, err
	}

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
			&item.NullReplyRootArticleTitle,
			&item.TotalReplyCount,
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

func (a *Article) Delete(id int, authorId int) (rootArticleId int, err error) {
	err = a.dbPool.QueryRow(context.Background(),
		"UPDATE posts SET deleted = true WHERE id = $1 AND author_id = $2 RETURNING (root_article_id)",
		id,
		authorId,
	).Scan(&rootArticleId)
	if rootArticleId == 0 {
		rootArticleId = id
	}
	return
}
