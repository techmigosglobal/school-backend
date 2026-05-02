# School Backend Implementation, Integration, and VPS Deployment Guide

This document explains what has been implemented in `school-backend`, how it integrates with the Flutter app (`/Users/kgt/Documents/schooldesk`), and how to deploy it on a VPS in production mode.

For GitHub → Hostinger VPS → Coolify deployments, use the root `docker-compose.coolify.yml` and `deploy/COOLIFY_DEPLOYMENT.md`. That compose file provisions the Flutter web app, API, worker, Postgres, and Redis with persistent Docker volumes.

## 1) What is implemented (backend architecture)

## Runtime entrypoint
- `main.go`
- Loads environment config, validates production requirements, initializes Redis-backed services, initializes database, sets up API routes, and starts server.
- Supports two runtime modes:
  - `APP_MODE=api` (default): starts Gin HTTP API server.
  - `APP_MODE=worker`: starts background notification worker.

## Configuration and environment
- `internal/config/config.go`
- Strong production validation:
  - `JWT_SECRET` required and minimum length 32.
  - `DATABASE_URL` required in production.
  - `REDIS_URL` and `REDIS_PASSWORD` required in production.
  - `ALLOWED_ORIGINS` required in production.
- Production-safe defaults:
  - `DISABLE_PUBLIC_REGISTRATION=true` (prod default).
  - `MIGRATE_ON_START=false` (prod default).
  - `USE_POSTGRES_ONLY=true` (prod default).

## Database layer
- `internal/database/database.go`
- Supports Postgres (primary/prod) and SQLite (local/dev fallback).
- Uses phased auto-migrations for stable schema creation.
- Empty databases bootstrap only the Principal account; all operational records are created through the app.
- If `USE_POSTGRES_ONLY=true`, startup fails when `DATABASE_URL` is not Postgres.

## Security and API guardrails
- `internal/middleware/auth.go`
  - JWT auth parsing and validation.
  - Role-based access control via `RBACMiddleware`.
  - School scope enforcement via `SchoolScopeMiddleware`:
    - Prevents cross-school access.
    - Injects token school scope into query params to force tenant scoping.
  - Strict CORS allowlist from `ALLOWED_ORIGINS`.
- `internal/middleware/request_id.go`
  - Adds request correlation (`X-Request-ID`) for debugging and incident tracing.
- `internal/handlers/helpers.go` and `internal/models/dto.go`
  - Standardized success/error response envelope including `request_id`, `code`, and `details` fields.

## Redis-backed platform services
- `internal/platform/redis_client.go`
  - Redis URL parsing, password override, ping validation, timeout tuning.
- `internal/services/cache.go`
  - Shared cache service used by read endpoints.
- `internal/services/rate_limit.go`
  - Per-endpoint and per-subject (user/IP) request throttling.
- `internal/services/session_store.go`
  - Session revocation/JTI invalidation and refresh token storage.
- `internal/services/job_queue.go` + `internal/worker/notifications.go`
  - Stream-based job queue and notification worker consumer.

## API modules implemented
- `auth` (login/refresh/logout/profile/register when enabled)
- `schools`, `academic-years`, `grades`, `sections`, `departments`, `subjects`, `rooms`
- `staff` (CRUD + leave balances + attendance)
- `students` (CRUD + enrollments + attendance + fees + marks + transport)
- `attendance` (sessions, marks, summary, staff attendance)
- `exams` (types, exams, schedules, marks, report-cards, grading)
- `fees` (categories, structures, invoices, payments, concessions)
- `leave` (types, applications, approval, balances)
- `timetable` (slots/substitutions/section timetable)
- `announcements`, `events`, `notifications`

## 2) Frontend integration (how app and backend are connected)

The Flutter app is API-first and wired to this backend under `/api/v1`.

## Integration points in frontend
- API base URL and environment validation:
  - `lib/core/config/env_config.dart`
- HTTP client and auth/session handling:
  - `lib/services/backend_api_client.dart`
- App boot sync:
  - `lib/main.dart`
- Post-login sync:
  - `lib/features/auth/presentation/controllers/auth_controller.dart`
- Domain hydration (students/staff/leaves/exams/fees/timetable/notifications):
  - `lib/services/production_data_sync_service.dart`
- Local cache and role data refresh:
  - `lib/services/local_storage_service.dart`
  - `lib/services/role_access_service.dart`

## Request flow
1. App starts, initializes `BackendApiClient`.
2. If token exists, app fetches profile and synchronizes domain data from backend.
3. UI reads from synchronized cache keys (no dependency on hardcoded mock seed repositories).
4. Login path validates role from backend token/profile and then triggers full sync.

## Mock/fake data status
- Dedicated mock repositories have been removed from active flow.
- Runtime data is pulled from backend APIs and persisted to local cache.
- Backend demo seeding is disabled; the app starts from the Principal account and real operational records.

## 3) Production environment variables

Use `.env.example` as baseline in production:

```env
PORT=8080
APP_MODE=api
ENVIRONMENT=production
JWT_SECRET=replace_with_min_32_char_random_secret
DATABASE_URL=postgres://username:password@postgres:5432/schooldesk?sslmode=disable
DATABASE_DSN=school.db
ALLOWED_ORIGINS=https://app.example.com,https://admin.example.com
REDIS_URL=redis://:replace_with_strong_password@schooldesk-redis:6379/0
REDIS_PASSWORD=replace_with_strong_password
REDIS_DB=0
CACHE_TTL_SECONDS=120
RATE_LIMIT_WINDOW_SECONDS=60
RATE_LIMIT_MAX_LOGIN=5
RATE_LIMIT_MAX_API=120
DISABLE_PUBLIC_REGISTRATION=true
BOOTSTRAP_PRINCIPAL_EMAIL=principal@yourdomain.com
BOOTSTRAP_PRINCIPAL_PASSWORD=replace_with_strong_initial_password
MIGRATE_ON_START=false
USE_POSTGRES_ONLY=true
REQUIRE_HTTPS_API_BASE_URL=true
```

Notes:
- `DATABASE_DSN` is ignored when Postgres is used; keep it only for local fallback.
- For first deploy, run migrations once with `MIGRATE_ON_START=true`, then set back to `false`.
- For an empty production database, the bootstrap Principal account uses `BOOTSTRAP_PRINCIPAL_EMAIL`, username `PRINC`, and `BOOTSTRAP_PRINCIPAL_PASSWORD`. The password must be at least 12 characters in production.

## 4) VPS deployment runbook

## Prerequisites
- VPS with Ubuntu 22.04+ (or equivalent).
- Installed: `git`, `golang`, `postgresql`, `redis`, `nginx`, `certbot`.
- DNS A record for API domain (example `api.yourdomain.com`) pointing to VPS.

## Step A: Clone and build
```bash
cd /opt
sudo git clone git@github.com:techmigosglobal/school-backend.git
cd school-backend
sudo go mod download
sudo go build -o school-backend main.go
```

## Step B: Configure environment
```bash
sudo cp .env.example .env
sudo nano .env
```
- Replace all placeholder secrets and endpoints.
- Keep `ENVIRONMENT=production`.
- Set `ALLOWED_ORIGINS` to exact frontend domains.

## Step C: Create systemd service (API)
Create `/etc/systemd/system/school-backend.service`:

```ini
[Unit]
Description=School Desk Backend API
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/school-backend
EnvironmentFile=/opt/school-backend/.env
ExecStart=/opt/school-backend/school-backend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable school-backend
sudo systemctl start school-backend
sudo systemctl status school-backend
```

## Step D: Nginx reverse proxy + TLS
Create `/etc/nginx/sites-available/school-backend`:

```nginx
server {
    listen 80;
    server_name api.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Enable site and reload:
```bash
sudo ln -s /etc/nginx/sites-available/school-backend /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

Enable TLS:
```bash
sudo certbot --nginx -d api.yourdomain.com
```

## Step E: Worker service (optional but recommended)
Create `/etc/systemd/system/school-backend-worker.service`:

```ini
[Unit]
Description=School Desk Backend Notification Worker
After=network.target

[Service]
Type=simple
User=www-data
WorkingDirectory=/opt/school-backend
EnvironmentFile=/opt/school-backend/.env
Environment=APP_MODE=worker
ExecStart=/opt/school-backend/school-backend
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Start worker:
```bash
sudo systemctl daemon-reload
sudo systemctl enable school-backend-worker
sudo systemctl start school-backend-worker
```

## Step F: Frontend production API target
Build Flutter with production API endpoint:

```bash
flutter build web \
  --release \
  --dart-define=APP_ENV=production \
  --dart-define=API_BASE_URL=https://api.yourdomain.com/api/v1
```

## 5) Verification checklist

## Backend health
```bash
curl -sS https://api.yourdomain.com/health
```
Expected:
```json
{"status":"healthy"}
```

## Root info
```bash
curl -sS https://api.yourdomain.com/
```
Expected: JSON response with service metadata.

## Logs
```bash
sudo journalctl -u school-backend -f
sudo journalctl -u school-backend-worker -f
```

## 6) Git/operations workflow used

For each backend change:
1. Implement in `school-backend`.
2. Run `go test ./...`.
3. Run frontend checks in app repo (`flutter analyze`, `flutter test`) to verify integration health.
4. Commit to `school-backend` git repo and push `main`.

## 7) Troubleshooting quick map

- Startup fails with config validation in production:
  - Check missing `JWT_SECRET`, `DATABASE_URL`, `REDIS_URL`, `REDIS_PASSWORD`, `ALLOWED_ORIGINS`.
- 401 on protected APIs:
  - Verify Bearer token and refresh flow.
- 403 cross-school access denied:
  - Token school scope and requested `school_id` mismatch.
- CORS failure from frontend:
  - Ensure exact origin is present in `ALLOWED_ORIGINS`.
- High latency or no cache/rate limiting/session revocation:
  - Verify Redis connectivity and password.
