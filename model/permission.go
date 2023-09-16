package model

import "time"

type Permission struct {
	Id        int
	FrontId   string
	Name      string
	CreatedAt time.Time
}
