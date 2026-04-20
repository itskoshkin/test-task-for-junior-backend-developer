# Task Service

## Запуск

1. Установить зависимости:
   ```bash
   npm install
   ```
2. Запустить dev-сервер:
   ```bash
   npm run dev
   ```

## Переменные окружения

Пример лежит в [`.env.example`](./frontend/.env.example).

- `VITE_API_PROXY_TARGET` — адрес бэкенда для dev proxy (куда Vite проксирует `/api` и `/swagger`)
- `VITE_API_BASE_URL` — переопределить базовый URL API (по умолчанию `/api/v1`)

## Демо

[Web UI](https://task-tracker-service.itskoshkin.ru) и [Swagger](https://task-tracker-service.itskoshkin.ru/swagger)
