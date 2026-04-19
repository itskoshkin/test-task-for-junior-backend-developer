DROP INDEX IF EXISTS uniq_tasks_template_date;
DROP INDEX IF EXISTS idx_tasks_template_id;
DROP INDEX IF EXISTS idx_tasks_due_date;

ALTER TABLE tasks
	DROP COLUMN IF EXISTS template_id,
	DROP COLUMN IF EXISTS due_date;

DROP INDEX IF EXISTS idx_task_templates_active;
DROP TABLE IF EXISTS task_templates;
