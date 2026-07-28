# Automatic Learning Loop v1 — Test Report

Date: 2026-07-28

## Scope

- learning-note persistence and migration;
- capture, list, promote, and reject service rules;
- authenticated REST and CLI contracts;
- agent skill capture and resolution protocol;
- bilingual Knowledge view and aggregate metrics;
- full release build.

## Results

| Gate | Result |
|---|---|
| `cd server && go test ./... -count=1` | Passed, including real-server e2e |
| `cd cli && go test ./... -count=1` | Passed |
| `cd web && pnpm lint` | Passed |
| `cd web && pnpm exec vitest run --maxWorkers=1` | Passed, 140 tests |
| `cd web && pnpm build` | Passed |
| `./scripts/test-skill.sh` | Passed |
| `./scripts/build.sh` | Passed |
| local v15 migration and `/healthz` | Passed |
| three-task pending/promoted recall smoke | Passed |
| bilingual Knowledge view browser smoke | Passed, zero console errors |

## Verified invariants

- only live source-task owner can capture or resolve a learning note;
- blank promotion evidence is rejected;
- promotion updates note and creates one capsule in one transaction;
- repeated promotion is idempotent and returns the existing capsule;
- rejected and pending notes never enter recall;
- task history records capture, promotion, and rejection;
- HTTP shapes stay aligned across server, CLI, and web fixtures;
- Knowledge view exposes pending, promoted, rejected, and promotion-rate
  metrics without claiming causal time saved.

## End-to-end recall smoke

Project: `automatic-learning-smoke-20260728`

1. Source task captured a pending human-correction note: use
   `one-flow` `notion-to-prd` before PRD analysis.
2. Second task generated a context snapshot before promotion and recalled zero
   capsules.
3. Note was promoted with scoped Markdown evidence.
4. Third matching task generated a new context snapshot and recalled the
   promoted capsule exactly once.
5. Metrics reported one learning note, one promoted note, zero pending notes,
   one active capsule, and a 100% promotion rate.

Browser verification confirmed both Chinese and English candidate labels,
promoted candidate detail, active capsule evidence, scope, fingerprints, and
source-task provenance. Browser console contained no warnings or errors.

## Known non-blocking warning

Vite reports production chunks above 500 kB. Existing lazy-loaded Markdown
editor chunk remains the main contributor. Build succeeds; bundle splitting is
separate performance work.
