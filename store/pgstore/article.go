package pgstore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

type Article struct {
	dbPool *pgxpool.Pool
}

func (a *Article) List(page, pageSize, userId int) ([]*model.Article, error) {
	sqlStr := `
SELECT tp.id, tp.title, u.name as author_name, tp.author_id, tp.content, tp.created_at, tp.updated_at, tp.depth, p2.title as root_article_title, (
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
) AS total_reply_count,
(
SELECT type FROM post_votes WHERE post_id = tp.id AND user_id = $3
) AS user_vote_type,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = tp.id AND type = 'up'
) AS vote_up,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = tp.id AND type = 'down'
) AS vote_down
FROM posts tp
LEFT JOIN posts p2 ON tp.root_article_id = p2.id
LEFT JOIN users u ON u.id = tp.author_id
WHERE tp.deleted = false AND tp.reply_to = 0
ORDER BY tp.created_at DESC
OFFSET $1
LIMIT $2;`

	var args []any
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 0 {
		args = []any{0, nil, userId}
	} else {
		if pageSize < 1 {
			pageSize = defaultPageSize
		}
		args = []any{pageSize * (page - 1), pageSize, userId}
	}

	// fmt.Println("page", page)
	// fmt.Println("pageSize", pageSize)
	fmt.Println("args: ", args)

	rows, err := a.dbPool.Query(context.Background(), sqlStr, args...)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return nil, err
	}

	defer rows.Close()

	var list []*model.Article
	for rows.Next() {
		var currUserState model.CurrUserState
		item := model.Article{
			CurrUserState: &currUserState,
		}
		// var item model.Article
		err := rows.Scan(
			&item.Id,
			&item.Title,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyDepth,
			&item.NullReplyRootArticleTitle,
			&item.TotalReplyCount,
			&item.CurrUserState.NullVoteType,
			&item.VoteUp,
			&item.VoteDown,
		)
		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return nil, err
		}

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

func (a *Article) Create(title, content string, authorId, replyTo int) (int, error) {
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
		title,
		authorId,
		content,
		replyTo,
	).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func validArticleUpdateField(key string) bool {
	allowedFields := []string{"Title", "Content", "Weight"}
	for _, field := range allowedFields {
		if key == field {
			return true
		}
	}
	return false
}

func (a *Article) Update(item *model.Article, fieldNames []string) (int, error) {
	for _, field := range fieldNames {
		if !validArticleUpdateField(field) {
			return 0, errors.New(fmt.Sprintf("'%s' is not allowed to update", field))
		}
	}

	var updateStr []string
	var updateVals []any
	itemVal := reflect.ValueOf(*item)

	dbFieldNameMap := map[string]string{
		"Title":   "title",
		"Content": "content",
		"Weight":  "weight",
	}
	for idx, field := range fieldNames {
		updateStr = append(updateStr, fmt.Sprintf("%s = $%d", dbFieldNameMap[field], idx+1))
		updateVals = append(updateVals, itemVal.FieldByName(field))
	}

	sqlStr := "UPDATE posts SET " + strings.Join(updateStr, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING(id)", len(updateStr)+1)
	updateVals = append(updateVals, item.Id)

	// fmt.Println("update sql string: ", sqlStr)
	// fmt.Println("update vals: ", updateVals)

	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr, updateVals...).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *Article) Item(id, userId int) (*model.Article, error) {
	var currUserState model.CurrUserState
	item := model.Article{
		CurrUserState: &currUserState,
	}
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
) AS total_reply_count,
(
SELECT type FROM post_votes WHERE post_id = p.id AND user_id = $2
) AS user_vote_type,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = p.id AND type = 'up'
) AS vote_up,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = p.id AND type = 'down'
) AS vote_down
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
WHERE p.id = $1;`
	err := a.dbPool.QueryRow(context.Background(), sqlStr, id, userId).Scan(
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
		&item.CurrUserState.NullVoteType,
		&item.VoteUp,
		&item.VoteDown,
	)
	if err != nil {
		return nil, err
	}

	item.FormatNullValues()
	item.FormatTimeStr()
	item.CalcScore()
	item.CalcWeight()
	return &item, nil
}

func (a *Article) ItemTree(id, userId int) ([]*model.Article, error) {
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
) AS total_reply_count,
(
SELECT type FROM post_votes WHERE post_id = ar.id AND user_id = $3
) AS user_vote_type,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = ar.id AND type = 'up'
) AS vote_up,
(
SELECT COUNT(*) FROM post_votes
WHERE post_id = ar.id AND type = 'down'
) AS vote_down
FROM articleTree ar
JOIN users u ON ar.author_id = u.id
LEFT JOIN posts p2 ON ar.root_article_id = p2.id
ORDER BY ar.created_at;`

	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), userId)
	if err != nil {
		return nil, err
	}

	var list []*model.Article
	for rows.Next() {
		var userState model.CurrUserState
		item := model.Article{
			CurrUserState: &userState,
		}

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
			&item.CurrUserState.NullVoteType,
			&item.VoteUp,
			&item.VoteDown,
		)

		if err != nil {
			return nil, err
		}

		// fmt.Printf("row item: %+v\n", &item)
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

func (a *Article) Vote(id, userId int, voteType string) error {
	err, vt := a.VoteCheck(id, userId)
	// fmt.Println("check error: ", err)
	// fmt.Println("check vote type: ", vt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// var aId int
			err = a.dbPool.QueryRow(
				context.Background(),
				`INSERT INTO post_votes (post_id, user_id, type) VALUES ($1, $2, $3) RETURNING (post_id, user_id)`,
				id,
				userId,
				voteType,
			).Scan(nil)

			// fmt.Println("after insert, aId: ", aId)
			// fmt.Println("after insert: ", err)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if vt == voteType {
			err = a.dbPool.QueryRow(
				context.Background(),
				`DELETE FROM post_votes WHERE post_id = $1 AND user_id = $2 RETURNING (post_id, user_id)`,
				id,
				userId,
			).Scan(nil)
			if err != nil {
				return err
			}
		} else {
			// fmt.Println("change vote type to: ", voteType)
			err = a.dbPool.QueryRow(
				context.Background(),
				`UPDATE post_votes SET type = $1 WHERE post_id = $2 AND user_id = $3 RETURNING (post_id, user_id)`,
				voteType,
				id,
				userId,
			).Scan(nil)

			// fmt.Println("after change vote type: ", err)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (a *Article) VoteCheck(id, userId int) (error, string) {
	var vt string
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT type FROM post_votes WHERE post_id = $1 AND user_id = $2`,
		id,
		userId,
	).Scan(&vt)
	if err != nil {
		return err, ""
	}

	return nil, vt
}
