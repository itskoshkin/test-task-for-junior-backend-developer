# TODO

## Decisions Made

- **Template + instances.** A separate `task_templates` entity describes the rule. Concrete tasks for a date (`tasks`) are instances of the template. A one-off task = `tasks` with `template_id = NULL`.
- **Custom discriminated rule type**, not RRULE. In the DB: `rule_type TEXT` + `rule_params JSONB`.
- **Lazy materialization.** A row in `tasks` is created only on the first mutation of a specific day (status changed → materialized). Before that, occurrences are virtual and computed on the fly.
- **Timezone:** rules are treated as a “day” (`DATE` type without time). The server runs in UTC.
- **Read window is mandatory:** `GET /tasks` requires `from`/`to`, maximum 366 days per request.

### Stage 1. Domain

- [x] Value object `RecurrenceRule` (interface or sealed-like discriminated type):
  - `Daily{EveryN int}` — every Nth day (N ≥ 1)
  - `Monthly{DaysOfMonth []int}` — days from 1 to 30 (31 is not allowed; for the end of month use `LastDayOfMonth`)
  - `SpecificDates{Dates []Date}` — explicit list
  - `WeekdayParity{Parity enum{even,odd}}` — based on the parity of the day number in the month
  - `LastDayOfMonth{}` — triggers on the last day of each month (28/29/30/31 depending on the month and leap year)
- [x] `Rule.Validate() error`
- [x] `Rule.Occurrences(from, to Date) []Date` — all rule dates within the window (inclusive)
- [x] Entity `TaskTemplate{ID, Title, Description, Rule, StartDate, EndDate *Date, CreatedAt, UpdatedAt}`
- [x] Extend `Task`: `TemplateID *int64`, `DueDate Date` (required)
- [x] Generator tests (table-driven):
  - [x] `Monthly` with day 29 or 30 in a February that doesn't have that day — skip, do not shift
  - [x] `LastDayOfMonth`: February non-leap → 28, February leap → 29, April → 30, January → 31
  - [x] Leap year (29.02 is reachable via `SpecificDates` / `Monthly{29}`)
  - [x] `EveryN=1`, `EveryN=7`
  - [x] Even/odd parity, including the 31st (31 falls under odd parity, not under even)
  - [x] Empty `SpecificDates` list — validation error
  - [x] `EndDate < StartDate` — validation error
  - [x] `EndDate = nil` — infinite rule, returns the entire window range

### Stage 2. Persistence

- [x] Migration `0002_add_recurrence.up.sql`:
  - Table `task_templates (id BIGSERIAL, title, description, rule_type TEXT, rule_params JSONB, start_date DATE, end_date DATE NULL, created_at, updated_at)`
  - In `tasks`: add `template_id BIGINT REFERENCES task_templates(id) ON DELETE CASCADE NULL`, `due_date DATE NULL`
  - Indexes: `idx_tasks_due_date`, `idx_tasks_template_id`, `idx_templates_active (start_date, end_date)`
  - Unique key `(template_id, due_date) WHERE template_id IS NOT NULL` — so an instance cannot be materialized twice
- [x] Down migration `0002_add_recurrence.down.sql`
- [x] Repository `TaskTemplateRepository`: Create / GetByID / Update / Delete / ListActiveInRange(from; to)
- [x] Update `TaskRepository`:
  - `ListInRange(from, to Date) []Task`
  - `UpsertInstance(templateID, date, status)` — idempotent materialization

### Stage 3. Use case

- [x] `TemplateService.Create/GetByID/Update/Delete` with rule validation
- [x] `TaskService.ListInRange(from, to)`:
  1. Query materialized instances in the window
  2. Generate virtual occurrences for each active template
  3. Merge: a materialized instance always has priority over a virtual one on the same date
- [x] `TaskService.UpdateOccurrenceStatus(templateID, date, status)` — materializes if not yet materialized
- [x] Request window validation (max 366 days, `from <= to`)

### Stage 4. Transport

- [x] DTO `RecurrenceRuleDTO` with discriminator `type`, custom `UnmarshalJSON`
- [x] Resource `/api/v1/task-templates`:
  - `POST` — create template
  - `GET /{id}` / `PUT /{id}` / `DELETE /{id}`
- [x] `/api/v1/tasks`:
  - `POST` — one-off task (`due_date` required, `template_id = NULL`)
  - `GET /?from=YYYY-MM-DD&to=YYYY-MM-DD` — window instances (materialized + virtual)
  - `PATCH /status` — for both materialized tasks (`id`) and virtual occurrences (`template_id` + `due_date`)
- [x] Update `internal/transport/http/docs/openapi.json`

### Stage 5. Tests

- [ ] Date generator — table-driven, each type × edge cases
- [ ] DTO and `RecurrenceRule` validation (negative N, empty arrays, day > 30, end < start)
- [ ] Use case: verify merge of virtual + materialized instances on a mock repository
- [ ] Handler: happy path + 400/404 for each endpoint
- [ ] One e2e test via docker-compose: create template → read window → change occurrence status → read again

### Stage 6. Documentation

- [ ] README.md — section “Assumptions and Decisions”:
  - Why template+instances, why not RRULE
  - Timezone, interpretation of a day
  - `Monthly` days 29/30 in months that don't have them — skipped, not shifted; for the end of month use `LastDayOfMonth`
  - Parity is based on the numeric day of the month (31 is odd)
  - `end_date` is inclusive and may be `null` (infinite rule)
  - Deleting a template cascades to delete materialized instances
- [ ] README.md — curl examples for the new flow
- [x] Swagger up to date

## Open Questions

- [ ] `assignee_id` for a template — current `Task` does not have it. Should it be added as part of this assignment? Probably not: the task is about recurrence, not assignment.
- [ ] List pagination — keep the `from/to` window without `limit/offset`. If the window is too large, cut it off server-side (`maxspan` 366 days).
- [ ] “Edit the whole series / only this occurrence” — in v1, do the basic version: editing the template affects future virtual occurrences; editing a materialized instance affects only that one.
