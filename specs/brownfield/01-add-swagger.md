# 01 — Swagger UI (swaggo)

Brownfield playbook. Run only after Gate 4 is green. Adding these modules is **high impact**: they are not in `specs/greenfield/TECH_STACK.md`. Wait for a new human signature before editing `backend/go.mod`.

Swagger publishes the HTTP API that already exists. It does not change status codes, the schema, or redirect behavior. The written contract stays `specs/contracts/openapi.yaml`.

**Modules:** `github.com/swaggo/swag` **v1.16.6** and `github.com/swaggo/http-swagger/v2` **v2.0.2**. Serve the same paths the legal-path playbook allowlists.

| Item | Value |
|:-----|:------|
| Module | `backend/` (`url-shortener`) |
| chi | v5.5.0 |
| Go | 1.27.4 |
| JSON API | `POST /api/v1/shorten` |
| Redirect | `GET /short/{shortCode}` → **302** (omit from the document) |

After enablement: `http://localhost:8080/swagger-ui.html` and `http://localhost:8080/v3/api-docs`.

## 1. go.mod

```text
github.com/swaggo/swag v1.16.6
github.com/swaggo/http-swagger/v2 v2.0.2
```

Generate the docs package from `backend/` with `swag init -g main.go -o web/docs`. Check with `go list -m github.com/swaggo/swag github.com/swaggo/http-swagger/v2`.

## 2. Paths

Mount Swagger UI at `/swagger-ui.html` (and the assets it requests under `/swagger-ui/`). The library’s default `/swagger/index.html` is not the public path. Mount the generated document at:

* `GET /v3/api-docs` — OpenAPI JSON
* `GET /v3/api-docs.yaml` — OpenAPI YAML

Both are off unless enabled. Dev default is on. Prod default is off unless the env var is set:

| Variable | Dev default | Prod default |
|:---------|:------------|:-------------|
| `OPENAPI_ENABLED` | `true` | `false` |

The document includes only `/api/v1/**`. Do not register `GET /short/{shortCode}` on the OpenAPI spec.

## 3. Document metadata

swag comments set title **URL Shortener API**, version `0.1.0`, and a server URL from `APP_BASE_URL`. The value is documentation only.

Do not add a Bearer/JWT security scheme. The create API stays public.

## 4. Types

Annotate `ShortenRequest`, `ShortenResponse`, and `ErrorResponse`. Comments must not change paths, status codes, or validation. Document **201**, **400**, and **503** on `POST /api/v1/shorten` as in `openapi.yaml`.

## 5. What this playbook does not do

- No auth middleware in this change (`03-security-jsoup.md` is separate).
- No schema change.
- No replacement for `web/url_shortener_api_test.go`.

## 6. Smoke check

1. Start the backend on port 8080.
2. `GET /swagger-ui.html` shows **URL Shortener API** `0.1.0`.
3. The catalog lists `POST /api/v1/shorten` and omits `GET /short/{shortCode}`.
4. Try-it-out with a valid `https://…` URL returns **201**.
5. `GET /v3/api-docs` is OpenAPI 3 JSON.
6. From `backend/`, `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` stay green (`total:` ≥ 80%).
