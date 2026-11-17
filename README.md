# URL shortener

This project is implemented by Go + React. The React client shows that sentence in the header.

## Toolchain

- Backend: Go 1.27.4 and the `go` command. Module root: `backend/`.
- Client: Node.js 24, React 19, TypeScript, Vite 6. Module root: `frontend/`.
- Database: MySQL 8.0.
- Docker is optional. This tree has no `backend/Dockerfile` and no root `docker-compose.yml`.

## Run the backend

Dev defaults are in `specs/greenfield/TECH_STACK.md`. `APP_ENV=dev` listens on `0.0.0.0:8080`, uses `APP_BASE_URL=http://localhost:8080`, and applies `backend/sql/schema.sql` to MySQL on `localhost:3306` database `shortener_dev`. The lab `DB_PASSWORD` default is `Tiger`. Production must set `DB_PASSWORD` itself.

```powershell
cd backend
go run .
```

## Run the React client

The dev server is `http://localhost:5173`. Requests to `/api` are proxied to `http://localhost:8080`. The client sources live in `frontend/`.

```powershell
cd frontend
npm ci
npm run dev
```

`npm run build` typechecks and builds the client. `npm test` runs Vitest.

## Tests

From `backend/`:

```powershell
go test ./... -count=1 -covermode=atomic "-coverpkg=./..." "-coverprofile=coverage.out"
go tool cover "-func=coverage.out"
```

The gate requires every test to pass and `total:` statement coverage of at least 80%. The measured run recorded at 2026-11-17T20:22:23Z is 83.5% (44 tests, zero skipped).

## HTTP

- `POST /api/v1/shorten` creates a short URL and returns 201 JSON.
- `GET /short/{code}` redirects with 302.
- `GET /swagger-ui.html` and `GET /v3/api-docs` publish the create API.

## Where the specs live

- `specs/greenfield/` — product, design, toolchain, tests, and the stage graph
- `specs/brownfield/` — Swagger, legal paths, public access and HTML sanitization, then these docs
- `GATE2_APPROVAL.txt` — human signatures
- `SDLC_AUDIT_LOG.json` — lineage
- `SDLC_METRICS.json` — measured coverage and repair counts

## Limits

Codes are sequential Base62 of the MySQL id. The sequence starts at 56800235. The public-access layer is a demo: there is no login and no CSRF challenge. `stage1_epoch_ms` and `stage7_epoch_ms` are null. Stage 7 is open. `docker compose up --build` was run once from the workspace root and exited 1 with `no configuration file provided: not found`. No image was produced.
