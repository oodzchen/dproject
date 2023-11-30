package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/utils"
)

var afterUpdateWeights func() error

type Article struct {
	dbPool *pgxpool.Pool
}

func (a *Article) List(
	page, pageSize int,
	sortType model.ArticleSortType,
	categoryFrontId string,
	pinned, deleted, includeReplies bool,
	keywords string,
) ([]*model.Article, int, error) {
	// fmt.Println("page, pageSize: ", page, pageSize)
	// fmt.Println("category front id:", categoryFrontId)
	// fmt.Println("sortType:", string(sortType))

	var orderSqlStrHead, orderSqlStrTail string
	switch sortType {
	case model.ListSortBest:
		orderSqlStrHead = ` ORDER BY p.list_weight DESC `
		orderSqlStrTail = ` ORDER BY tp.list_weight DESC `
	case model.ListSortHot:
		orderSqlStrHead = ` ORDER BY p.participate_count DESC `
		orderSqlStrTail = ` ORDER BY tp.participate_count DESC `
	case model.ListSortOldest:
		orderSqlStrHead = ` ORDER BY p.created_at`
		orderSqlStrTail = ` ORDER BY tp.created_at`
	case model.ListSortLatest:
		fallthrough
	default:
		orderSqlStrHead = ` ORDER BY p.created_at DESC `
		orderSqlStrTail = ` ORDER BY tp.created_at DESC `
	}

	var args []any
	var conditions []string
	if page < 1 {
		page = DefaultPage
	}

	if pageSize < 0 {
		args = []any{0, nil}
	} else {
		if pageSize < 1 {
			pageSize = DefaultPageSize
		}
		args = []any{pageSize * (page - 1), pageSize}
	}

	sqlStr := `
WITH postIds AS (
    SELECT p.id, COUNT(p.id) OVER() AS total FROM posts p`
	if strings.TrimSpace(categoryFrontId) != "" {
		args = append(args, categoryFrontId)
		conditions = append(conditions, `p.category_id = c.id`)
		sqlStr += fmt.Sprintf(" LEFT JOIN categories c ON c.front_id = $%d ", len(args))
	}

	if deleted {
		conditions = append(conditions, `p.deleted = true`)
	} else {
		conditions = append(conditions, `p.deleted = false`)
	}

	if !includeReplies {
		conditions = append(conditions, `p.reply_to = 0`)
	}

	if pinned {
		conditions = append(conditions, `(p.pinned_expire_at is not null AND p.pinned_expire_at > NOW())`)
		// sqlStr += ` AND (p.pinned_expire_at is not null AND p.pinned_expire_at > NOW()) `
	} else {
		conditions = append(conditions, `(p.pinned_expire_at is null OR p.pinned_expire_at <= NOW())`)
		// sqlStr += ` AND (p.pinned_expire_at is null OR p.pinned_expire_at <= NOW())`
	}

	if strings.TrimSpace(keywords) != "" {
		sqlStr += ` LEFT JOIN users u ON u.id = p.author_id `
		args = append(args, fmt.Sprintf("%s%s%s", "%%", keywords, "%%"))
		conditions = append(conditions, fmt.Sprintf("p.title ILIKE $%d OR u.username ILIKE $%d", len(args), len(args)))
	}

	if len(conditions) > 0 {
		sqlStr += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	sqlStr += orderSqlStrHead + `
    OFFSET $1
    LIMIT $2
)
SELECT tp.id, tp.title, COALESCE(tp.url, ''), u.username as author_name, tp.author_id, tp.content, tp.created_at, tp.updated_at, tp.depth, tp.list_weight, tp.reply_weight, tp.participate_count, p2.title as root_article_title, COUNT(p3.id) AS total_reply_count, tp.locked, tp.pinned_expire_at,

(
SELECT COUNT(post_id) FROM post_votes
WHERE post_id = tp.id AND type = 'up'
) AS vote_up,
(
SELECT COUNT(post_id) FROM post_votes
WHERE post_id = tp.id AND type = 'down'
) AS vote_down,
pi.total AS total,
c.id AS category_id,
c.front_id AS category_front_id,
c.name AS category_name
FROM posts tp
JOIN postIds pi ON tp.id = pi.id
LEFT JOIN posts p2 ON tp.root_article_id = p2.id
LEFT JOIN posts p3 ON tp.id = p3.root_article_id AND p3.deleted = false
LEFT JOIN users u ON u.id = tp.author_id
LEFT JOIN categories c ON c.id = tp.category_id
GROUP BY tp.id, u.username, p2.title, pi.total, c.id` + orderSqlStrTail

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
		// var currUserState model.CurrUserState
		var category model.Category
		item := model.Article{
			// CurrUserState: &currUserState,
			Category: &category,
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
			&item.Locked,
			&item.NullPinnedExpireAt,
			&item.VoteUp,
			&item.VoteDown,
			&total,

			&category.Id,
			&category.FrontId,
			&category.Name,
		)

		if err != nil {
			fmt.Printf("Collect rows error: %v\n", err)
			return nil, 0, err
		}

		item.CategoryFrontId = category.FrontId

		list = append(list, &item)
	}

	for _, item := range list {
		item.CalcScore()
		item.FormatNullValues()
		item.UpdateDisplayTitle()
		item.GenSummary(200)
		item.UpdatePinnedState()
	}

	return list, total, nil
}

func (a *Article) ListUserState(ids []int, userId int) ([]*model.Article, error) {
	sqlStr := `
SELECT p.id,
(
  SELECT type FROM post_votes WHERE post_id = p.id AND user_id = $1
) AS user_vote_type
FROM posts p
WHERE p.id = ANY($2)
`
	rows, err := a.dbPool.Query(context.Background(), sqlStr, userId, ids)
	if err != nil {
		return nil, err
	}

	var list []*model.Article
	for rows.Next() {
		var article model.Article
		var userState model.CurrUserState

		err = rows.Scan(&article.Id, &userState.NullVoteType)
		if err != nil {
			return nil, err
		}

		userState.FormatNullValues()

		article.CurrUserState = &userState
		list = append(list, &article)
	}

	return list, nil
}

func (a *Article) Count(frontId string) (int, error) {
	var count int
	var args []any
	sqlStr := `SELECT COUNT(*) FROM posts WHERE reply_to = 0 AND deleted = false`
	if frontId != "" {
		args = append(args, frontId)
		sqlStr = `SELECT COUNT(p.*) FROM posts p
LEFT JOIN categories c ON c.front_id = $1
WHERE p.reply_to = 0 AND p.deleted = false AND p.category_id = c.id`
	}

	err := a.dbPool.QueryRow(context.Background(), sqlStr, args...).Scan(&count)
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

func (a *Article) Create(title, url, content string, authorId, replyToId int, categoryFrontId string, pinnedExpireAt time.Time, locked bool) (int, error) {
	var id int
	args := []any{
		title,
		authorId,
		content,
		replyToId,
		url,
		categoryFrontId,
	}

	sqlStr := `
INSERT INTO posts (title, author_id, content, reply_to, url, root_article_id, depth, category_id, pinned_expire_at, locked)
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
    ),
    (
        CASE WHEN $4 = 0 THEN (SELECT id FROM categories WHERE front_id = $6)
             ELSE (SELECT p.category_id FROM posts p WHERE $4 = p.id)
        END
    ) `
	if !pinnedExpireAt.IsZero() {
		args = append(args, pinnedExpireAt)
		sqlStr += fmt.Sprintf(", $%d ", len(args))
	} else {
		sqlStr += ", null "
	}

	args = append(args, locked)
	sqlStr += fmt.Sprintf(", $%d ", len(args))

	sqlStr += `
)
RETURNING (id);`

	// fmt.Println("create article sql:", sqlStr)
	// fmt.Println("create article args:", args)
	err := a.dbPool.QueryRow(context.Background(), sqlStr,
		args...,
	).Scan(&id)
	if err != nil {
		return 0, err
	}

	err = a.ToggleVote(id, authorId, "up")
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

	if replyToId == 0 {
		wg.Done()
	} else {
		go func() {
			defer wg.Done()
			err = a.updateWeights(replyToId)
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

func (a *Article) UpdateRootArticle(id int, title, content, link, categoryFrontId string, pinnedExpireAt time.Time, locked bool) (int, error) {
	sqlStr := `UPDATE posts SET
title = $2,
content = $3,
url = $4,
category_id = (
  SELECT id FROM categories WHERE front_id = $5
),
updated_at = NOW(),
pinned_expire_at = $6,
locked = $7
WHERE id = $1`

	args := []any{id, title, content, link, categoryFrontId}

	if pinnedExpireAt.IsZero() {
		args = append(args, nil)
	} else {
		args = append(args, pinnedExpireAt)
	}

	args = append(args, locked)

	_, err := a.dbPool.Exec(context.Background(), sqlStr, args...)
	if err != nil {
		return 0, err
	}

	err = a.updateWeights(id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *Article) UpdateReply(id int, content string, pinnedExpireAt time.Time, locked bool) (int, error) {
	sqlStr := `UPDATE posts SET content = $2, updated_at = NOW(), pinned_expire_at = $3, locked = $4 WHERE id = $1`

	args := []any{id, content}

	if pinnedExpireAt.IsZero() {
		args = append(args, nil)
	} else {
		args = append(args, pinnedExpireAt)
	}

	args = append(args, locked)

	_, err := a.dbPool.Exec(context.Background(), sqlStr, args...)
	if err != nil {
		return 0, err
	}

	err = a.updateWeights(id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

// func (a *Article) Update(item *model.Article, fieldNames []string) (int, error) {
// 	for _, field := range fieldNames {
// 		if !validArticleUpdateField(field) {
// 			return 0, errors.New(fmt.Sprintf("'%s' is not allowed to update", field))
// 		}
// 	}

// 	var updateStr []string
// 	var updateVals []any
// 	itemVal := reflect.ValueOf(*item)

// 	dbFieldNameMap := map[string]string{
// 		"Title":           "title",
// 		"Content":         "content",
// 		"Weight":          "weight",
// 		"Link":            "url",
// 		"CategoryFrontId": "category_id",
// 	}

// 	for idx, field := range fieldNames {
// 		updateStr = append(updateStr, fmt.Sprintf("%s = $%d", dbFieldNameMap[field], idx+1))
// 		updateVals = append(updateVals, itemVal.FieldByName(field))
// 	}

// 	updateStr = append(updateStr, "updated_at = NOW()")

// 	updateVals = append(updateVals, item.Id)
// 	sqlStr := "UPDATE posts SET " + strings.Join(updateStr, ", ") + fmt.Sprintf(" WHERE id = $%d RETURNING(id)", len(updateVals))

// 	// fmt.Println("update sql string: ", sqlStr)
// 	// fmt.Println("update vals: ", updateVals)

// 	var id int
// 	err := a.dbPool.QueryRow(context.Background(), sqlStr, updateVals...).Scan(&id)
// 	if err != nil {
// 		return 0, err
// 	}

// 	err = a.updateWeights(id)
// 	if err != nil {
// 		return 0, err
// 	}

// 	return id, nil
// }

func (a *Article) Item(id, userId int) (*model.Article, error) {
	sqlStr := `
SELECT p.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title as root_article_title, p.locked, p.pinned_expire_at,

COUNT(DISTINCT p3.id) AS children_count,
COUNT(DISTINCT pv1.id) AS vote_up_count,
COUNT(DISTINCT pv2.id) AS vote_down_count,

pv.type AS user_vote_type,
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
) AS subscribed,
COALESCE(r2.front_id, '') AS curr_react_front_id,

COALESCE(r.id, 0) AS react_id,
COALESCE(r.emoji, '') AS react_emoji,
COALESCE(r.front_id, '') AS react_front_id,
COALESCE(r.describe, '') AS react_describe,
COALESCE(r.created_at, NOW()) AS react_created_at,

c.id AS category_id,
c.front_id AS category_front_id,
c.name AS category_name
FROM posts p
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p.id = p3.reply_to
LEFT JOIN post_votes pv ON pv.post_id = p.id AND pv.user_id = $2
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
LEFT JOIN post_reacts pr ON pr.post_id = p.id
LEFT JOIN reacts r ON r.id = pr.react_id
LEFT JOIN post_reacts pr2 ON pr2.post_id = p.id AND pr2.user_id = $2
LEFT JOIN reacts r2 ON r2.id = pr2.react_id
LEFT JOIN categories c ON c.id = p.category_id
WHERE p.id = $1
GROUP BY p.id, p.title, p.url, u.username, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p2.title, pv.type, r.id, pr.id, c.id, r2.id, pr2.id;`

	var userState model.CurrUserState
	var category model.Category
	var react model.ArticleReact
	item := model.Article{
		CurrUserState: &userState,
		Category:      &category,
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
		&item.ReplyToId,
		&item.ReplyDepth,
		&item.ReplyRootArticleId,
		&item.NullReplyRootArticleTitle,
		&item.Locked,
		&item.NullPinnedExpireAt,

		&item.ChildrenCount,
		&item.VoteUp,
		&item.VoteDown,

		&userState.NullVoteType,
		&userState.Saved,
		&userState.Subscribed,
		&userState.ReactFrontId,

		&react.Id,
		&react.Emoji,
		&react.FrontId,
		&react.Describe,
		&react.CreatedAt,

		&category.Id,
		&category.FrontId,
		&category.Name,
	)

	if err != nil {
		return nil, err
	}

	if react.Id > 0 {
		// fmt.Println("react: ", react)
		if item.Reacts == nil {
			item.Reacts = []*model.ArticleReact{&react}
		} else {
			item.Reacts = append(item.Reacts, &react)
		}
	}

	item.CurrUserState.FormatNullValues()

	item.CategoryFrontId = category.FrontId

	item.FormatNullValues()
	item.FormatReactCounts()
	item.CalcScore()
	item.CheckShowScore(userId)
	item.UpdatePinnedState()

	return &item, nil
}

func (a *Article) ReplyTree(page, pageSize, id int, sortType model.ArticleSortType, pinned bool) ([]*model.Article, error) {
	var orderSqlStrTail string
	switch sortType {
	case model.ReplySortBest:
		// orderSqlStrHead = ` ORDER BY reply_weight DESC `
		orderSqlStrTail = ` ORDER BY p.reply_weight DESC `
	case model.ListSortOldest:
		// orderSqlStrHead = ` ORDER BY reply_weight DESC `
		orderSqlStrTail = ` ORDER BY p.created_at `
	case model.ListSortLatest:
		fallthrough
	default:
		// orderSqlStrHead = ` ORDER BY created_at DESC `
		orderSqlStrTail = ` ORDER BY p.created_at DESC `
	}

	sqlStr := `
WITH RECURSIVE articleTree AS (
     SELECT id, reply_to, created_at, reply_weight, 1 AS cur_depth
     FROM posts
     WHERE reply_to = $1 `

	if pinned {
		sqlStr += ` AND pinned_expire_at is not null AND pinned_expire_at > NOW() `
	} else {
		sqlStr += ` AND pinned_expire_at is null OR pinned_expire_at <= NOW()`
	}

	sqlStr += `
     UNION ALL
     SELECT p.id, p.reply_to, p.created_at, p.reply_weight, ar.cur_depth + 1
     FROM posts p
     JOIN articleTree ar
     ON p.reply_to = ar.id
     WHERE ar.cur_depth < $2 `

	if pinned {
		sqlStr += ` AND p.pinned_expire_at is not null AND p.pinned_expire_at > NOW() `
	} else {
		sqlStr += ` AND p.pinned_expire_at is null OR p.pinned_expire_at <= NOW()`
	}

	sqlStr += `
), groupedList AS (
    SELECT p.*, ROW_NUMBER() OVER (PARTITION BY p.reply_to ` + orderSqlStrTail + `) AS rn
    FROM articleTree p
)
SELECT ar.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title AS root_article_title, p.locked, p.pinned_expire_at,
COUNT(DISTINCT p3.id) AS children_count,
COUNT(DISTINCT pv1.id) AS vote_up_count,
COUNT(DISTINCT pv2.id) AS vote_down_count,
COALESCE(r.id, 0) AS react_id,
COALESCE(r.emoji, '') AS react_emoji,
COALESCE(r.front_id, '') AS react_front_id,
COALESCE(r.describe, '') AS react_describe,
COALESCE(r.created_at, NOW()) AS react_created_at,
c.id AS category_id,
c.front_id AS category_front_id,
c.name AS category_name
FROM groupedList ar
LEFT JOIN posts p ON p.id = ar.id
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p.id = p3.reply_to
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
LEFT JOIN post_reacts pr ON pr.post_id = p.id
LEFT JOIN reacts r ON r.id = pr.react_id
LEFT JOIN categories c ON c.id = p.category_id
WHERE (ar.cur_depth = 1 AND ar.rn BETWEEN $3 AND $4) OR (ar.cur_depth > 1 AND ar.rn BETWEEN 0 AND $5)
GROUP BY ar.id, p.id, p.title, p.url, u.username, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title, r.id, pr.id, c.id
` + orderSqlStrTail

	if page < 1 {
		page = DefaultPage
	}

	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	// fmt.Println("item tree sql:", sqlStr)

	// rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), userId, pageSize*(page-1), pageSize)
	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), pageSize*(page-1), pageSize*page, pageSize)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.Article
	listMap := make(map[int]*model.Article)
	for rows.Next() {
		var react model.ArticleReact
		var item model.Article
		var category model.Category

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
			&item.ReplyToId,
			&item.ReplyDepth,
			&item.ReplyRootArticleId,
			&item.Weight,
			&item.NullReplyRootArticleTitle,
			&item.Locked,
			&item.NullPinnedExpireAt,
			&item.ChildrenCount,

			&item.VoteUp,
			&item.VoteDown,

			&react.Id,
			&react.Emoji,
			&react.FrontId,
			&react.Describe,
			&react.CreatedAt,

			&category.Id,
			&category.FrontId,
			&category.Name,
		)

		// fmt.Println("item.id: ", item.Id)
		// fmt.Println("userState: ", userState)
		// fmt.Println("react: ", react)

		if err != nil {
			return nil, err
		}

		var article *model.Article
		if item.Id > 0 {
			if v, ok := listMap[item.Id]; ok {
				article = v
			} else {
				article = &item
				list = append(list, article)
				listMap[article.Id] = article
			}

			if react.Id > 0 {
				// fmt.Println("react: ", react)
				if article.Reacts == nil {
					article.Reacts = []*model.ArticleReact{&react}
				} else {
					article.Reacts = append(article.Reacts, &react)
				}
			}

			item.Category = &category
			item.CategoryFrontId = category.FrontId
		}
	}

	for _, item := range list {
		item.FormatNullValues()
		item.FormatReactCounts()
		item.CalcScore()
		item.UpdatePinnedState()
	}

	return list, nil
}

func (a *Article) ReplyList(page, pageSize, id int, sortType model.ArticleSortType, pinned bool) ([]*model.Article, error) {
	var orderSqlStrTail string
	switch sortType {
	case model.ReplySortBest:
		// orderSqlStrHead = ` ORDER BY reply_weight DESC `
		orderSqlStrTail = ` ORDER BY p.reply_weight DESC `
	case model.ListSortOldest:
		// orderSqlStrHead = ` ORDER BY reply_weight DESC `
		orderSqlStrTail = ` ORDER BY p.created_at `
	case model.ListSortLatest:
		fallthrough
	default:
		// orderSqlStrHead = ` ORDER BY created_at DESC `
		orderSqlStrTail = ` ORDER BY p.created_at DESC `
	}

	sqlStr := `
WITH RECURSIVE articleTree AS (
     SELECT id, reply_to, created_at, reply_weight
     FROM posts
     WHERE reply_to = $1 `

	if pinned {
		sqlStr += ` AND pinned_expire_at is not null AND pinned_expire_at > NOW() `
	} else {
		sqlStr += ` AND pinned_expire_at is null OR pinned_expire_at <= NOW()`
	}

	sqlStr += `
     UNION ALL
     SELECT p.id, p.reply_to, p.created_at, p.reply_weight
     FROM posts p
     JOIN articleTree ar
     ON p.reply_to = ar.id `

	if pinned {
		sqlStr += ` WHERE p.pinned_expire_at is not null AND p.pinned_expire_at > NOW() `
	} else {
		sqlStr += ` WHERE p.pinned_expire_at is null OR p.pinned_expire_at <= NOW()`
	}

	sqlStr += `
), pagedList AS (
    SELECT * FROM articleTree p ` + orderSqlStrTail + `
    OFFSET $2
    LIMIT $3
)
SELECT ar.id, p.title, COALESCE(p.url, ''), u.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title AS root_article_title, p.locked, p.pinned_expire_at,
COUNT(DISTINCT p3.id) AS children_count,
COUNT(DISTINCT pv1.id) AS vote_up_count,
COUNT(DISTINCT pv2.id) AS vote_down_count,
COALESCE(r.id, 0) AS react_id,
COALESCE(r.emoji, '') AS react_emoji,
COALESCE(r.front_id, '') AS react_front_id,
COALESCE(r.describe, '') AS react_describe,
COALESCE(r.created_at, NOW()) AS react_created_at,
c.id AS category_id,
c.front_id AS category_front_id,
c.name AS category_name,

COALESCE(p4.id, 0),
COALESCE(p4.title, ''),
COALESCE(p4.url,''),
COALESCE(u2.username, '') AS author_name2,
COALESCE(p4.author_id, 0),
COALESCE(p4.content, ''),
COALESCE(p4.created_at, NOW()),
COALESCE(p4.updated_at, NOW()),
COALESCE(p4.deleted, false),
COALESCE(p4.reply_to, 0),
COALESCE(p4.depth, 0),
COALESCE(p4.root_article_id, 0),
COALESCE(p4.reply_weight, 0)

FROM posts p
JOIN pagedList ar ON p.id = ar.id
LEFT JOIN users u ON p.author_id = u.id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p.id = p3.reply_to
LEFT JOIN posts p4 ON p.reply_to = p4.id AND p4.reply_to != 0 AND p4.id != $1
LEFT JOIN users u2 ON p4.author_id = u2.id
LEFT JOIN post_votes pv1 ON pv1.post_id = p.id AND pv1.type = 'up'
LEFT JOIN post_votes pv2 ON pv2.post_id = p.id AND pv2.type = 'down'
LEFT JOIN post_reacts pr ON pr.post_id = p.id
LEFT JOIN reacts r ON r.id = pr.react_id
LEFT JOIN categories c ON c.id = p.category_id
GROUP BY ar.id, p.id, p.title, p.url, u.username, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, p.root_article_id, p.reply_weight, p2.title, r.id, pr.id, c.id, p4.id, u2.username` + orderSqlStrTail

	if page < 1 {
		page = DefaultPage
	}

	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	// fmt.Println("item tree sql:", sqlStr)

	// rows, err := a.dbPool.Query(context.Background(), sqlStr, id, utils.GetReplyDepthSize(), userId, pageSize*(page-1), pageSize)
	rows, err := a.dbPool.Query(context.Background(), sqlStr, id, pageSize*(page-1), pageSize)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var list []*model.Article
	listMap := make(map[int]*model.Article)
	for rows.Next() {
		var react model.ArticleReact
		var item model.Article
		var category model.Category
		var parent model.Article

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
			&item.ReplyToId,
			&item.ReplyDepth,
			&item.ReplyRootArticleId,
			&item.Weight,
			&item.NullReplyRootArticleTitle,
			&item.Locked,
			&item.NullPinnedExpireAt,
			&item.ChildrenCount,

			&item.VoteUp,
			&item.VoteDown,
			&react.Id,
			&react.Emoji,
			&react.FrontId,
			&react.Describe,
			&react.CreatedAt,

			&category.Id,
			&category.FrontId,
			&category.Name,

			&parent.Id,
			&parent.Title,
			&parent.Link,
			&parent.AuthorName,
			&parent.AuthorId,
			&parent.Content,
			&parent.CreatedAt,
			&parent.UpdatedAt,
			&parent.Deleted,
			&parent.ReplyToId,
			&parent.ReplyDepth,
			&parent.ReplyRootArticleId,
			&parent.Weight,
		)

		// fmt.Println("item.id: ", item.Id)
		// fmt.Println("userState: ", userState)
		// fmt.Println("react: ", react)

		if err != nil {
			return nil, err
		}

		var article *model.Article
		if item.Id > 0 {
			if v, ok := listMap[item.Id]; ok {
				article = v
			} else {
				article = &item
				list = append(list, article)
				listMap[article.Id] = article
			}

			if react.Id > 0 {
				// fmt.Println("react: ", react)
				if article.Reacts == nil {
					article.Reacts = []*model.ArticleReact{&react}
				} else {
					article.Reacts = append(article.Reacts, &react)
				}
			}

			item.Category = &category
			item.CategoryFrontId = category.FrontId

			if parent.Id > 0 {
				item.ReplyToArticle = &parent
			}
		}
	}

	for _, item := range list {
		item.FormatNullValues()
		item.FormatReactCounts()
		item.CalcScore()
		item.UpdatePinnedState()
	}

	return list, nil
}

func (a *Article) ItemTreeUserState(ids []int, userId int) ([]*model.Article, error) {
	// fmt.Println("item tree user state ids:", ids)
	// fmt.Println("item tree user id:", userId)
	sqlStr := `
SELECT p.id,
(
  SELECT type FROM post_votes WHERE post_id = p.id AND user_id = $1
) AS user_vote_type,
(
  SELECT
    CASE
      WHEN COUNT(*) > 0 THEN TRUE
      ELSE FALSE
    END
   FROM post_saves WHERE post_id = p.id AND user_id = $1
) AS saved,
(
  SELECT
    CASE
      WHEN COUNT(*) > 0 THEN TRUE
      ELSE FALSE
    END
   FROM post_subs WHERE post_id = p.id AND user_id = $1
) AS subscribed,
COALESCE(r.front_id, '') AS curr_react_front_id
FROM posts p
LEFT JOIN post_reacts pr ON pr.post_id = p.id AND pr.user_id = $1
LEFT JOIN reacts r ON r.id = pr.react_id
WHERE p.id = ANY($2)
`
	rows, err := a.dbPool.Query(context.Background(), sqlStr, userId, ids)
	if err != nil {
		return nil, err
	}

	var list []*model.Article
	for rows.Next() {
		var article model.Article
		var userState model.CurrUserState

		err = rows.Scan(
			&article.Id,
			&userState.NullVoteType,
			&userState.Saved,
			&userState.Subscribed,
			&userState.ReactFrontId,
		)
		if err != nil {
			return nil, err
		}

		userState.FormatNullValues()
		// fmt.Println("react front id:", userState.ReactFrontId)
		article.CurrUserState = &userState
		list = append(list, &article)
	}

	return list, nil
}

func (a *Article) GetReactList() ([]*model.ArticleReact, error) {
	rows, err := a.dbPool.Query(context.Background(), `SELECT id, emoji, front_id, describe, created_at FROM reacts ORDER BY created_at`)
	if err != nil {
		return nil, err
	}

	var list []*model.ArticleReact
	for rows.Next() {
		var react model.ArticleReact
		err := rows.Scan(
			&react.Id,
			&react.Emoji,
			&react.FrontId,
			&react.Describe,
			&react.CreatedAt,
		)

		if err != nil {
			return nil, err
		}

		list = append(list, &react)
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

func (a *Article) ToggleVote(id, userId int, voteType string) error {
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

func (a *Article) ToggleSave(id, userId int) error {
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

func (a *Article) ToggleReact(id, userId, reactId int) error {
	err, rt := a.ReactCheck(id, userId)
	// fmt.Println("check error: ", err)
	// fmt.Println("check vote type: ", rt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// var aId int
			err = a.dbPool.QueryRow(
				context.Background(),
				`INSERT INTO post_reacts (post_id, user_id, react_id) VALUES ($1, $2, $3) RETURNING (post_id, user_id)`,
				id,
				userId,
				reactId,
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
		if rt == reactId {
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
				`UPDATE post_reacts SET react_id = $1 WHERE post_id = $2 AND user_id = $3 RETURNING (post_id, user_id)`,
				reactId,
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

func (a *Article) ReactCheck(id, userId int) (error, int) {
	var rt int
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT react_id FROM post_reacts WHERE post_id = $1 AND user_id = $2`,
		id,
		userId,
	).Scan(&rt)
	if err != nil {
		return err, 0
	}

	return nil, rt
}

func (a *Article) ReactItem(reactId int) (*model.ArticleReact, error) {
	var react model.ArticleReact
	err := a.dbPool.QueryRow(
		context.Background(),
		`SELECT id, emoji, front_id, describe, created_at FROM reacts WHERE id = $1`,
		reactId,
	).Scan(
		&react.Id,
		&react.Emoji,
		&react.FrontId,
		&react.Describe,
		&react.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &react, nil
}

func (a *Article) ToggleSubscribe(id, userId int) error {
	err, subscribed := a.subscribeCheck(id, userId)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	sqlStr := `INSERT INTO post_subs (post_id, user_id) VALUES ($1, $2)`
	if subscribed {
		sqlStr = `DELETE FROM post_subs WHERE post_id = $1 AND user_id = $2`
	}

	_, err = a.dbPool.Exec(
		context.Background(),
		sqlStr,
		id,
		userId,
	)

	// if !subscribed {
	// 	_, err = a.dbPool.Exec(
	// 		context.Background(),
	// 		`INSERT INTO post_subs (post_id, user_id) VALUES ($1, $2)`,
	// 		id,
	// 		userId,
	// 	)

	// } else {
	// 	_, err = a.dbPool.Exec(
	// 		context.Background(),
	// 		`DELETE FROM post_subs WHERE post_id = $1 AND user_id = $2`,
	// 		id,
	// 		userId,
	// 	)
	// }

	if err != nil {
		return err
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
 SELECT an.id FROM ancestors an
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

func (a *Article) Notify(senderUserId, sourceArticleId, contentArticleId int) error {
	sqlStr := `
WITH RECURSIVE parentPosts AS (
  SELECT id, reply_to FROM posts WHERE id = $2
  UNION ALL
  SELECT p1.id, p1.reply_to FROM posts p1
  JOIN parentPosts pp ON pp.reply_to = p1.id AND pp.reply_to != 0
)
INSERT INTO messages (sender_id, reciever_id, source_article_id, content_id, type)
SELECT $1, ps.user_id, pp.id, $3, 'reply' FROM parentPosts pp
INNER JOIN post_subs ps ON ps.post_id = pp.id AND ps.user_id != $1;
`
	_, err := a.dbPool.Exec(context.Background(), sqlStr, senderUserId, sourceArticleId, contentArticleId)

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

	if afterUpdateWeights != nil {
		err = afterUpdateWeights()
		if err != nil {
			return err
		}

		// go func() {
		// 	err = afterUpdateWeights()
		// 	if err != nil {
		// 		fmt.Println("error in afterUpdateWeights:", err)
		// 	}
		// }()
	}

	return nil
}

func (a *Article) SetAfterUpdateWeights(fn func() error) {
	afterUpdateWeights = fn
}

func (a *Article) Tag(id int, tagFrontId string) error {
	return nil
}

func (a *Article) AddHistory(
	articleId, operatorId int,
	curr, prev time.Time,
	titleDelta, urlDelta, contentDelta, categoryFrontDelta string,
	isHidden bool,
) (int, error) {
	sqlStr := `INSERT INTO post_history (post_id, operator_id, curr, prev, version_num, title_delta, url_delta, content_delta, category_front_delta, is_hidden)
VALUES ($1, $2, $3, $4, (
   SELECT COALESCE(MAX(version_num)+1, 1) FROM post_history WHERE post_id = $1
) ,$5, $6, $7, $8, $9) RETURNING (id)`
	var id int
	err := a.dbPool.QueryRow(
		context.Background(),
		sqlStr,
		articleId,
		operatorId,
		curr,
		prev,
		titleDelta,
		urlDelta,
		contentDelta,
		categoryFrontDelta,
		isHidden,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

func (a *Article) ListHistory(articleId int) ([]*model.ArticleLog, error) {
	sqlStr := `SELECT
ph.id,
ph.post_id,
ph.created_at,
ph.operator_id,
ph.curr,
ph.prev,
ph.version_num,
ph.title_delta,
ph.url_delta,
ph.content_delta,
ph.category_front_delta,
ph.is_hidden,

u.username AS username

FROM post_history ph
LEFT JOIN users u ON u.id = ph.operator_id
WHERE ph.post_id = $1
ORDER BY ph.version_num DESC`

	rows, err := a.dbPool.Query(context.Background(), sqlStr, articleId)
	if err != nil {
		return nil, err
	}

	var list []*model.ArticleLog
	for rows.Next() {
		var log model.ArticleLog
		var user model.User
		err := rows.Scan(
			&log.Id,
			&log.PrimaryArticleId,
			&log.CreatedAt,
			&log.OperatorId,
			&log.CurrEditTime,
			&log.PrevEditTime,
			&log.VersionNum,
			&log.TitleDelta,
			&log.URLDelta,
			&log.ContentDelta,
			&log.CategoryFrontIdDelta,
			&log.IsHidden,

			&user.Name,
		)

		if err != nil {
			return nil, err
		}
		log.Operator = &user

		list = append(list, &log)
	}

	return list, nil
}

func (a *Article) Lock(id int) error {
	locked, err := a.CheckLocked(id)
	if err != nil {
		return err
	}

	newLockState := true
	if locked {
		newLockState = false
	}

	_, err = a.dbPool.Exec(context.Background(), `UPDATE posts SET locked = $2 WHERE id = $1`, id, newLockState)

	if err != nil {
		return err
	}

	return nil
}

func (a *Article) CheckLocked(id int) (bool, error) {
	var locked bool
	err := a.dbPool.QueryRow(context.Background(), `SELECT locked FROM posts WHERE id = $1`, id).Scan(&locked)
	if err != nil {
		return false, err
	}
	return locked, nil
}

func (a *Article) Pin(id int, expireAt time.Time) error {
	_, err := a.dbPool.Exec(context.Background(), `UPDATE posts SET pinned_expire_at = $2 WHERE id = $1`, id, expireAt)

	if err != nil {
		return err
	}

	return nil
}

func (a *Article) Unpin(id int) error {
	_, err := a.dbPool.Exec(context.Background(), `UPDATE posts SET pinned_expire_at = null WHERE id = $1`, id)

	if err != nil {
		return err
	}

	return nil
}

func (a *Article) ToggleHideHistory(historyId int, isHidden bool) error {
	_, err := a.dbPool.Exec(context.Background(), `UPDATE post_history SET is_hidden = $2 WHERE id = $1`, historyId, isHidden)

	if err != nil {
		return err
	}

	return nil
}

// func (a *Article) DeletedList() ([]*model.Article, error) {
// 	var list []*model.Article

// 	rows, err := a.dbPool.Query(context.Background(), `SELECT * FROM posts WHERE deleted = true`)
// 	if err != nil {
// 		return nil, err
// 	}

// 	for rows.Next() {

// 	}

// 	return list, nil
// }
