package service

import (
	"github.com/oodzchen/dproject/model"
	"github.com/oodzchen/dproject/store"
)

type User struct {
	Store *store.Store
}

func (u *User) Register(email string, password string, name string) (int, error) {
	user := &model.User{
		Email:    email,
		Name:     name,
		Password: password,
	}

	user.Sanitize()

	err := user.Valid()
	if err != nil {
		return 0, err
	}

	// log.Printf("user model is %v", user)

	err = user.EncryptPassword()
	if err != nil {
		return 0, err
	}

	// fmt.Printf("Password value: %s\n", user.Password)
	return u.Store.User.Create(user.Email, user.Password, user.Name)
}