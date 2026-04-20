package handlers

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type TemplateHandler struct {
	useCase taskusecase.TemplateUsecase
}

func NewTemplateHandler(useCase taskusecase.TemplateUsecase) *TemplateHandler {
	return &TemplateHandler{useCase: useCase}
}

func (h *TemplateHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req templateMutationDTO
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := req.Rule.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	created, err := h.useCase.Create(r.Context(), taskusecase.CreateTemplateInput{
		Title:       req.Title,
		Description: req.Description,
		Rule:        rule,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeTemplate(w, http.StatusCreated, created)
}

func (h *TemplateHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	tpl, err := h.useCase.GetByID(r.Context(), id)
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeTemplate(w, http.StatusOK, tpl)
}

func (h *TemplateHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	var req templateMutationDTO
	if err = decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	rule, err := req.Rule.toDomain()
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	updated, err := h.useCase.Update(r.Context(), id, taskusecase.UpdateTemplateInput{
		Title:       req.Title,
		Description: req.Description,
		Rule:        rule,
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	writeTemplate(w, http.StatusOK, updated)
}

func (h *TemplateHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := getTemplateIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	if err = h.useCase.Delete(r.Context(), id); err != nil {
		writeUsecaseError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TemplateHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	templates, err := h.useCase.List(r.Context(), taskusecase.ListTemplatesInput{
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeUsecaseError(w, err)
		return
	}

	response := make([]templateDTO, 0, len(templates))
	for i := range templates {
		var dto templateDTO
		dto, err = newTemplateDTO(&templates[i])
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		response = append(response, dto)
	}

	writeJSON(w, http.StatusOK, response)
}

func writeTemplate(w http.ResponseWriter, status int, tpl *taskdomain.Template) {
	dto, err := newTemplateDTO(tpl)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, status, dto)
}

func getTemplateIDFromRequest(r *http.Request) (int64, error) {
	rawID := mux.Vars(r)["id"]
	if rawID == "" {
		return 0, errors.New("missing template id")
	}

	id, err := strconv.ParseInt(rawID, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid template id")
	}

	return id, nil
}

func parsePagination(r *http.Request) (limit, offset int, err error) {
	q := r.URL.Query()

	if raw := q.Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid limit")
		}
	}
	if raw := q.Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil {
			return 0, 0, errors.New("invalid offset")
		}
	}

	return limit, offset, nil
}
