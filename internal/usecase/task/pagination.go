package task

import (
	"fmt"
)

const (
	defaultListLimit = 20
	maxListLimit     = 100
)

type Pagination struct {
	Limit  int
	Offset int
}

func (p Pagination) normalize() (limit, offset int, err error) {
	if p.Offset < 0 {
		return 0, 0, fmt.Errorf("%w: offset must be >= 0", ErrInvalidInput)
	}
	if p.Limit < 0 {
		return 0, 0, fmt.Errorf("%w: limit must be >= 0", ErrInvalidInput)
	}

	limit = p.Limit
	if limit == 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	return limit, p.Offset, nil
}
