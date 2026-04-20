package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type fakeUseCase struct {
	createIn        taskusecase.CreateInput
	createOut       *taskdomain.Task
	createErr       error
	getByIDID       int64
	getByIDOut      *taskdomain.Task
	getByIDErr      error
	updateID        int64
	updateIn        taskusecase.UpdateInput
	updateOut       *taskdomain.Task
	updateErr       error
	deleteID        int64
	deleteErr       error
	listPage        taskusecase.ListTasksInput
	listOut         []taskdomain.Task
	listErr         error
	rangeFrom       taskdomain.Date
	rangeTo         taskdomain.Date
	rangePage       taskusecase.ListTasksInput
	rangeOut        []taskdomain.Task
	rangeErr        error
	updateStatusIn  taskusecase.UpdateOccurrenceStatusInput
	updateStatusOut *taskdomain.Task
	updateStatusErr error
}

func (u *fakeUseCase) Create(_ context.Context, in taskusecase.CreateInput) (*taskdomain.Task, error) {
	u.createIn = in
	return u.createOut, u.createErr
}

func (u *fakeUseCase) GetByID(_ context.Context, id int64) (*taskdomain.Task, error) {
	u.getByIDID = id
	return u.getByIDOut, u.getByIDErr
}

func (u *fakeUseCase) Update(_ context.Context, id int64, in taskusecase.UpdateInput) (*taskdomain.Task, error) {
	u.updateID = id
	u.updateIn = in
	return u.updateOut, u.updateErr
}

func (u *fakeUseCase) Delete(_ context.Context, id int64) error {
	u.deleteID = id
	return u.deleteErr
}

func (u *fakeUseCase) List(_ context.Context, page taskusecase.ListTasksInput) ([]taskdomain.Task, error) {
	u.listPage = page
	return u.listOut, u.listErr
}

func (u *fakeUseCase) ListInRange(_ context.Context, from, to taskdomain.Date, page taskusecase.ListTasksInput) ([]taskdomain.Task, error) {
	u.rangeFrom, u.rangeTo, u.rangePage = from, to, page
	return u.rangeOut, u.rangeErr
}

func (u *fakeUseCase) UpdateOccurrenceStatus(_ context.Context, in taskusecase.UpdateOccurrenceStatusInput) (*taskdomain.Task, error) {
	u.updateStatusIn = in
	return u.updateStatusOut, u.updateStatusErr
}

func newTestRouter(uc taskusecase.UseCase) *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	h := NewTaskHandler(uc)
	r.HandleFunc("/api/v1/tasks", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/tasks", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/tasks/status", h.UpdateStatus).Methods(http.MethodPatch)
	r.HandleFunc("/api/v1/tasks/{id:[0-9]+}", h.GetByID).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/tasks/{id:[0-9]+}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/tasks/{id:[0-9]+}", h.Delete).Methods(http.MethodDelete)
	return r
}

func do(t *testing.T, r http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	var req *http.Request
	if reader != nil {
		req = httptest.NewRequest(method, target, reader)
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, target, nil)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

func sampleTask() *taskdomain.Task {
	return &taskdomain.Task{
		ID:        42,
		Title:     "Write docs",
		Status:    taskdomain.StatusNew,
		DueDate:   taskdomain.NewDate(2026, time.April, 19),
		CreatedAt: time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2026, time.April, 19, 12, 0, 0, 0, time.UTC),
	}
}

func TestTaskHandler_Create(t *testing.T) {
	t.Run("201 created", func(t *testing.T) {
		uc := &fakeUseCase{createOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":"Write docs"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body)
		}
		if uc.createIn.Title != "Write docs" {
			t.Errorf("createIn.Title = %q", uc.createIn.Title)
		}
	})

	t.Run("passes due_date when provided", func(t *testing.T) {
		uc := &fakeUseCase{createOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks",
			`{"title":"Write docs","due_date":"2026-04-25"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body)
		}
		want := taskdomain.NewDate(2026, time.April, 25)
		if uc.createIn.DueDate == nil || !uc.createIn.DueDate.Equal(want) {
			t.Errorf("createIn.DueDate = %+v, want %s", uc.createIn.DueDate, want)
		}
	})

	t.Run("leaves due_date nil when omitted", func(t *testing.T) {
		uc := &fakeUseCase{createOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":"Write docs"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body)
		}
		if uc.createIn.DueDate != nil {
			t.Errorf("createIn.DueDate = %+v, want nil", uc.createIn.DueDate)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on unknown field", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":"x","extra":"nope"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on invalid input from use-case", func(t *testing.T) {
		uc := &fakeUseCase{createErr: taskusecase.ErrInvalidInput}
		rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":""}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestTaskHandler_GetByID(t *testing.T) {
	t.Run("200 ok", func(t *testing.T) {
		uc := &fakeUseCase{getByIDOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks/42", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}
		if uc.getByIDID != 42 {
			t.Errorf("getByIDID = %d, want 42", uc.getByIDID)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeUseCase{getByIDErr: taskdomain.ErrNotFound}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks/42", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("404 on non-numeric id (mux rejects)", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks/abc", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestTaskHandler_Update(t *testing.T) {
	t.Run("200 ok", func(t *testing.T) {
		uc := &fakeUseCase{updateOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPut, "/api/v1/tasks/42",
			`{"title":"Renamed","status":"done"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.updateID != 42 || uc.updateIn.Title != "Renamed" {
			t.Errorf("unexpected call: id=%d in=%+v", uc.updateID, uc.updateIn)
		}
		if uc.updateIn.DueDate.Set {
			t.Errorf("updateIn.DueDate.Set = true, want false")
		}
	})

	t.Run("200 clears due_date with null", func(t *testing.T) {
		uc := &fakeUseCase{updateOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPut, "/api/v1/tasks/42",
			`{"title":"Renamed","status":"done","due_date":null}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if !uc.updateIn.DueDate.Set || uc.updateIn.DueDate.Value != nil {
			t.Errorf("updateIn.DueDate = %+v, want Set=true Value=nil", uc.updateIn.DueDate)
		}
	})

	t.Run("200 sets due_date when provided", func(t *testing.T) {
		uc := &fakeUseCase{updateOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPut, "/api/v1/tasks/42",
			`{"title":"Renamed","status":"done","due_date":"2026-04-25"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		want := taskdomain.NewDate(2026, time.April, 25)
		if !uc.updateIn.DueDate.Set || uc.updateIn.DueDate.Value == nil || !uc.updateIn.DueDate.Value.Equal(want) {
			t.Errorf("updateIn.DueDate = %+v, want Set=true Value=%s", uc.updateIn.DueDate, want)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeUseCase{updateErr: taskdomain.ErrNotFound}
		rec := do(t, newTestRouter(uc), http.MethodPut, "/api/v1/tasks/42",
			`{"title":"x","status":"new"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodPut, "/api/v1/tasks/42", `{not-json`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestTaskHandler_Delete(t *testing.T) {
	t.Run("204 ok", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodDelete, "/api/v1/tasks/42", "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if uc.deleteID != 42 {
			t.Errorf("deleteID = %d, want 42", uc.deleteID)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeUseCase{deleteErr: taskdomain.ErrNotFound}
		rec := do(t, newTestRouter(uc), http.MethodDelete, "/api/v1/tasks/42", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestTaskHandler_List(t *testing.T) {
	t.Run("200 without window", func(t *testing.T) {
		uc := &fakeUseCase{listOut: []taskdomain.Task{*sampleTask()}}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", rec.Code)
		}

		var body []map[string]any
		if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if len(body) != 1 {
			t.Errorf("len(body) = %d, want 1", len(body))
		}
	})

	t.Run("200 with window", func(t *testing.T) {
		uc := &fakeUseCase{rangeOut: []taskdomain.Task{*sampleTask()}}
		rec := do(t, newTestRouter(uc), http.MethodGet,
			"/api/v1/tasks?from=2026-04-18&to=2026-04-25", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.rangeFrom.Day != 18 || uc.rangeTo.Day != 25 {
			t.Errorf("range mismatch: %s..%s", uc.rangeFrom, uc.rangeTo)
		}
	})

	t.Run("400 when only one of from/to", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks?from=2026-04-18", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on bad date", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodGet,
			"/api/v1/tasks?from=not-a-date&to=2026-04-25", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on non-numeric limit", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks?limit=abc", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("propagates pagination", func(t *testing.T) {
		uc := &fakeUseCase{}
		do(t, newTestRouter(uc), http.MethodGet, "/api/v1/tasks?limit=5&offset=10", "")
		if uc.listPage.Limit != 5 || uc.listPage.Offset != 10 {
			t.Errorf("listPage = %+v, want {5, 10}", uc.listPage)
		}
	})
}

func TestTaskHandler_UpdateStatus(t *testing.T) {
	t.Run("200 by id", func(t *testing.T) {
		uc := &fakeUseCase{updateStatusOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPatch, "/api/v1/tasks/status",
			`{"id":42,"status":"done"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.updateStatusIn.ID != 42 || uc.updateStatusIn.Status != taskdomain.StatusDone {
			t.Errorf("updateStatusIn = %+v", uc.updateStatusIn)
		}
	})

	t.Run("200 by template_id + due_date", func(t *testing.T) {
		uc := &fakeUseCase{updateStatusOut: sampleTask()}
		rec := do(t, newTestRouter(uc), http.MethodPatch, "/api/v1/tasks/status",
			`{"template_id":7,"due_date":"2026-04-19","status":"done"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.updateStatusIn.TemplateID != 7 || uc.updateStatusIn.DueDate.Day != 19 {
			t.Errorf("updateStatusIn = %+v", uc.updateStatusIn)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		uc := &fakeUseCase{}
		rec := do(t, newTestRouter(uc), http.MethodPatch, "/api/v1/tasks/status", `{bad`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 when use-case says invalid input", func(t *testing.T) {
		uc := &fakeUseCase{updateStatusErr: taskusecase.ErrInvalidInput}
		rec := do(t, newTestRouter(uc), http.MethodPatch, "/api/v1/tasks/status",
			`{"status":"done"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("404 when template not found", func(t *testing.T) {
		uc := &fakeUseCase{updateStatusErr: taskdomain.ErrNotFound}
		rec := do(t, newTestRouter(uc), http.MethodPatch, "/api/v1/tasks/status",
			`{"template_id":999,"due_date":"2026-04-19","status":"done"}`)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

// sanity: response body is valid JSON and preserves key shape
func TestTaskHandler_Create_ResponseShape(t *testing.T) {
	uc := &fakeUseCase{createOut: sampleTask()}
	rec := do(t, newTestRouter(uc), http.MethodPost, "/api/v1/tasks", `{"title":"x"}`)

	var body map[string]any
	if err := json.NewDecoder(bytes.NewReader(rec.Body.Bytes())).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	for _, key := range []string{"id", "title", "status", "due_date", "virtual"} {
		if _, ok := body[key]; !ok {
			t.Errorf("response missing key %q; got %v", key, body)
		}
	}
}
