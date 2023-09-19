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

// type PermissionModule string

// const (
// 	PermissionModuleUser       PermissionModule = "user"
// 	PermissionModuleArticle                     = "article"
// 	PermissionModulePermission                  = "permission"
// 	PermissionModuleRole                        = "role"
// )

// var PermissionModuleMap = map[PermissionModule]bool{
// 	PermissionModuleUser:       true,
// 	PermissionModuleArticle:    true,
// 	PermissionModulePermission: true,
// 	PermissionModuleRole:       true,
// }

type Permission struct {
	Id        int
	FrontId   string
	Name      string
	CreatedAt time.Time
	// Module    PermissionModule
	Module string
}

type PermissionListItem struct {
	Module string
	List   []*Permission
}

// func ValidPermissionModule(module string) bool {
// 	return PermissionModuleMap[PermissionModule(module)]
// }

// func GetPermissionModuleOptions() []PermissionModule {
// 	return []PermissionModule{
// 		PermissionModuleUser,
// 		PermissionModuleArticle,
// 		PermissionModulePermission,
// 		PermissionModuleRole,
// 	}
// }

func permissionValidErr(str string) error {
	return errors.Join(ErrValidPermissionFailed, errors.New(str))
}

const PermissionFrontIdMaxLen int = 50
const PermissionNameMaxLen int = 50

func (p *Permission) TrimSpace() {
	p.FrontId = strings.TrimSpace(p.FrontId)
	p.Name = strings.TrimSpace(p.Name)
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
		return permissionValidErr(fmt.Sprintf("require field: %s", lackField))
	}

	// if ok := ValidPermissionModule(string(p.Module)); !ok {
	// 	return permissionValidErr("module not exist")
	// }

	if utf8.RuneCountInString(p.FrontId) > PermissionFrontIdMaxLen {
		return permissionValidErr(fmt.Sprintf("front id length limit in %d characters", PermissionFrontIdMaxLen))
	}

	reFrontId := regexp.MustCompile(`^[\w\d_]{1,` + strconv.Itoa(PermissionFrontIdMaxLen) + `}$`)
	if !reFrontId.Match([]byte(p.FrontId)) {
		return permissionValidErr("front id format error")
	}

	if utf8.RuneCountInString(p.Name) > PermissionNameMaxLen {
		return permissionValidErr(fmt.Sprintf("name length limit in %d characters", PermissionNameMaxLen))
	}

	reName := regexp.MustCompile(`^[\w\d\s]{1,` + strconv.Itoa(PermissionNameMaxLen) + `}$`)
	if !reName.Match([]byte(p.Name)) {
		return permissionValidErr("name format error")
	}

	return nil
}
