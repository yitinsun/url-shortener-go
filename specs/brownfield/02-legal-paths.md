# 02 — Legal paths and friendly HTML

Brownfield playbook. Run after `01-add-swagger.md`. Greenfield already specifies `GET /short/{shortCode}`, `shortUrl = {APP_BASE_URL}/short/{shortCode}`, and HTML **404** for a missing or expired code. This playbook adds the allowlist fence and returns that same HTML page for every other URL. It does not rename spec folders and it does not change the redirect path.

No new Go module requirements. No auth middleware (that is `03-security-jsoup.md`). No schema change.

## Legal URL allowlist

Host is `http://localhost:8080` in local `dev` / `test`.

| Legal pattern | Methods | Notes |
|:--------------|:--------|:------|
| `/api/v1/shorten` | **POST** | JSON create. Other methods are illegal. |
| `/short/{shortCode}` | **GET** | `{shortCode}` is Base62 `[0-9A-Za-z]{1,10}`. |
| `/swagger-ui.html` | GET | From playbook 01. |
| `/swagger-ui/**` | GET | UI assets. |
| `/v3/api-docs` | GET | OpenAPI JSON. |
| `/v3/api-docs.yaml` | GET | OpenAPI YAML. |
| `/v3/api-docs/**` | GET | OpenAPI extras. |

**Illegal examples:** `/`, `/index.html`, `/foo`, `/b9K2x1` (no `/short/` prefix), `GET /api/v1/shorten`, `/short/` with no code.

There is no actuator endpoint. Do not allowlist `/actuator/**` or `/metrics`.

Implement `LegalPaths` plus chi middleware (`func(http.Handler) http.Handler`) on the router, outside the handlers:

1. Legal pattern → call the next handler.
2. Anything else → **404** friendly HTML. Do not fall through to a handler that might 302 or return JSON.

Package file: `backend/web/legal_path.go`.

## HTML vs JSON

| Request | Body |
|:--------|:-----|
| `POST /api/v1/shorten` 400 / 503 | JSON `ErrorResponse` |
| `GET /short/{shortCode}` missing or expired | HTML 404 |
| Illegal path | HTML 404 |

chi’s and `net/http`’s default plain-text 404 must not be reachable. The page must include a visible title, a short explanation, the escaped request path, and links to `POST /api/v1/shorten`, `/swagger-ui.html`, and `/short/{code}`. It must not show a stack trace. Escape `<`, `&`, and `"`.

`ERR_CONNECTION_REFUSED` means nothing is listening on 8080. That is not this 404.

## Tests

Keep the greenfield cases. Add:

| Case | Expect |
|:-----|:-------|
| `GET /short/{activeCode}` | 302 + `Location` |
| `GET /short/unknown` | 404, `text/html`, friendly heading, no framework default body |
| `GET /b9K2x1` | 404 HTML |
| `GET /` and `GET /nope` | 404 HTML |
| `GET /api/v1/shorten` | 404 HTML |
| `POST /api/v1/shorten` valid | 201 JSON |
| `GET /v3/api-docs` | 200 JSON containing `/api/v1/shorten` and not a bare `/{shortCode}` path |

From `backend/`, `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` must stay green (`total:` ≥ 80%). No `t.Skip`.
