package model

import (
	"fmt"
	"net/mail"
	"regexp"
	"time"

	"github.com/oodzchen/dproject/utils"
	"golang.org/x/crypto/bcrypt"
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
	PasswordHased bool
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
		return utils.NewError(fmt.Sprintf("require field: %s", lackField))
	}

	parsedMailAddr, err := mail.ParseAddress(u.Email)
	if err != nil {
		return err
	}
	u.Email = parsedMailAddr.Address

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
