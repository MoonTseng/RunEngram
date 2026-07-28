# Spec: Docs Task Type

## Context

Taskline currently supports two task types: `feature` and `bug`. Documentation
maintenance is common enough that it should be a first-class task type instead
of being modeled as a feature with labels or title conventions.

## Goals

- Add `docs` as a valid task type across the server, CLI, web UI, and agent
  skill contract.
- Preserve the existing default task type as `feature`.
- Allow old databases to accept `docs` through a migration that updates the
  SQLite `tasks.type` CHECK constraint.
- Keep frontend display compact and scannable while making docs tasks visually
  distinct from features and bugs.

## Non-Goals

- Do not add new workflow states.
- Do not change runnable-task ordering or dependency behavior by type.
- Do not split docs tasks into a separate resource; they remain normal tasks.
- Do not introduce a broader taxonomy system in this change.

## Required Changes

- Server model: add `TaskTypeDocs` and include it in `TaskType.Valid`.
- Service/store errors: mention `feature`, `bug`, and `docs`.
- Database:
  - Add a new migration to rebuild `tasks` with `type IN ('feature','bug','docs')`.
  - Add the same SQL under `server/internal/store/schema/`.
  - Keep historical migration files unchanged.
- Tests:
  - Server store/service coverage for creating and updating docs tasks.
  - E2E coverage that POST/PATCH with `type: "docs"` round-trips.
  - CLI command/help tests that accept and document `docs`.
  - Web tests for API type union use, editor dropdown, and task-card styling.
- Web:
  - Extend `TaskType` union to include `docs`.
  - Add `docs` to the task type select.
  - Give docs tasks a distinct card accent.
- Docs/skills:
  - Update README examples or docs where task types are described.
  - Update `ARCHITECTURE.md`, `PRODUCT.md`, `AGENTS.md`, and
    `skills/taskline-management/SKILL.md` references from two types to three.

## Acceptance Criteria

- `taskline task create --project <project> --title <title> --type docs` works.
- `taskline task update <id> --type docs` works.
- REST create/update task endpoints accept and return `type: "docs"`.
- Existing DBs migrate without losing tasks, labels, dependencies, docs, links,
  or image metadata.
- The web task editor offers `docs` in the Type dropdown and renders docs task
  cards with a distinct visual accent.
- Full verification passes:
  - `cd server && mise exec -- go test ./...`
  - `cd cli && mise exec -- go test ./...`
  - `cd web && mise exec -- pnpm lint && mise exec -- pnpm test -- --run && mise exec -- pnpm build`
  - Browser smoke against the rebuilt local server.
