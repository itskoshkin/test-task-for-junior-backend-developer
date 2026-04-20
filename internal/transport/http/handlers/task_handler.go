package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TaskHandler struct {
	useCase taskusecase.UseCase
}

func NewTaskHandler(useCase taskusecase.UseCase) *TaskHandler {
	return &TaskHandler{useCase: useCase}
}

func (h *TaskHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createTaskDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.useCase.Create(r.Context(), taskusecase.CreateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		DueDate:     req.DueDate,
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, newTaskDTO(created))
}

func (h *TaskHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	task, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(task))
}

func (h *TaskHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req updateTaskDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.useCase.Update(r.Context(), id, taskusecase.UpdateInput{
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		DueDate: taskusecase.OptionalDatePatch{
			Set:   req.DueDate.Set,
			Value: req.DueDate.Value,
		},
	})
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func (h *TaskHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err = h.useCase.Delete(r.Context(), id); err != nil {
		writeUseCaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fromStr, toStr := q.Get("from"), q.Get("to")

	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	page := taskusecase.ListTasksInput{Limit: limit, Offset: offset}

	var tasks []taskdomain.Task

	switch {
	case fromStr == "" && toStr == "":
		tasks, err = h.useCase.List(r.Context(), page)
	case fromStr != "" && toStr != "":
		from, parseErr := taskdomain.ParseDate(fromStr)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid from"))
			return
		}
		to, parseErr := taskdomain.ParseDate(toStr)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, errors.New("invalid to"))
			return
		}
		tasks, err = h.useCase.ListInRange(r.Context(), from, to, page)
	default:
		writeError(w, http.StatusBadRequest, errors.New("from and to must be provided together"))
		return
	}

	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	response := make([]taskDTO, 0, len(tasks))
	for i := range tasks {
		response = append(response, newTaskDTO(&tasks[i]))
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *TaskHandler) UpdateStatus(w http.ResponseWriter, r *http.Request) {
	var req updateStatusDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	input := taskusecase.UpdateOccurrenceStatusInput{Status: req.Status}
	if req.ID != nil {
		input.ID = *req.ID
	}
	if req.TemplateID != nil {
		input.TemplateID = *req.TemplateID
	}
	if req.DueDate != nil {
		input.DueDate = *req.DueDate
	}

	updated, err := h.useCase.UpdateOccurrenceStatus(r.Context(), input)
	if err != nil {
		writeUseCaseError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, newTaskDTO(updated))
}

func getIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing task id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil {
		return 0, errors.New("invalid task id")
	}

	if id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}

func decodeJSON(r *http.Request, dst any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		return err
	}

	return nil
}

func writeUseCaseError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{
		"error": err.Error(),
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(payload)
}
