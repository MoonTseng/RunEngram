# Exploration Capsules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Turn RunEngram from task visualization into a learning loop that freezes task-start context, preserves verified exploration, recalls it for later work, and measures whether reuse helped.

**Architecture:** Keep task orchestration unchanged. Add a capability layer for immutable context snapshots, exploration capsules, and reuse outcomes; a service layer for deterministic capsule matching and honest learning metrics; CLI/skill adapters for Codex; and a read-focused Knowledge view. SQLite remains the only persistence dependency. Snapshot creation is explicit after claim so claim semantics stay atomic and existing clients do not break.

**Tech Stack:** Go 1.24, modernc SQLite, Hertz, Cobra, React 19, TypeScript, TanStack Query, Tailwind CSS, Vitest.

---

## Product contract

- A claimed task can request one immutable context snapshot. Repeated reads return identical task/capsule content.
- Snapshot suggestions only include active capsules from the same project.
- Matching is deterministic and explainable: normalized token overlap across task title, description, labels and capsule title, summary, scope, labels, fingerprints.
- Capsule evidence is mandatory. “Knowledge” without evidence cannot enter reusable memory.
- One capsule/task pair has one current outcome: `used`, `helpful`, `rejected`, or `stale`.
- Metrics expose counts and helpful rate only. No invented “hours saved”.
- Existing task APIs, state machine, local auth, and one-binary deployment remain compatible.

### Task 1: Persist learning assets

**Files:**
- Create: `server/migrations/0014_learning_assets.sql`
- Create: `server/internal/store/schema/0014_learning_assets.sql`
- Modify: `server/api/model/model.go`
- Modify: `server/internal/store/store.go`
- Modify: `server/internal/store/store_test.go`

- [ ] Write store tests first:
  - capsule create/list/get/update status;
  - same-project source-task validation;
  - snapshot create-once immutability;
  - usage upsert for `(capsule_id, task_id)`;
  - migration version advances to 14.
- [ ] Run `(cd server && GOPROXY=https://goproxy.cn,direct go test ./internal/store -run 'Capsule|Snapshot|Usage|MigrationsRunOnce' -count=1)` and confirm failure.
- [ ] Add migration to both canonical and embedded schema copies:

```sql
CREATE TABLE exploration_capsules (
  id TEXT PRIMARY KEY,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  source_task_id TEXT NOT NULL DEFAULT '',
  title TEXT NOT NULL,
  summary TEXT NOT NULL,
  scope TEXT NOT NULL DEFAULT '',
  evidence TEXT NOT NULL,
  labels TEXT NOT NULL DEFAULT '[]',
  fingerprints TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL DEFAULT 'active'
    CHECK (status IN ('active','stale','archived')),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE INDEX idx_capsules_project_status
  ON exploration_capsules(project_id, status, updated_at DESC);

CREATE TABLE context_snapshots (
  id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL UNIQUE,
  project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  payload TEXT NOT NULL,
  created_at INTEGER NOT NULL
);

CREATE TABLE capsule_usages (
  id TEXT PRIMARY KEY,
  capsule_id TEXT NOT NULL REFERENCES exploration_capsules(id) ON DELETE CASCADE,
  task_id TEXT NOT NULL,
  outcome TEXT NOT NULL CHECK (outcome IN ('used','helpful','rejected','stale')),
  notes TEXT NOT NULL DEFAULT '',
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  UNIQUE(capsule_id, task_id)
);
CREATE INDEX idx_capsule_usages_task ON capsule_usages(task_id);
```

- [ ] Add model types:

```go
type CapsuleStatus string
type CapsuleOutcome string

type ExplorationCapsule struct {
    ID, ProjectID, SourceTaskID string
    Title, Summary, Scope, Evidence string
    Labels, Fingerprints []string
    Status CapsuleStatus
    UseCount, HelpfulCount, RejectedCount int
    CreatedAt, UpdatedAt int64
}

type ContextSnapshot struct {
    ID, TaskID, ProjectID string
    Task Task
    SuggestedCapsules []ExplorationCapsule
    CreatedAt int64
}

type CapsuleUsage struct {
    ID, CapsuleID, TaskID string
    Outcome CapsuleOutcome
    Notes string
    CreatedAt, UpdatedAt int64
}
```

- [ ] Add migration embed/version 14 and store CRUD methods. Store snapshot as complete JSON payload, not references, preserving immutability.
- [ ] Run focused tests; then `(cd server && GOPROXY=https://goproxy.cn,direct go test ./internal/store -count=1)`.
- [ ] Commit: `feat(server): persist exploration capsules`

### Task 2: Add matching, context, and metrics services

**Files:**
- Create: `server/internal/service/learning.go`
- Create: `server/internal/service/learning_test.go`
- Modify: `server/internal/service/service.go`

- [ ] Write service tests first:
  - evidence/title/summary validation;
  - project-scoped capsule matching and stable rank;
  - existing snapshot returned unchanged after new capsule appears;
  - usage task/capsule project mismatch rejected;
  - helpful rate denominator is `helpful + rejected`, not all usages.
- [ ] Run `(cd server && GOPROXY=https://goproxy.cn,direct go test ./internal/service -run 'Capsule|Context|Learning' -count=1)` and confirm failure.
- [ ] Implement normalized matcher:

```go
score = 5*labelOverlap +
        4*fingerprintOverlap +
        3*titleOverlap +
        2*summaryOverlap +
        scopeOverlap
```

Tokenize lower-case Unicode letter/digit runs, deduplicate terms, reject zero-score results, order by score DESC then updated_at DESC then id ASC, limit 5.

- [ ] Implement:
  - `CreateCapsule`, `ListCapsules`, `UpdateCapsule`;
  - `GetOrCreateTaskContext`;
  - `RecordCapsuleUsage`;
  - `GetLearningMetrics`.
- [ ] Run focused and full service tests.
- [ ] Commit: `feat(server): add verified learning loop`

### Task 3: Expose REST and CLI contracts

**Files:**
- Modify: `server/api/handler/handler.go`
- Modify: `server/tests/e2e_test.go`
- Modify: `cli/client/client.go`
- Modify: `cli/client/client_test.go`
- Create: `cli/cmd/capsule.go`
- Create: `cli/cmd/capsule_test.go`
- Modify: `cli/cmd/task.go`
- Modify: `cli/cmd/task_test.go`

- [ ] Write API/CLI tests first for:

```text
GET    /api/v1/tasks/:id/context
GET    /api/v1/projects/:project/capsules
POST   /api/v1/projects/:project/capsules
PATCH  /api/v1/capsules/:id
POST   /api/v1/capsules/:id/usages
GET    /api/v1/projects/:project/learning-metrics
```

- [ ] Add CLI commands:

```text
taskline task context <task-id>
taskline capsule list [--query text] [--status active]
taskline capsule create --source-task ID --title TEXT --summary TEXT \
  --scope TEXT --evidence-file PATH [--label VALUE] [--fingerprint VALUE]
taskline capsule use <capsule-id> --task ID --outcome helpful [--notes TEXT]
taskline capsule archive <capsule-id>
```

All non-TTY output uses `internal/output`; evidence file content, not path, goes to API.

- [ ] Run focused API/CLI tests, then full server and CLI suites.
- [ ] Commit: `feat(cli): expose task learning workflow`

### Task 4: Make Codex use memory by default

**Files:**
- Modify: `skills/taskline-management/SKILL.md`
- Modify: `scripts/test-skill.sh`
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `PRODUCT.md`
- Modify: `使用说明.md`

- [ ] Add skill contract:
  - after claim/resume, run `taskline task context "$TASK_ID"` before code exploration;
  - cite capsule IDs actually used;
  - record `helpful`, `rejected`, or `stale`;
  - before completion, create capsule only for verified reusable findings;
  - never promote guesses, task-specific prose, secrets, or raw chat.
- [ ] Add smoke assertions for new commands and mandatory workflow terms.
- [ ] Document a generic URL-service migration example: snapshot before analysis, then capture verified call-site and migration evidence after tests.
- [ ] Explain product distinction: Codex executes one task; RunEngram preserves verified cross-task engineering memory and measures reuse.
- [ ] Run `./scripts/test-skill.sh`.
- [ ] Commit: `docs: define evidence-backed memory workflow`

### Task 5: Add Knowledge view

**Files:**
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/hooks/queries.ts`
- Modify: `web/src/lib/i18n.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/App.test.tsx`
- Create: `web/src/components/KnowledgeView.tsx`
- Create: `web/src/components/KnowledgeView.test.tsx`

- [ ] Write component/API tests first:
  - third navigation tab renders;
  - metrics display honest counts and helpful rate;
  - capsule evidence, scope, fingerprints, source task and status visible;
  - query/status filters work;
  - empty state explains how Codex creates first capsule;
  - Chinese and English labels exist.
- [ ] Add mirrored TypeScript types and API calls.
- [ ] Add `knowledge` to `View`, URL parser, toggle, workspace rendering.
- [ ] Build readable, responsive view with minimum body text `14px`; avoid tiny helper copy.
- [ ] Run `(cd web && pnpm lint && pnpm test && pnpm build)`.
- [ ] Commit: `feat(web): surface reusable engineering memory`

### Task 6: End-to-end verification and release

**Files:**
- Modify only if verification exposes defects.

- [ ] Run:

```bash
(cd server && GOPROXY=https://goproxy.cn,direct go test ./...)
(cd cli && GOPROXY=https://goproxy.cn,direct go test ./...)
(cd web && pnpm lint && pnpm test && pnpm build)
./scripts/test-skill.sh
./scripts/build.sh
```

- [ ] Start isolated server with temporary DB/docs and smoke:
  - create project/task;
  - claim;
  - create capsule with evidence;
  - fetch task context twice and compare;
  - record helpful outcome;
  - verify metrics;
  - verify `/` serves bundled UI.
- [ ] Review diff for secrets, internal URLs, CamScanner proprietary details, GitLab references, and accidental generated/runtime files.
- [ ] Merge `codex/exploration-capsules` into `main`, push `origin/main`, reinstall/restart local RunEngram, and verify `http://127.0.0.1:8787/healthz`.
