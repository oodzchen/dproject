package model

import (
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
}

func (u *User) FormatTimeStr() {
	u.RegisterAtStr = utils.FormatTime(u.RegisterAt, "YYYY年MM月DD日")
}
