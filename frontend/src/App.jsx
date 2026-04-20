import {
  startTransition,
  useDeferredValue,
  useEffect,
  useRef,
  useState,
} from "react";
import {
  createTask,
  createTemplate,
  deleteTask,
  deleteTemplate,
  getTaskById,
  getTemplateById,
  listTasks,
  listTasksInRange,
  listTemplates,
  updateTask,
  updateTaskStatus,
  updateTemplate,
} from "./api.js";

const STATUS_OPTIONS = [
  { value: "new", label: "Новая" },
  { value: "in_progress", label: "В работе" },
  { value: "done", label: "Выполнена" },
];

const RULE_OPTIONS = [
  { value: "daily", label: "Каждый n-й день" },
  { value: "monthly", label: "Определенные числа месяца" },
  { value: "specific_dates", label: "Конкретные даты" },
  { value: "monthday_parity", label: "Четные / нечетные дни" },
  { value: "last_day_of_month", label: "Последний день месяца" },
];

const NAV_ITEMS = [
  { label: "Пациенты", active: false, icon: "square" },
  { label: "Расписание", active: false, icon: "square" },
  { label: "Задачи", active: true, icon: "tasks" },
  { label: "Отчеты", active: false, icon: "square" },
];

const WORKSPACE_TABS = [
  { key: "all", label: "Все" },
  { key: "today", label: "Сегодня" },
  { key: "week", label: "Неделя" },
  { key: "window", label: "По диапазону" },
  { key: "templates", label: "Повторы" },
  { key: "undated", label: "Без даты" },
];

const TASK_PAGE_LIMIT = 12;
const TEMPLATE_PAGE_LIMIT = 8;
const TOAST_TIMEOUT_MS = 4200;
const TASK_ACCUMULATION_LIMIT = 200;

const DATE_ONLY_FORMATTER = new Intl.DateTimeFormat("ru-RU", {
  dateStyle: "medium",
  timeZone: "UTC",
});

const DATE_TIME_FORMATTER = new Intl.DateTimeFormat("ru-RU", {
  dateStyle: "medium",
  timeStyle: "short",
});

function toInputDate(date) {
  const year = String(date.getFullYear());
  const month = String(date.getMonth() + 1).padStart(2, "0");
  const day = String(date.getDate()).padStart(2, "0");
  return `${year}-${month}-${day}`;
}

function addDays(dateString, amount) {
  const [year, month, day] = dateString.split("-").map(Number);
  const date = new Date(year, month - 1, day);
  date.setDate(date.getDate() + amount);
  return toInputDate(date);
}

function getDefaultWindowRange(from = toInputDate(new Date())) {
  return {
    from,
    to: addDays(from, 21),
  };
}

function getWeekRange(from = toInputDate(new Date())) {
  return {
    from,
    to: addDays(from, 6),
  };
}

function createEmptyTaskDraft() {
  return {
    title: "",
    description: "",
    status: "new",
    dueDate: "",
  };
}

function createEmptyTemplateDraft() {
  return {
    title: "",
    description: "",
    ruleType: "daily",
    dailyEveryN: "1",
    monthlyDays: "",
    specificDates: "",
    parity: "odd",
    startDate: toInputDate(new Date()),
    hasEndDate: false,
    endDate: "",
  };
}

function createPageState(limit) {
  return {
    limit,
    offset: 0,
  };
}

function normalizePage(page, fallbackLimit) {
  return {
    limit: page?.limit ?? fallbackLimit,
    offset: page?.offset ?? 0,
  };
}

function statusLabel(status) {
  return STATUS_OPTIONS.find((option) => option.value === status)?.label || status;
}

function ruleTypeLabel(type) {
  return RULE_OPTIONS.find((option) => option.value === type)?.label || type;
}

function paneLabel(key) {
  return WORKSPACE_TABS.find((tab) => tab.key === key)?.label || key;
}

function isTaskCollectionPane(pane) {
  return pane === "all" || pane === "today" || pane === "undated";
}

function taskPaneEmptyLabel(pane) {
  if (pane === "today") {
    return "На сегодня задач нет.";
  }
  if (pane === "undated") {
    return "Задач без даты пока нет.";
  }
  return "Задач пока нет.";
}

function pageSliceLabel(page, itemsCount) {
  if (itemsCount === 0) {
    return "Нет записей";
  }

  const from = page.offset + 1;
  const to = page.offset + itemsCount;
  return `${from}-${to}`;
}

function sliceItems(items, page) {
  return items.slice(page.offset, page.offset + page.limit);
}

function formatDateOnly(value) {
  if (!value) {
    return "—";
  }

  if (/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    const [year, month, day] = value.split("-").map(Number);
    const date = new Date(Date.UTC(year, month - 1, day, 12));
    return DATE_ONLY_FORMATTER.format(date);
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return DATE_ONLY_FORMATTER.format(date);
}

function formatCompactDate(value) {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) {
    return "Выбрать дату";
  }

  const [year, month, day] = value.split("-");
  return `${day}.${month}.${year}`;
}

function formatDateTime(value) {
  if (!value) {
    return "—";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return value;
  }

  return DATE_TIME_FORMATTER.format(date);
}

function isBeforeDate(left, right) {
  return Boolean(left && right && left < right);
}

function taskKey(task) {
  return task.id > 0
    ? `task:${task.id}`
    : `template:${task.template_id || "unknown"}:${task.due_date || "unknown"}`;
}

function toTaskDraft(task) {
  return {
    title: task?.title || "",
    description: task?.description || "",
    status: task?.status || "new",
    dueDate: task?.due_date || "",
  };
}

function parseIntegerList(raw, min, max, label) {
  const values = raw
    .split(/[\s,;]+/)
    .map((item) => item.trim())
    .filter(Boolean)
    .map((item) => Number.parseInt(item, 10));

  if (values.length === 0) {
    throw new Error(`${label}: укажите хотя бы одно значение.`);
  }

  if (values.some((value) => !Number.isInteger(value))) {
    throw new Error(`${label}: допустимы только целые числа.`);
  }

  const unique = [...new Set(values)].sort((left, right) => left - right);
  if (unique.some((value) => value < min || value > max)) {
    throw new Error(`${label}: значения должны быть в диапазоне ${min}-${max}.`);
  }

  return unique;
}

function parseDateList(raw) {
  const values = raw
    .split(/[\s,;]+/)
    .map((item) => item.trim())
    .filter(Boolean);

  if (values.length === 0) {
    throw new Error("Для specific_dates укажите хотя бы одну дату.");
  }

  const unique = [...new Set(values)].sort();

  if (unique.some((value) => !/^\d{4}-\d{2}-\d{2}$/.test(value))) {
    throw new Error("Даты должны быть в формате YYYY-MM-DD.");
  }

  return unique;
}

function ruleSummary(rule) {
  if (!rule) {
    return "Правило не задано";
  }

  switch (rule.type) {
    case "daily":
      return `Каждый ${rule.params?.every_n || 1}-й день`;
    case "monthly":
      return `Числа месяца: ${rule.params?.days_of_month?.join(", ") || "—"}`;
    case "specific_dates":
      return `Даты: ${rule.params?.dates?.join(", ") || "—"}`;
    case "monthday_parity":
      return rule.params?.parity === "even"
        ? "Только четные дни месяца"
        : "Только нечетные дни месяца";
    case "last_day_of_month":
      return "Последний день месяца";
    default:
      return rule.type;
  }
}

function templateWindowLabel(template) {
  return template.end_date
    ? `${formatDateOnly(template.start_date)} - ${formatDateOnly(template.end_date)}`
    : `С ${formatDateOnly(template.start_date)} без даты окончания`;
}

function ruleDraftToPayload(draft) {
  switch (draft.ruleType) {
    case "daily": {
      const everyN = Number.parseInt(draft.dailyEveryN, 10);
      if (!Number.isInteger(everyN) || everyN < 1) {
        throw new Error("Для daily укажите every_n больше либо равный 1.");
      }

      return {
        type: "daily",
        params: { every_n: everyN },
      };
    }
    case "monthly":
      return {
        type: "monthly",
        params: {
          days_of_month: parseIntegerList(
            draft.monthlyDays,
            1,
            30,
            "Для monthly",
          ),
        },
      };
    case "specific_dates":
      return {
        type: "specific_dates",
        params: {
          dates: parseDateList(draft.specificDates),
        },
      };
    case "monthday_parity":
      if (!["even", "odd"].includes(draft.parity)) {
        throw new Error("Для monthday_parity выберите even или odd.");
      }

      return {
        type: "monthday_parity",
        params: { parity: draft.parity },
      };
    case "last_day_of_month":
      return {
        type: "last_day_of_month",
        params: {},
      };
    default:
      throw new Error("Неизвестный тип правила.");
  }
}

function buildTemplatePayload(draft) {
  const title = draft.title.trim();
  if (!title) {
    throw new Error("Название шаблона обязательно.");
  }

  if (!draft.startDate) {
    throw new Error("Укажите start_date шаблона.");
  }

  if (draft.hasEndDate && !draft.endDate) {
    throw new Error("Если включен end_date, укажите дату окончания.");
  }

  return {
    title,
    description: draft.description.trim(),
    rule: ruleDraftToPayload(draft),
    start_date: draft.startDate,
    end_date: draft.hasEndDate ? draft.endDate : null,
  };
}

function templateToDraft(template) {
  const next = createEmptyTemplateDraft();
  next.title = template.title || "";
  next.description = template.description || "";
  next.startDate = template.start_date || next.startDate;
  next.hasEndDate = Boolean(template.end_date);
  next.endDate = template.end_date || "";
  next.ruleType = template.rule?.type || "daily";

  switch (template.rule?.type) {
    case "daily":
      next.dailyEveryN = String(template.rule.params?.every_n || 1);
      break;
    case "monthly":
      next.monthlyDays = (template.rule.params?.days_of_month || []).join(", ");
      break;
    case "specific_dates":
      next.specificDates = (template.rule.params?.dates || []).join("\n");
      break;
    case "monthday_parity":
      next.parity = template.rule.params?.parity || "odd";
      break;
    default:
      break;
  }

  return next;
}

function matchesTaskSearch(task, query) {
  if (!query) {
    return true;
  }

  const haystack = [task.title, task.description].join(" ").toLowerCase();

  return haystack.includes(query);
}

function matchesTemplateSearch(template, query) {
  if (!query) {
    return true;
  }

  const haystack = [
    template.id,
    template.title,
    template.description,
    template.rule?.type,
    ruleTypeLabel(template.rule?.type),
    formatDateOnly(template.start_date),
    formatDateOnly(template.end_date),
    templateWindowLabel(template),
  ]
    .join(" ")
    .toLowerCase();

  return haystack.includes(query);
}

function rulePreviewText(draft) {
  try {
    return ruleSummary(ruleDraftToPayload(draft));
  } catch (error) {
    return error.message;
  }
}

function StatusBadge({ status }) {
  return <span className={`status-badge ${status}`}>{statusLabel(status)}</span>;
}

function SidebarGlyph({ type }) {
  if (type === "tasks") {
    return (
      <svg className="sidebar-svg" viewBox="0 0 16 16" aria-hidden="true">
        <rect x="2" y="2" width="12" height="12" rx="2.5" fill="none" stroke="currentColor" strokeWidth="1.7" />
        <path d="M5 5.5H11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <path d="M5 8H11" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
        <path d="M5 10.5H9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" />
      </svg>
    );
  }

  return (
    <svg className="sidebar-svg" viewBox="0 0 16 16" aria-hidden="true">
      <rect x="2" y="2" width="12" height="12" rx="3" fill="none" stroke="currentColor" strokeWidth="1.7" />
    </svg>
  );
}

function SidebarNav({ referenceDate, onOpenReferenceDatePicker }) {
  return (
    <aside className="app-sidebar">
      <div className="sidebar-main">
        <div className="sidebar-brand">МИС</div>
        <nav className="sidebar-nav" aria-label="Разделы">
          {NAV_ITEMS.map((item) => (
            <button
              key={item.label}
              type="button"
              className={`sidebar-item${item.active ? " active" : ""}`}
              disabled={!item.active}
              aria-current={item.active ? "page" : undefined}
            >
              <span className="sidebar-icon" aria-hidden="true">
                <SidebarGlyph type={item.icon} />
              </span>
              <span>{item.label}</span>
            </button>
          ))}
        </nav>
      </div>

      <label className="sidebar-date-picker">
        <span className="sidebar-date-label">Симуляция сегодняшней даты</span>
        <button
          className="sidebar-date-button"
          type="button"
          onClick={onOpenReferenceDatePicker}
          aria-label="Выбрать дату симуляции"
        >
          {formatCompactDate(referenceDate)}
        </button>
      </label>
    </aside>
  );
}

function WorkspaceTabs({ activePane, onChange }) {
  return (
    <div className="workspace-tabs" role="tablist" aria-label="Режим просмотра">
      {WORKSPACE_TABS.map((tab) => (
        <button
          key={tab.key}
          type="button"
          role="tab"
          aria-selected={activePane === tab.key}
          className={`workspace-tab${activePane === tab.key ? " active" : ""}`}
          onClick={() => onChange(tab.key)}
        >
          {tab.label}
        </button>
      ))}
    </div>
  );
}

function PaginationControls({ page, count, loading, onPrev, onNext }) {
  return (
    <div className="pagination-row">
      <button
        className="ghost-button"
        type="button"
        onClick={onPrev}
        disabled={page.offset === 0 || loading}
      >
        Назад
      </button>
      <span className="page-label">Показаны {pageSliceLabel(page, count)}</span>
      <button
        className="ghost-button"
        type="button"
        onClick={onNext}
        disabled={loading || count < page.limit}
      >
        Вперед
      </button>
    </div>
  );
}

function TaskRow({ task, isActive, onOpen }) {
  const isOpenable = task.id > 0;

  return (
    <button
      type="button"
      className={`record-row task-row${isActive ? " active" : ""}${isOpenable ? "" : " disabled-row"}`}
      onClick={isOpenable ? () => onOpen(task.id) : undefined}
      aria-disabled={!isOpenable}
    >
      <div className="record-main">
        <div className="record-overline">{task.id > 0 ? `Задача #${task.id}` : "По расписанию"}</div>
        <h3>{task.title}</h3>
        <p>{task.description || "Без описания"}</p>
      </div>
      <div className="record-aside task-row-aside">
        <StatusBadge status={task.status} />
        <div className="task-row-meta">
          <span className={`record-chip${task.overdue ? " danger" : ""}`}>
            {task.overdue
              ? `${formatDateOnly(task.due_date)} (просрочено)`
              : formatDateOnly(task.due_date)}
          </span>
          <span className="record-chip muted">
            {task.template_id ? `Из повтора #${task.template_id}` : "Разовая"}
          </span>
        </div>
      </div>
    </button>
  );
}

function TemplateRow({ template, isActive, onOpen }) {
  return (
    <button
      type="button"
      className={`record-row template-row${isActive ? " active" : ""}`}
      onClick={() => onOpen(template.id)}
    >
      <div className="record-main">
        <div className="record-overline">Повтор #{template.id}</div>
        <h3>{template.title}</h3>
        <p>{template.description || "Без описания"}</p>
      </div>
      <div className="record-aside template-row-aside">
        <span className="record-chip accent">{ruleTypeLabel(template.rule?.type)}</span>
        <span className="record-chip muted">{templateWindowLabel(template)}</span>
      </div>
    </button>
  );
}

function OccurrenceRow({ task, isUpdating, onStatusChange, onOpen }) {
  const isOpenable = !task.virtual && task.id > 0;
  const occurrenceMetaLabel = task.template_id ? `Из повтора #${task.template_id}` : "Разовая";

  function handleOpen() {
    if (isOpenable) {
      onOpen(task.id);
    }
  }

  return (
    <article
      className={`record-row occurrence-row${task.virtual ? " virtual" : ""}${
        isOpenable ? " clickable" : ""
      }`}
      onClick={isOpenable ? handleOpen : undefined}
      onKeyDown={
        isOpenable
          ? (event) => {
              if (event.key === "Enter" || event.key === " ") {
                event.preventDefault();
                handleOpen();
              }
            }
          : undefined
      }
      role={isOpenable ? "button" : undefined}
      tabIndex={isOpenable ? 0 : undefined}
    >
      <div className="record-main">
        <div className="record-overline">{formatDateOnly(task.due_date)}</div>
        <h3>{task.title}</h3>
        <p>{task.description || "Без описания"}</p>
      </div>

      <div className="record-aside occurrence-row-aside">
        <select
          className={`status-select ${task.status}`}
          value={task.status}
          onClick={(event) => event.stopPropagation()}
          onKeyDown={(event) => event.stopPropagation()}
          onChange={(event) => onStatusChange(task, event.target.value)}
          disabled={isUpdating}
          aria-label={`Статус задачи ${task.title}`}
        >
          {STATUS_OPTIONS.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
        <span className="record-chip muted occurrence-row-chip">{occurrenceMetaLabel}</span>
      </div>
    </article>
  );
}

function RuleFields({ draft, onChange }) {
  switch (draft.ruleType) {
    case "daily":
      return (
        <label className="field">
          <span>Каждый n-й день</span>
          <input
            type="number"
            min="1"
            value={draft.dailyEveryN}
            onChange={(event) => onChange({ dailyEveryN: event.target.value })}
          />
        </label>
      );
    case "monthly":
      return (
        <label className="field">
          <span>Числа месяца</span>
          <input
            type="text"
            value={draft.monthlyDays}
            onChange={(event) => onChange({ monthlyDays: event.target.value })}
            placeholder="Например: 1, 15, 30"
          />
        </label>
      );
    case "specific_dates":
      return (
        <label className="field field-span">
          <span>Даты</span>
          <textarea
            value={draft.specificDates}
            onChange={(event) => onChange({ specificDates: event.target.value })}
            placeholder={"2026-04-21\n2026-04-30"}
          />
        </label>
      );
    case "monthday_parity":
      return (
        <label className="field">
          <span>Четность дня</span>
          <select
            value={draft.parity}
            onChange={(event) => onChange({ parity: event.target.value })}
          >
            <option value="odd">Нечетные</option>
            <option value="even">Четные</option>
          </select>
        </label>
      );
    case "last_day_of_month":
      return <div className="helper-box field-span">Срабатывает в последний день каждого месяца.</div>;
    default:
      return null;
  }
}

function ModalShell({ title, subtitle, children, footer, onClose }) {
  return (
    <div className="modal-overlay" role="presentation" onClick={onClose}>
      <div
        className="modal-window"
        role="dialog"
        aria-modal="true"
        aria-label={title}
        onClick={(event) => event.stopPropagation()}
      >
        <div className="modal-header">
          <div>
            <div className="modal-title">{title}</div>
            {subtitle ? <p className="modal-subtitle">{subtitle}</p> : null}
          </div>
          <button className="modal-close" type="button" onClick={onClose} aria-label="Закрыть">
            ×
          </button>
        </div>
        <div className="modal-body">{children}</div>
        <div className="modal-footer">{footer}</div>
      </div>
    </div>
  );
}

function TaskComposer({
  task,
  draft,
  isSubmitting,
  isDeleting,
  onChange,
  onClose,
  onSubmit,
  onDelete,
}) {
  const title = task ? `Редактирование задачи #${task.id}` : "Создание новой задачи";
  const subtitle = "Параметры задачи";

  return (
    <ModalShell
      title={title}
      subtitle={subtitle}
      onClose={onClose}
      footer={(
        <>
          <button className="primary-button" type="submit" form="task-composer" disabled={isSubmitting}>
            {isSubmitting ? "Сохраняем..." : task ? "Сохранить задачу" : "Создать задачу"}
          </button>
          {task ? (
            <button className="danger-button" type="button" onClick={onDelete} disabled={isDeleting}>
              {isDeleting ? "Удаляем..." : "Удалить"}
            </button>
          ) : null}
          <button className="ghost-button" type="button" onClick={onClose}>
            Отмена
          </button>
        </>
      )}
    >
      <form id="task-composer" className="composer-form" onSubmit={onSubmit}>
        <label className="field field-span">
          <span>Название</span>
          <input
            type="text"
            value={draft.title}
            onChange={(event) => onChange({ title: event.target.value })}
            placeholder="Введите краткое название задачи..."
          />
        </label>

        <label className="field field-span">
          <span>Описание</span>
          <textarea
            value={draft.description}
            onChange={(event) => onChange({ description: event.target.value })}
            placeholder="Подробно опишите задачу, пациента, действия..."
          />
        </label>

        <label className="field">
          <span>Статус</span>
          <select
            value={draft.status}
            onChange={(event) => onChange({ status: event.target.value })}
          >
            {STATUS_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span>Срок выполнения</span>
          <input
            type="date"
            value={draft.dueDate}
            onChange={(event) => onChange({ dueDate: event.target.value })}
          />
        </label>

        {task ? (
          <div className="detail-box">
            <span>Создана</span>
            <strong>{formatDateTime(task.created_at)}</strong>
            <small>Изменена {formatDateTime(task.updated_at)}</small>
          </div>
        ) : null}
      </form>
    </ModalShell>
  );
}

function TemplateComposer({
  template,
  draft,
  isSubmitting,
  isDeleting,
  onChange,
  onClose,
  onSubmit,
  onDelete,
}) {
  const title = template ? `Редактирование повтора #${template.id}` : "Создание повтора";
  const subtitle = "Параметры повторения";

  return (
    <ModalShell
      title={title}
      subtitle={subtitle}
      onClose={onClose}
      footer={(
        <>
          <button
            className="primary-button"
            type="submit"
            form="template-composer"
            disabled={isSubmitting}
          >
            {isSubmitting ? "Сохраняем..." : template ? "Сохранить повтор" : "Создать повтор"}
          </button>
          {template ? (
            <button className="danger-button" type="button" onClick={onDelete} disabled={isDeleting}>
              {isDeleting ? "Удаляем..." : "Удалить"}
            </button>
          ) : null}
          <button className="ghost-button" type="button" onClick={onClose}>
            Отмена
          </button>
        </>
      )}
    >
      <form id="template-composer" className="composer-form" onSubmit={onSubmit}>
        <label className="field field-span">
          <span>Название</span>
          <input
            type="text"
            value={draft.title}
            onChange={(event) => onChange({ title: event.target.value })}
            placeholder="Введите краткое название повтора..."
          />
        </label>

        <label className="field field-span">
          <span>Описание</span>
          <textarea
            value={draft.description}
            onChange={(event) => onChange({ description: event.target.value })}
            placeholder="Подробно опишите повторяемую задачу..."
          />
        </label>

        <label className="field">
          <span>Тип повтора</span>
          <select
            value={draft.ruleType}
            onChange={(event) => onChange({ ruleType: event.target.value })}
          >
            {RULE_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {option.label}
              </option>
            ))}
          </select>
        </label>

        <label className="field">
          <span>Дата начала</span>
          <input
            type="date"
            value={draft.startDate}
            onChange={(event) => onChange({ startDate: event.target.value })}
          />
        </label>

        <RuleFields draft={draft} onChange={onChange} />

        <div className="field">
          <span>Дата окончания</span>
          <input
            type="date"
            value={draft.endDate}
            onChange={(event) => onChange({ endDate: event.target.value })}
            disabled={!draft.hasEndDate}
          />
          <label className="checkbox-row">
            <input
              type="checkbox"
              checked={draft.hasEndDate}
              onChange={(event) =>
                onChange({
                  hasEndDate: event.target.checked,
                  endDate: event.target.checked ? draft.endDate : "",
                })
              }
            />
            <span>Использовать дату окончания</span>
          </label>
        </div>

        <div className="helper-box field-span">
          <strong>{rulePreviewText(draft)}</strong>
        </div>
      </form>
    </ModalShell>
  );
}

function ReferenceDateDialog({ value, onChange, onClose }) {
  const [draft, setDraft] = useState(value);

  useEffect(() => {
    setDraft(value);
  }, [value]);

  return (
    <ModalShell
      title="Симуляция сегодняшней даты"
      subtitle="Выберите день, который фронтенд будет считать текущим."
      onClose={onClose}
      footer={(
        <>
          <button
            className="primary-button"
            type="button"
            onClick={() => {
              if (!draft) {
                return;
              }
              onChange(draft);
              onClose();
            }}
            disabled={!draft || draft === value}
          >
            Применить
          </button>
          <button className="ghost-button" type="button" onClick={onClose}>
            Отмена
          </button>
        </>
      )}
    >
      <div className="reference-date-dialog">
        <label className="field field-span">
          <span>Дата</span>
          <input
            autoFocus
            type="date"
            value={draft}
            onChange={(event) => setDraft(event.target.value)}
          />
        </label>
      </div>
    </ModalShell>
  );
}

function ToastViewport({ toasts, onDismiss }) {
  if (toasts.length === 0) {
    return null;
  }

  return (
    <div className="toast-viewport" aria-live="polite" aria-atomic="false">
      {toasts.map((toast) => (
        <div key={toast.id} className={`toast toast-${toast.tone}`} role="status">
          <span className="toast-text">{toast.text}</span>
          <button
            className="toast-close"
            type="button"
            onClick={() => onDismiss(toast.id)}
            aria-label="Закрыть уведомление"
          >
            ×
          </button>
        </div>
      ))}
    </div>
  );
}

export default function App() {
  const didInitRef = useRef(false);
  const nextToastIdRef = useRef(0);
  const toastTimersRef = useRef(new Map());
  const initialReferenceDate = useRef(toInputDate(new Date())).current;

  const [activePane, setActivePane] = useState("all");
  const [composer, setComposer] = useState(null);
  const [isReferenceDateDialogOpen, setIsReferenceDateDialogOpen] = useState(false);
  const [referenceDate, setReferenceDate] = useState(initialReferenceDate);

  const [tasks, setTasks] = useState([]);
  const [windowTasks, setWindowTasks] = useState([]);
  const [weekTasks, setWeekTasks] = useState([]);
  const [templates, setTemplates] = useState([]);

  const [selectedTask, setSelectedTask] = useState(null);
  const [selectedTemplate, setSelectedTemplate] = useState(null);

  const [taskDraft, setTaskDraft] = useState(createEmptyTaskDraft());
  const [templateDraft, setTemplateDraft] = useState(createEmptyTemplateDraft());

  const [taskPage, setTaskPage] = useState(createPageState(TASK_PAGE_LIMIT));
  const [windowPage, setWindowPage] = useState(createPageState(TASK_PAGE_LIMIT));
  const [weekPage, setWeekPage] = useState(createPageState(TASK_PAGE_LIMIT));
  const [templatePage, setTemplatePage] = useState(createPageState(TEMPLATE_PAGE_LIMIT));

  const [windowRange, setWindowRange] = useState(() => getDefaultWindowRange(initialReferenceDate));

  const [taskSearch, setTaskSearch] = useState("");
  const [windowSearch, setWindowSearch] = useState("");
  const [weekSearch, setWeekSearch] = useState("");
  const [templateSearch, setTemplateSearch] = useState("");

  const [toasts, setToasts] = useState([]);

  const [isTasksLoading, setIsTasksLoading] = useState(true);
  const [isWindowLoading, setIsWindowLoading] = useState(true);
  const [isWeekLoading, setIsWeekLoading] = useState(true);
  const [isTemplatesLoading, setIsTemplatesLoading] = useState(true);
  const [isTaskDetailLoading, setIsTaskDetailLoading] = useState(false);
  const [isTemplateDetailLoading, setIsTemplateDetailLoading] = useState(false);
  const [isTaskSubmitting, setIsTaskSubmitting] = useState(false);
  const [isTaskDeleting, setIsTaskDeleting] = useState(false);
  const [isTemplateSubmitting, setIsTemplateSubmitting] = useState(false);
  const [isTemplateDeleting, setIsTemplateDeleting] = useState(false);
  const [statusUpdatingKey, setStatusUpdatingKey] = useState("");

  const deferredTaskSearch = useDeferredValue(taskSearch.trim().toLowerCase());
  const deferredWindowSearch = useDeferredValue(windowSearch.trim().toLowerCase());
  const deferredWeekSearch = useDeferredValue(weekSearch.trim().toLowerCase());
  const deferredTemplateSearch = useDeferredValue(templateSearch.trim().toLowerCase());

  useEffect(
    () => () => {
      toastTimersRef.current.forEach((timerId) => {
        window.clearTimeout(timerId);
      });
      toastTimersRef.current.clear();
    },
    [],
  );

  function dismissToast(id) {
    const timerId = toastTimersRef.current.get(id);
    if (timerId) {
      window.clearTimeout(timerId);
      toastTimersRef.current.delete(id);
    }

    startTransition(() => {
      setToasts((current) => current.filter((toast) => toast.id !== id));
    });
  }

  function pushToast(tone, text) {
    const id = nextToastIdRef.current + 1;
    nextToastIdRef.current = id;

    startTransition(() => {
      setToasts((current) => [...current, { id, tone, text }]);
    });

    const timerId = window.setTimeout(() => {
      dismissToast(id);
    }, TOAST_TIMEOUT_MS);

    toastTimersRef.current.set(id, timerId);
  }

  const visibleTasks = tasks.filter((task) =>
    matchesTaskSearch(task, deferredTaskSearch),
  );
  const visibleWindowTasks = windowTasks.filter((task) =>
    matchesTaskSearch(task, deferredWindowSearch),
  );
  const visibleWeekTasks = weekTasks.filter((task) =>
    matchesTaskSearch(task, deferredWeekSearch),
  );
  const visibleTemplates = templates.filter((template) =>
    matchesTemplateSearch(template, deferredTemplateSearch),
  );

  useEffect(() => {
    if (didInitRef.current) {
      return;
    }

    didInitRef.current = true;

    void Promise.all([
      refreshTasks({ page: createPageState(TASK_PAGE_LIMIT) }),
      refreshWindowTasks(getDefaultWindowRange(initialReferenceDate), {
        page: createPageState(TASK_PAGE_LIMIT),
      }),
      refreshWeekTasks(initialReferenceDate, {
        page: createPageState(TASK_PAGE_LIMIT),
      }),
      refreshTemplates(createPageState(TEMPLATE_PAGE_LIMIT)),
    ]);
  }, []);

  async function refreshTasks(options = {}) {
    const baseDate = options.baseDate ?? referenceDate;
    const requestedPane = options.pane ?? (isTaskCollectionPane(activePane) ? activePane : "all");
    const pane = isTaskCollectionPane(requestedPane) ? requestedPane : "all";
    const nextPage = normalizePage(options.page ?? taskPage, TASK_PAGE_LIMIT);
    setIsTasksLoading(true);

    try {
      if (pane === "all") {
        const items = await listTasks(nextPage);
        const normalizedItems = items.map((task) => ({
          ...task,
          overdue: isBeforeDate(task.due_date, baseDate) && task.status !== "done",
        }));

        startTransition(() => {
          setTaskPage(nextPage);
          setTasks(normalizedItems);

          if (selectedTask) {
            const fresh = normalizedItems.find((task) => task.id === selectedTask.id);
            if (fresh) {
              setSelectedTask(fresh);
            }
          }
        });
        return;
      }

      if (pane === "undated") {
        const materializedItems = await listTasks({
          limit: TASK_ACCUMULATION_LIMIT,
          offset: 0,
        });
        const undatedItems = materializedItems
          .filter((task) => !task.due_date)
          .map((task) => ({ ...task, overdue: false }));
        const items = sliceItems(undatedItems, nextPage);

        startTransition(() => {
          setTaskPage(nextPage);
          setTasks(items);

          if (selectedTask) {
            const fresh = undatedItems.find((task) => task.id === selectedTask.id);
            if (fresh) {
              setSelectedTask(fresh);
            }
          }
        });
        return;
      }

      const [todayItems, materializedItems] = await Promise.all([
        listTasksInRange({
          from: baseDate,
          to: baseDate,
          limit: TASK_ACCUMULATION_LIMIT,
          offset: 0,
        }),
        listTasks({
          limit: TASK_ACCUMULATION_LIMIT,
          offset: 0,
        }),
      ]);

      const overdueItems = materializedItems
        .filter((task) => isBeforeDate(task.due_date, baseDate) && task.status !== "done")
        .map((task) => ({ ...task, overdue: true }));

      const currentDayItems = todayItems.map((task) => ({
        ...task,
        overdue: false,
      }));

      const mergedItems = [...overdueItems, ...currentDayItems];
      const items = sliceItems(mergedItems, nextPage);

      startTransition(() => {
        setTaskPage(nextPage);
        setTasks(items);

        if (selectedTask) {
          const fresh = mergedItems.find((task) => task.id === selectedTask.id);
          if (fresh) {
            setSelectedTask(fresh);
          }
        }
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTasksLoading(false);
    }
  }

  async function refreshWindowTasks(range = windowRange, options = {}) {
    const nextRange = {
      from: range.from,
      to: range.to,
    };
    const nextPage = normalizePage(options.page ?? windowPage, TASK_PAGE_LIMIT);

    if (!nextRange.from || !nextRange.to) {
      pushToast("error", "Укажите обе границы диапазона.");
      return;
    }

    setIsWindowLoading(true);

    try {
      const items = await listTasksInRange({
        ...nextRange,
        ...nextPage,
      });

      startTransition(() => {
        setWindowRange(nextRange);
        setWindowPage(nextPage);
        setWindowTasks(items);
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsWindowLoading(false);
    }
  }

  async function refreshWeekTasks(baseDate = referenceDate, options = {}) {
    const nextRange = getWeekRange(baseDate);
    const nextPage = normalizePage(options.page ?? weekPage, TASK_PAGE_LIMIT);
    setIsWeekLoading(true);

    try {
      const items = await listTasksInRange({
        ...nextRange,
        ...nextPage,
      });

      startTransition(() => {
        setWeekPage(nextPage);
        setWeekTasks(items);

        if (selectedTask) {
          const fresh = items.find((task) => task.id === selectedTask.id);
          if (fresh) {
            setSelectedTask(fresh);
          }
        }
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsWeekLoading(false);
    }
  }

  async function refreshTemplates(page = templatePage) {
    const nextPage = normalizePage(page, TEMPLATE_PAGE_LIMIT);
    setIsTemplatesLoading(true);

    try {
      const items = await listTemplates(nextPage);

      startTransition(() => {
        setTemplatePage(nextPage);
        setTemplates(items);

        if (selectedTemplate) {
          const fresh = items.find((template) => template.id === selectedTemplate.id);
          if (fresh) {
            setSelectedTemplate(fresh);
          }
        }
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTemplatesLoading(false);
    }
  }

  function openTaskComposer() {
    startTransition(() => {
      setSelectedTask(null);
      setTaskDraft(createEmptyTaskDraft());
      setComposer({ type: "task", mode: "create" });
    });
  }

  function openTemplateComposer() {
    startTransition(() => {
      setSelectedTemplate(null);
      setTemplateDraft(createEmptyTemplateDraft());
      setComposer({ type: "template", mode: "create" });
    });
  }

  function closeComposer() {
    setComposer(null);
  }

  function openPrimaryComposer() {
    if (activePane === "templates") {
      openTemplateComposer();
      return;
    }

    openTaskComposer();
  }

  function handleReferenceDateChange(nextDate) {
    if (!nextDate) {
      return;
    }

    startTransition(() => {
      setReferenceDate(nextDate);
    });

    if (activePane === "week") {
      void refreshWeekTasks(nextDate, {
        page: createPageState(weekPage.limit),
      });
      return;
    }

    if (activePane === "all" || activePane === "today") {
      void refreshTasks({
        page: createPageState(taskPage.limit),
        pane: activePane,
        baseDate: nextDate,
      });
      return;
    }

    if (activePane === "window") {
      const nextRange = getDefaultWindowRange(nextDate);
      startTransition(() => {
        setWindowRange(nextRange);
      });
      void refreshWindowTasks(nextRange, {
        page: createPageState(windowPage.limit),
      });
    }
  }

  function openReferenceDateDialog() {
    setIsReferenceDateDialogOpen(true);
  }

  function closeReferenceDateDialog() {
    setIsReferenceDateDialogOpen(false);
  }

  function handlePaneChange(nextPane) {
    startTransition(() => {
      setActivePane(nextPane);
    });

    if (isTaskCollectionPane(nextPane)) {
      void refreshTasks({
        page: createPageState(taskPage.limit),
        pane: nextPane,
      });
      return;
    }

    if (nextPane === "week") {
      void refreshWeekTasks(referenceDate, {
        page: createPageState(weekPage.limit),
      });
    }
  }

  async function openTask(id) {
    setIsTaskDetailLoading(true);

    try {
      const task = await getTaskById(id);

      startTransition(() => {
        setSelectedTask(task);
        setTaskDraft(toTaskDraft(task));
        setComposer({ type: "task", mode: "edit" });
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTaskDetailLoading(false);
    }
  }

  async function openTemplate(id) {
    setIsTemplateDetailLoading(true);

    try {
      const template = await getTemplateById(id);

      startTransition(() => {
        setSelectedTemplate(template);
        setTemplateDraft(templateToDraft(template));
        setComposer({ type: "template", mode: "edit" });
      });
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTemplateDetailLoading(false);
    }
  }

  async function handleTaskSubmit(event) {
    event.preventDefault();

    const payload = {
      title: taskDraft.title.trim(),
      description: taskDraft.description.trim(),
      status: taskDraft.status,
      due_date: selectedTask
        ? taskDraft.dueDate || null
        : taskDraft.dueDate || undefined,
    };

    if (!payload.title) {
      pushToast("error", "Название задачи обязательно.");
      return;
    }

    setIsTaskSubmitting(true);

    try {
      if (selectedTask) {
        const updated = await updateTask(selectedTask.id, payload);

        startTransition(() => {
          setSelectedTask(updated);
          setTaskDraft(toTaskDraft(updated));
          setComposer(null);
        });

        pushToast("success", `Задача #${selectedTask.id} обновлена.`);
      } else {
        const created = await createTask(payload);

        startTransition(() => {
          setSelectedTask(created);
          setTaskDraft(toTaskDraft(created));
          setComposer(null);
        });

        pushToast("success", `Задача #${created.id} создана.`);
      }

      await Promise.all([
        refreshTasks({
          page: selectedTask ? taskPage : createPageState(taskPage.limit),
          pane: "all",
        }),
        refreshWindowTasks(windowRange, { page: windowPage }),
        refreshWeekTasks(referenceDate, { page: weekPage }),
      ]);
      setActivePane("all");
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTaskSubmitting(false);
    }
  }

  async function handleTaskDelete() {
    if (!selectedTask) {
      return;
    }

    const confirmed = window.confirm(`Удалить задачу #${selectedTask.id}?`);
    if (!confirmed) {
      return;
    }

    setIsTaskDeleting(true);

    try {
      await deleteTask(selectedTask.id);

      startTransition(() => {
        setSelectedTask(null);
        setTaskDraft(createEmptyTaskDraft());
        setComposer(null);
      });

      pushToast("success", `Задача #${selectedTask.id} удалена.`);

      await Promise.all([
        refreshTasks({
          page:
            tasks.length === 1 && taskPage.offset > 0
              ? {
                  limit: taskPage.limit,
                  offset: Math.max(0, taskPage.offset - taskPage.limit),
                }
              : taskPage,
          pane: activePane,
        }),
        refreshWindowTasks(windowRange, { page: windowPage }),
        refreshWeekTasks(referenceDate, { page: weekPage }),
      ]);
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTaskDeleting(false);
    }
  }

  async function handleOccurrenceStatusChange(task, status) {
    const key = taskKey(task);
    setStatusUpdatingKey(key);

    try {
      const payload =
        task.id > 0
          ? { id: task.id, status }
          : { template_id: task.template_id, due_date: task.due_date, status };

      const updated = await updateTaskStatus(payload);

      if (updated.id > 0) {
        startTransition(() => {
          setSelectedTask(updated);
        });
      }

      pushToast(
        "success",
        task.id > 0
          ? `Статус задачи #${task.id} обновлен.`
          : `Задача на ${formatDateOnly(task.due_date)} сохранена в БД.`,
      );

      await Promise.all([
        refreshWindowTasks(windowRange, { page: windowPage }),
        refreshWeekTasks(referenceDate, { page: weekPage }),
        refreshTasks({
          page: task.virtual ? createPageState(taskPage.limit) : taskPage,
          pane: activePane,
        }),
      ]);
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setStatusUpdatingKey("");
    }
  }

  async function handleRangeSubmit(event) {
    event.preventDefault();
    await refreshWindowTasks(windowRange, {
      page: createPageState(windowPage.limit),
    });
  }

  async function handleTemplateSubmit(event) {
    event.preventDefault();

    let payload;
    try {
      payload = buildTemplatePayload(templateDraft);
    } catch (error) {
      pushToast("error", error.message);
      return;
    }

    setIsTemplateSubmitting(true);

    try {
      if (selectedTemplate) {
        const updated = await updateTemplate(selectedTemplate.id, payload);

        startTransition(() => {
          setSelectedTemplate(updated);
          setTemplateDraft(templateToDraft(updated));
          setComposer(null);
        });

        pushToast("success", `Повтор #${selectedTemplate.id} обновлен.`);
      } else {
        const created = await createTemplate(payload);

        startTransition(() => {
          setSelectedTemplate(created);
          setTemplateDraft(templateToDraft(created));
          setComposer(null);
        });

        pushToast("success", `Повтор #${created.id} создан.`);
      }

      await Promise.all([
        refreshTemplates(selectedTemplate ? templatePage : createPageState(templatePage.limit)),
        refreshWindowTasks(windowRange, { page: windowPage }),
        refreshWeekTasks(referenceDate, { page: weekPage }),
        refreshTasks({ page: taskPage, pane: "all" }),
      ]);
      setActivePane("templates");
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTemplateSubmitting(false);
    }
  }

  async function handleTemplateDelete() {
    if (!selectedTemplate) {
      return;
    }

    const confirmed = window.confirm(`Удалить повтор #${selectedTemplate.id}?`);
    if (!confirmed) {
      return;
    }

    setIsTemplateDeleting(true);

    try {
      await deleteTemplate(selectedTemplate.id);

      startTransition(() => {
        setSelectedTemplate(null);
        setTemplateDraft(createEmptyTemplateDraft());
        setComposer(null);
      });

      pushToast("success", `Повтор #${selectedTemplate.id} удален.`);

      await Promise.all([
        refreshTemplates(
          templates.length === 1 && templatePage.offset > 0
            ? {
                limit: templatePage.limit,
                offset: Math.max(0, templatePage.offset - templatePage.limit),
              }
            : templatePage,
        ),
        refreshWindowTasks(windowRange, { page: windowPage }),
        refreshWeekTasks(referenceDate, { page: weekPage }),
        refreshTasks({ page: taskPage, pane: "all" }),
      ]);
    } catch (error) {
      pushToast("error", error.message);
    } finally {
      setIsTemplateDeleting(false);
    }
  }

  async function changeTaskPage(direction) {
    const nextOffset =
      direction === "next"
        ? taskPage.offset + taskPage.limit
        : Math.max(0, taskPage.offset - taskPage.limit);

    await refreshTasks({
      page: {
        limit: taskPage.limit,
        offset: nextOffset,
      },
      pane: activePane,
    });
  }

  async function changeWindowPage(direction) {
    const nextOffset =
      direction === "next"
        ? windowPage.offset + windowPage.limit
        : Math.max(0, windowPage.offset - windowPage.limit);

    await refreshWindowTasks(windowRange, {
      page: {
        limit: windowPage.limit,
        offset: nextOffset,
      },
    });
  }

  async function changeWeekPage(direction) {
    const nextOffset =
      direction === "next"
        ? weekPage.offset + weekPage.limit
        : Math.max(0, weekPage.offset - weekPage.limit);

    await refreshWeekTasks(referenceDate, {
      page: {
        limit: weekPage.limit,
        offset: nextOffset,
      },
    });
  }

  async function changeTemplatePage(direction) {
    const nextOffset =
      direction === "next"
        ? templatePage.offset + templatePage.limit
        : Math.max(0, templatePage.offset - templatePage.limit);

    await refreshTemplates({
      limit: templatePage.limit,
      offset: nextOffset,
    });
  }

  function renderPane() {
    if (activePane === "templates") {
      return (
        <>
          <div className="surface-toolbar">
            <label className="toolbar-search">
              <span>Поиск</span>
              <input
                type="search"
                value={templateSearch}
                onChange={(event) => setTemplateSearch(event.target.value)}
                placeholder="По названию, описанию или типу"
              />
            </label>

            <div className="toolbar-meta">
              <span className="counter">
                {isTemplatesLoading ? "Загрузка..." : `${visibleTemplates.length} записей`}
              </span>
              <button
                className="ghost-button"
                type="button"
                onClick={() => refreshTemplates(templatePage)}
                disabled={isTemplatesLoading}
              >
                Обновить
              </button>
            </div>
          </div>

          <div className="record-list">
            {isTemplatesLoading ? (
              <div className="empty-state">Загружаем повторы...</div>
            ) : visibleTemplates.length > 0 ? (
              visibleTemplates.map((template) => (
                <TemplateRow
                  key={template.id}
                  template={template}
                  isActive={template.id === selectedTemplate?.id}
                  onOpen={openTemplate}
                />
              ))
            ) : (
              <div className="empty-state">Повторов пока нет.</div>
            )}
          </div>

          <PaginationControls
            page={templatePage}
            count={templates.length}
            loading={isTemplatesLoading}
            onPrev={() => changeTemplatePage("prev")}
            onNext={() => changeTemplatePage("next")}
          />
        </>
      );
    }

    if (activePane === "week") {
      const weekRange = getWeekRange(referenceDate);

      return (
        <>
          <div className="surface-toolbar week-toolbar">
            <div className="week-range-summary">
              <div className="week-range-item">
                <span>С</span>
                <strong>{formatDateOnly(weekRange.from)}</strong>
              </div>
              <div className="week-range-item">
                <span>По</span>
                <strong>{formatDateOnly(weekRange.to)}</strong>
              </div>
            </div>

            <button
              className="ghost-button"
              type="button"
              onClick={() => refreshWeekTasks(referenceDate, { page: weekPage })}
              disabled={isWeekLoading}
            >
              Обновить
            </button>
          </div>

          <div className="surface-toolbar secondary">
            <label className="toolbar-search">
              <span>Поиск</span>
              <input
                type="search"
                value={weekSearch}
                onChange={(event) => setWeekSearch(event.target.value)}
                placeholder="По дате, шаблону, названию или статусу"
              />
            </label>

            <span className="counter">
              {isWeekLoading ? "Загрузка..." : `${visibleWeekTasks.length} записей`}
            </span>
          </div>

          <div className="record-list">
            {isWeekLoading ? (
              <div className="empty-state">Загружаем задачи...</div>
            ) : visibleWeekTasks.length > 0 ? (
              visibleWeekTasks.map((task) => (
                <OccurrenceRow
                  key={taskKey(task)}
                  task={task}
                  isUpdating={statusUpdatingKey === taskKey(task)}
                  onStatusChange={handleOccurrenceStatusChange}
                  onOpen={openTask}
                />
              ))
            ) : (
              <div className="empty-state">На выбранной неделе задач нет.</div>
            )}
          </div>

          <PaginationControls
            page={weekPage}
            count={weekTasks.length}
            loading={isWeekLoading}
            onPrev={() => changeWeekPage("prev")}
            onNext={() => changeWeekPage("next")}
          />
        </>
      );
    }

    if (activePane === "window") {
      return (
        <>
          <form className="surface-toolbar range-toolbar" onSubmit={handleRangeSubmit}>
            <label className="toolbar-search compact">
              <span>С</span>
              <input
                type="date"
                value={windowRange.from}
                onChange={(event) =>
                  setWindowRange((current) => ({
                    ...current,
                    from: event.target.value,
                  }))
                }
              />
            </label>

            <label className="toolbar-search compact">
              <span>По</span>
              <input
                type="date"
                value={windowRange.to}
                onChange={(event) =>
                  setWindowRange((current) => ({
                    ...current,
                    to: event.target.value,
                  }))
                }
              />
            </label>

            <button className="primary-button" type="submit" disabled={isWindowLoading}>
              Показать
            </button>

            <button
              className="ghost-button"
              type="button"
              onClick={() => {
                const nextRange = getDefaultWindowRange(referenceDate);
                setWindowRange(nextRange);
                void refreshWindowTasks(nextRange, {
                  page: createPageState(windowPage.limit),
                });
              }}
            >
              Сбросить
            </button>
          </form>

          <div className="surface-toolbar secondary">
            <label className="toolbar-search">
              <span>Поиск</span>
              <input
                type="search"
                value={windowSearch}
                onChange={(event) => setWindowSearch(event.target.value)}
                placeholder="По дате, шаблону, названию или статусу"
              />
            </label>

            <span className="counter">
              {isWindowLoading ? "Загрузка..." : `${visibleWindowTasks.length} записей`}
            </span>
          </div>

          <div className="record-list">
            {isWindowLoading ? (
              <div className="empty-state">Загружаем задачи...</div>
            ) : visibleWindowTasks.length > 0 ? (
              visibleWindowTasks.map((task) => (
                <OccurrenceRow
                  key={taskKey(task)}
                  task={task}
                  isUpdating={statusUpdatingKey === taskKey(task)}
                  onStatusChange={handleOccurrenceStatusChange}
                  onOpen={openTask}
                />
              ))
            ) : (
              <div className="empty-state">На выбранном диапазоне задач нет.</div>
            )}
          </div>

          <PaginationControls
            page={windowPage}
            count={windowTasks.length}
            loading={isWindowLoading}
            onPrev={() => changeWindowPage("prev")}
            onNext={() => changeWindowPage("next")}
          />
        </>
      );
    }

    return (
      <>
        <div className="surface-toolbar">
          <label className="toolbar-search">
            <span>Поиск</span>
            <input
              type="search"
              value={taskSearch}
              onChange={(event) => setTaskSearch(event.target.value)}
              placeholder="По названию, описанию или статусу"
            />
          </label>

          <div className="toolbar-meta">
            <span className="counter">
              {isTasksLoading ? "Загрузка..." : `${visibleTasks.length} записей`}
            </span>
            <button
              className="ghost-button"
              type="button"
              onClick={() => refreshTasks({ page: taskPage })}
              disabled={isTasksLoading}
            >
              Обновить
            </button>
          </div>
        </div>

        <div className="record-list">
          {isTasksLoading ? (
            <div className="empty-state">Загружаем задачи...</div>
          ) : visibleTasks.length > 0 ? (
            visibleTasks.map((task) => (
              <TaskRow
                key={taskKey(task)}
                task={task}
                isActive={task.id > 0 && task.id === selectedTask?.id}
                onOpen={openTask}
              />
            ))
          ) : (
            <div className="empty-state">{taskPaneEmptyLabel(activePane)}</div>
          )}
        </div>

        <PaginationControls
          page={taskPage}
          count={tasks.length}
          loading={isTasksLoading}
          onPrev={() => changeTaskPage("prev")}
          onNext={() => changeTaskPage("next")}
        />
      </>
    );
  }

  return (
    <div className="app-shell">
      <div className="app-frame">
        <SidebarNav
          referenceDate={referenceDate}
          onOpenReferenceDatePicker={openReferenceDateDialog}
        />

        <main className="app-content">
          <div className="app-content-inner">
            <header className="workspace-header">
              <div>
                <div className="workspace-eyebrow">ЗАДАЧИ</div>
                <h1>{paneLabel(activePane)}</h1>
              </div>

              <div className="workspace-header-actions">
                <WorkspaceTabs activePane={activePane} onChange={handlePaneChange} />
                <button
                  className="create-button"
                  type="button"
                  onClick={openPrimaryComposer}
                  aria-label={activePane === "templates" ? "Создать повтор" : "Создать задачу"}
                  title={activePane === "templates" ? "Создать повтор" : "Создать задачу"}
                >
                  <svg className="create-button-glyph" viewBox="0 0 24 24" aria-hidden="true">
                    <path d="M12 5V19" />
                    <path d="M5 12H19" />
                  </svg>
                </button>
              </div>
            </header>

            <section className="workspace-surface">
              {renderPane()}
            </section>
          </div>
        </main>
      </div>

      {composer?.type === "task" ? (
        <TaskComposer
          task={composer.mode === "edit" ? selectedTask : null}
          draft={taskDraft}
          isSubmitting={isTaskSubmitting}
          isDeleting={isTaskDeleting}
          onChange={(patch) =>
            setTaskDraft((current) => ({
              ...current,
              ...patch,
            }))
          }
          onClose={closeComposer}
          onSubmit={handleTaskSubmit}
          onDelete={handleTaskDelete}
        />
      ) : null}

      {isReferenceDateDialogOpen ? (
        <ReferenceDateDialog
          value={referenceDate}
          onChange={handleReferenceDateChange}
          onClose={closeReferenceDateDialog}
        />
      ) : null}

      {composer?.type === "template" ? (
        <TemplateComposer
          template={composer.mode === "edit" ? selectedTemplate : null}
          draft={templateDraft}
          isSubmitting={isTemplateSubmitting}
          isDeleting={isTemplateDeleting}
          onChange={(patch) =>
            setTemplateDraft((current) => ({
              ...current,
              ...patch,
            }))
          }
          onClose={closeComposer}
          onSubmit={handleTemplateSubmit}
          onDelete={handleTemplateDelete}
        />
      ) : null}

      {isTaskDetailLoading || isTemplateDetailLoading ? (
        <div className="busy-overlay" aria-hidden="true">
          <div className="busy-badge">Открываем…</div>
        </div>
      ) : null}

      <ToastViewport toasts={toasts} onDismiss={dismissToast} />
    </div>
  );
}
