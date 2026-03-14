package service

import "errors"

// User

var ErrFailedUserCreate = errors.New("failed to create user")
var ErrTgUserNotFound = errors.New("user not found")

// Task

var ErrTaskAlreadyExists = errors.New("task already exists for user")
