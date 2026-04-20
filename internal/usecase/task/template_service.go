package task

import (
	"context"
	"fmt"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type TemplateService struct {
	repo TemplateRepository
	now  func() time.Time
}

func NewTemplateService(repo TemplateRepository) *TemplateService {
	return &TemplateService{
		repo: repo,
		now:  func() time.Time { return time.Now().UTC() },
	}
}

func (s *TemplateService) Create(ctx context.Context, input CreateTemplateInput) (*taskdomain.Template, error) {
	normalized, err := validateCreateTemplateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskdomain.Template{
		Title:       normalized.Title,
		Description: normalized.Description,
		Rule:        normalized.Rule,
		StartDate:   normalized.StartDate,
		EndDate:     normalized.EndDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.Create(ctx, model)
}

func (s *TemplateService) GetByID(ctx context.Context, id int64) (*taskdomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *TemplateService) Update(ctx context.Context, id int64, input UpdateTemplateInput) (*taskdomain.Template, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateTemplateInput(input)
	if err != nil {
		return nil, err
	}

	model := &taskdomain.Template{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Rule:        normalized.Rule,
		StartDate:   normalized.StartDate,
		EndDate:     normalized.EndDate,
		UpdatedAt:   s.now(),
	}

	return s.repo.Update(ctx, model)
}

func (s *TemplateService) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *TemplateService) List(ctx context.Context, page ListTemplatesInput) ([]taskdomain.Template, error) {
	limit, offset, err := page.normalize()
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, limit, offset)
}

func validateCreateTemplateInput(input CreateTemplateInput) (CreateTemplateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateTemplateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if err := validateRuleAndDates(input.Rule, input.StartDate, input.EndDate); err != nil {
		return CreateTemplateInput{}, err
	}

	return input, nil
}

func validateUpdateTemplateInput(input UpdateTemplateInput) (UpdateTemplateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateTemplateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if err := validateRuleAndDates(input.Rule, input.StartDate, input.EndDate); err != nil {
		return UpdateTemplateInput{}, err
	}

	return input, nil
}

func validateRuleAndDates(rule taskdomain.RecurrenceRule, start taskdomain.Date, end *taskdomain.Date) error {
	if rule == nil {
		return fmt.Errorf("%w: rule is required", ErrInvalidInput)
	}
	if err := rule.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}

	if start.IsZero() {
		return fmt.Errorf("%w: start_date is required", ErrInvalidInput)
	}
	if end != nil && end.Before(start) {
		return fmt.Errorf("%w: end_date must be on or after start_date", ErrInvalidInput)
	}

	return nil
}

