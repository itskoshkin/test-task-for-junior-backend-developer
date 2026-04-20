package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type fakeTemplateRepo struct {
	createIn   *taskdomain.Template
	createOut  *taskdomain.Template
	updateIn   *taskdomain.Template
	updateOut  *taskdomain.Template
	getOut     *taskdomain.Template
	getErr     error
	deleteErr  error
	listLimit  int
	listOffset int
	listOut    []taskdomain.Template
}

func (r *fakeTemplateRepo) Create(_ context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error) {
	r.createIn = tpl
	if r.createOut != nil {
		return r.createOut, nil
	}
	cp := *tpl
	cp.ID = 42
	return &cp, nil
}

func (r *fakeTemplateRepo) GetByID(_ context.Context, _ int64) (*taskdomain.Template, error) {
	return r.getOut, r.getErr
}

func (r *fakeTemplateRepo) Update(_ context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error) {
	r.updateIn = tpl
	if r.updateOut != nil {
		return r.updateOut, nil
	}
	return tpl, nil
}

func (r *fakeTemplateRepo) Delete(_ context.Context, _ int64) error { return r.deleteErr }

func (r *fakeTemplateRepo) List(_ context.Context, limit, offset int) ([]taskdomain.Template, error) {
	r.listLimit = limit
	r.listOffset = offset
	return r.listOut, nil
}

func (r *fakeTemplateRepo) ListActiveInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Template, error) {
	return nil, nil
}

func fixedClock(y int, m time.Month, d int) func() time.Time {
	return func() time.Time { return time.Date(y, m, d, 0, 0, 0, 0, time.UTC) }
}

func newTemplateService(repo *fakeTemplateRepo) *TemplateService {
	return &TemplateService{repo: repo, now: fixedClock(2026, time.April, 19)}
}

func TestTemplateService_Create(t *testing.T) {
	validStart := taskdomain.NewDate(2026, time.April, 1)
	validRule := taskdomain.DailyRule{EveryN: 1}

	tests := []struct {
		name    string
		input   CreateTemplateInput
		wantErr bool
	}{
		{
			name: "ok",
			input: CreateTemplateInput{
				Title:     "  Clean room  ",
				Rule:      validRule,
				StartDate: validStart,
			},
		},
		{
			name: "empty title",
			input: CreateTemplateInput{
				Title:     "   ",
				Rule:      validRule,
				StartDate: validStart,
			},
			wantErr: true,
		},
		{
			name: "nil rule",
			input: CreateTemplateInput{
				Title:     "Clean",
				StartDate: validStart,
			},
			wantErr: true,
		},
		{
			name: "invalid rule",
			input: CreateTemplateInput{
				Title:     "Clean",
				Rule:      taskdomain.DailyRule{EveryN: 0},
				StartDate: validStart,
			},
			wantErr: true,
		},
		{
			name: "missing start_date",
			input: CreateTemplateInput{
				Title: "Clean",
				Rule:  validRule,
			},
			wantErr: true,
		},
		{
			name: "end before start",
			input: CreateTemplateInput{
				Title:     "Clean",
				Rule:      validRule,
				StartDate: validStart,
				EndDate:   datePtr(taskdomain.NewDate(2026, time.March, 1)),
			},
			wantErr: true,
		},
		{
			name: "end equal to start is ok",
			input: CreateTemplateInput{
				Title:     "Clean",
				Rule:      validRule,
				StartDate: validStart,
				EndDate:   datePtr(validStart),
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeTemplateRepo{}
			svc := newTemplateService(repo)

			_, err := svc.Create(context.Background(), tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				if !errors.Is(err, ErrInvalidInput) {
					t.Errorf("error should wrap ErrInvalidInput, got %v", err)
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
			if repo.createIn.Title != "Clean room" && repo.createIn.Title != "Clean" {
				t.Errorf("title not trimmed: %q", repo.createIn.Title)
			}
			if repo.createIn.CreatedAt.IsZero() || repo.createIn.UpdatedAt.IsZero() {
				t.Error("timestamps not set")
			}
		})
	}
}

func TestTemplateService_GetByID_InvalidID(t *testing.T) {
	svc := newTemplateService(&fakeTemplateRepo{})
	if _, err := svc.GetByID(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
	if _, err := svc.GetByID(context.Background(), -5); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTemplateService_Update_InvalidID(t *testing.T) {
	svc := newTemplateService(&fakeTemplateRepo{})
	_, err := svc.Update(context.Background(), 0, UpdateTemplateInput{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTemplateService_Delete_InvalidID(t *testing.T) {
	svc := newTemplateService(&fakeTemplateRepo{})
	if err := svc.Delete(context.Background(), 0); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("expected ErrInvalidInput, got %v", err)
	}
}

func TestTemplateService_List_Pagination(t *testing.T) {
	tests := []struct {
		name       string
		in         ListTemplatesInput
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{name: "defaults", in: ListTemplatesInput{}, wantLimit: defaultListLimit, wantOffset: 0},
		{name: "explicit", in: ListTemplatesInput{Limit: 50, Offset: 100}, wantLimit: 50, wantOffset: 100},
		{name: "clamped to max", in: ListTemplatesInput{Limit: 9999}, wantLimit: maxListLimit},
		{name: "negative offset", in: ListTemplatesInput{Offset: -1}, wantErr: true},
		{name: "negative limit", in: ListTemplatesInput{Limit: -1}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeTemplateRepo{}
			svc := newTemplateService(repo)

			_, err := svc.List(context.Background(), tc.in)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidInput) {
					t.Fatalf("expected ErrInvalidInput, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if repo.listLimit != tc.wantLimit {
				t.Errorf("limit = %d, want %d", repo.listLimit, tc.wantLimit)
			}
			if repo.listOffset != tc.wantOffset {
				t.Errorf("offset = %d, want %d", repo.listOffset, tc.wantOffset)
			}
		})
	}
}

func datePtr(d taskdomain.Date) *taskdomain.Date { return &d }
