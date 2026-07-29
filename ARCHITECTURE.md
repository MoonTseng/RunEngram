# Architecture

How RunEngram's current task execution kernel is wired together. Existing
binary, package, and environment names retain `taskline` for alpha
compatibility. For the *why* see `PRODUCT.md`; for build, test, and
contribution mechanics see `AGENTS.md`.

## Components

```
   ┌──────────────────┐         HTTP /api/v1/*         ┌──────────────────┐
   │   taskline CLI   │  ────────────────────────────▶ │ taskline-server  │
   │  (cobra, JSON-   │ ◀────────────────────────────  │  (Hertz + SQLite)│
   │   first output)  │            JSON                │                  │
   └──────────────────┘                                │  ┌────────────┐  │
                                                       │  │ embedded   │  │
   ┌──────────────────┐         HTTP /api/v1/*         │  │ React UI   │  │
   │   Browser (UI)   │ ◀────────────────────────────▶ │  │ (go:embed) │  │
   │  React + Vite    │       static + REST            │  └────────────┘  │
   └──────────────────┘                                └──────────────────┘
                                                                │
                                                                ▼
                                                       ┌──────────────────┐
                                                       │  ./data/         │
                                                       │   ├ taskline.db  │
                                                       │   ├ images/<id>/ │
                                                       │   └ docs/<id>/   │
                                                       └──────────────────┘
```

One binary (`taskline-server`) serves both the REST API and the React
SPA. SQLite is one file on disk; image attachments live alongside it as
plain files keyed by task id; task docs are Markdown files stored in the
configured docs directory with only file references kept in SQLite.

## Server layering

`server/` is a single Go module with a strict downward-only import
graph:

```
  cmd/taskline-server/         ← process entrypoint, slog, config wiring
       │
       ▼
  api/handler/                 ← Hertz routes, JSON encode/decode, CORS,
       │                         SPA fallback, status-code mapping
       ▼
  internal/service/            ← name resolution (id-or-name), state-machine
       │                         validation, runnable filter orchestration
       ▼
  internal/store/              ← SQLite. CRUD, dep DAG, cycle check.
       │                         Returns ErrNotFound / ErrConflict sentinels.
       ▼
  api/model/                   ← Project, Task, TaskState, TaskType.
                                 Imported by every layer; imports nothing.
```

`internal/config/` is a sibling of service/store: it's loaded by `cmd/`
once and passed through to the handler (for `ImagesDir` and `DocsDir`).

### Why the split

- The handler layer never touches SQL. It maps HTTP ↔ service calls and
  errors ↔ statuses, nothing else.
- The service layer never touches HTTP. It owns invariants (state
  transitions, project resolution by id-or-name) and calls the store.
- The store layer is the only place that knows about SQLite. It returns
  sentinel errors so the handler can map them to status codes without
  string matching.

## Data model

```sql
projects(id, name UNIQUE, description, created_at, updated_at)
tasks   (id, project_id → projects.id, title, description,
         type ∈ {feature,bug,docs},
         state ∈ {pending,start,spec,dev,test,review,done}, priority,
         labels JSON string array,
         owner, claimed_at, lease_expires_at, completed_at,
         created_at, updated_at)
task_deps   (task_id → tasks.id, depends_on_task_id → tasks.id,
             PRIMARY KEY(task_id, depends_on_task_id),
             CHECK(task_id ≠ depends_on_task_id))
task_images (id, task_id → tasks.id, filename, mime_type,
             size_bytes, storage_path, uploaded_at)
task_docs   (id, task_id → tasks.id, title, storage_path,
             created_at, updated_at)
task_links  (id, task_id → tasks.id, url, label, created_at)
task_events (id, task_id, actor, action, summary, details JSON, created_at)
learning_notes
            (id, project_id → projects.id, source_task_id,
             kind ∈ {human-correction,agent-recovery},
             trigger, guidance, scope, labels JSON, fingerprints JSON,
             producer, status ∈ {pending,promoted,rejected},
             evidence, capsule_id, rejection_reason,
             created_at, updated_at, resolved_at)
agent_runs  (id, task_id → tasks.id, project_id → projects.id,
             agent_name,
             agent_tool ∈ {codex,claude-code,pi,other},
             workflow_key (single-loop, engineering-flow, or adapter slug),
             workflow_version,
             status ∈ {running,blocked,completed,failed},
             summary, next_step,
             started_at, updated_at, completed_at)
run_nodes   (id, run_id → agent_runs.id, node_key, title, capability, kind,
             position, depends_on JSON, status, attempt, summary, next_step,
             artifact_ids JSON, evidence, input_fingerprint,
             started_at, completed_at, updated_at)
run_interrupts
            (id, run_id → agent_runs.id, node_key,
             kind ∈ {approval,question,choice,conflict},
             prompt, options JSON,
             status ∈ {pending,answered,rejected},
             response, requested_by, responded_by, created_at, resolved_at)
```

Attachment and dependency FKs use `ON DELETE CASCADE`. Cascade is what makes
`DELETE /api/v1/tasks/:id` "just work" without app-level cleanup.
`task_events.task_id` intentionally has no FK: append-only history remains
queryable by task ID after the task itself is deleted.
`learning_notes.source_task_id` also intentionally has no FK so a promoted or
rejected learning keeps its provenance after task deletion. Its
`project_id` does use `ON DELETE CASCADE`.

Indexes:
- `idx_tasks_project_state(project_id, state)` — list-by-state filter
- `idx_tasks_priority(project_id, priority DESC)` — runnable ordering
- `idx_task_deps_dep(depends_on_task_id)` — reverse-dep traversal
- `idx_task_images_task(task_id)` — task detail attachment lookup
- `idx_task_docs_task(task_id)` — task detail doc lookup
- `idx_task_links_task(task_id)` — task detail link lookup
- `idx_task_events_task_created(task_id, created_at DESC)` — newest history first
- `idx_learning_notes_project_status(project_id, status, updated_at DESC)` —
  candidate review and metrics
- `idx_learning_notes_task(source_task_id, created_at DESC)` — task provenance
- `idx_agent_runs_active_task(task_id) WHERE status IN ('running','blocked')`
  — at most one resumable run per task
- `idx_agent_runs_task_updated(task_id, updated_at DESC)` — resume lookup
- `idx_agent_runs_project_status(project_id, status)` — run metrics
- `idx_run_nodes_run_position(run_id, position)` — ordered Work Graph restore
- `idx_run_interrupts_run_status(run_id, status)` — pending human decisions

Schema lives twice: once at `server/migrations/0001_init.sql` (for tools
that read the migration history) and once at
`server/internal/store/schema/0001_init.sql` (`go:embed`-ed into the
binary so a fresh database can be created without shipping the migrations
directory). Keep them identical.

## Agent run loop

Task workflow and Agent runtime are separate state machines:

```text
task: pending ⇄ start ⇄ spec ⇄ dev ⇄ test ⇄ review ⇄ done
run:             running ⇄ blocked ───────────────▶ completed | failed
```

Claim ownership gates both. Starting a run for a claimed task creates one
`agent_runs` row; repeating `run start` for the same owner resumes the active
row and appends `run.resumed`. A partial unique index prevents concurrent
active runs for one task.

Checkpoints store only a compact summary and next concrete action. Normalized
events live in append-only `task_events` and carry `run_id`:

- `run.started`, `run.resumed`;
- `tool.called`;
- `checkpoint.saved`, `run.blocked`;
- `verification.passed`;
- `learning.discovered`;
- `run.completed`, `run.failed`.

`GET /tasks/:id/resume` returns immutable task-start context, latest run, and
recent events. CLI, Web UI, Codex, Claude Code, Pi, and future adapters consume
the same protocol; none owns server-side loop semantics.

### Work Graph overlay

`workflow_template=single-loop` preserves the compact run/checkpoint model.
`workflow_template=engineering-flow` creates eight ordered `run_nodes`.
`workflow_definition` accepts any portable JSON Workflow Adapter with 1–32
nodes. A node is `ready` only when all dependency nodes are `completed` or
`skipped`. Agents write stage results, artifact IDs, input fingerprints, and
evidence through the CLI. The server rejects duplicate keys, unknown
dependencies, cycles, dependency skips, and completion while a node or human
interrupt remains open.

This graph is an outer durability layer, not an Agent engine:

```text
RunEngram Work Graph
  requirement → design → plan → implement → refactor → verify → review → gate
       │            │        │        │          │         │        │
       └──────── each node contains an ordinary Agent tool loop ────────────┘
```

An installed project skill, command, or human SOP maps outputs onto these
nodes. Codex, Claude Code, or Pi still owns tool selection and iteration inside
a node.
`run_interrupts` model explicit questions or approvals; a human browser or a
different CLI actor resolves them, after which the same node and run resume.
The `final-gate` rejects self-approval by the executing Agent. No queue worker
or second execution runtime is required.

The Action Console derives honest counters from stored receipts:
`completed_node_count`, `verified_node_count`, unique `artifact_count`, recalled
memory count, and `open_interrupt_count`. It does not estimate causal time
savings.

`learning.discovered` is the bridge from execution into learning. It creates a
pending Learning Note and links its ID back to the run event. Pending notes can
be edited by the live task owner. Promoted or rejected notes are immutable;
correcting promoted knowledge requires archiving its capsule and capturing a
new candidate.

## Task operation history

Mutation handlers resolve an actor once per request. A valid bearer token wins
and contributes the registered agent name; otherwise `X-Taskline-Client`
distinguishes `web` and `cli`, with `api` as the neutral fallback. The handler
places that actor in the request context and never constructs event payloads.

The service owns the event vocabulary, summaries, task snapshots, and
structured field differences. It synchronously appends a history event after
each successful task, claim, dependency, image, document, or link mutation.
The store owns only JSON persistence and newest-first retrieval. Task update
events retain full before/after values; document content updates record that
content changed without duplicating full Markdown bodies into SQLite.

## State machine

```
pending ⇄ start ──▶ spec ──▶ dev ──▶ test ──▶ review ──▶ done
              ▲         ▲         ▲        ▲          ▲
              └─────────┴─────────┴────────┴──────────┘
              any move between known states is allowed
              any state may also transition into pending
```

Implemented as a membership set in `model.stateOrder`. `CanTransitionTo`
only rejects unknown state names — direction is the agent's call. The
service layer enforces membership before calling `store.UpdateTask`.
Directional jumps remain legal. Dropping `review → dev` is intentional
(a review can surface a defect that legitimately reopens the implementation).
`test` is the local verification stage after implementation and before
review: unit tests, API e2e, browser smoke, and test coverage review
belong there. Teams with PR and CI may attach those results during `review`;
teams without them can record manual review evidence and continue.

The store records `completed_at` when a task enters `done`, clears it when the
task leaves `done`, and preserves it across ordinary edits and heartbeats. This
is the stable work-end timestamp; `updated_at` remains the timestamp of the most
recent mutation and must not be used as completion evidence.

Optional entry rules can be registered by target state through
`server/internal/service/workflow.go`. The default registry is empty, so
`review` and `done` do not require GitHub, PR, or CI access. Rules run only
when the state actually changes, keeping ordinary edits and same-state
updates free of external calls. `PullRequestVerifier` and its concrete
GraphQL adapter remain available for a future opt-in delivery policy rather
than being imposed on every project. Store `Force` only bypasses claim
ownership; it is not needed for ordinary manual completion.

`pending` lives off the main pipeline: tasks created without
`auto_start=true` land there, and any state may transition into it to
"park" work. The runnable query skips both `done` and `pending`.

There's no automatic transition triggered by completing dependencies —
"runnable" is a *query*, not a state. State only moves when an agent
(or human) PATCHes the task.

## Dependency DAG and the runnable query

`task_deps` is a many-to-many edge table. The runnable filter is a
single SQL query:

```sql
SELECT … FROM tasks t
 WHERE t.project_id = ?
   AND t.state NOT IN ('done','pending')
   AND NOT EXISTS (
         SELECT 1 FROM task_deps d
           JOIN tasks dt ON dt.id = d.depends_on_task_id
          WHERE d.task_id = t.id AND dt.state <> 'done'
   )
 ORDER BY t.priority DESC, t.created_at ASC;
```

Cycle prevention is application-side: before inserting an edge
`(task → dep)`, the store walks `dep`'s transitive deps and refuses if
it can reach `task`. SQLite has no native graph reachability, and the
DAG is small enough that a DFS per insert is fine.

Adding an existing edge is a no-op (the unique-key violation is caught
and swallowed) so dependency-add is idempotent for agents retrying on
network blips.

Task search is project-scoped lexical matching, not a separate index or
vector store. The handler validates `q` / `limit`, the service ranks
short-id, title, description, label, type, and state matches, and the
store remains a task persistence layer. This keeps the search feature in
the same local-first shape as the rest of the product; a future semantic
search feature would need an explicit persistence/indexing design rather
than being hidden inside the current store.

## Automatic learning loop

Automatic learning is a controlled state machine, not transcript storage:

```text
human correction / successful recovery
                  │
                  ▼
       learning note: pending
             │             │
   verified evidence       └── reject + reason
             ▼
 learning note: promoted + one active Exploration Capsule
                                      │
                                      ▼
                         future same-project recall
```

Only the live owner of the source task may capture a learning note. Any
authenticated workspace Agent may edit, promote, or reject a pending candidate,
so review still works after the source lease expires. Permission roles remain
outside the current single-user/local alpha.
Capture accepts structured `trigger`, `guidance`, `scope`,
labels, fingerprints, and producer fields; raw conversations, secrets, and
hidden reasoning are outside the contract.

Promotion requires non-empty verification evidence and assigns either
`project-rule` or `experience`. One store transaction
updates the pending note and creates its Exploration Capsule together,
preventing a promoted note without recallable knowledge. Repeating promotion
is idempotent: the existing capsule is returned and no duplicate is created.
Reject records a reason and never creates a capsule.

Pending and rejected notes remain visible for audit and metrics but are never
injected into task context. Active project rules use a separate per-task
budget without relevance filtering. Active experience is ranked by task text, labels, fingerprints, scope,
trigger, confidence, and observed reuse, then fitted to a context budget.
Agents can perform dynamic recall when new execution context appears without
mutating the immutable task-start snapshot. The public skill drives capture at high-signal execution
moments and resolves candidates after test evidence exists; the server
enforces capture ownership, review authentication, status, evidence, and
atomicity independently of any agent tool.

Automatic capture is deliberately narrow. Explicit reusable project
conventions, human corrections that change execution, and verified recovery
paths become candidates. Routine reads, successful commands, temporary paths,
task-only wording, secrets, raw transcripts, hidden reasoning, and guesses do
not become project memory.

### Typed Memory Graph

Exploration Capsules remain the recall unit. `memory_relations` adds a small
typed adjacency layer instead of introducing a graph database:

- `derived-from` connects a memory to a task or artifact;
- `validated-by` connects supporting evidence;
- `applies-to` gives an explicit module, platform, version, or scenario scope;
- `supersedes` replaces a capsule while preserving history and marking the old
  capsule stale;
- `conflicts-with` keeps competing conclusions visible;
- `caused-by` captures a reusable causal dependency.

The service validates relation and target kinds, rejects self edges, keeps
capsule targets inside one project, prevents supersession cycles, and treats
reverse duplicate conflicts as conflicts. Reads attach inbound and outbound
edges to each capsule.

Recall stays deterministic and budgeted. Explicit `applies-to` overlap adds a
ranking signal. The result includes:

- `context_revision`, a stable digest of selected capsule versions and
  relation IDs;
- one explanation per selected capsule with score and structured reason codes;
- warnings for active `conflicts-with` edges.

This preserves the immutable task-start snapshot while making later dynamic
recall auditable. Agents refresh recall after a material context change rather
than assuming one initial top-k list represents all project knowledge.

Promoted capsule edits accept `expected_updated_at`. The store performs the
update as compare-and-swap and returns `ErrConflict` when another reviewer
already changed the row. Material corrections should create a new capsule and
link it with `supersedes`; in-place edits remain available for wording and
metadata fixes.

## Project deletion

`DELETE /api/v1/projects/:project` resolves ID or name, snapshots task
attachment paths, removes task history owned by the project, then deletes the
project. Foreign-key cascades remove tasks, dependencies, metadata,
snapshots, runs, learning notes, and capsules. The handler deletes Markdown
and image files on a best-effort basis after the database transaction.

## Web UI delivery

`server/web/embed.go` exposes the bundle via two paths, in priority:

1. **Embedded** (`//go:embed all:dist`) — the production path. `pnpm
   build` writes into `server/web/dist/`; `go build` rolls it into the
   binary. A `.gitkeep` placeholder lets `go:embed` succeed on a fresh
   checkout where `pnpm build` hasn't run yet. The web `prebuild` step
   preserves that placeholder while replacing generated assets, and
   `FS()` detects the placeholder-only case and falls through.
2. **External `./dev-web/`** — if a directory by that name exists in the
   server working directory, it's served from disk. Useful for iterating
   on the UI without rebuilding the server.

When both miss, the server runs API-only and `serveUI` returns 404.

The handler registers API routes first, then mounts `serveUI` as a
catch-all on `NoRoute`. Unknown paths fall through to `index.html` so
the SPA's client-side router handles deep links.

## Web UI state and ordering

The React app keeps project and view selection in the URL without adding
a full router:

- `?project=<name|id>` selects a project and survives reload/share.
  The app prefers writing the project name, but still resolves saved id
  links.
- `?view=graph` opens the dependency graph. Kanban is the default and
  clears the `view` query param.

Kanban column sorting is component-local UI state. The default "next
execution order" mirrors the agent mental model by putting unblocked
tasks before blocked tasks, then sorting by priority and creation time;
other column sort options are browse conveniences. The canonical
runnable ordering remains the server-side SQL query used by `task next`.

The task search dialog is also a derived UI: it calls
`GET /api/v1/projects/:project/tasks/search`, opens the selected task in
the normal editor, and does not own task state.

## CLI ↔ server protocol

The CLI is a thin REST wrapper. `cli/client/client.go` is a hand-written
HTTP client (no codegen, no shared types) so the CLI module can stay
independent of the server module. Domain shapes are duplicated and kept
in sync by hand — drift here is the single most likely place for bugs,
so a CLI-side e2e test suite exercises the round-trip.

Agent preflight uses `GET /api/v1/status`. Without authorization it proves
server reachability. With a bearer token it also validates the identity and
returns that agent's live claims across projects. The store query uses the
existing owner/lease index and returns only the compact status shape; local
facts such as CLI version, config path, and `$TASKLINE_PROJECT` are composed by
the CLI. `POST /api/v1/agents/register` rejects a request that already carries
a valid token so an existing checkout identity cannot be replaced accidentally.

Canonical JSON fixtures in `testdata/http_contract/` guard the duplicated
HTTP shapes across server, CLI, and web tests. They are intentionally a
test-only drift net: they do not make the CLI import server packages, do
not introduce code generation, and do not change the runtime contract.
When adding or renaming a public JSON field, update the fixture and the
three local shape tests together.

Output formatting is centralized in `cli/internal/output`:

- `Resolve(flag)` picks JSON when stdout isn't a TTY (the default for
  agents), table otherwise.
- `Render` takes both a JSON value and a table-rendering closure so each
  command declares both shapes once.

## Configuration

Server config (`server/internal/config/config.go`) is environment
variables with optional `.env` overlay (process env wins). All paths
auto-`MkdirAll` on first boot:

- `TASKLINE_DB` — SQLite file (default `./data/taskline.db`)
- `TASKLINE_LISTEN` — listen addr (default `:8787`)
- `TASKLINE_IMAGES_DIR` — image storage root (default `./data/images`)
- `TASKLINE_DOCS_DIR` — markdown doc storage root (default `./data/docs`)

GitHub state verification reads `TASKLINE_GITHUB_TOKEN`, `GITHUB_TOKEN`, then
`GH_TOKEN`. When none is set it falls back to `gh auth token`, including common
Homebrew paths for LaunchAgent deployments. Tokens stay in memory and are not
written to taskline configuration or SQLite.

The checked-in `.env.example` intentionally points local runtime state at
ignored `./.cache/data/...`; the defaults above are what the server uses
when no `.env` value is present.

CLI config:

- `TASKLINE_SERVER` — base URL (default `http://127.0.0.1:8787`)
- `TASKLINE_PROJECT` — default `--project` value (so agents don't have
  to pass it on every subcommand)
- `.config/taskline/agent.json` — checkout-local agent id and bearer token;
  `taskline status` validates it before queue work

## Concurrency

`db.SetMaxOpenConns(1)`. SQLite under `modernc.org/sqlite` doesn't
reliably share PRAGMA state across connections, so we serialize access.
For a single-user, single-agent workload this is the right tradeoff —
correctness over throughput. WAL is enabled so reads don't block writes
within that single connection's transaction queue.

If contention ever matters, lift the cap and move PRAGMA setup into a
connection initializer.

## Test strategy

- **Unit**: `service_test.go` and `store_test.go` cover happy paths and
  edge cases (cycle rejection, invalid-state rejection, idempotent dep
  insert). `:memory:` SQLite for speed.
- **End-to-end**: `server/tests/e2e_test.go` boots a real Hertz server
  on a random port and exercises the HTTP surface, including the SPA
  fallback. This is the regression net for handler ↔ service wiring.
- **HTTP contract drift guard**: `testdata/http_contract/` contains
  canonical JSON fixtures round-tripped by server model tests, CLI client
  tests, and web API-shape tests. This preserves module independence while
  making field drift visible in normal test runs.
- **CLI**: lives in the CLI module; uses an `httptest.Server` to fake
  the backend.
- **Web**: Vitest component tests, ESLint, and `pnpm build`.
- **Skills**: `scripts/test-skill.sh` checks public and internal
  `SKILL.md` frontmatter plus load-bearing section headings.
