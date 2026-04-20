package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type ruleDTO struct {
	Type   taskdomain.RecurrenceType `json:"type"`
	Params json.RawMessage           `json:"params,omitempty"`
}

func (d ruleDTO) toDomain() (taskdomain.RecurrenceRule, error) {
	params := d.Params
	if len(params) == 0 {
		params = json.RawMessage(`{}`)
	}

	return taskdomain.DecodeRule(d.Type, params)
}

func ruleDTOFromDomain(rule taskdomain.RecurrenceRule) (ruleDTO, error) {
	rtype, params, err := taskdomain.EncodeRule(rule)
	if err != nil {
		return ruleDTO{}, fmt.Errorf("rule dto: %w", err)
	}

	return ruleDTO{Type: rtype, Params: params}, nil
}

type templateMutationDTO struct {
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Rule        ruleDTO          `json:"rule"`
	StartDate   taskdomain.Date  `json:"start_date"`
	EndDate     *taskdomain.Date `json:"end_date,omitempty"`
}

type templateDTO struct {
	ID          int64            `json:"id"`
	Title       string           `json:"title"`
	Description string           `json:"description"`
	Rule        ruleDTO          `json:"rule"`
	StartDate   taskdomain.Date  `json:"start_date"`
	EndDate     *taskdomain.Date `json:"end_date,omitempty"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
}

func newTemplateDTO(tpl *taskdomain.Template) (templateDTO, error) {
	rule, err := ruleDTOFromDomain(tpl.Rule)
	if err != nil {
		return templateDTO{}, err
	}

	return templateDTO{
		ID:          tpl.ID,
		Title:       tpl.Title,
		Description: tpl.Description,
		Rule:        rule,
		StartDate:   tpl.StartDate,
		EndDate:     tpl.EndDate,
		CreatedAt:   tpl.CreatedAt,
		UpdatedAt:   tpl.UpdatedAt,
	}, nil
}
