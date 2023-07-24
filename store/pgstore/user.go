package pgstore

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/oodzchen/dproject/model"
)

type User struct {
	dbPool *pgxpool.Pool
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
		return -1, err
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
