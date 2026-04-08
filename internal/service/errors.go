package service

import "errors"

// User

var ErrFailedUserCreate = errors.New("failed to create user")
var ErrTgUserNotFound = errors.New("user not found")
var ErrInvalidGoal = errors.New("invalid goal")
var ErrInvalidDifficulty = errors.New("invalid difficulty")

// LeetCode

var ErrInvalidLeetCodeUsername = errors.New("invalid leetcode username")

// Task

var ErrTaskAlreadyExists = errors.New("task already exists for user")
var ErrTaskNotFound = errors.New("task not found for user")
