# Greenfield scenario

**Intent:** build the URL shortener backend from specs. This file is an evaluation index. It does not generate code.

**Decomposition**

1. Requirements → `specs/greenfield/PRD.md` and `specs/contracts/openapi.yaml` (Gate 1: named APIs, Go 1.27.4, MySQL 8, Base62).
2. Architecture → `SYSTEM_DESIGN.md`, `TECH_STACK.md` (Gate 2: a human line `[APPROVED: …]` in `GATE2_APPROVAL.txt`).
3. Implementation in parallel, after approval (Gate 3 sync):
   - **3A** `backend/sql/schema.sql`
   - **3B** `encoding.Encode` / `encoding.Decode` + unit tests
   - **3C** `backend/go.mod`, `database/sql` repository, chi handlers, error response
4. Tests / compliance (Gate 4): from `backend/`, `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out`, then `go tool cover -func=coverage.out` with `total:` ≥ 80%, bound SQL, no `t.Skip`.

**Orchestration path:** Stages 1→2→HITL→3A/3B/3C→4, as in `specs/greenfield/ORCHESTRATION_MANIFEST.md`.

**Validation (when Stage 4 runs):** Base62 tests, service tests, cleanup tests, `web/url_shortener_api_test.go`. Commands: the two Gate 4 lines above, on Go 1.27.4, from `backend/`.

**Outputs present in the workspace:**

- `GATE2_APPROVAL.txt` contains `[APPROVED: yitin]`. Stage 3 is unlocked.
- Gate 3 files are present: `backend/sql/schema.sql`, `backend/go.mod`, `backend/go.sum`, and `backend/**/*.go` (config, Base62, `database/sql` repository, chi handlers, cleanup).
- Gate 4 measurement, `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out` from `backend/` recorded at 2026-11-17T20:22:23Z: `go tool cover -func` total statement coverage **83.5%**. 44 tests passed, with zero skipped. `SDLC_METRICS.json` records `line_coverage_ratio` `0.835`.
- `SDLC_AUDIT_LOG.json` and `SDLC_METRICS.json` exist. `stage1_epoch_ms`, `stage7_epoch_ms`, and `end_to_end_latency_ms` are null. Stage 7 is open.
- Stage 6 artifacts are absent: no `backend/Dockerfile` and no root `docker-compose.yml`. `docker compose up --build` from the workspace root exited 1 with `no configuration file provided: not found`. No image was produced.

Policy: `specs/greenfield/ORCHESTRATION_MANIFEST.md`.
