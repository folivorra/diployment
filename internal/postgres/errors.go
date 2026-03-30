package postgres

import "errors"

var (
	// users
	ErrUserNotFound = errors.New("user is not founded in db")

	// projects
	ErrProjectAlreadyExist = errors.New("project already exists in db")
	ErrProjectNotFound     = errors.New("project is not founded in db")
)
