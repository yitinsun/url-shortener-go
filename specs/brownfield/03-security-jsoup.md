# 03 — Public access demo and HTML sanitization

Brownfield playbook. Run after `02-legal-paths.md`. This is a **demo**, not product login. No users, JWT, or login page.

The Java reference used Spring Security (`permitAll`, CSRF off) and jsoup 1.18.1. This module has neither. The same behavior is:

1. Public-access middleware that documents the routes as public, adds no `Authorization` challenge, and never returns **401**.
2. HTML-sanitize middleware that rejects HTML/script in `longUrl` using **bluemonday v1.0.27** (the HTML sanitizer on the allow-list for this playbook).

Both are outside `TECH_STACK.md`. Wait for a new human signature before editing `backend/go.mod`. bluemonday is the only new module. Do not add an auth framework.

On a finding, return the existing **400** JSON from the error type. The `message` is `longUrl contains disallowed HTML`. Do not invent a second error shape.

`LegalPaths` stays the allowlist and runs **first**. This middleware runs after it, so illegal URLs still get HTML 404.

## go.mod

```text
github.com/microcosm-cc/bluemonday v1.0.27
```

Do not add OAuth2 or JWT modules.

## Public access

`backend/web/public_access.go`:

- Do not require a CSRF token, so `POST /api/v1/shorten` from Swagger and curl still works.
- Do not add form login, HTTP basic, or a logout route.
- Documented routes (`/api/v1/shorten`, `/short/**`, Swagger, `/v3/api-docs/**`) are public.
- Unknown paths still reach the legal-path middleware (HTML 404, not 401).

Requiring authentication is a later re-plan, not this demo.

## Bluemonday

Buffer `POST` JSON so the handler can still read the body. Treat markup as unsafe. A plain `https://example.com/path` must pass.

Reject rather than silently clean. Apply only to `longUrl` on `POST /api/v1/shorten`.

Unsafe when `bluemonday.StrictPolicy().Sanitize(value)` is not equal to `value`, or when the fragment parses to one or more HTML elements. That is the same reject-or-keep decision the reference service made with jsoup’s empty safelist.

## Tests

- Existing API tests still pass (POST is not blocked).
- `longUrl` containing HTML or `<script>` → **400** JSON whose message is `longUrl contains disallowed HTML`.
- Sanitizer unit tests: clean URL vs `<script>` / `<img>`.

## Files

| File | Role |
|:-----|:-----|
| `backend/go.mod` | bluemonday v1.0.27 |
| `web/public_access.go` | public routes, no 401 |
| `web/html_sanitizer.go` | `LooksUnsafe` |
| `web/html_sanitize.go` | middleware after the allowlist |
| `apperror/error.go` | map the finding to 400 JSON |
