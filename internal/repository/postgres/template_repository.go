package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type TemplateRepository struct {
	pool *pgxpool.Pool
}

func NewTemplateRepository(pool *pgxpool.Pool) *TemplateRepository {
	return &TemplateRepository{pool: pool}
}

const templateColumns = `id, title, description, rule_type, rule_params, start_date, end_date, created_at, updated_at`

func (r *TemplateRepository) Create(ctx context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error) {
	ruleType, ruleParams, err := taskdomain.EncodeRule(tpl.Rule)
	if err != nil {
		return nil, err
	}

	const query = `INSERT INTO task_templates (title, description, rule_type, rule_params, start_date, end_date, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
					RETURNING ` + templateColumns

	row := r.pool.QueryRow(ctx, query,
		tpl.Title,
		tpl.Description,
		ruleType,
		ruleParams,
		tpl.StartDate,
		tpl.EndDate,
		tpl.CreatedAt,
		tpl.UpdatedAt,
	)

	return scanTemplate(row)
}

func (r *TemplateRepository) GetByID(ctx context.Context, id int64) (*taskdomain.Template, error) {
	const query = `SELECT ` + templateColumns + ` FROM task_templates WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *TemplateRepository) Update(ctx context.Context, tpl *taskdomain.Template) (*taskdomain.Template, error) {
	ruleType, ruleParams, err := taskdomain.EncodeRule(tpl.Rule)
	if err != nil {
		return nil, err
	}

	const query = `UPDATE task_templates
					SET title = $1,
						description = $2,
						rule_type = $3,
						rule_params = $4,
						start_date = $5,
						end_date = $6,
						updated_at = $7
					WHERE id = $8
					RETURNING ` + templateColumns

	row := r.pool.QueryRow(ctx, query,
		tpl.Title,
		tpl.Description,
		ruleType,
		ruleParams,
		tpl.StartDate,
		tpl.EndDate,
		tpl.UpdatedAt,
		tpl.ID,
	)

	updated, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *TemplateRepository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM task_templates WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *TemplateRepository) List(ctx context.Context, limit, offset int) ([]taskdomain.Template, error) {
	const query = `SELECT ` + templateColumns + `FROM task_templates ORDER BY id DESC LIMIT $1 OFFSET $2`

	return r.queryTemplates(ctx, query, limit, offset)
}

func (r *TemplateRepository) ListActiveInRange(ctx context.Context, from, to taskdomain.Date) ([]taskdomain.Template, error) {
	const query = `SELECT ` + templateColumns + ` FROM task_templates WHERE start_date <= $2 AND (end_date IS NULL OR end_date >= $1) ORDER BY id ASC`

	return r.queryTemplates(ctx, query, from, to)
}

func (r *TemplateRepository) queryTemplates(ctx context.Context, query string, args ...any) ([]taskdomain.Template, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	templates := make([]taskdomain.Template, 0)
	for rows.Next() {
		var tpl *taskdomain.Template
		tpl, err = scanTemplate(rows)
		if err != nil {
			return nil, err
		}

		templates = append(templates, *tpl)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return templates, nil
}

func scanTemplate(scanner taskScanner) (*taskdomain.Template, error) {
	var (
		tpl        taskdomain.Template
		ruleType   string
		ruleParams []byte
	)

	if err := scanner.Scan(
		&tpl.ID,
		&tpl.Title,
		&tpl.Description,
		&ruleType,
		&ruleParams,
		&tpl.StartDate,
		&tpl.EndDate,
		&tpl.CreatedAt,
		&tpl.UpdatedAt,
	); err != nil {
		return nil, err
	}

	rule, err := taskdomain.DecodeRule(taskdomain.RecurrenceType(ruleType), json.RawMessage(ruleParams))
	if err != nil {
		return nil, err
	}
	tpl.Rule = rule

	return &tpl, nil
}
