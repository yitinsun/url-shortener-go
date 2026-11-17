# Automated Test Verification Suite Plan

Mandatory tests and the agent loop for Gate 4. From `backend/`, the suite must pass with **100%** test success and **statement coverage >= 80%** on the `total:` line of `go tool cover -func`. Never use `t.Skip` to drop a mandatory case. Do not pass `-short`.

Coverage is enforced by the commands named in `TECH_STACK.md`. Do not invent a percentage in `specs/scenarios/01-greenfield.md` or `SDLC_METRICS.json`. Copy `line_coverage_ratio` from that `total:` line after a real run (`80.0%` → `0.80`).

```text
go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## 1. Unit Testing Matrix

### 1.1 `encoding/base62_test.go`

No chi router and no database. Assert:

* **Pivot:** `Encode(56800235)` is alphanumeric `[0-9A-Za-z]+` and length **>= 5** (and **<= 7** for this pivot).
* **Alphabet:** encoded forms contain no `+` or `/`.
* **Max:** `Encode(uint64(math.MaxInt64))` completes without overflow; `Decode(Encode(uint64(math.MaxInt64)))` returns that same value. `BIGINT` ids fit in `uint64` for this range; do not use a wrapping 32-bit type.
* **Round-trip:** for a set of random positive ids, `Decode(Encode(id)) == id`.

### 1.2 `service/url_shortener_test.go`

Use the repository interface with an in-memory fake. No MySQL.

| Case | Setup | Expect |
|:-----|:------|:-------|
| Missing resource | Repo empty for short code | not-found error |
| Expired mapping | `expires_at` in the past | not-found error |
| Idempotent create | Active row for same `longUrl` | No second insert; return existing code |
| Default TTL | `ttlInDays` absent | `expires_at` ≈ now + **730** days |
| Self-loop | `longUrl` host equals `APP_BASE_URL` host | validation error (400 path) |
| Oversize URL | length > 2048 | validation error |

### 1.3 `schedule/cleanup_test.go`

The scheduled delete uses a fixed SQL string with `?` placeholders. Assert a cutoff instant is bound and the statement is not built with `fmt.Sprintf` or concatenation.

## 2. HTTP Integration Testing (`web/url_shortener_api_test.go`)

Build the real chi router from `web.NewRouter` and call it with `httptest`. The pool is MySQL 8.0 from testcontainers-go, schema from `backend/sql/schema.sql`, including `AUTO_INCREMENT=56800235`. Seed with the repository or with `POST` — not a fake pool.

The error type must serialize the error JSON from `PRD.md` / `SYSTEM_DESIGN.md` for `POST /api/v1/shorten`. Redirect misses use HTML.

### 2.1 `POST /api/v1/shorten`

| Case | Assert |
|:-----|:-------|
| Valid HTTP(S) URL | **201**; body has `shortUrl`, `shortCode`, `expiresAt`; `shortCode` length 5–7; `shortUrl` starts with `{APP_BASE_URL}/short/` |
| Second POST of the same `longUrl` | **201** with the identical `shortCode` |
| `"longUrl": "not-a-valid-url"` | **400** and error JSON `status`/`message` |
| Self-loop to `APP_BASE_URL` | **400** |

### 2.2 `GET /short/{shortCode}`

| Case | Setup | Assert |
|:-----|:------|:-------|
| Active mapping | Persist unexpired row (or create via POST) | **302** and `Location` equals `longUrl` |
| Unknown code | GET a code that was never inserted | **404** friendly HTML (not JSON, not chi’s or `net/http`’s plain-text default) |
| Expired mapping | Persist row with `expires_at` in the past | **404** friendly HTML (HTTP layer, not only service unit) |

## 3. Automated Agent Workflow Cycle

The two commands in the header are the Gate 4 command. Do not pass `-short` or `t.Skip`. The Docker **image build** (`go build`) does not run tests; skip them in the image only when Gate 4 has already passed on that source tree.

```
[ Compile ] ──> [ go test -coverprofile + go tool cover -func ]
                      │
         ┌────────────┴────────────┐
         ▼                         ▼
[ All tests pass + coverage ]   [ Failures or total < 80% ]
         │                         │
         ▼                         ▼
[ Gate 5: container phase ]   [ Read compiler / test output ]
         (Stage 6)                 │
                                   ▼
                          [ Patch failing package ]
                                   │
                                   ▼
                          [ Retry; max 3 times ]
                                   │
                    exhausted ──> [ Safe-Stop / rollback to Gate 2 ]
```

This loop **is** the Bounded Retry in `ORCHESTRATION_MANIFEST.md`. Passing tests do **not** rewrite packages. Gate 4C (secrets / parameterized SQL) is a compliance review of the same tree, not an extra test binary — still must pass before Stage 6.
