package pgstore

import "github.com/oodzchen/dproject/utils"

type PGStore struct{}

type DBConfig struct {
	DSN string
}

var pgDB *DB

const defaultPage = 1
const defaultPageSize = 50

func New(config *DBConfig) *PGStore {
	pgDB = &DB{config.DSN, nil}
	return &PGStore{}
}

// func (pg *PGStore) ConfigDB(dsn string) *PGStore {
// 	pgDB = &DB{dsn, nil}
// 	return &PGStore{}
// }

func (pg *PGStore) ConnectDB() error {
	err := CheckDB(true)
	if err != nil {
		return err
	}
	return pgDB.Connect()
}

func (pg *PGStore) CloseDB() {
	pgDB.Close()
}

func CheckDB(beforeConnect bool) error {
	if pgDB == nil {
		return utils.NewErrorWithId("store.pgstore.CheckDB", "Database config is required")
	}

	if beforeConnect {
		return nil
	}

	if pgDB.Pool == nil {
		return utils.NewErrorWithId("store.pgstore.CheckDB", "No database connection")
	}

	return nil
}

func (pg *PGStore) NewArticle() (any, error) {
	err := CheckDB(false)
	if err != nil {
		return nil, err
	}
	return &Article{pgDB.Pool}, nil
}

func (pg *PGStore) NewUser() (any, error) {
	err := CheckDB(false)
	if err != nil {
		return nil, err
	}
	return &User{pgDB.Pool}, nil
}
