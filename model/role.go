package model

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type UserRoleFrontId string

const (
	DefaultUserRoleCommon    UserRoleFrontId = "common_user"
	DefaultUserRoleBanned                    = "banned_user"
	DefaultUserRoleModerator                 = "moderator"
	DefaultUserRoleAdmin                     = "admin"
)

type Role struct {
	Id                   int
	FrontId              string
	Name                 string
	CreatedAt            time.Time
	IsDefault            bool
	Permissions          []*Permission
	FormattedPermissions []*PermissionListItem
}

func roleValidErr(str string) error {
	return errors.Join(AppErrRoleValidFailed, errors.New(", "+str))
}

const RoleFrontIdMaxLen int = 50
const RoleNameMaxLen int = 50

func (r *Role) TrimSpace() {
	r.FrontId = strings.TrimSpace(r.FrontId)
	r.Name = strings.TrimSpace(r.Name)
}

func (r *Role) Valid(isUpdate bool) error {
	lackField := ""

	if !isUpdate && r.FrontId == "" {
		lackField = "front id"
	}

	if r.Name == "" {
		lackField = "name"
	}

	if lackField != "" {
		return roleValidErr(fmt.Sprintf("require field: %s", lackField))
	}

	if !isUpdate {
		if utf8.RuneCountInString(r.FrontId) > PermissionFrontIdMaxLen {
			return roleValidErr(fmt.Sprintf("front id length limit in %d characters", PermissionFrontIdMaxLen))
		}

		reFrontId := regexp.MustCompile(`^[\w\d_]{1,` + strconv.Itoa(PermissionFrontIdMaxLen) + `}$`)
		if !reFrontId.Match([]byte(r.FrontId)) {
			return roleValidErr("front id format error")
		}
	}

	if utf8.RuneCountInString(r.Name) > PermissionNameMaxLen {
		return roleValidErr(fmt.Sprintf("name length limit in %d characters", PermissionNameMaxLen))
	}

	reName := regexp.MustCompile(`^[\w\d\s]{1,` + strconv.Itoa(PermissionNameMaxLen) + `}$`)
	if !reName.Match([]byte(r.Name)) {
		return roleValidErr("name format error")
	}

	return nil
}
