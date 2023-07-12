package model

import "time"

type Article struct {
	Id           int
	Title        string
	AuthorName   string
	AuthorId     int
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CreatedAtStr string
	UpdatedAtStr string
}
