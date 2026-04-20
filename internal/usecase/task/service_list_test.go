package task

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

func TestService_List_Pagination(t *testing.T) {
	tests := []struct {
		name       string
		in         ListTasksInput
		wantLimit  int
		wantOffset int
		wantErr    bool
	}{
		{name: "defaults", in: ListTasksInput{}, wantLimit: defaultListLimit, wantOffset: 0},
		{name: "explicit", in: ListTasksInput{Limit: 50, Offset: 100}, wantLimit: 50, wantOffset: 100},
		{name: "clamped to max", in: ListTasksInput{Limit: 9999}, wantLimit: maxListLimit},
		{name: "negative offset", in: ListTasksInput{Offset: -1}, wantErr: true},
		{name: "negative limit", in: ListTasksInput{Limit: -1}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &fakeTaskRepo{}
			svc := &Service{repo: repo, templates: &fakeTemplateRepoForRange{}}

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

func TestService_ListInRange_PaginationOverMergedOccurrences(t *testing.T) {
	tpl := taskdomain.Template{
		ID:        7,
		Title:     "Daily",
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: dd(2026, time.April, 1),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tpl}},
	}

	from := dd(2026, time.April, 18)
	to := dd(2026, time.April, 27) // 10 occurrences

	got, err := svc.ListInRange(context.Background(), from, to, ListTasksInput{Limit: 3, Offset: 4})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("want 3 occurrences after paging, got %d", len(got))
	}

	want := []taskdomain.Date{
		dd(2026, time.April, 22),
		dd(2026, time.April, 23),
		dd(2026, time.April, 24),
	}
	for i, d := range want {
		if !got[i].DueDate.Equal(d) {
			t.Errorf("occ[%d] = %s, want %s", i, got[i].DueDate, d)
		}
	}
}

func TestService_ListInRange_PaginationOffsetBeyondEnd(t *testing.T) {
	tpl := taskdomain.Template{
		ID:        7,
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: dd(2026, time.April, 1),
	}
	svc := &Service{
		repo:      &fakeTaskRepo{},
		templates: &fakeTemplateRepoForRange{active: []taskdomain.Template{tpl}},
	}

	got, err := svc.ListInRange(context.Background(),
		dd(2026, time.April, 18), dd(2026, time.April, 20),
		ListTasksInput{Limit: 10, Offset: 100})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d items", len(got))
	}
}
