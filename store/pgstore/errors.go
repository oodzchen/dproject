package pgstore

import "fmt"

type DBError struct {
	errorId string
	message string
}

func NewDBError(id string, msg string) error {
	return &DBError{id, msg}
}

func (err *DBError) Error() string {
	return fmt.Sprintf("Error id: %s\nError message: %s\n", err.errorId, err.message)
}
