package model

import "time"

type Role struct {
	Id        int
	FrontId   string
	Name      string
	CreatedAt time.Time
}
