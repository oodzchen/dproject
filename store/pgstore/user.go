package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	dbPool *pgxpool.Pool
}

func (u *User) List() ([]*model.User, error) {
	sqlStr := "select id, name, email, created_at from users where deleted = false"
	rows, err := u.dbPool.Query(context.Background(), sqlStr)

	if err != nil {
		return nil, err
	}

	var list []*model.User
	for rows.Next() {
		var item model.User
		err := rows.Scan(
			&item.Id,
			&item.Name,
			&item.Email,
			&item.RegisterAt,
		)

		if err != nil {
			return nil, err
		}

		item.FormatTimeStr()

		list = append(list, &item)
	}

	return list, nil
}

func (u *User) Create(item *model.User) (int, error) {
	sqlStr := fmt.Sprintf("insert into users (email, password, name) values ('%s', '%s', '%s') returning (id)",
		item.Email,
		item.Password,
		item.Name,
	)

	var id int
	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (u *User) Update(item *model.User) (int, error) {
	sqlStr := fmt.Sprintf("update users set introduction = %s, password = %s where id = %d",
		item.Introduction,
		item.Password,
		item.Id)
	var id int
	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}

func (u *User) Item(id int) (*model.User, error) {
	var item model.User
	sqlStr := fmt.Sprintf("select id, email, name, created_at from users where id = %d", id)

	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(&item.Id, &item.Email, &item.Name, &item.RegisterAt)
	if err != nil {
		return nil, err
	}

	item.FormatTimeStr()

	return &item, nil
}

func (u *User) Delete(id int) error {
	sqlStr := fmt.Sprintf("update users set deleted = true where id = %d", id)

	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Ban(id int) error {
	sqlStr := fmt.Sprintf("update users set banned = true where id = %d", id)

	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(nil)
	if err != nil {
		return err
	}
	return nil
}

func (u *User) Login(email string, pwd string) (int, error) {
	var id int
	var hasedPwd string
	sqlStr := fmt.Sprintf("select id, password from users where email = '%s'\n", email)

	fmt.Printf("sql string: %s", sqlStr)

	err := u.dbPool.QueryRow(context.Background(), sqlStr).Scan(&id, &hasedPwd)
	if err != nil {
		return 0, err
	}

	fmt.Printf("login query hashed password: %s\n", hasedPwd)

	err = bcrypt.CompareHashAndPassword([]byte(hasedPwd), []byte(pwd))

	if err != nil {
		return 0, err
	}

	fmt.Printf("pass!\n")

	return id, nil
}
