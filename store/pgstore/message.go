package pgstore

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type Message struct {
	dbPool *pgxpool.Pool
}

func (m *Message) List(userId int, status string, page, pageSize int) ([]*model.Message, int, error) {
	sqlStr := `SELECT m.id, m.sender_id, u.username AS sender_name, m.reciever_id, u1.username AS reciever_name, m.created_at, m.is_read, m.type,
COALESCE(p.id, 0),
COALESCE(p.title, ''),
COALESCE(p.url, ''),
COALESCE(u2.username, ''),
COALESCE(p.author_id, 0),
COALESCE(p.content, ''),
COALESCE(p.created_at, NOW()),
COALESCE(p.updated_at, NOW()),
COALESCE(p.deleted, false),
COALESCE(p.reply_to, 0),
COALESCE(p.depth, 0),
COALESCE(p2.title, ''),
(
SELECT
  CASE
    WHEN COUNT(*) > 0 THEN TRUE
    ELSE FALSE
  END
 FROM post_subs WHERE post_id = p.id AND user_id = m.reciever_id
) AS subscribed,

COALESCE(p3.id, 0),
COALESCE(p3.title, ''),
COALESCE(p3.url, ''),
COALESCE(u3.username, ''),
COALESCE(p3.author_id, 0),
COALESCE(p3.content, ''),
COALESCE(p3.created_at, NOW()),
COALESCE(p3.updated_at, NOW()),
COALESCE(p3.deleted, false),
COALESCE(p3.reply_to, 0),
COALESCE(p3.depth, 0),
COALESCE(p4.title, ''),

COALESCE(c.id, 0),
COALESCE(c.front_id, ''),
COALESCE(c.name, ''),
COALESCE(c.describe, ''),
COALESCE(c.author_id, 0),
COALESCE(c.approved, false),
COALESCE(c.approval_comment, ''),
COALESCE(c.created_at, NOW()),

COUNT(*) OVER() AS total
FROM messages m
LEFT JOIN users u ON u.id = m.sender_id
LEFT JOIN users u1 ON u1.id = m.reciever_id
LEFT JOIN posts p ON p.id = m.source_article_id
LEFT JOIN users u2 ON u2.id = p.author_id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p3.id = m.content_id
LEFT JOIN users u3 ON u3.id = p3.author_id
LEFT JOIN posts p4 ON p3.root_article_id = p4.id
LEFT JOIN categories c ON c.id = m.source_category_id`

	var args []any
	var conditions []string
	conditionCount := 0
	if page < 1 {
		page = DefaultPage
	}
	if pageSize < 1 {
		pageSize = DefaultPageSize
	}

	if userId > 0 {
		args = append(args, userId)
		conditions = append(conditions, fmt.Sprintf("m.reciever_id = $%d", len(args)))
		conditionCount += 1
	}

	if status != "" {
		switch status {
		case "unread":
			conditions = append(conditions, "m.is_read = false")
		case "read":
			conditions = append(conditions, "m.is_read = true")
		}
		conditionCount += 1
	}

	if conditionCount > 0 {
		sqlStr += ` WHERE ` + strings.Join(conditions, " AND ")
	}

	args = append(args, pageSize*(page-1), pageSize)
	sqlStr += fmt.Sprintf(" ORDER BY m.created_at DESC, m.id OFFSET $%d LIMIT $%d", len(args)-1, len(args))

	// fmt.Println("message list sqlStr: ", sqlStr)
	// fmt.Println("message list args: ", args)

	rows, err := m.dbPool.Query(context.Background(), sqlStr, args...)
	if err != nil {
		return nil, 0, err
	}

	var list []*model.Message
	var total int
	for rows.Next() {
		var item model.Message
		var userState model.CurrUserState
		var sourceArticle model.Article
		var sourceCategory model.Category
		var contentArticle model.Article

		err := rows.Scan(
			&item.Id,
			&item.SenderUserId,
			&item.SenderUserName,
			&item.RecieverUserId,
			&item.RecieverUserName,
			&item.CreatedAt,
			&item.IsRead,
			&item.Type,

			&sourceArticle.Id,
			&sourceArticle.Title,
			&sourceArticle.Link,
			&sourceArticle.AuthorName,
			&sourceArticle.AuthorId,
			&sourceArticle.Content,
			&sourceArticle.CreatedAt,
			&sourceArticle.UpdatedAt,
			&sourceArticle.Deleted,
			&sourceArticle.ReplyToId,
			&sourceArticle.ReplyDepth,
			&sourceArticle.ReplyRootArticleTitle,
			&userState.Subscribed,

			&contentArticle.Id,
			&contentArticle.Title,
			&contentArticle.Link,
			&contentArticle.AuthorName,
			&contentArticle.AuthorId,
			&contentArticle.Content,
			&contentArticle.CreatedAt,
			&contentArticle.UpdatedAt,
			&contentArticle.Deleted,
			&contentArticle.ReplyToId,
			&contentArticle.ReplyDepth,
			&contentArticle.ReplyRootArticleTitle,

			&sourceCategory.Id,
			&sourceCategory.FrontId,
			&sourceCategory.Name,
			&sourceCategory.Describe,
			&sourceCategory.AuthorId,
			&sourceCategory.Approved,
			&sourceCategory.ApprovalComment,
			&sourceCategory.CreatedAt,

			&total,
		)

		if err != nil {
			return nil, 0, err
		}

		sourceArticle.CurrUserState = &userState
		item.SourceArticle = &sourceArticle
		item.ContentArticle = &contentArticle
		item.SourceCategory = &sourceCategory

		list = append(list, &item)
	}

	// fmt.Println("total: ", total)

	return list, total, nil
}

// func (m *Message) Create(senderUserId, reciverUserId, sourceArticleId, contentArticleId int) (int, error) {
// 	var id int
// 	err := m.dbPool.QueryRow(
// 		context.Background(),
// 		`INSERT INTO messages
// (sender_id, reciver_id, source_article_id, content_id)
// VALUES
// ($1, $2, $3, $4)
// RETURNING (id)`,
// 		senderUserId,
// 		reciverUserId,
// 		sourceArticleId,
// 		contentArticleId,
// 	).Scan(&id)

// 	if err != nil {
// 		return 0, err
// 	}

// 	return id, nil
// }

func (m *Message) Read(messageId int) error {
	_, err := m.dbPool.Exec(context.Background(), `UPDATE messages SET is_read = true WHERE id = $1`, messageId)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) ReadMany(messageIds []any) error {
	var placeholdArr []string
	for idx := range messageIds {
		placeholdArr = append(placeholdArr, fmt.Sprintf("$%d", idx+1))
	}

	sqlStr := `UPDATE messages SET is_read = true WHERE id IN (` + strings.Join(placeholdArr, ", ") + `)`

	// fmt.Println("Read many messages sql: ", sqlStr)

	_, err := m.dbPool.Exec(context.Background(), sqlStr, messageIds...)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) ReadAll(userId int) error {
	sqlStr := `UPDATE messages SET is_read = true WHERE reciever_id = $1`

	// fmt.Println("Read many messages sql: ", sqlStr)

	_, err := m.dbPool.Exec(context.Background(), sqlStr, userId)
	if err != nil {
		return err
	}

	return nil
}

func (m *Message) UnreadCount(userId int) (int, error) {
	var total int
	err := m.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE reciever_id = $1 AND is_read = false`, userId).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
