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

## Known non-blocking warning

Vite reports production chunks above 500 kB. Existing lazy-loaded Markdown
editor chunk remains the main contributor. Build succeeds; bundle splitting is
separate performance work.
