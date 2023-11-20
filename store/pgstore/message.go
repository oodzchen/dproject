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
	sqlStr := `SELECT m.id, m.sender_id, u.username AS sender_name, m.reciever_id, u1.username AS reciever_name, p3.content AS content, m.created_at, m.is_read,
p.id, p.title, COALESCE(p.url, ''), u2.username AS author_name, p.author_id, p.content, p.created_at, p.updated_at, p.deleted, p.reply_to, p.depth, COALESCE(p2.title, ''),
(
SELECT
  CASE
    WHEN COUNT(*) > 0 THEN TRUE
    ELSE FALSE
  END
 FROM post_subs WHERE post_id = p.id AND user_id = m.reciever_id
) AS subscribed,
COUNT(*) OVER() AS total
FROM messages m
LEFT JOIN users u ON u.id = m.sender_id
LEFT JOIN users u1 ON u1.id = m.reciever_id
LEFT JOIN posts p ON p.id = m.source_id
LEFT JOIN users u2 ON u2.id = p.author_id
LEFT JOIN posts p2 ON p.root_article_id = p2.id
LEFT JOIN posts p3 ON p3.id = m.content_id`

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
		var article model.Article

		err := rows.Scan(
			&item.Id,
			&item.SenderUserId,
			&item.SenderUserName,
			&item.RecieverUserId,
			&item.RecieverUserName,
			&item.Content,
			&item.CreatedAt,
			&item.IsRead,

			&article.Id,
			&article.Title,
			&article.Link,
			&article.AuthorName,
			&article.AuthorId,
			&article.Content,
			&article.CreatedAt,
			&article.UpdatedAt,
			&article.Deleted,
			&article.ReplyTo,
			&article.ReplyDepth,
			&article.ReplyRootArticleTitle,
			&userState.Subscribed,

			&total,
		)

		if err != nil {
			return nil, 0, err
		}

		article.CurrUserState = &userState
		item.SourceArticle = &article

		list = append(list, &item)
	}

	// fmt.Println("total: ", total)

	return list, total, nil
}

func (m *Message) Create(senderUserId, reciverUserId, sourceArticleId, contentArticleId int) (int, error) {
	var id int
	err := m.dbPool.QueryRow(
		context.Background(),
		`INSERT INTO messages
(sender_id, reciver_id, source_id, content_id)
VALUES
($1, $2, $3, $4)
RETURNING (id)`,
		senderUserId,
		reciverUserId,
		sourceArticleId,
		contentArticleId,
	).Scan(&id)

	if err != nil {
		return 0, err
	}

	return id, nil
}

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

func (m *Message) UnreadCount(userId int) (int, error) {
	var total int
	err := m.dbPool.QueryRow(context.Background(), `SELECT COUNT(*) FROM messages WHERE reciever_id = $1 AND is_read = false`, userId).Scan(&total)
	if err != nil {
		return 0, err
	}
	return total, nil
}
