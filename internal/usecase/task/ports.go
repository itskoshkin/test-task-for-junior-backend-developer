package task

import (
	"context"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]taskdomain.Task, error)
	ListInRange(ctx context.Context, from, to taskdomain.Date) ([]taskdomain.Task, error)
	UpdateStatus(ctx context.Context, id int64, status taskdomain.Status, updatedAt time.Time) (*taskdomain.Task, error)
	UpsertInstance(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
}

type TemplateRepository interface {
	Create(ctx context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Template, error)
	Update(ctx context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, limit, offset int) ([]taskdomain.Template, error)
	ListActiveInRange(ctx context.Context, from, to taskdomain.Date) ([]taskdomain.Template, error)
}

type UseCase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page ListTasksInput) ([]taskdomain.Task, error)
	ListInRange(ctx context.Context, from, to taskdomain.Date, page ListTasksInput) ([]taskdomain.Task, error)
	UpdateOccurrenceStatus(ctx context.Context, input UpdateOccurrenceStatusInput) (*taskdomain.Task, error)
}

type ListTasksInput = Pagination

type UpdateOccurrenceStatusInput struct {
	ID         int64
	TemplateID int64
	DueDate    taskdomain.Date
	Status     taskdomain.Status
}

type OptionalDatePatch struct {
	Set   bool
	Value *taskdomain.Date
}

type CreateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	DueDate     *taskdomain.Date
}

type UpdateInput struct {
	Title       string
	Description string
	Status      taskdomain.Status
	DueDate     OptionalDatePatch
}

type TemplateUseCase interface {
	Create(ctx context.Context, input CreateTemplateInput) (*taskdomain.Template, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Template, error)
	Update(ctx context.Context, id int64, input UpdateTemplateInput) (*taskdomain.Template, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context, page ListTemplatesInput) ([]taskdomain.Template, error)
}

type CreateTemplateInput struct {
	Title       string
	Description string
	Rule        taskdomain.RecurrenceRule
	StartDate   taskdomain.Date
	EndDate     *taskdomain.Date
}

type UpdateTemplateInput struct {
	Title       string
	Description string
	Rule        taskdomain.RecurrenceRule
	StartDate   taskdomain.Date
	EndDate     *taskdomain.Date
}

type ListTemplatesInput = Pagination
