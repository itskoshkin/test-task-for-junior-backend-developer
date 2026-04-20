package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type fakeCRUDRepo struct {
	createIn   *taskdomain.Task
	createOut  *taskdomain.Task
	createErr  error
	getOut     *taskdomain.Task
	getErr     error
	updateIn   *taskdomain.Task
	updateOut  *taskdomain.Task
	updateErr  error
	deleteID   int64
	deleteErr  error
	listLimit  int
	listOffset int
	listOut    []taskdomain.Task
}

func (r *fakeCRUDRepo) Create(_ context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	r.createIn = t
	if r.createErr != nil {
		return nil, r.createErr
	}
	if r.createOut != nil {
		return r.createOut, nil
	}
	cp := *t
	cp.ID = 1
	return &cp, nil
}

func (r *fakeCRUDRepo) GetByID(_ context.Context, _ int64) (*taskdomain.Task, error) {
	return r.getOut, r.getErr
}

func (r *fakeCRUDRepo) Update(_ context.Context, t *taskdomain.Task) (*taskdomain.Task, error) {
	r.updateIn = t
	if r.updateErr != nil {
		return nil, r.updateErr
	}
	if r.updateOut != nil {
		return r.updateOut, nil
	}
	return t, nil
}

func (r *fakeCRUDRepo) Delete(_ context.Context, id int64) error {
	r.deleteID = id
	return r.deleteErr
}

func (r *fakeCRUDRepo) List(_ context.Context, limit, offset int) ([]taskdomain.Task, error) {
	r.listLimit = limit
	r.listOffset = offset
	return r.listOut, nil
}

func (r *fakeCRUDRepo) ListInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeCRUDRepo) UpdateStatus(_ context.Context, _ int64, _ taskdomain.Status, _ time.Time) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeCRUDRepo) UpsertInstance(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

func newCRUDService(repo *fakeCRUDRepo) *Service {
	return &Service{
		repo:      repo,
		templates: &fakeTemplateRepoForRange{},
		now:       fixedClock(2026, time.April, 19),
	}
}

func TestService_Create(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateInput
		wantErr bool
	}{
		{name: "ok", input: CreateInput{Title: "  Write docs  "}},
		{name: "ok with status", input: CreateInput{Title: "Write docs", Status: taskdomain.StatusInProgress}},
		{name: "empty title", input: CreateInput{Title: "   "}, wantErr: true},
		{name: "invalid status", input: CreateInput{Title: "x", Status: "garbage"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCRUDRepo{}
			svc := newCRUDService(repo)

			got, err := svc.Create(context.Background(), tc.input)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
				if repo.createIn != nil {
					t.Error("repo.Create should not be called on invalid input")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.createIn == nil {
				t.Fatal("repo.Create was not called")
			}
			if repo.createIn.Title != "Write docs" {
				t.Errorf("title not trimmed: %q", repo.createIn.Title)
			}
			if repo.createIn.Status == "" {
				t.Error("status should default to 'new' when empty")
			}
			if got.ID == 0 {
				t.Error("created task should have an ID")
			}
		})
	}
}

func TestService_Create_DefaultsStatusToNew(t *testing.T) {
	repo := &fakeCRUDRepo{}
	svc := newCRUDService(repo)

	if _, err := svc.Create(context.Background(), CreateInput{Title: "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.createIn.Status != taskdomain.StatusNew {
		t.Errorf("default status = %q, want %q", repo.createIn.Status, taskdomain.StatusNew)
	}
}

func TestService_Create_SetsDueDateWhenProvided(t *testing.T) {
	repo := &fakeCRUDRepo{}
	svc := newCRUDService(repo)
	due := taskdomain.NewDate(2026, time.April, 25)

	if _, err := svc.Create(context.Background(), CreateInput{Title: "x", DueDate: &due}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.createIn.DueDate.Equal(due) {
		t.Errorf("due_date = %s, want %s", repo.createIn.DueDate, due)
	}
}

func TestService_Create_LeavesDueDateZeroWhenOmitted(t *testing.T) {
	repo := &fakeCRUDRepo{}
	svc := newCRUDService(repo)

	if _, err := svc.Create(context.Background(), CreateInput{Title: "x"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.createIn.DueDate.IsZero() {
		t.Errorf("due_date = %s, want zero", repo.createIn.DueDate)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	svc := newCRUDService(&fakeCRUDRepo{})
	for _, id := range []int64{0, -1} {
		if _, err := svc.GetByID(context.Background(), id); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("id=%d: expected ErrInvalidInput, got %v", id, err)
		}
	}
}

func TestService_GetByID_PassesErrorThrough(t *testing.T) {
	repo := &fakeCRUDRepo{getErr: taskdomain.ErrNotFound}
	svc := newCRUDService(repo)

	_, err := svc.GetByID(context.Background(), 42)
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestService_Update(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		input   UpdateInput
		wantErr bool
	}{
		{name: "ok", id: 1, input: UpdateInput{Title: "Renamed", Status: taskdomain.StatusDone}},
		{name: "id=0", id: 0, input: UpdateInput{Title: "x", Status: taskdomain.StatusNew}, wantErr: true},
		{name: "empty title", id: 1, input: UpdateInput{Title: "  ", Status: taskdomain.StatusNew}, wantErr: true},
		{name: "invalid status", id: 1, input: UpdateInput{Title: "x", Status: "bad"}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeCRUDRepo{
				getOut: &taskdomain.Task{
					ID:      tc.id,
					DueDate: taskdomain.NewDate(2026, time.April, 19),
				},
			}
			svc := newCRUDService(repo)

			_, err := svc.Update(context.Background(), tc.id, tc.input)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
				if repo.updateIn != nil {
					t.Error("repo.Update should not be called on invalid input")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.updateIn == nil || repo.updateIn.ID != tc.id {
				t.Errorf("repo.Update called with wrong id: %+v", repo.updateIn)
			}
			if repo.getOut == nil {
				t.Fatal("repo.GetByID should provide current task")
			}
			if repo.updateIn.UpdatedAt.IsZero() {
				t.Error("UpdatedAt should be set")
			}
		})
	}
}

func TestService_Update_DueDateOmittedKeepsExisting(t *testing.T) {
	existing := taskdomain.NewDate(2026, time.April, 19)
	repo := &fakeCRUDRepo{
		getOut: &taskdomain.Task{ID: 1, DueDate: existing},
	}
	svc := newCRUDService(repo)

	_, err := svc.Update(context.Background(), 1, UpdateInput{
		Title:  "Renamed",
		Status: taskdomain.StatusDone,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updateIn.DueDate.Equal(existing) {
		t.Errorf("due_date = %s, want existing %s", repo.updateIn.DueDate, existing)
	}
}

func TestService_Update_DueDateNullClearsExisting(t *testing.T) {
	existing := taskdomain.NewDate(2026, time.April, 19)
	repo := &fakeCRUDRepo{
		getOut: &taskdomain.Task{ID: 1, DueDate: existing},
	}
	svc := newCRUDService(repo)

	_, err := svc.Update(context.Background(), 1, UpdateInput{
		Title:  "Renamed",
		Status: taskdomain.StatusDone,
		DueDate: OptionalDatePatch{
			Set: true,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updateIn.DueDate.IsZero() {
		t.Errorf("due_date = %s, want zero", repo.updateIn.DueDate)
	}
}

func TestService_Update_DueDateValueReplacesExisting(t *testing.T) {
	existing := taskdomain.NewDate(2026, time.April, 19)
	replacement := taskdomain.NewDate(2026, time.April, 25)
	repo := &fakeCRUDRepo{
		getOut: &taskdomain.Task{ID: 1, DueDate: existing},
	}
	svc := newCRUDService(repo)

	_, err := svc.Update(context.Background(), 1, UpdateInput{
		Title:  "Renamed",
		Status: taskdomain.StatusDone,
		DueDate: OptionalDatePatch{
			Set:   true,
			Value: &replacement,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !repo.updateIn.DueDate.Equal(replacement) {
		t.Errorf("due_date = %s, want %s", repo.updateIn.DueDate, replacement)
	}
}

func TestService_Delete_InvalidID(t *testing.T) {
	svc := newCRUDService(&fakeCRUDRepo{})
	if err := svc.Delete(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Delete_PassesThrough(t *testing.T) {
	repo := &fakeCRUDRepo{deleteErr: taskdomain.ErrNotFound}
	svc := newCRUDService(repo)

	err := svc.Delete(context.Background(), 42)
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
	if repo.deleteID != 42 {
		t.Errorf("delete id = %d, want 42", repo.deleteID)
	}
}
