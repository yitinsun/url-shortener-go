---
name: url-shortener-sdlc
description: >-
  Runs the gated spec-first Go SDLC for this repo (PRD, design, stack,
  tests, Gate 2 HITL, go test coverage, audit log). Use when implementing the URL
  shortener backend from specs/greenfield, or when the user mentions AGENTS.md,
  Gate 2, or the backend SDLC.
disable-model-invocation: true
---

# URL shortener backend SDLC

Cursor loads this file when the skill is attached. Product behavior lives in `specs/greenfield/`. Do not invent APIs, Go module requirements, or coverage.

## Instructions

1. Read `AGENTS.md`.
2. Read `specs/greenfield/INSTRUCTIONS.md` and `specs/greenfield/ORCHESTRATION_MANIFEST.md`.
3. Read `PRD.md`, `SYSTEM_DESIGN.md`, `TECH_STACK.md`, and `TESTING_PLAN.md` in `specs/greenfield/` for their roles only.
4. Keep `specs/contracts/openapi.yaml` aligned with `PRD.md`.
5. Honor Gate 2: no `backend/go.mod`, `backend/go.sum`, `backend/**/*.go`, or `backend/sql/**` until `GATE2_APPROVAL.txt` contains `[APPROVED: …]`.
6. After approval, run 3A schema, 3B Base62, and 3C the Go module together, then from `backend/` run `go test ./... -count=1 -covermode=atomic -coverpkg=./... -coverprofile=coverage.out` and `go tool cover -func=coverage.out` (`total:` ≥ 80%).
7. Brownfield playbooks in `specs/brownfield/` start only after Gate 4. `specs/scenarios/` records evaluation; it does not generate code.
8. Do not create `frontend/src/**` in this skill. The React specs live in `specs/frontend/` and are a separate pass.
