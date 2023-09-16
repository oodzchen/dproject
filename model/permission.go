package model

import (
	"errors"
	"fmt"
	"time"
)

type PermissionModule string

const (
	PermissionModuleUser       PermissionModule = "user"
	PermissionModuleArticle                     = "article"
	PermissionModulePermission                  = "permission"
	PermissionModuleRole                        = "role"
)

var PermissionModuleMap = map[PermissionModule]bool{
	PermissionModuleUser:       true,
	PermissionModuleArticle:    true,
	PermissionModulePermission: true,
	PermissionModuleRole:       true,
}

type Permission struct {
	Id        int
	FrontId   string
	Name      string
	CreatedAt time.Time
	Module    PermissionModule
}

func ValidPermissionModule(module string) bool {
	return PermissionModuleMap[PermissionModule(module)]
}

func GetPermissionModuleOptions() []PermissionModule {
	return []PermissionModule{
		PermissionModuleUser,
		PermissionModuleArticle,
		PermissionModulePermission,
		PermissionModuleRole,
	}
}

func permissionValidErr(str string) error {
	return errors.Join(ErrValidPermissionFailed, errors.New(str))
}

func (p *Permission) Valid() error {
	lackField := ""

	if p.Module == "" {
		lackField = "module"
	}
	if p.FrontId == "" {
		lackField = "front id"
	}
	if p.Name == "" {
		lackField = "name"
	}

	if lackField != "" {
		return userValidErr(fmt.Sprintf("require field: %s", lackField))
	}

	// ok := utils.ValidateEmail(p.Email)
	// if !ok {
	// 	return userValidErr("email format error")
	// }

	// reUsername := regexp.MustCompile(`^[\p{L}\p{N}\s]+$`)
	// if !reUsername.Match([]byte(p.Name)) {
	// 	return userValidErr("username format error")
	// }

	// rePassword := regexp.MustCompile(`[A-Za-z\d[:graph:]]{8,}`)
	// reLetter := regexp.MustCompile(`[A-Za-z]`)
	// reNum := regexp.MustCompile(`\d`)
	// reNotaion := regexp.MustCompile(`[[:graph:]]`)
	// originalPwd := []byte(p.Password)
	// if !rePassword.Match(originalPwd) || !reLetter.Match(originalPwd) || !reNum.Match(originalPwd) || !reNotaion.Match(originalPwd) {
	// 	return userValidErr("password format error")
	// }

	return nil
}
