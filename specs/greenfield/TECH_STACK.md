# Technology Stack Profile: Go URL Shortener (chi & MySQL)

This document is the **only** allowed backend toolchain for v1. Direct module requirements not listed here are high-impact (see `ORCHESTRATION_MANIFEST.md` §5.1). Transitive modules that `go mod tidy` records in `go.sum` are not a new dependency decision.

**Explicitly out of v1 (do not add):** Redis, an auth or CSRF module, a rate-limiter, Safe Browsing clients, a migration tool (no goose, golang-migrate, atlas, or GORM AutoMigrate), swaggo, bluemonday, a metrics/actuator endpoint. Schema is applied by executing `backend/sql/schema.sql` as described below, plus the DDL in `SYSTEM_DESIGN.md`.

**Frontend:** Node.js 24 and React are specified in `specs/frontend/TECH_STACK.md`. They are not dependencies of `backend/go.mod`.

## 1. Core Tooling Versions

* **Language:** Go **1.27.4**. `go.mod` language line `go 1.27`, `toolchain go1.27.4`. Host `windows/amd64` on this machine.
* **HTTP:** chi **v5.5.0** on `net/http`
* **Build:** `go` command. Module path `url-shortener`. Module root `backend/` (`backend/go.mod`)
* **Database:** `database/sql` with `github.com/go-sql-driver/mysql` **v1.10.2**. The driver speaks MySQL 8 `caching_sha2_password` without cgo.
* **Test DB:** testcontainers-go **v0.45.0** and `modules/mysql` **v0.45.0**, image `mysql:8.0`. Tests must not use `shortener_dev` or `shortener_prod`.
* **Coverage:** `go test` cover profile and `go tool cover -func` (stdlib, not a module). **Minimum 80% statement coverage** on the `total:` line. Store that ratio in `SDLC_METRICS.json` as `line_coverage_ratio` (80% is `0.80`). Go reports statements; do not substitute a hand-counted percentage.

Gate 4, from `backend/`:

```text
go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

The second command fails the gate when `total:` is below **80.0%**.

## 2. Module allow-list (`backend/go.mod`)

```text
module url-shortener

go 1.27

toolchain go1.27.4

require (
    github.com/go-chi/chi/v5 v5.5.0
    github.com/go-sql-driver/mysql v1.10.2
)
```

Test-only imports (may appear in the same `require` block after `go mod tidy`; only `_test.go` files may import them):

```text
github.com/testcontainers/testcontainers-go v0.45.0
github.com/testcontainers/testcontainers-go/modules/mysql v0.45.0
```

Start image `mysql:8.0` with that module. Do not add `github.com/testcontainers/testcontainers-go/modules/compose` or other modules.

SQL is `database/sql` (`Query`, `QueryRow`, `Exec`, `Prepare`) with `?` placeholders and bound arguments. Do not interpolate values into SQL strings. Do not add an ORM.

`CGO_ENABLED=0` is the production build. The allow-listed driver does not need cgo.

## 3. Configuration

There is no `application.yml`. The process reads environment variables. Lab defaults below apply only when `APP_ENV=dev`. Production has **no** password default: a missing `DB_PASSWORD` in prod is a startup failure. Never commit a literal production password in Go source.

`APP_BASE_URL` builds `shortUrl` in API responses. It has no trailing slash.

| Variable | Dev default | Prod |
|:---------|:------------|:-----|
| `APP_ENV` | `dev` | `prod` |
| `APP_BASE_URL` | `http://localhost:8080` | required |
| `BIND_ADDR` | `0.0.0.0:8080` | `0.0.0.0:8080` |
| `DB_HOST` | `localhost` | required |
| `DB_PORT` | `3306` | `3306` |
| `DB_NAME` | `shortener_dev` | `shortener_prod` |
| `DB_USER` | `root` | required |
| `DB_PASSWORD` | `Tiger` (lab only) | required, no default |

Build the MySQL DSN from those parts:

```text
user:password@tcp(host:port)/dbname?parseTime=true&loc=UTC&charset=utf8mb4&collation=utf8mb4_unicode_ci
```

* **dev:** on startup, run `backend/sql/schema.sql`. Ignore “already exists” on restart. Do not `ALTER` the table afterward.
* **prod:** do not run `schema.sql`. Compose or a DBA applies it, including `AUTO_INCREMENT=56800235`.
* **test:** `APP_ENV=test`. Integration tests start MySQL 8.0 with testcontainers-go, create the schema from the same file, and build the chi router against that DSN. Do not point tests at a developer’s local `shortener_dev`.

`docker-compose.yml` (workspace root, Stage 6) must pass `DB_HOST`, `DB_NAME`, `DB_USER`, `DB_PASSWORD`, and `APP_BASE_URL` into the API service. Compose may set lab passwords in `environment:`. Go reads only the variables.

## 4. Development Constraints & Coding Standards

* **API types:** Request and response bodies are structs in `backend/web/dto.go` with explicit camelCase `json` tags. `ttlInDays` is `*int64` (omitted or JSON `null` means the 730-day default). Do not add a builder module.
* **Rows:** a plain struct in `backend/repo/url_mapping.go`. The repository is an interface so service tests can use an in-memory fake.
* **Database access:** the `database/sql` implementation of that interface is the only production database code. SQL built with `fmt.Sprintf`, `+`, or string interpolation is banned. Bound `?` arguments are required, including the batched expiry delete.
* **Scheduling:** a goroutine and `time.NewTicker` started from `main`. No extra scheduler module.
* **Logging:** `log/slog`. No extra logging module.
* **Layout when the module is generated:**

```text
backend/
├── go.mod
├── go.sum
├── sql/schema.sql
├── main.go
├── config/config.go
├── encoding/base62.go
├── apperror/error.go
├── repo/url_mapping.go
├── service/url_shortener.go
├── schedule/cleanup.go
└── web/
    ├── dto.go
    ├── router.go
    ├── shorten.go
    └── redirect.go
```
