package model

import "time"

type Message struct {
	Id               int
	Content          string
	SenderUserId     int
	SenderUserName   string
	RecieverUserId   int
	RecieverUserName string
	SourceArticle    *Article
	IsRead           bool
	CreatedAt        *time.Time
}
