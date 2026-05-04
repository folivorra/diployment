package postgres

import "errors"

var (
	// users
	ErrUserNotFound = errors.New("user not found")

	// projects
	ErrProjectAlreadyExist = errors.New("project already exists")
	ErrProjectNotFound     = errors.New("project not found")

	// jobs
	ErrJobNotFound = errors.New("job not found")
)
