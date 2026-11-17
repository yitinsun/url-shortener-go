# Brownfield scenario

**Intent:** change the running backend after Gate 4 without skipping the dependency allow-list, tests, or error pages. This file is an evaluation index. The playbooks generate the changes.

**Decomposition:** one playbook → impact → code and tests → coverage gate from `backend/`. A new direct `go.mod` requirement or a table change waits for a human signature.

| Order | Change | Playbook | Packages | API / data |
|:------|:-------|:---------|:---------|:-----------|
| 1 | OpenAPI UI | `01-add-swagger.md` | `backend/go.mod`, swag comments, DTO schemas | `/swagger-ui.html`, `/v3/api-docs` |
| 2 | Allowlist and HTML 404 for illegal paths | `02-legal-paths.md` | legal-path middleware, `LegalPaths`, friendly HTML | Fence around the greenfield routes. Redirect stays `GET /short/{code}` |
| 3 | Public access + HTML sanitizer | `03-security-jsoup.md` | public-access middleware, HTML-sanitize middleware, bluemonday v1.0.27 | 400 JSON if `longUrl` contains HTML. Routes stay public (no 401) |
| 4 | Docs and measured metrics | `04-refine.md` | README, audit log, scenario outputs | No schema |

**Orchestration path:** post–Gate 4, one playbook at a time. Greenfield already specifies `GET /short/{shortCode}`. Do not reopen that path as a second design.

**Validation:** after each playbook, the greenfield API tests still pass, plus the cases named in that playbook. Command remains `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` (`total:` ≥ 80%).

**Outputs in the workspace:**

- Swagger is applied: `github.com/swaggo/swag` v1.16.6 and `github.com/swaggo/http-swagger/v2` v2.0.2 are in `backend/go.mod`. Document routes are `GET /swagger-ui.html`, `GET /swagger-ui/**`, `GET /v3/api-docs`, and `GET /v3/api-docs.yaml`. `GET /short/{shortCode}` stays out of the OpenAPI document. `OPENAPI_ENABLED` defaults on outside prod.
- Legal paths are applied: `backend/web/legal_path.go` (`LegalPaths`, `LegalPath`). Illegal URLs, including `GET /api/v1/shorten`, return the friendly HTML 404. Redirect stays `GET /short/{code}`.
- Public access and the HTML sanitizer are applied. `AllowPublic` adds no authentication challenge. `HtmlSanitize` runs after `LegalPath` and rejects HTML in `longUrl` with **400** JSON, message `longUrl contains disallowed HTML`. The module is `github.com/microcosm-cc/bluemonday` v1.0.27. A query `&` stays allowed.
- Docs files that exist: `README.md` describes the Go backend, the React client commands, the Gate 4 coverage command, and the public-access demo. `SDLC_AUDIT_LOG.json` and `SDLC_METRICS.json` exist. `line_coverage_ratio` is `0.835` from the 2026-11-17T20:22:23Z `go tool cover -func` run. `stage1_epoch_ms`, `stage7_epoch_ms`, and `end_to_end_latency_ms` are null. `retry_rollback_frequency` is `0` and `mttr_ms` is null. Stage 7 is open.
- Client sources are in `frontend/`. The client calls `POST /api/v1/shorten` only. `npm test` on Node.js 24 passed 10 tests. `npm run build` succeeded. No coverage percentage is defined for the client.
- No schema change. No actuator. No click-count column.
- No container image. `backend/Dockerfile` and root `docker-compose.yml` are absent. `docker compose up --build` from the workspace root exited 1 with `no configuration file provided: not found`. Stage 7 is open.
