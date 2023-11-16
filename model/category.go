package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

type CategoryState string

const (
	CategoryStateAll        CategoryState = "all"
	CategoryStateApproved                 = "approved"
	CategoryStateUnapproved               = "unapproved"
)

type Category struct {
	Id              int
	FrontId         string
	Name            string
	Describe        string
	AuthorId        string
	CreatedAt       time.Time
	Approved        bool
	ApprovalComment string
}

func categoryValidErr(str string) error {
	return errors.Join(AppErrCategoryValidFailed, errors.New(", "+str))
}

func (p *Category) TrimSpace() {
	p.Name = strings.TrimSpace(p.Name)
}

func (p *Category) Valid() error {
	lackField := ""

	if p.Name == "" {
		lackField = "name"
	}

	if lackField != "" {
		return categoryValidErr(fmt.Sprintf("require field: %s", lackField))
	}

	return nil
}
