package model

import (
	"fmt"
	"time"

	"github.com/oodzchen/dproject/utils"
)

type User struct {
	Id            int
	Name          string
	Email         string
	Password      string
	RegisterAt    time.Time
	RegisterAtStr string
	Introduction  string
	IsAdmin       bool
	Deleted       bool
	Banned        bool
}

func (u *User) FormatTimeStr() {
	u.RegisterAtStr = utils.FormatTime(u.RegisterAt, "YYYY年MM月DD日")
}

func (u *User) Valid() error {
	lackField := ""

	if u.Email == "" {
		lackField = "email"
	}
	if u.Name == "" {
		lackField = "username"
	}
	if u.Password == "" {
		lackField = "password"
	}

	if lackField != "" {
		return utils.NewError(fmt.Sprintf("Require field: %s", lackField))
	}
	return nil
}
