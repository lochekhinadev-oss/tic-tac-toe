## Дисклеймер

Этот проект представляет собой учебную open-source реализацию классической игры «Крестики-нолики».

Репозиторий является публичным и может свободно использоваться, изучаться, изменяться и дорабатываться всеми желающими.

Обратите внимание, что проект находится в стадии активной разработки. Архитектура, функциональность и отдельные части приложения могут изменяться со временем, а некоторые функции могут быть незавершёнными, экспериментальными или нестабильными.

Предложения, обратная связь и любые улучшения приветствуются.

## Health Checks

| Контейнер | Проверка | Назначение |
|---|---|---|
| `backend` | `GET /healthz` | Процесс жив |
| `backend` | `GET /readyz` | База доступна, сервис готов |
| `frontend` | `GET /healthz` | Nginx со статикой жив |
| `gateway` | `GET /healthz` | Входная точка жива |
| `cleanup` | `GET /healthz` | Worker запущен |
| `cleanup` | `GET /metrics` | Метрики и состояние worker'а |
| `db` | `pg_isready` | Postgres готов принимать подключения |
| `redis` | `redis-cli ping` | Redis отвечает |

`seed` — одноразовый job, отдельная health-ручка ему не нужна.

## Monitoring

| Сервис | URL | Что там есть |
|---|---|---|
| `backend` | `https://localhost:3000/tic-tac-toe/metrics` | Prometheus metrics |
| `cleanup` | `http://localhost:9091/metrics` | Prometheus metrics |
| `blackbox-exporter` | `http://localhost:9115` | HTTP health probes for app endpoints |
| `prometheus` | `http://localhost:9090` | Scrape targets и TSDB |
| `grafana` | `http://localhost:3001` | Dashboards, datasource уже настроен |
| `alertmanager` | `http://localhost:9093` | Приёмник alert rules Prometheus |

Prometheus уже скрапит:
- `backend`
- `cleanup`
- `postgres-exporter`
- `redis-exporter`
- `blackbox-exporter`

Blackbox probes cover:
- `https://gateway:443/tic-tac-toe/healthz`
- `https://gateway:443/tic-tac-toe/frontend-healthz`
- `https://gateway:443/tic-tac-toe/readyz`
- `http://backend:8080/healthz`
- `http://backend:8080/readyz`
- `http://cleanup:9091/healthz`
- `http://frontend:80/healthz`

В Grafana уже provisioned четыре дашборда:
- `Tic-Tac-Toe System Overview`
- `Tic-Tac-Toe API Overview`
- `Tic-Tac-Toe Auth & Game`
- `Tic-Tac-Toe Cleanup & Infra`

На `Tic-Tac-Toe System Overview` есть кликабельные links на:
- gateway health
- backend readyz
- Prometheus
- Grafana
- Alertmanager

Prometheus alert rules уже загружены и покрывают:
- падение backend/cleanup/exporters;
- рост 5xx rate;
- рост p95 latency;
- всплески auth errors и authz denials;
- ошибки cleanup и отсутствие успешных cleanup runs.

## Run modes

- `make run` - only app stack: `backend`, `frontend`, `gateway`, `db`, `redis`, `cleanup`
- `make observability-up` - only observability stack: `prometheus`, `grafana`, `alertmanager`, `postgres-exporter`, `redis-exporter`
- `make full-up` - app + observability together

Alertmanager wired for Telegram personal chat via `env/monitoring.env`:
- `TELEGRAM_BOT_TOKEN`
- `TELEGRAM_CHAT_ID`

If both values are empty, Alertmanager stays on the local blackhole receiver and does not send notifications.
