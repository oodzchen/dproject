package pgstore

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

type Article struct {
	dbPool *pgxpool.Pool
}

func (a *Article) List(page, pageSize, userId int, sortType model.ArticleSortType) ([]*model.Article, int, error) {
	// 	sqlStr := `
	// SELECT tp.id, tp.title, COALESCE(tp.url, ''), u.username as author_name, tp.author_id, tp.content, tp.created_at, tp.updated_at, tp.depth, p2.title as root_article_title, (
	// 	WITH RECURSIVE replies AS (
	// 	   SELECT id
	// 	   FROM posts
	// 	   WHERE reply_to = tp.id AND deleted = false
	// 	   UNION ALL
	// 	   SELECT p.id
	// 	   FROM posts p
	// 	   INNER JOIN replies pr
	// 	   ON p.reply_to = pr.id
	// 	   WHERE p.deleted = false
	// 	)
	// 	SELECT COUNT(id)
	// 	FROM replies
	// ) AS total_reply_count,
	// (
	// SELECT type FROM post_votes WHERE post_id = tp.id AND user_id = $3
	// ) AS user_vote_type,
	// (
	// SELECT COUNT(post_id) FROM post_votes
	// WHERE post_id = tp.id AND type = 'up'
	// ) AS vote_up,
	// (
	// SELECT COUNT(post_id) FROM post_votes
	// WHERE post_id = tp.id AND type = 'down'
	// ) AS vote_down,
	// (SELECT COUNT(user_id) FROM (
	//   WITH RECURSIVE postTree AS (
	//     SELECT id, author_id FROM posts WHERE reply_to = tp.id
	//     UNION ALL
	//     SELECT p1.id, p1.author_id FROM posts p1
	//     JOIN postTree pt
	//     ON p1.reply_to = pt.id
	//   )
	//   SELECT user_id
	//     FROM (
	//       SELECT user_id FROM post_votes WHERE post_id = tp.id
	//       UNION ALL
	//       SELECT user_id FROM post_saves WHERE post_id = tp.id
	//       UNION ALL
	//       SELECT user_id FROM post_reacts WHERE post_id = tp.id
	//       UNION ALL
	//       SELECT author_id AS user_id FROM postTree
	//     ) AS p_users GROUP BY user_id
	// ) AS participate_count)
	// FROM posts tp
	// LEFT JOIN posts p2 ON tp.root_article_id = p2.id
	// LEFT JOIN users u ON u.id = tp.author_id
	// WHERE tp.deleted = false AND tp.reply_to = 0
	// ORDER BY tp.created_at DESC
	// OFFSET $1
	// LIMIT $2;`

	var orderSqlStrHead, orderSqlStrTail string
	switch sortType {
	case model.ListSortBest:
		orderSqlStrHead = ` ORDER BY list_weight DESC `
		orderSqlStrTail = ` ORDER BY tp.list_weight DESC `
	case model.ListSortHot:
		orderSqlStrHead = ` ORDER BY participate_count DESC `
		orderSqlStrTail = ` ORDER BY tp.participate_count DESC `
	case model.ListSortLatest:
		fallthrough
	default:
		orderSqlStrHead = ` ORDER BY created_at DESC `
		orderSqlStrTail = ` ORDER BY tp.created_at DESC `
	}

	sqlStr := `
WITH postIds AS (
    SELECT id, COUNT(id) OVER() AS total FROM posts
    WHERE deleted = false AND reply_to = 0` + orderSqlStrHead + `
    OFFSET $1
    LIMIT $2
)
SELECT tp.id, tp.title, COALESCE(tp.url, ''), u.username as author_name, tp.author_id, tp.content, tp.created_at, tp.updated_at, tp.depth, tp.list_weight, tp.reply_weight, tp.participate_count, p2.title as root_article_title, COUNT(p3.id) AS total_reply_count,
(
SELECT type FROM post_votes WHERE post_id = tp.id AND user_id = $3
) AS user_vote_type,
(
SELECT COUNT(post_id) FROM post_votes
WHERE post_id = tp.id AND type = 'up'
) AS vote_up,
(
SELECT COUNT(post_id) FROM post_votes
WHERE post_id = tp.id AND type = 'down'
) AS vote_down,
pi.total AS total
FROM posts tp
JOIN postIds pi ON tp.id = pi.id
LEFT JOIN posts p2 ON tp.root_article_id = p2.id
LEFT JOIN posts p3 ON tp.id = p3.root_article_id AND p3.deleted = false
LEFT JOIN users u ON u.id = tp.author_id
GROUP BY tp.id, u.username, p2.title, pi.total` + orderSqlStrTail

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
	// fmt.Println("args: ", args)
	// fmt.Println("article list sql: ", sqlStr)

	rows, err := a.dbPool.Query(context.Background(), sqlStr, args...)

	if err != nil {
		fmt.Printf("Query database error: %v\n", err)
		return nil, 0, err
	}

	defer rows.Close()

	var total int
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
			&item.Link,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.ReplyDepth,
			&item.ListWeight,
			&item.Weight,
			&item.ParticipateCount,
			&item.NullReplyRootArticleTitle,
			&item.TotalReplyCount,
			&item.CurrUserState.NullVoteType,
			&item.VoteUp,
			&item.VoteDown,
			&total,
		)
		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return nil, 0, err
		}

		list = append(list, &item)
	}

	return list, total, nil
}

func (a *Article) Count() (int, error) {
	var count int
	err := a.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM posts WHERE reply_to = 0 AND deleted = false;`).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (a *Article) ListLatestCount(start, end time.Time) (int, error) {
	var count int
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM posts WHERE reply_to = 0 AND deleted = false AND created_at BETWEEN $1 AND $2;`,
		start,
		end,
	).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (a *Article) CountTotalReply(id int) (int, error) {
	var count int
	sqlStr := `
WITH RECURSIVE replyTree AS(
  SELECT id FROM posts WHERE reply_to = $1 AND deleted = false
  UNION ALL
  SELECT p1.id FROM posts p1
  JOIN replyTree rt ON p1.reply_to = rt.id AND p1.deleted = false
)
SELECT COUNT(*) FROM replyTree`

	err := a.dbPool.QueryRow(context.Background(), sqlStr, id).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (a *Article) Create(title, url, content string, authorId, replyTo int) (int, error) {
	var id int
	sqlStr := `
INSERT INTO posts (title, author_id, content, reply_to, url, root_article_id, depth)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
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
		url,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	var wg sync.WaitGroup
	var ch = make(chan error, 2)

	wg.Add(2)

	go func() {
		defer wg.Done()
		err = a.updateWeights(id)
		if err != nil {
			ch <- err
		}
	}()

	if replyTo == 0 {
		wg.Done()
	} else {
		go func() {
			defer wg.Done()
			err = a.updateWeights(replyTo)
			if err != nil {
				ch <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for res := range ch {
		switch v := res.(type) {
		case error:
			err = v
		}
	}

	if err != nil {
		return 0, err
	}

	return id, nil
}

func validArticleUpdateField(key string) bool {
	allowedFields := []string{"Title", "Content", "Weight", "Link"}
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
		"Link":    "url",
	}
	for idx, field := range fieldNames {
		updateStr = append(updateStr, fmt.Sprintf("%s = $%d", dbFieldNameMap[field], idx+1))
		updateVals = append(updateVals, itemVal.FieldByName(field))
	}

	updateStr = append(updateStr, "updated_at = NOW()")

	updateVals = append(updateVals, item.Id)
	sqlStr := "UPDATE posts SET " + strings.Join(updateStr, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING(id)", len(updateVals))

	// fmt.Println("update sql string: ", sqlStr)
	// fmt.Println("update vals: ", updateVals)

	var id int
	err := a.dbPool.QueryRow(context.Background(), sqlStr, updateVals...).Scan(&id)
	if err != nil {
		return 0, err
	}

	err = a.updateWeights(id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *Article) Item(id, userId int) (*model.Article, error) {
	// 	sqlStr := `
	// SELECT p.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title as root_article_title, 0 AS total_reply_count,
	// (
	// SELECT type FROM post_votes WHERE post_id = p.id AND user_id = $2
	// ) AS user_vote_type,
	// (
	// SELECT COUNT(*) FROM post_votes
	// WHERE post_id = p.id AND type = 'up'
	// ) AS vote_up,
	// (
	// SELECT COUNT(*) FROM post_votes
	// WHERE post_id = p.id AND type = 'down'
	// ) AS vote_down,
	// (
	// SELECT
	//   CASE
	//     WHEN COUNT(*) > 0 THEN TRUE
	//     ELSE FALSE
	//   END
	//  FROM post_saves WHERE post_id = p.id AND user_id = $2
	// ) AS saved,
	// (
	// SELECT
	//   CASE
	//     WHEN COUNT(*) > 0 THEN TRUE
	//     ELSE FALSE
	//   END
	//  FROM post_subs WHERE post_id = p.id AND user_id = $2
	// ) AS subscribed,
	// (
	// SELECT type FROM post_reacts WHERE post_id = p.id AND user_id = $2
	// ) AS user_react_type,
	// COALESCE(reacts.grinning, 0),
	// COALESCE(reacts.confused, 0),
	// COALESCE(reacts.eyes, 0),
	// COALESCE(reacts.party, 0),
	// COALESCE(reacts.thanks, 0)
	// FROM posts p
	// LEFT OUTER JOIN(
	//  SELECT post_id,
	//    SUM(CASE WHEN type = 'grinning' THEN count ELSE 0 END) AS grinning,
	//    SUM(CASE WHEN type = 'confused' THEN count ELSE 0 END) AS confused,
	//    SUM(CASE WHEN type = 'eyes' THEN count ELSE 0 END) AS eyes,
	//    SUM(CASE WHEN type = 'party' THEN count ELSE 0 END) AS party,
	//    SUM(CASE WHEN type = 'thanks' THEN count ELSE 0 END) AS thanks
	//    FROM
	//    (
	//      SELECT post_id, type, COUNT(*) AS count FROM post_reacts
	//      GROUP BY type, post_id
	//    ) AS react_types GROUP BY post_id
	// ) AS reacts ON reacts.post_id = p.id
	// LEFT JOIN users u ON p.author_id = u.id
	// LEFT JOIN posts p2 ON p.root_article_id = p2.id
	// WHERE p.id = $1;`

	sqlStr := `
SELECT p.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title as root_article_title, 0 AS total_reply_count,
pv.type AS user_vote_type,
COUNT(pv1.id) AS vote_up,
COUNT(pv2.id) AS vote_down,
(
SELECT
  CASE
    WHEN COUNT(*) > 0 THEN TRUE
    ELSE FALSE
  END
 FROM post_saves WHERE post_id = p.id AND user_id = $2
) AS saved,
(
SELECT
  CASE
    WHEN COUNT(*) > 0 THEN TRUE
    ELSE FALSE
  END
 FROM post_subs WHERE post_id = p.id AND user_id = $2
) AS subscribed
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN post_votes pv ON pv.post_id = p.id AND pv.user_id = $2
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
WHERE p.id = $1
GROUP BY p.id, p.title, p.url, u.username, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title, pv.type;`

	var userState model.CurrUserState
	item := model.Article{
		CurrUserState: &userState,
	}
	err := a.dbPool.QueryRow(context.Background(), sqlStr, id, userId).Scan(
		&item.Id,
		&item.Title,
		&item.Link,
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
		&item.CurrUserState.Saved,
		&item.CurrUserState.Subscribed,
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

func (a *Article) ItemTree(page, pageSize, id, userId int, sortType model.ArticleSortType) ([]*model.Article, error) {
	// 	sqlStr := `
	// WITH RECURSIVE articleTree AS (
	//      SELECT id, title, url, author_id, content, created_at, updated_at, deleted, reply_to, depth, 0 AS cur_depth, root_article_id
	//      FROM posts
	//      WHERE id = $1
	//      UNION ALL
	//      SELECT p.id, p.title, p.url, p.author_id, p.content, p.created_at,p.updated_at, p.deleted, p.reply_to, p.depth, ar.cur_depth + 1, p.root_article_id
	//      FROM posts p
	//      JOIN articleTree ar
	//      ON p.reply_to = ar.id
	//      WHERE ar.cur_depth < $2
	// )
	// SELECT ar.id, ar.title, COALESCE(ar.url, ''), u.username as author_name, ar.author_id, ar.content, ar.created_at, ar.updated_at, ar.deleted, ar.reply_to, ar.depth, ar.root_article_id, p2.title as root_article_title, (
	// 	WITH RECURSIVE replies AS (
	// 	   SELECT id
	// 	   FROM posts
	// 	   WHERE reply_to = ar.id AND deleted = false
	// 	   UNION ALL
	// 	   SELECT p.id
	// 	   FROM posts p
	// 	   INNER JOIN replies pr
	// 	   ON p.reply_to = pr.id
	// 	   WHERE p.deleted = false
	// 	)
	// 	SELECT COUNT(*)
	// 	FROM replies
	// ) AS total_reply_count,
	// (
	// SELECT type FROM post_votes WHERE post_id = ar.id AND user_id = $3
	// ) AS user_vote_type,
	// (
	// SELECT COUNT(*) FROM post_votes
	// WHERE post_id = ar.id AND type = 'up'
	// ) AS vote_up,
	// (
	// SELECT COUNT(*) FROM post_votes
	// WHERE post_id = ar.id AND type = 'down'
	// ) AS vote_down,
	// (
	// SELECT
	//   CASE
	//     WHEN COUNT(*) > 0 THEN TRUE
	//     ELSE FALSE
	//   END
	//  FROM post_saves WHERE post_id = ar.id AND user_id = $3
	// ) AS saved,
	// (
	// SELECT
	//   CASE
	//     WHEN COUNT(*) > 0 THEN TRUE
	//     ELSE FALSE
	//   END
	//  FROM post_subs WHERE post_id = ar.id AND user_id = $3
	// ) AS subscribed,
	// (
	// SELECT type FROM post_reacts WHERE post_id = ar.id AND user_id = $3
	// ) AS user_react_type,
	// COALESCE(reacts.grinning, 0),
	// COALESCE(reacts.confused, 0),
	// COALESCE(reacts.eyes, 0),
	// COALESCE(reacts.party, 0),
	// COALESCE(reacts.thanks, 0)
	// FROM articleTree ar
	// LEFT OUTER JOIN(
	//  SELECT post_id,
	//    SUM(CASE WHEN type = 'grinning' THEN count ELSE 0 END) AS grinning,
	//    SUM(CASE WHEN type = 'confused' THEN count ELSE 0 END) AS confused,
	//    SUM(CASE WHEN type = 'eyes' THEN count ELSE 0 END) AS eyes,
	//    SUM(CASE WHEN type = 'party' THEN count ELSE 0 END) AS party,
	//    SUM(CASE WHEN type = 'thanks' THEN count ELSE 0 END) AS thanks
	//    FROM
	//    (
	//      SELECT post_id, type, COUNT(*) AS count FROM post_reacts
	//      GROUP BY type, post_id
	//    ) AS react_types GROUP BY post_id
	// ) AS reacts ON reacts.post_id = ar.id
	// JOIN users u ON ar.author_id = u.id
	// LEFT JOIN posts p2 ON ar.root_article_id = p2.id
	// ORDER BY ar.created_at;`

	// var orderSqlStrHead, orderSqlStrTail string
	// switch sortType {
	// case model.ReplySortBest:
	// 	orderSqlStrHead = ` ORDER BY reply_weight DESC `
	// 	orderSqlStrTail = ` ORDER BY p.reply_weight DESC `
	// case model.ListSortLatest:
	// 	fallthrough
	// default:
	// 	orderSqlStrHead = ` ORDER BY created_at DESC `
	// 	orderSqlStrTail = ` ORDER BY p.created_at DESC `
	// }

	// fmt.Println("orderStr:", orderSqlStrHead)

	sqlStr := `
WITH RECURSIVE articleTree AS (
     SELECT id, title, url, author_id, content, created_at, updated_at, deleted, reply_to, depth, 0 AS cur_depth, root_article_id, reply_weight
     FROM posts
     WHERE id = $1
     UNION ALL
     SELECT p.id, p.title, p.url, p.author_id, p.content, p.created_at,p.updated_at, p.deleted, p.reply_to, p.depth, ar.cur_depth + 1, p.root_article_id, p.reply_weight
     FROM posts p
     JOIN articleTree ar
     ON p.reply_to = ar.id
     WHERE ar.cur_depth < $2
)
SELECT p.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title AS root_article_title,
COUNT(p3.id) AS children_count,
pv.type AS vote_type,
COUNT(pv1.id) AS vote_up_count,
COUNT(pv2.id) AS vote_down_count,
(
  SELECT
    CASE
      WHEN COUNT(*) > 0 THEN TRUE
      ELSE FALSE
    END
   FROM post_saves WHERE post_id = p.id AND user_id = $3
) AS saved,
(
  SELECT
    CASE
      WHEN COUNT(*) > 0 THEN TRUE
      ELSE FALSE
    END
   FROM post_subs WHERE post_id = p.id AND user_id = $3
) AS subscribed,
null
FROM articleTree p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p.id = p3.reply_to
LEFT JOIN post_votes pv ON pv.post_id = p.id AND pv.user_id = $3
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
GROUP BY p.id, p.title, p.url, u.username, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title, pv.type`
	if page < 1 {
		page = defaultPage
	}

	if pageSize < 1 {
		pageSize = defaultPageSize
	}

	// rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), userId, pageSize*(page-1), pageSize)
	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), userId)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.Article
	for rows.Next() {
		var userState model.CurrUserState
		item := model.Article{
			CurrUserState: &userState,
		}

		err = rows.Scan(
			&item.Id,
			&item.Title,
			&item.Link,
			&item.AuthorName,
			&item.AuthorId,
			&item.Content,
			&item.CreatedAt,
			&item.UpdatedAt,
			&item.Deleted,
			&item.ReplyTo,
			&item.ReplyDepth,
			&item.ReplyRootArticleId,
			&item.Weight,
			&item.NullReplyRootArticleTitle,
			// &item.TotalReplyCount,
			&item.ChildrenCount,
			&item.CurrUserState.NullVoteType,
			&item.VoteUp,
			&item.VoteDown,
			&item.CurrUserState.Saved,
			&item.CurrUserState.Subscribed,
			&item.CurrUserState.NullReactType,
		)

		if err != nil {
			return nil, err
		}

		// fmt.Printf("row item: %+v\n", &item)
		list = append(list, &item)
	}

	return list, nil
}

func (a *Article) Delete(id int) (rootArticleId int, err error) {
	err = a.dbPool.QueryRow(context.Background(),
		"UPDATE posts SET deleted = true WHERE id = $1 RETURNING (root_article_id)",
		id,
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

	err = a.updateWeights(id)
	if err != nil {
		return err
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

func (a *Article) Save(id, userId int) error {
	err, saved := a.saveCheck(id, userId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if !saved {
		_, err = a.dbPool.Exec(
			context.Background(),
			`INSERT INTO post_saves (post_id, user_id) VALUES ($1, $2)`,
			id,
			userId,
		)

		if err != nil {
			return err
		}
	} else {
		_, err = a.dbPool.Exec(
			context.Background(),
			`DELETE FROM post_saves WHERE post_id = $1 AND user_id = $2`,
			id,
			userId,
		)
		if err != nil {
			return err
		}
	}

	err = a.updateWeights(id)
	if err != nil {
		return err
	}

	return nil
}

func (a *Article) saveCheck(id, userId int) (error, bool) {
	var count int
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM post_saves WHERE post_id = $1 AND user_id = $2`,
		id,
		userId,
	).Scan(&count)
	if err != nil {
		return err, false
	}

	return nil, count > 0
}

func (a *Article) React(id, userId int, reactType string) error {
	err, rt := a.ReactCheck(id, userId)
	// fmt.Println("check error: ", err)
	// fmt.Println("check vote type: ", rt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// var aId int
			err = a.dbPool.QueryRow(
				context.Background(),
				`INSERT INTO post_reacts (post_id, user_id, type) VALUES ($1, $2, $3) RETURNING (post_id, user_id)`,
				id,
				userId,
				reactType,
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
		if rt == reactType {
			err = a.dbPool.QueryRow(
				context.Background(),
				`DELETE FROM post_reacts WHERE post_id = $1 AND user_id = $2 RETURNING (post_id, user_id)`,
				id,
				userId,
			).Scan(nil)
			if err != nil {
				return err
			}
		} else {
			// fmt.Println("change vote type to: ", reactType)
			err = a.dbPool.QueryRow(
				context.Background(),
				`UPDATE post_reacts SET type = $1 WHERE post_id = $2 AND user_id = $3 RETURNING (post_id, user_id)`,
				reactType,
				id,
				userId,
			).Scan(nil)

			// fmt.Println("after change vote type: ", err)

			if err != nil {
				return err
			}
		}
	}

	err = a.updateWeights(id)
	if err != nil {
		return err
	}

	return nil
}

func (a *Article) ReactCheck(id, userId int) (error, string) {
	var rt string
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT type FROM post_reacts WHERE post_id = $1 AND user_id = $2`,
		id,
		userId,
	).Scan(&rt)
	if err != nil {
		return err, ""
	}

	return nil, rt
}

func (a *Article) Subscribe(id, userId int) error {
	err, subscribed := a.subscribeCheck(id, userId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	if !subscribed {
		_, err = a.dbPool.Exec(
			context.Background(),
			`INSERT INTO post_subs (post_id, user_id) VALUES ($1, $2)`,
			id,
			userId,
		)

		if err != nil {
			return err
		}
	} else {
		_, err = a.dbPool.Exec(
			context.Background(),
			`DELETE FROM post_subs WHERE post_id = $1 AND user_id = $2`,
			id,
			userId,
		)
		if err != nil {
			return err
		}
	}

	err = a.updateWeights(id)
	if err != nil {
		return err
	}

	return nil
}

// Check if user already subscribe in ancestor node
func (a *Article) CheckSubscribe(id, userId int) (int, error) {
	sqlStr := `WITH RECURSIVE ancestors AS (
  SELECT p.id, p.reply_to FROM posts p
  WHERE p.id = $1
  
  UNION ALL
  
  SELECT p1.id, p1.reply_to FROM posts p1
  JOIN ancestors an ON p1.id = an.reply_to
 )
 SELECT id FROM ancestors an
 INNER JOIN post_subs ps ON ps.post_id = an.id AND ps.user_id = $2
`
	// fmt.Println("article id: ", id)
	// fmt.Println("user id: ", userId)
	// fmt.Println("check subscribe sql: ", sqlStr)
	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, userId)
	if err != nil {
		return 0, err
	}

	// var count int
	var ancestorSubscribes []int
	var subedId int
	for rows.Next() {
		err := rows.Scan(&subedId)
		if err != nil {
			return len(ancestorSubscribes), err
		}

		ancestorSubscribes = append(ancestorSubscribes, subedId)
	}

	// fmt.Println("subed ids: ", ancestorSubscribes)

	return len(ancestorSubscribes), nil
}

func (a *Article) Notify(senderUserId, sourceArticleId int, content string) error {
	sqlStr := `
WITH RECURSIVE parentPosts AS (
  SELECT id, reply_to FROM posts WHERE id = $2
  UNION ALL
  SELECT p1.id, p1.reply_to FROM posts p1
  JOIN parentPosts pp ON pp.reply_to = p1.id AND pp.reply_to != 0
)
INSERT INTO messages (sender_id, reciever_id, source_id, content)
SELECT $1, ps.user_id, pp.id, $3 FROM parentPosts pp
INNER JOIN post_subs ps ON ps.post_id = pp.id AND ps.user_id != $1;
`
	_, err := a.dbPool.Exec(context.Background(), sqlStr, senderUserId, sourceArticleId, content)

	if err != nil {
		return err
	}

	return nil
}

func (a *Article) subscribeCheck(id, userId int) (error, bool) {
	var count int
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT COUNT(*) FROM post_subs WHERE post_id = $1 AND user_id = $2`,
		id,
		userId,
	).Scan(&count)
	if err != nil {
		return err, false
	}

	return nil, count > 0
}

func (a *Article) updateWeights(id int) error {
	if id == 0 {
		return nil
	}

	var voteUp, voteDown, participateCount int
	var createdAt time.Time

	err := a.dbPool.QueryRow(
		context.Background(),
		`WITH partiUsers AS (
  SELECT p2.author_id AS user_id FROM posts p2 WHERE p2.root_article_id = $1
  UNION ALL
  SELECT pv.user_id FROM post_votes pv WHERE pv.post_id = $1
  UNION ALL
  SELECT ps.user_id FROM post_saves ps WHERE ps.post_id = $1
  UNION ALL
  SELECT pr.user_id FROM post_reacts pr WHERE pr.post_id = $1
)
SELECT COUNT(pv1.id) AS vote_up_count, COUNT(pv2.id) AS vote_down_count, p.created_at,
(SELECT COUNT(DISTINCT user_id) FROM partiUsers) AS participate_count
FROM posts p
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
WHERE p.id = $1
GROUP BY p.created_at;`,
		id,
	).Scan(&voteUp, &voteDown, &createdAt, &participateCount)
	if err != nil {
		return err
	}

	// fmt.Println("vote up", voteUp)
	// fmt.Println("vote down", voteDown)
	// fmt.Println("participate count", participateCount)

	listWeight := model.CalcArticleListWeight(voteUp, voteDown, createdAt)
	replyWeight := model.CalcArticleReplyWeight(voteUp, voteDown)

	// fmt.Println("listWeight: ", listWeight)

	_, err = a.dbPool.Exec(
		context.Background(),
		`UPDATE posts SET list_weight = $1, participate_count = $2, reply_weight = $3 WHERE id = $4`,
		listWeight,
		participateCount,
		replyWeight,
		id,
	)
	if err != nil {
		return err
	}

	return nil
}
