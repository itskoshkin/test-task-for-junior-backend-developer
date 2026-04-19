package task

import (
	"errors"
)

var (
	ErrNotFound    = errors.New("task not found")
	ErrInvalidRule = errors.New("invalid recurrence rule")
)
