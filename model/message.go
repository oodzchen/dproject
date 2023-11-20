package model

import "time"

type MessageType string

const (
	MessageTypeReply    MessageType = "reply"
	MessageTypeCategory             = "category"
	MessageTypeSystem               = "system"
)

type Message struct {
	Id               int
	Content          string
	SenderUserId     int
	SenderUserName   string
	RecieverUserId   int
	RecieverUserName string
	SourceArticle    *Article
	SourceCategory   *Category
	ContentArticle   *Article
	IsRead           bool
	CreatedAt        *time.Time
	Type             MessageType
}
