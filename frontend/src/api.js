const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL || "/api/v1").replace(
  /\/$/,
  "",
);

async function request(path, options = {}) {
  const headers = new Headers(options.headers || {});
  let response;

  if (options.body != null && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  try {
    response = await fetch(`${API_BASE_URL}${path}`, {
      ...options,
      headers,
    });
  } catch (error) {
    throw new Error(
      "Не удалось подключиться к API. Проверьте, что Go-сервис запущен на localhost:8080.",
      { cause: error },
    );
  }

  if (response.status === 204) {
    return null;
  }

  const text = await response.text();
  const payload = text ? safeParse(text) : null;

  if (!response.ok) {
    const message =
      typeof payload === "object" && payload?.error
        ? payload.error
        : `Ошибка запроса (${response.status})`;

    throw new Error(message);
  }

  return payload;
}

function safeParse(text) {
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

function toQueryString(params) {
  const query = new URLSearchParams();

  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null || value === "") {
      continue;
    }

    query.set(key, String(value));
  }

  const serialized = query.toString();
  return serialized ? `?${serialized}` : "";
}

export function listTasks(page = {}) {
  return request(`/tasks${toQueryString(page)}`);
}

export function listTasksInRange(range) {
  return request(`/tasks${toQueryString(range)}`);
}

export function getTaskById(id) {
  return request(`/tasks/${id}`);
}

export function createTask(payload) {
  return request("/tasks", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateTask(id, payload) {
  return request(`/tasks/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteTask(id) {
  return request(`/tasks/${id}`, {
    method: "DELETE",
  });
}

export function updateTaskStatus(payload) {
  return request("/tasks/status", {
    method: "PATCH",
    body: JSON.stringify(payload),
  });
}

export function listTemplates(page = {}) {
  return request(`/task-templates${toQueryString(page)}`);
}

export function getTemplateById(id) {
  return request(`/task-templates/${id}`);
}

export function createTemplate(payload) {
  return request("/task-templates", {
    method: "POST",
    body: JSON.stringify(payload),
  });
}

export function updateTemplate(id, payload) {
  return request(`/task-templates/${id}`, {
    method: "PUT",
    body: JSON.stringify(payload),
  });
}

export function deleteTemplate(id) {
  return request(`/task-templates/${id}`, {
    method: "DELETE",
  });
}
