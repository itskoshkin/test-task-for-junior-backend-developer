package task

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type fakeTaskRepo struct {
	listInRange []taskdomain.Task
	listErr     error
}

func (r *fakeTaskRepo) Create(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeTaskRepo) GetByID(_ context.Context, _ int64) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeTaskRepo) Update(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeTaskRepo) Delete(_ context.Context, _ int64) error { return nil }

func (r *fakeTaskRepo) List(_ context.Context) ([]taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeTaskRepo) ListInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Task, error) {
	return r.listInRange, r.listErr
}

func (r *fakeTaskRepo) UpdateStatus(_ context.Context, _ int64, _ taskdomain.Status, _ time.Time) (*taskdomain.Task, error) {
	return nil, nil
}

func (r *fakeTaskRepo) UpsertInstance(_ context.Context, _ *taskdomain.Task) (*taskdomain.Task, error) {
	return nil, nil
}

type fakeTemplateRepoForRange struct {
	active []taskdomain.Template
}

func (r *fakeTemplateRepoForRange) Create(_ context.Context, _ *taskdomain.Template) (*taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForRange) GetByID(_ context.Context, _ int64) (*taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForRange) Update(_ context.Context, _ *taskdomain.Template) (*taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForRange) Delete(_ context.Context, _ int64) error { return nil }

func (r *fakeTemplateRepoForRange) List(_ context.Context, _, _ int) ([]taskdomain.Template, error) {
	return nil, nil
}

func (r *fakeTemplateRepoForRange) ListActiveInRange(_ context.Context, _, _ taskdomain.Date) ([]taskdomain.Template, error) {
	return r.active, nil
}

func int64Ptr(v int64) *int64 { return &v }

func dd(y int, m time.Month, day int) taskdomain.Date { return taskdomain.NewDate(y, m, day) }

func TestService_ListInRange_InvalidInput(t *testing.T) {
	svc := &Service{repo: &fakeTaskRepo{}, templates: &fakeTemplateRepoForRange{}}

	cases := []struct {
		name     string
		from, to taskdomain.Date
	}{
		{name: "zero from", from: taskdomain.Date{}, to: dd(2026, time.April, 30)},
		{name: "zero to", from: dd(2026, time.April, 1), to: taskdomain.Date{}},
		{name: "from after to", from: dd(2026, time.April, 30), to: dd(2026, time.April, 1)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.ListInRange(context.Background(), tc.from, tc.to)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("expected ErrInvalidInput, got %v", err)
			}
		})
	}
}

func TestService_ListInRange_PureVirtual(t *testing.T) {
	tpl := taskdomain.Template{
		ID:          7,
		Title:       "Drink water",
		Description: "2L",
		Rule:        taskdomain.DailyRule{EveryN: 1},
		StartDate:   dd(2026, time.April, 1),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tpl}},
	}

	got, err := svc.ListInRange(context.Background(), dd(2026, time.April, 18), dd(2026, time.April, 20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 occurrences, got %d", len(got))
	}

	for i, want := range []taskdomain.Date{dd(2026, time.April, 18), dd(2026, time.April, 19), dd(2026, time.April, 20)} {
		if !got[i].DueDate.Equal(want) {
			t.Errorf("occ[%d] due = %s, want %s", i, got[i].DueDate, want)
		}
		if got[i].ID != 0 {
			t.Errorf("virtual occ should have ID=0, got %d", got[i].ID)
		}
		if got[i].Status != taskdomain.StatusNew {
			t.Errorf("virtual occ status = %q, want %q", got[i].Status, taskdomain.StatusNew)
		}
		if got[i].TemplateID == nil || *got[i].TemplateID != 7 {
			t.Errorf("template_id mismatch: %+v", got[i].TemplateID)
		}
		if got[i].Title != "Drink water" {
			t.Errorf("title inherited wrong: %q", got[i].Title)
		}
	}
}

func TestService_ListInRange_MaterializedWinsOverVirtual(t *testing.T) {
	tpl := taskdomain.Template{
		ID:        7,
		Title:     "Drink water",
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: dd(2026, time.April, 1),
	}
	mat := taskdomain.Task{
		ID:         100,
		TemplateID: int64Ptr(7),
		Title:      "Drink water",
		Status:     taskdomain.StatusDone,
		DueDate:    dd(2026, time.April, 19),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{listInRange: []taskdomain.Task{mat}},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tpl}},
	}

	got, err := svc.ListInRange(context.Background(), dd(2026, time.April, 18), dd(2026, time.April, 20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3, got %d", len(got))
	}

	// The 19th should be the materialized one (status=done, id=100).
	var on19 *taskdomain.Task
	for i := range got {
		if got[i].DueDate.Equal(dd(2026, time.April, 19)) {
			on19 = &got[i]
			break
		}
	}
	if on19 == nil {
		t.Fatal("no task on April 19")
	}
	if on19.ID != 100 {
		t.Errorf("April 19 should be materialized (id=100), got id=%d", on19.ID)
	}
	if on19.Status != taskdomain.StatusDone {
		t.Errorf("April 19 status = %q, want done", on19.Status)
	}

	// The 18th and 20th should still be virtual (id=0).
	for _, d := range []taskdomain.Date{dd(2026, time.April, 18), dd(2026, time.April, 20)} {
		for _, tk := range got {
			if tk.DueDate.Equal(d) && tk.ID != 0 {
				t.Errorf("date %s expected virtual (id=0), got id=%d", d, tk.ID)
			}
		}
	}
}

func TestService_ListInRange_StandaloneTaskPassesThrough(t *testing.T) {
	standalone := taskdomain.Task{
		ID:      200,
		Title:   "One-off",
		Status:  taskdomain.StatusNew,
		DueDate: dd(2026, time.April, 19),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{listInRange: []taskdomain.Task{standalone}},
		templates: &fakeTemplateRepoForRange{},
	}

	got, err := svc.ListInRange(context.Background(), dd(2026, time.April, 18), dd(2026, time.April, 20))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(got, []taskdomain.Task{standalone}) {
		t.Errorf("standalone task should pass through unchanged, got %+v", got)
	}
}

func TestService_ListInRange_TemplateWindowClipping(t *testing.T) {
	tpl := taskdomain.Template{
		ID:        5,
		Title:     "Bounded",
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: dd(2026, time.April, 20),
		EndDate:   datePtr(dd(2026, time.April, 22)),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tpl}},
	}

	got, err := svc.ListInRange(context.Background(), dd(2026, time.April, 18), dd(2026, time.April, 25))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 occurrences (April 20..22), got %d", len(got))
	}

	for i, want := range []taskdomain.Date{dd(2026, time.April, 20), dd(2026, time.April, 21), dd(2026, time.April, 22)} {
		if !got[i].DueDate.Equal(want) {
			t.Errorf("occ[%d] = %s, want %s", i, got[i].DueDate, want)
		}
	}
}

func TestService_ListInRange_SortedByDueDate(t *testing.T) {
	tplA := taskdomain.Template{ID: 1, Title: "A", Rule: taskdomain.DailyRule{EveryN: 2}, StartDate: dd(2026, time.April, 18)}
	tplB := taskdomain.Template{ID: 2, Title: "B", Rule: taskdomain.DailyRule{EveryN: 3}, StartDate: dd(2026, time.April, 19)}
	svc := &Service{
		repo:      &fakeTaskRepo{},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tplA, tplB}},
	}

	got, err := svc.ListInRange(context.Background(), dd(2026, time.April, 18), dd(2026, time.April, 22))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	for i := 1; i < len(got); i++ {
		if got[i].DueDate.Before(got[i-1].DueDate) {
			t.Fatalf("not sorted at %d: %s after %s", i, got[i].DueDate, got[i-1].DueDate)
		}
	}
}
