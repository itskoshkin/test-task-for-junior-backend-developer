CREATE TABLE IF NOT EXISTS task_templates (
	id           BIGSERIAL PRIMARY KEY,
	title        TEXT NOT NULL,
	description  TEXT NOT NULL DEFAULT '',
	rule_type    TEXT NOT NULL CHECK (rule_type IN (
		'daily', 'monthly', 'specific_dates', 'monthday_parity', 'last_day_of_month'
	)),
	rule_params  JSONB NOT NULL,
	start_date   DATE NOT NULL,
	end_date     DATE,
	created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT   task_templates_end_after_start
		CHECK (end_date IS NULL OR end_date >= start_date)
);

CREATE INDEX IF NOT EXISTS idx_task_templates_active
	ON task_templates (start_date, end_date);

ALTER TABLE tasks
	ADD COLUMN IF NOT EXISTS template_id BIGINT
		REFERENCES task_templates(id) ON DELETE CASCADE,
	ADD COLUMN IF NOT EXISTS due_date DATE;

CREATE INDEX IF NOT EXISTS idx_tasks_due_date ON tasks (due_date);
CREATE INDEX IF NOT EXISTS idx_tasks_template_id ON tasks (template_id);
CREATE UNIQUE INDEX IF NOT EXISTS uniq_tasks_template_date
	ON tasks (template_id, due_date) WHERE template_id IS NOT NULL;
