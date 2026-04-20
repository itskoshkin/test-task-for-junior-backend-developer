package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type fakeTemplateUseCase struct {
	createIn  taskusecase.CreateTemplateInput
	createOut *taskdomain.Template
	createErr error
	getID     int64
	getOut    *taskdomain.Template
	getErr    error
	updateID  int64
	updateIn  taskusecase.UpdateTemplateInput
	updateOut *taskdomain.Template
	updateErr error
	deleteID  int64
	deleteErr error
	listPage  taskusecase.ListTemplatesInput
	listOut   []taskdomain.Template
	listErr   error
}

func (u *fakeTemplateUseCase) Create(_ context.Context, in taskusecase.CreateTemplateInput) (*taskdomain.Template, error) {
	u.createIn = in
	return u.createOut, u.createErr
}

func (u *fakeTemplateUseCase) GetByID(_ context.Context, id int64) (*taskdomain.Template, error) {
	u.getID = id
	return u.getOut, u.getErr
}

func (u *fakeTemplateUseCase) Update(_ context.Context, id int64, in taskusecase.UpdateTemplateInput) (*taskdomain.Template, error) {
	u.updateID = id
	u.updateIn = in
	return u.updateOut, u.updateErr
}

func (u *fakeTemplateUseCase) Delete(_ context.Context, id int64) error {
	u.deleteID = id
	return u.deleteErr
}

func (u *fakeTemplateUseCase) List(_ context.Context, page taskusecase.ListTemplatesInput) ([]taskdomain.Template, error) {
	u.listPage = page
	return u.listOut, u.listErr
}

func newTemplateTestRouter(uc taskusecase.TemplateUseCase) *mux.Router {
	r := mux.NewRouter().StrictSlash(true)
	h := NewTemplateHandler(uc)
	r.HandleFunc("/api/v1/task-templates", h.Create).Methods(http.MethodPost)
	r.HandleFunc("/api/v1/task-templates", h.List).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/task-templates/{id:[0-9]+}", h.GetByID).Methods(http.MethodGet)
	r.HandleFunc("/api/v1/task-templates/{id:[0-9]+}", h.Update).Methods(http.MethodPut)
	r.HandleFunc("/api/v1/task-templates/{id:[0-9]+}", h.Delete).Methods(http.MethodDelete)
	return r
}

func sampleTemplate() *taskdomain.Template {
	return &taskdomain.Template{
		ID:        7,
		Title:     "Drink water",
		Rule:      taskdomain.DailyRule{EveryN: 1},
		StartDate: taskdomain.NewDate(2026, time.April, 1),
	}
}

const validTemplateBody = `{"title":"Drink water","rule":{"type":"daily","params":{"every_n":1}},"start_date":"2026-04-01"}`

func TestTemplateHandler_Create(t *testing.T) {
	t.Run("201 created", func(t *testing.T) {
		uc := &fakeTemplateUseCase{createOut: sampleTemplate()}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPost, "/api/v1/task-templates", validTemplateBody)
		if rec.Code != http.StatusCreated {
			t.Fatalf("status = %d, want 201; body=%s", rec.Code, rec.Body)
		}
		if uc.createIn.Title != "Drink water" {
			t.Errorf("createIn.Title = %q", uc.createIn.Title)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		uc := &fakeTemplateUseCase{}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPost, "/api/v1/task-templates", `{bad`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on unknown rule type", func(t *testing.T) {
		uc := &fakeTemplateUseCase{}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPost, "/api/v1/task-templates",
			`{"title":"x","rule":{"type":"mystery","params":{}},"start_date":"2026-04-01"}`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})

	t.Run("400 on invalid input from use-case", func(t *testing.T) {
		uc := &fakeTemplateUseCase{createErr: taskusecase.ErrInvalidInput}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPost, "/api/v1/task-templates", validTemplateBody)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestTemplateHandler_GetByID(t *testing.T) {
	t.Run("200 ok", func(t *testing.T) {
		uc := &fakeTemplateUseCase{getOut: sampleTemplate()}
		rec := do(t, newTemplateTestRouter(uc), http.MethodGet, "/api/v1/task-templates/7", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.getID != 7 {
			t.Errorf("getID = %d, want 7", uc.getID)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeTemplateUseCase{getErr: taskdomain.ErrNotFound}
		rec := do(t, newTemplateTestRouter(uc), http.MethodGet, "/api/v1/task-templates/7", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestTemplateHandler_Update(t *testing.T) {
	t.Run("200 ok", func(t *testing.T) {
		uc := &fakeTemplateUseCase{updateOut: sampleTemplate()}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPut, "/api/v1/task-templates/7", validTemplateBody)
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.updateID != 7 {
			t.Errorf("updateID = %d, want 7", uc.updateID)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeTemplateUseCase{updateErr: taskdomain.ErrNotFound}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPut, "/api/v1/task-templates/7", validTemplateBody)
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})

	t.Run("400 on malformed JSON", func(t *testing.T) {
		uc := &fakeTemplateUseCase{}
		rec := do(t, newTemplateTestRouter(uc), http.MethodPut, "/api/v1/task-templates/7", `{bad`)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}

func TestTemplateHandler_Delete(t *testing.T) {
	t.Run("204 ok", func(t *testing.T) {
		uc := &fakeTemplateUseCase{}
		rec := do(t, newTemplateTestRouter(uc), http.MethodDelete, "/api/v1/task-templates/7", "")
		if rec.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", rec.Code)
		}
		if uc.deleteID != 7 {
			t.Errorf("deleteID = %d, want 7", uc.deleteID)
		}
	})

	t.Run("404 when not found", func(t *testing.T) {
		uc := &fakeTemplateUseCase{deleteErr: taskdomain.ErrNotFound}
		rec := do(t, newTemplateTestRouter(uc), http.MethodDelete, "/api/v1/task-templates/7", "")
		if rec.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", rec.Code)
		}
	})
}

func TestTemplateHandler_List(t *testing.T) {
	t.Run("200 ok", func(t *testing.T) {
		uc := &fakeTemplateUseCase{listOut: []taskdomain.Template{*sampleTemplate()}}
		rec := do(t, newTemplateTestRouter(uc), http.MethodGet, "/api/v1/task-templates?limit=5&offset=10", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200; body=%s", rec.Code, rec.Body)
		}
		if uc.listPage.Limit != 5 || uc.listPage.Offset != 10 {
			t.Errorf("listPage = %+v, want {5, 10}", uc.listPage)
		}
	})

	t.Run("400 on bad limit", func(t *testing.T) {
		uc := &fakeTemplateUseCase{}
		rec := do(t, newTemplateTestRouter(uc), http.MethodGet, "/api/v1/task-templates?limit=abc", "")
		if rec.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", rec.Code)
		}
	})
}
