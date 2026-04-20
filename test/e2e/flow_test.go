//go:build e2e

// Package e2e exercises the whole stack (HTTP → service → Postgres) against a running
// docker-compose deployment. Invoke with `go test -tags=e2e ./test/e2e/...` after
// `docker compose up -d` and the migrations applied.
package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

const (
	defaultBaseURL = "http://localhost:8080"
	windowDays     = 4 // inclusive range of size windowDays+1
)

type taskDTO struct {
	ID         int64  `json:"id"`
	TemplateID *int64 `json:"template_id,omitempty"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	DueDate    string `json:"due_date"`
	Virtual    bool   `json:"virtual"`
}

type templateDTO struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
}

func TestE2E_RecurringFlow(t *testing.T) {
	base := os.Getenv("API_URL")
	if base == "" {
		base = defaultBaseURL
	}
	client := &apiClient{base: base, t: t}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	start := today.Format("2006-01-02")
	end := today.AddDate(0, 0, windowDays).Format("2006-01-02")

	// 1. Create a daily template — it generates `windowDays+1` virtual occurrences in the window.
	var tpl templateDTO
	client.do(http.MethodPost, "/api/v1/task-templates", map[string]any{
		"title": "E2E daily",
		"rule": map[string]any{
			"type":   "daily",
			"params": map[string]int{"every_n": 1},
		},
		"start_date": start,
	}, http.StatusCreated, &tpl)

	t.Cleanup(func() {
		client.do(http.MethodDelete, fmt.Sprintf("/api/v1/task-templates/%d", tpl.ID), nil, http.StatusNoContent, nil)
	})
	if tpl.ID == 0 {
		t.Fatal("template ID must be non-zero")
	}

	// 2. Read window — everything for our template is virtual and "new".
	var first []taskDTO
	client.do(http.MethodGet, fmt.Sprintf("/api/v1/tasks?from=%s&to=%s&limit=100", start, end), nil, http.StatusOK, &first)

	occurs := filterByTemplate(first, tpl.ID)
	if len(occurs) != windowDays+1 {
		t.Fatalf("want %d occurrences in window, got %d", windowDays+1, len(occurs))
	}
	for _, o := range occurs {
		if !o.Virtual || o.ID != 0 || o.Status != "new" {
			t.Errorf("expected virtual/new, got %+v", o)
		}
	}

	// 3. Flip the 2nd occurrence to "done" — lazy materialization should kick in.
	target := occurs[1].DueDate
	var patched taskDTO
	client.do(http.MethodPatch, "/api/v1/tasks/status", map[string]any{
		"template_id": tpl.ID,
		"due_date":    target,
		"status":      "done",
	}, http.StatusOK, &patched)

	if patched.ID == 0 || patched.Virtual {
		t.Fatalf("patched occurrence should be materialized: %+v", patched)
	}
	if patched.Status != "done" || patched.DueDate != target {
		t.Errorf("patched task mismatch: %+v", patched)
	}

	// 4. Read window again — target date is materialized/"done", others still virtual.
	var second []taskDTO
	client.do(http.MethodGet, fmt.Sprintf("/api/v1/tasks?from=%s&to=%s&limit=100", start, end), nil, http.StatusOK, &second)

	occurs2 := filterByTemplate(second, tpl.ID)
	if len(occurs2) != windowDays+1 {
		t.Fatalf("want %d occurrences after patch, got %d", windowDays+1, len(occurs2))
	}

	var sawMaterialized bool
	for _, o := range occurs2 {
		if o.DueDate == target {
			if o.Virtual || o.ID == 0 || o.Status != "done" {
				t.Errorf("target date should be materialized/done: %+v", o)
			}
			sawMaterialized = true
			continue
		}
		if !o.Virtual || o.Status != "new" {
			t.Errorf("non-target date should stay virtual/new: %+v", o)
		}
	}
	if !sawMaterialized {
		t.Fatal("materialized occurrence for target date not found in second list")
	}
}

func filterByTemplate(tasks []taskDTO, templateID int64) []taskDTO {
	var out []taskDTO
	for _, t := range tasks {
		if t.TemplateID != nil && *t.TemplateID == templateID {
			out = append(out, t)
		}
	}
	return out
}

type apiClient struct {
	base string
	t    *testing.T
}

func (c *apiClient) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()

	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			c.t.Fatalf("marshal %s %s: %v", method, path, err)
		}
		reader = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, c.base+path, reader)
	if err != nil {
		c.t.Fatalf("build request %s %s: %v", method, path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatalf("do %s %s: %v", method, path, err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: status %d, want %d; body=%s", method, path, resp.StatusCode, wantStatus, raw)
	}
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("decode %s %s response: %v; body=%s", method, path, err, raw)
		}
	}
}
