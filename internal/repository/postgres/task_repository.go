package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

const taskColumns = `id, template_id, title, description, status, due_date, created_at, updated_at`

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `INSERT INTO tasks (template_id, title, description, status, due_date, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					RETURNING ` + taskColumns

	row := r.pool.QueryRow(ctx, query,
		task.TemplateID,
		task.Title,
		task.Description,
		task.Status,
		task.DueDate,
		task.CreatedAt,
		task.UpdatedAt,
	)
	created, err := scanTask(row)
	if err != nil {
		return nil, err
	}

	return created, nil
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM tasks WHERE id = $1`

	row := r.pool.QueryRow(ctx, query, id)
	found, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `UPDATE tasks
					SET title = $1,
						description = $2,
						status = $3,
						due_date = $4,
						updated_at = $5
					WHERE id = $6
					RETURNING ` + taskColumns

	row := r.pool.QueryRow(ctx, query, task.Title, task.Description, task.Status, task.DueDate, task.UpdatedAt, task.ID)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) UpdateStatus(ctx context.Context, id int64, status taskdomain.Status, updatedAt time.Time) (*taskdomain.Task, error) {
	const query = `UPDATE tasks
					SET status = $1, updated_at = $2
					WHERE id = $3
					RETURNING ` + taskColumns

	row := r.pool.QueryRow(ctx, query, status, updatedAt, id)
	updated, err := scanTask(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}

		return nil, err
	}

	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	const query = `DELETE FROM tasks WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}

	return nil
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]taskdomain.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM tasks ORDER BY id DESC LIMIT $1 OFFSET $2`

	return r.queryTasks(ctx, query, limit, offset)
}

func (r *Repository) ListInRange(ctx context.Context, from, to taskdomain.Date) ([]taskdomain.Task, error) {
	const query = `SELECT ` + taskColumns + ` FROM tasks WHERE due_date IS NOT NULL AND due_date BETWEEN $1 AND $2 ORDER BY due_date ASC, id ASC`

	return r.queryTasks(ctx, query, from, to)
}

// UpsertInstance materializes a virtual occurrence idempotently.
// The ON CONFLICT target matches the partial unique index on (template_id, due_date) — partial because standalone tasks (template_id IS NULL) must not participate in deduplication
func (r *Repository) UpsertInstance(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `INSERT INTO tasks (template_id, title, description, status, due_date, created_at, updated_at)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					ON CONFLICT (template_id, due_date) WHERE template_id IS NOT NULL
					DO UPDATE SET
						status = EXCLUDED.status,
						title = EXCLUDED.title,
						description = EXCLUDED.description,
						updated_at = EXCLUDED.updated_at
					RETURNING ` + taskColumns

	row := r.pool.QueryRow(ctx, query,
		task.TemplateID,
		task.Title,
		task.Description,
		task.Status,
		task.DueDate,
		task.CreatedAt,
		task.UpdatedAt,
	)

	return scanTask(row)
}

func (r *Repository) queryTasks(ctx context.Context, query string, args ...any) ([]taskdomain.Task, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		var task *taskdomain.Task
		task, err = scanTask(rows)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, *task)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tasks, nil
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task   taskdomain.Task
		status string
	)

	if err := scanner.Scan(
		&task.ID,
		&task.TemplateID,
		&task.Title,
		&task.Description,
		&status,
		&task.DueDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}

	task.Status = taskdomain.Status(status)

	return &task, nil
}
