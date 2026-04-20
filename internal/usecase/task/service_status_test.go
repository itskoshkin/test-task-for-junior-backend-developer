package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type fakeRepoForStatus struct {
	updateStatusID  int64
	updateStatusIn  taskdomain.Status
	updateStatusOut *taskdomain.Task
	updateStatusErr error
	upsertIn        *taskdomain.Task
	upsertOut       *taskdomain.Task
	upsertCalled    bool
}

func (r *fakeRepoForStatus) Create(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeRepoForStatus) GetByID(_ context.Context, _ int64) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeRepoForStatus) Update(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeRepoForStatus) Delete(_ context.Context, _ int64) error { return nil }

func (r *fakeRepoForStatus) List(_ context.Context, _, _ int) ([]taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeRepoForStatus) ListInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeRepoForStatus) UpdateStatus(_ context.Context, id int64, status taskdomain.Status, _ time.Time) (*taskdomain.Task, error) {
	r.updateStatusID = id
	r.updateStatusIn = status
	return r.updateStatusOut, r.updateStatusErr
}

func (r *fakeRepoForStatus) UpsertInstance(_ context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	r.upsertCalled = true
	r.upsertIn = t
	if r.upsertOut != nil {
		return r.upsertOut, nil
	}
	cp := *t
	cp.ID = 999
	return &cp, nil
}

type fakeTemplateRepoForStatus struct {
	getOut *taskdomain.Template
	getErr error
}

func (r *fakeTemplateRepoForStatus) Create(_ context.Context, _ *taskdomain.Template) (*taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForStatus) GetByID(_ context.Context, _ int64) (*taskdomain.Template, error) {
	return r.getOut, r.getErr
}

func (r *fakeTemplateRepoForStatus) Update(_ context.Context, _ *taskdomain.Template) (*taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForStatus) Delete(_ context.Context, _ int64) error { return nil }

func (r *fakeTemplateRepoForStatus) List(_ context.Context, _, _ int) ([]taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForStatus) ListActiveInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Template, error) {
	return nil, nil
}

func newStatusService(repo *fakeRepoForStatus, templates *fakeTemplateRepoForStatus) *Service {
	return &Service{
		repo:      repo,
		templates: templates,
		now:       fixedClock(2026, time.April, 19),
	}
}

func TestUpdateOccurrenceStatus_InvalidStatus(t *testing.T) {
	svc := newStatusService(&fakeRepoForStatus{}, &fakeTemplateRepoForStatus{})
	_, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		ID: 1, Status: "garbage",
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateOccurrenceStatus_MissingKey(t *testing.T) {
	svc := newStatusService(&fakeRepoForStatus{}, &fakeTemplateRepoForStatus{})
	_, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		Status: taskdomain.StatusDone,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateOccurrenceStatus_ByID_CallsUpdateStatus(t *testing.T) {
	repo := &fakeRepoForStatus{
		updateStatusOut: &taskdomain.Task{ID: 100, Status: taskdomain.StatusDone},
	}
	svc := newStatusService(repo, &fakeTemplateRepoForStatus{})

	got, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		ID: 100, Status: taskdomain.StatusDone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.updateStatusID != 100 || repo.updateStatusIn != taskdomain.StatusDone {
		t.Errorf("UpdateStatus called with wrong args: id=%d status=%q", repo.updateStatusID, repo.updateStatusIn)
	}
	if got.ID != 100 {
		t.Errorf("unexpected returned task: %+v", got)
	}
	if repo.upsertCalled {
		t.Error("UpsertInstance should not be called when ID is set")
	}
}

func TestUpdateOccurrenceStatus_Virtual_Materializes(t *testing.T) {
	tpl := &taskdomain.Template{
		ID:          7,
		Title:       "Drink water",
		Description: "2L",
		Rule:        taskdomain.DailyRule{EveryN: 1},
		StartDate:   dd(2026, time.April, 1),
	}
	repo := &fakeRepoForStatus{}
	svc := newStatusService(repo, &fakeTemplateRepoForStatus{getOut: tpl})

	due := dd(2026, time.April, 19)
	got, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		TemplateID: 7, DueDate: due, Status: taskdomain.StatusDone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.upsertCalled {
		t.Fatal("UpsertInstance was not called")
	}
	if repo.upsertIn.TemplateID == nil || *repo.upsertIn.TemplateID != 7 {
		t.Errorf("template_id mismatch: %+v", repo.upsertIn.TemplateID)
	}
	if !repo.upsertIn.DueDate.Equal(due) {
		t.Errorf("due_date mismatch: got %s want %s", repo.upsertIn.DueDate, due)
	}
	if repo.upsertIn.Status != taskdomain.StatusDone {
		t.Errorf("status mismatch: %q", repo.upsertIn.Status)
	}
	if repo.upsertIn.Title != "Drink water" || repo.upsertIn.Description != "2L" {
		t.Errorf("title/description not inherited: %+v", repo.upsertIn)
	}
	if got.ID == 0 {
		t.Error("returned task should have ID after upsert")
	}
}

func TestUpdateOccurrenceStatus_Virtual_DateNotOnSchedule(t *testing.T) {
	tpl := &taskdomain.Template{
		ID:        7,
		Title:     "Odd days only",
		Rule:      taskdomain.MonthdayParityRule{Parity: taskdomain.ParityOdd},
		StartDate: dd(2026, time.April, 1),
	}
	repo := &fakeRepoForStatus{}
	svc := newStatusService(repo, &fakeTemplateRepoForStatus{getOut: tpl})

	// April 18 is even — not on the odd schedule.
	_, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		TemplateID: 7, DueDate: dd(2026, time.April, 18), Status: taskdomain.StatusDone,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.upsertCalled {
		t.Error("UpsertInstance should not be called for off-schedule date")
	}
}

func TestUpdateOccurrenceStatus_Virtual_DateOutsideTemplateWindow(t *testing.T) {
	tpl := &taskdomain.Template{
		ID:        7,
		Title:     "Bounded",
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: dd(2026, time.April, 10),
		EndDate:   datePtr(dd(2026, time.April, 20)),
	}
	repo := &fakeRepoForStatus{}
	svc := newStatusService(repo, &fakeTemplateRepoForStatus{getOut: tpl})

	_, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		TemplateID: 7, DueDate: dd(2026, time.April, 25), Status: taskdomain.StatusDone,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for date after end_date, got %v", err)
	}

	_, err = svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		TemplateID: 7, DueDate: dd(2026, time.April, 5), Status: taskdomain.StatusDone,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for date before start_date, got %v", err)
	}
}

func TestUpdateOccurrenceStatus_Virtual_TemplateNotFound(t *testing.T) {
	repo := &fakeRepoForStatus{}
	svc := newStatusService(repo, &fakeTemplateRepoForStatus{getErr: taskdomain.ErrNotFound})

	_, err := svc.UpdateOccurrenceStatus(context.Background(), UpdateOccurrenceStatusInput{
		TemplateID: 7, DueDate: dd(2026, time.April, 19), Status: taskdomain.StatusDone,
	})
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
