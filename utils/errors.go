package utils

import "fmt"

type AppError struct {
	errId   string
	message string
}

func NewError(msg string) error {
	return &AppError{"", msg}
}

func NewErrorWithId(id string, msg string) error {
	return &AppError{id, msg}
}

func (err *AppError) Error() string {
	var str string
	if err.errId != "" {
		str += fmt.Sprintf("Error id: %s\n", err.errId)
	}

	if err.message != "" {
		str += fmt.Sprintf("%s\n", err.message)
	}
	return str
}
