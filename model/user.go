package model

import (
	"errors"
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/microcosm-cc/bluemonday"
	"golang.org/x/crypto/bcrypt"
)

// https://regex101.com/r/RzBwPX/1
const ReEmailStr = `^(?P<name>[a-zA-Z0-9.!#$%&'*+/=?^_ \x60{|}~-]+)@(?P<domain>[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*)$`

const ReUsernameStr = `^[a-zA-Z0-9][a-zA-Z0-9._-]+[a-zA-Z0-9]$`
const ReUsernameMiddleStr = `^[a-zA-Z0-9._-]+$`
const ReUsernameEdgeStr = `^[a-zA-Z0-9]+$`

const MaxEmailLen = 254
const MaxUsernameLen = 20

type User struct {
	Id               int
	Name             string
	Email            string
	Password         string
	RegisteredAt     time.Time
	RegisteredAtStr  string
	NullIntroduction pgtype.Text
	Introduction     string
	Deleted          bool
	Banned           bool
	PasswordHased    bool
	RoleName         string
	RoleFrontId      string
	Permissions      []*Permission
	Super            bool
}

// func (u *User) FormatTimeStr() {
// 	u.RegisteredAtStr = utils.FormatTime(u.RegisteredAt, "YYYY年MM月DD日")
// }

func (u *User) FormatNullVals() {
	if u.NullIntroduction.Valid {
		u.Introduction = u.NullIntroduction.String
	}
}

func (u *User) Sanitize(p *bluemonday.Policy) {
	// u.Introduction = p.Sanitize(u.Introduction)
	u.Introduction = html.EscapeString(u.Introduction)
}

func userValidErr(str string) error {
	return errors.Join(AppErrUserValidFailed, errors.New(", "+str))
}

func (u *User) TrimSpace() {
	u.Email = strings.TrimSpace(u.Email)
	u.Name = strings.TrimSpace(u.Name)
}

func (u *User) Valid() error {
	lackField := ""

	if u.Email == "" {
		lackField = translator.LocalTpl("Email")
	}
	if u.Name == "" {
		lackField = translator.LocalTpl("Username")
	}
	if u.Password == "" {
		lackField = translator.LocalTpl("Password")
	}

	if lackField != "" {
		return userValidErr(translator.LocalTpl("Required", "FieldNames", lackField))
	}

	if err := ValidateEmail(u.Email); err != nil {
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Email")))
	}

	if err := ValidUsername(u.Name); err != nil {
		return err
	}

	rePassword := regexp.MustCompile(`[A-Za-z\d[:graph:]]{8,}`)
	reLetter := regexp.MustCompile(`[A-Za-z]`)
	reNum := regexp.MustCompile(`\d`)
	reNotaion := regexp.MustCompile(`[[:graph:]]`)
	originalPwd := []byte(u.Password)
	if !rePassword.Match(originalPwd) || !reLetter.Match(originalPwd) || !reNum.Match(originalPwd) || !reNotaion.Match(originalPwd) {
		// return userValidErr("password format error")
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Password")))
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

// Validate email string
func ValidateEmail(email string) error {
	if len(email) > MaxEmailLen {
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Email")))
	}
	re := regexp.MustCompile(ReEmailStr)
	if !re.Match([]byte(email)) {
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Email")))
	}
	return nil
}

func ValidUsername(username string) error {
	if len(username) > MaxUsernameLen {
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Username")))
	}

	reUsername := regexp.MustCompile(ReUsernameStr)
	if !reUsername.Match([]byte(username)) {
		return userValidErr(translator.LocalTpl("FormatError", "FieldNames", translator.LocalTpl("Username")))
	}
	return nil
}

func ExtractNameFromEmail(email string) string {
	name := strings.Split(email, "@")[0]
	reUsername := regexp.MustCompile(ReUsernameStr)
	reUsernameMiddle := regexp.MustCompile(ReUsernameMiddleStr)
	reUsernameEdge := regexp.MustCompile(ReUsernameEdgeStr)

	var res []string
	if reUsername.Match([]byte(name)) {
		return name
	} else {
		if !reUsernameEdge.Match([]byte(name[:1])) {
			name = name[1:]
			if len(name) < 1 {
				return ""
			}
		}

		if !reUsernameEdge.Match([]byte(name[len(name)-1:])) {
			name = name[:len(name)-1]
			if len(name) < 1 {
				return ""
			}
		}

		for _, rune := range name {
			if reUsernameMiddle.Match([]byte(string(rune))) {
				res = append(res, string(rune))
			} else {
				res = append(res, ".")
			}
		}
	}

	return strings.Join(res, "")
}
