package model

import (
	"fmt"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
	"github.com/oodzchen/dproject/utils"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	Id               int
	Name             string
	Email            string
	Password         string
	RegisteredAt     time.Time
	RegisteredAtStr  string
	NullIntroduction pgtype.Text
	Introduction     string
	IsAdmin          bool
	Deleted          bool
	Banned           bool
	PasswordHased    bool
}

func (u *User) FormatTimeStr() {
	u.RegisteredAtStr = utils.FormatTime(u.RegisteredAt, "YYYY年MM月DD日")
}

func (u *User) FormatNullVals() {
	if u.NullIntroduction.Valid {
		u.Introduction = u.NullIntroduction.String
	}
}

func (u *User) Sanitize() {
	p := bluemonday.NewPolicy()
	u.Introduction = p.Sanitize(u.Introduction)
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
		return utils.NewError(fmt.Sprintf("require field: %s", lackField))
	}

	ok := utils.ValidateEmail(u.Email)
	if !ok {
		return utils.NewError("email format error")
	}

	reUsername := regexp.MustCompile(`^[\p{L}\p{N}\s]+$`)
	if !reUsername.Match([]byte(u.Name)) {
		return utils.NewError("username format error")
	}

	rePassword := regexp.MustCompile(`[A-Za-z\d[:graph:]]{8,}`)
	reLetter := regexp.MustCompile(`[A-Za-z]`)
	reNum := regexp.MustCompile(`\d`)
	reNotaion := regexp.MustCompile(`[[:graph:]]`)
	originalPwd := []byte(u.Password)
	if !rePassword.Match(originalPwd) || !reLetter.Match(originalPwd) || !reNum.Match(originalPwd) || !reNotaion.Match(originalPwd) {
		return utils.NewError("password format error")
	}

	return nil
}

func (u *User) EncryptPassword() error {
	hashedPwd, err := hashPassword(u.Password)
	if err != nil {
		return err
	}

	u.Password = hashedPwd
	u.PasswordHased = true
	return nil
}

func hashPassword(pwd string) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", nil
	}

	return string(hashedPwd), nil
}
