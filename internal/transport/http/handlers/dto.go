package handlers

import (
	"encoding/json"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type createTaskDTO struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	Status      taskdomain.Status  `json:"status"`
	DueDate     *taskdomain.Date   `json:"due_date,omitempty"`
}

type optionalNullableDate struct {
	Set   bool
	Value *taskdomain.Date
}

func (o *optionalNullableDate) UnmarshalJSON(data []byte) error {
	o.Set = true

	if string(data) == "null" {
		o.Value = nil
		return nil
	}

	var d taskdomain.Date
	if err := json.Unmarshal(data, &d); err != nil {
		return err
	}

	o.Value = &d
	return nil
}

type updateTaskDTO struct {
	Title       string               `json:"title"`
	Description string               `json:"description"`
	Status      taskdomain.Status    `json:"status"`
	DueDate     optionalNullableDate `json:"due_date"`
}

type taskDTO struct {
	ID          int64             `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
	DueDate     taskdomain.Date   `json:"due_date"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
	return taskDTO{
		ID:          task.ID,
		Title:       task.Title,
		Description: task.Description,
		Status:      task.Status,
		DueDate:     task.DueDate,
		CreatedAt:   task.CreatedAt,
		UpdatedAt:   task.UpdatedAt,
	}
}
