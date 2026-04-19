package task

import (
	"context"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository interface {
	Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
	ListInRange(ctx context.Context, from, to taskdomain.Date) ([]taskdomain.Task, error)
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

type Usecase interface {
	Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error)
	GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
	Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error)
	Delete(ctx context.Context, id int64) error
	List(ctx context.Context) ([]taskdomain.Task, error)
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

type TemplateUsecase interface {
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

type ListTemplatesInput struct {
	Limit  int
	Offset int
}
