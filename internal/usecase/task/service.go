package task

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Service struct {
	repo      Repository
	templates TemplateRepository
	now       func() time.Time
}

func NewService(repo Repository, templates TemplateRepository) *Service {
	return &Service{
		repo:      repo,
		templates: templates,
		now:       func() time.Time { return time.Now().UTC() },
	}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*taskdomain.Task, error) {
	normalized, err := validateCreateInput(input)
	if err != nil {
		return nil, err
	}

	now := s.now()
	model := &taskdomain.Task{
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if normalized.DueDate != nil {
		model.DueDate = *normalized.DueDate
	}

	created, err := s.repo.Create(ctx, model)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*taskdomain.Task, error) {
	if id <= 0 {
		return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	normalized, err := validateUpdateInput(input)
	if err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	dueDate := current.DueDate
	if normalized.DueDate.Set {
		if normalized.DueDate.Value == nil {
			dueDate = taskdomain.Date{}
		} else {
			dueDate = *normalized.DueDate.Value
		}
	}

	model := &taskdomain.Task{
		ID:          id,
		Title:       normalized.Title,
		Description: normalized.Description,
		Status:      normalized.Status,
		DueDate:     dueDate,
		UpdatedAt:   s.now(),
	}

	updated, err := s.repo.Update(ctx, model)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, id)
}

func (s *Service) List(ctx context.Context, page ListTasksInput) ([]taskdomain.Task, error) {
	limit, offset, err := page.normalize()
	if err != nil {
		return nil, err
	}

	return s.repo.List(ctx, limit, offset)
}

func (s *Service) ListInRange(ctx context.Context, from, to taskdomain.Date, page ListTasksInput) ([]taskdomain.Task, error) {
	if from.IsZero() || to.IsZero() {
		return nil, fmt.Errorf("%w: from and to are required", ErrInvalidInput)
	}
	if from.After(to) {
		return nil, fmt.Errorf("%w: from must be <= to", ErrInvalidInput)
	}

	limit, offset, err := page.normalize()
	if err != nil {
		return nil, err
	}

	materialized, err := s.repo.ListInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	templates, err := s.templates.ListActiveInRange(ctx, from, to)
	if err != nil {
		return nil, err
	}

	merged := mergeOccurrences(materialized, templates, from, to)
	return pageSlice(merged, limit, offset), nil
}

func pageSlice(tasks []taskdomain.Task, limit, offset int) []taskdomain.Task {
	if offset >= len(tasks) {
		return []taskdomain.Task{}
	}
	end := offset + limit
	if end > len(tasks) {
		end = len(tasks)
	}
	return tasks[offset:end]
}

type occurrenceKey struct {
	templateID int64
	dueDate    taskdomain.Date
}

func mergeOccurrences(
	materialized []taskdomain.Task,
	templates []taskdomain.Template,
	from, to taskdomain.Date,
) []taskdomain.Task {
	// A materialized row wins over its virtual counterpart: if (template_id, due_date) already exists in the DB, skip generating a virtual occurrence for that slot.
	seen := make(map[occurrenceKey]struct{}, len(materialized))
	for _, t := range materialized {
		if t.TemplateID != nil {
			seen[occurrenceKey{*t.TemplateID, t.DueDate}] = struct{}{}
		}
	}

	out := make([]taskdomain.Task, 0, len(materialized))
	out = append(out, materialized...)

	for _, tpl := range templates {
		windowFrom := from
		if tpl.StartDate.After(windowFrom) {
			windowFrom = tpl.StartDate
		}
		windowTo := to
		if tpl.EndDate != nil && tpl.EndDate.Before(windowTo) {
			windowTo = *tpl.EndDate
		}
		if windowFrom.After(windowTo) {
			continue
		}

		occurrences := tpl.Rule.Occurrences(tpl.StartDate, windowFrom, windowTo)
		for _, occ := range occurrences {
			if _, ok := seen[occurrenceKey{tpl.ID, occ}]; ok {
				continue
			}
			out = append(out, virtualOccurrence(tpl, occ))
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].DueDate.Equal(out[j].DueDate) {
			return out[i].ID < out[j].ID
		}
		return out[i].DueDate.Before(out[j].DueDate)
	})

	return out
}

// UpdateOccurrenceStatus has two callers: the UI showing a materialized row (passes id) and the UI flipping a still-virtual occurrence (passes template_id + due_date, which triggers lazy materialization via UpsertInstance)
func (s *Service) UpdateOccurrenceStatus(ctx context.Context, input UpdateOccurrenceStatusInput) (*taskdomain.Task, error) {
	if !input.Status.Valid() {
		return nil, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	if input.ID > 0 {
		return s.repo.UpdateStatus(ctx, input.ID, input.Status, s.now())
	}

	if input.TemplateID <= 0 || input.DueDate.IsZero() {
		return nil, fmt.Errorf("%w: id or (template_id, due_date) is required", ErrInvalidInput)
	}

	tpl, err := s.templates.GetByID(ctx, input.TemplateID)
	if err != nil {
		return nil, err
	}

	if !occurrenceMatchesTemplate(tpl, input.DueDate) {
		return nil, fmt.Errorf("%w: %s is not a valid occurrence of template %d", ErrInvalidInput, input.DueDate, tpl.ID)
	}

	now := s.now()
	instance := &taskdomain.Task{
		TemplateID:  &tpl.ID,
		Title:       tpl.Title,
		Description: tpl.Description,
		Status:      input.Status,
		DueDate:     input.DueDate,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	return s.repo.UpsertInstance(ctx, instance)
}

func occurrenceMatchesTemplate(tpl *taskdomain.Template, due taskdomain.Date) bool {
	if due.Before(tpl.StartDate) {
		return false
	}
	if tpl.EndDate != nil && due.After(*tpl.EndDate) {
		return false
	}

	// Probe with a single-day window instead of adding a Contains method to the rule
	for _, d := range tpl.Rule.Occurrences(tpl.StartDate, due, due) {
		if d.Equal(due) {
			return true
		}
	}

	return false
}

func virtualOccurrence(tpl taskdomain.Template, due taskdomain.Date) taskdomain.Task {
	id := tpl.ID
	return taskdomain.Task{
		TemplateID:  &id,
		Title:       tpl.Title,
		Description: tpl.Description,
		Status:      taskdomain.StatusNew,
		DueDate:     due,
	}
}

func validateCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return CreateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = taskdomain.StatusNew
	}

	if !input.Status.Valid() {
		return CreateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}

func validateUpdateInput(input UpdateInput) (UpdateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Description = strings.TrimSpace(input.Description)

	if input.Title == "" {
		return UpdateInput{}, fmt.Errorf("%w: title is required", ErrInvalidInput)
	}

	if !input.Status.Valid() {
		return UpdateInput{}, fmt.Errorf("%w: invalid status", ErrInvalidInput)
	}

	return input, nil
}
