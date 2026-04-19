package task

import (
	"time"
)

type Template struct {
	ID          int64          `json:"id"`
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Rule        RecurrenceRule `json:"-"`
	StartDate   Date           `json:"start_date"`
	EndDate     *Date          `json:"end_date,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}
