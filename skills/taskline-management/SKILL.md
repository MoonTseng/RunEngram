---
name: taskline-management
description: |
  Manage structured project work through RunEngram: create feature, bug, or
  docs tasks; plan dependencies; claim runnable work; advance the
  pending → start → spec → dev → test → review → done lifecycle; record
  evidence; or answer what remains. Use for requests such as "create a task",
  "queue this", "track this", "what should I work on next", "block this on",
  "mark this in review", or any project, kanban, backlog, or agent-workflow
  request. Explicit "taskline-management" prompt prefixes must trigger this
  skill. Support default create-only behavior plus run/spec/pending and
  执行/方案/待规划 modes. When invoked without a payload, drain the current
  project's runnable queue. Skip one-off notes with no state or follow-up.
---

# taskline — task management for AI agents

The `taskline` CLI is your only interface to taskline. It tracks
projects and the tasks (features / bugs / docs) inside them, enforces a
seven-state lifecycle (`pending → start → spec → dev → test → review → done`),
models inter-task dependencies as a DAG, and answers "what's runnable
now?".

RunEngram's task protocol is tool-neutral. Codex is the default producer, but
Claude Code and other coding agents use the same CLI, task context snapshots,
exploration capsules, and reuse outcomes. Never fork project knowledge by AI
vendor.

**Always go through the CLI.** Don't `curl` anywhere, don't try to read
or write the database, don't shell out to internal endpoints — even if
the CLI doesn't expose the exact verb you want. If you find a real
gap, file a taskline task to extend the CLI; don't work around it.
Where taskline runs and how it stores data is not your concern.

The CLI is built for agents, not humans at a terminal:

- JSON on stdout when not a TTY (your case). Pass `--format json` to
  force it; you almost never want `--format table`.
- Stable exit codes (0 success, non-zero error). Diagnostics on stderr.
- One subcommand per verb. No interactive prompts.
- If a command fails with "connection refused" or similar, tell the
  user — don't try to start anything yourself.

## When to use

Reach for taskline whenever the user's ask has *structure* — state,
ordering, dependencies, more than one item, "what's next?". Examples:

- "Track this as a feature in `<project>`"
- "What should I pick up next?"
- "Block `<task A>` on `<task B>`"
- "Mark `<id>` review / done"
- "Show me the open bugs / what's still in dev"
- "Wipe the done tasks from `<project>`"
- "Use taskline-management" / "按照 taskline-management skill 执行"

If the user explicitly names or invokes `taskline-management` and gives
no additional instruction, treat that as "work the current project's
runnable queue": resolve the project from `--project`, `$TASKLINE_PROJECT`,
or the current repository name when it is unambiguous. If the project
cannot be resolved, ask only for the project name. Once a project is
known, export `TASKLINE_PROJECT` for the session or pass `--project`
on every project-scoped command. Before doing queue work, make sure
the current working directory has a valid agent identity by running
`taskline status --format json`. Register only when it reports
`"registered": false`, then run status again. If status fails because
the local identity or token is invalid, stop and fix that identity
instead of registering another name over it. Then keep pulling
`taskline task next --claim --format json` after each completed task
until it returns the literal `null`. A task is not yours until that
claim command succeeds and the returned `owner` equals the agent name
registered in this directory. Do not stop after one task or one PR
unless the runnable queue is exhausted or a real blocker prevents
progress.

Skip taskline when the user just wants a one-line note, a scratch
todo, or an answer that doesn't survive past this turn — reply
directly. taskline is the wrong tool for content that has no
follow-up.

## Prompt shorthand

Treat `taskline-management` as a deterministic prompt prefix. Optional
full-width or ASCII brackets only delimit the payload; strip them before
creating the task.

| Prompt | Required behavior |
| --- | --- |
| `taskline-management <requirement>` | Create one runnable task and stop. Do not claim it or modify code. |
| `taskline-management run <requirement>` / `taskline-management 执行 <需求>` | Create, claim that exact task, read its context, and execute only that task through the stage playbook. |
| `taskline-management spec <requirement>` / `taskline-management 方案 <需求>` | Create and claim that exact task, produce and attach the `Spec` document, then stop before code changes. |
| `taskline-management pending <requirement>` / `taskline-management 待规划 <需求>` | Create one task with `--auto-start=false` and stop. |
| `taskline-management` with no payload | Claim and drain the current project's runnable queue as documented below. |

Resolve the project in this order:

1. explicit `project:<name>` or `项目:<名称>` in the prompt;
2. `$TASKLINE_PROJECT`;
3. an exact, case-insensitive match between the current repository name and a
   project returned by `taskline project list`;
4. the only project returned by `taskline project list`;
5. otherwise ask only for the project name.

Remove the explicit project selector from the stored task description. Preserve
the rest of the requirement text instead of compressing it into a few sentences.
Derive a concise title from the first heading or first complete sentence. Infer
`bug` only for an explicit defect/fix/crash request and `docs` only for a
documentation-only request; otherwise use `feature`. Use priority `0` unless the
prompt supplies a priority.

Create-only and pending modes do not need an agent identity. Before `run`,
`spec`, or bare queue-drain mode, run `taskline status --format json`. Register
only when it reports `registered=false`; if no agent name was supplied and no
valid identity exists, ask only for the agent name. Never infer or replace an
identity silently.

For `run` and `spec`, claim the ID returned by `taskline task create` with
`taskline task claim <id>`; do not use `task next --claim`, because another
higher-priority task may be selected. Immediately read
`taskline task context <id>`. `run` stops after that created task completes or
hits a real blocker; it does not drain unrelated queued work. `spec` attaches
the `Spec` task document and stops before any source-code modification.

## Environment

```bash
export TASKLINE_PROJECT="demo"   # default project so you can omit --project
taskline status --format json
taskline register --name "agent-a"  # only when registered=false
taskline status --format json
```

`--project` overrides `$TASKLINE_PROJECT`. A project is referenced by
**name** (`demo`) or **id** (`9b…uuid`) — both work everywhere.
Export `TASKLINE_PROJECT` once at the start of a session that's
focused on a single project.

`taskline register --name <agent>` writes `.config/taskline/agent.json`
in the current working directory. That file contains the bearer token
used by claim, heartbeat, release, and normal update flows; it is
intentionally not global because multiple agents may share one machine.
If a claiming command fails with `agent identity required`, register
the current directory first, then verify it with `taskline status`.
Do not pass or invent owner strings; the
server derives owner from the registered token.

## Domain model

| Field         | Notes                                                                      |
| ------------- | -------------------------------------------------------------------------- |
| `id`          | UUID, generated for you on create                                          |
| `project_id`  | UUID of owning project                                                     |
| `title`       | required, short                                                            |
| `description` | optional, longer prose                                                     |
| `type`        | `feature` (default), `bug`, or `docs`                                      |
| `state`       | `pending`, `start`, `spec`, `dev`, `test`, `review`, `done`                |
| `priority`    | integer; **higher = runs sooner** (default 0)                              |
| `labels`      | task-local GitHub-style text labels, ordered and deduped by the server     |
| `owner`       | optional multi-agent owner; empty means unclaimed                          |
| `claimed_at`  | unix milliseconds when the current owner claimed the task                  |
| `lease_expires_at` | unix milliseconds when the current owner lease expires              |
| `completed_at` | stable unix milliseconds when the task most recently entered `done`; zero otherwise |
| `depends_on`  | list of task ids; the task is blocked until **every** dep reaches `done`  |
| `images`      | optional binary attachments; each image includes a `url` for retrieval     |
| `docs`        | optional Markdown docs; each doc includes a raw-content `url`              |

Every mutation also appends a task history event with `actor`, `action`,
`summary`, structured `details`, and `created_at`. A registered agent token is
recorded as that agent name; otherwise the actor is `web`, `cli`, or `api`.
Use `taskline task history <id>` whenever you need durable operation context or
the exact before/after values for title, description, state, type, priority, or
labels.

**State machine.** Any state may transition toward any other named state, but
target-state evidence rules still apply. Drop-backs (`review` → `dev` when a
defect surfaces) and directional jumps are legal; entering `review` requires
an attached valid GitHub PR, and entering `done` requires that PR to be merged
with review threads resolved and CI green or not configured. `--force` does
not bypass these gates. Unknown state names are rejected — don't invent new
ones.
`test` is the local verification stage between implementation and
review: test review, unit tests, API e2e, browser smoke, and any other
checks that should pass before PR review/CI begins.

**`pending` is the parking lot.** Tasks in `pending` are explicitly
**not runnable**: `task next` and `task list --runnable` skip them.
Use it when you want to capture work without offering it to the queue
yet (rough drafts, future ideas, things that need refinement). Any
state may transition into `pending` — drop a task back into the lot
whenever it should stop being a candidate. Move it to `start` (or
further along) when it's ready to be worked.

**Runnable.** A task is runnable when its state is neither `done` nor
`pending` AND every task it depends on has state `done`. Runnable
queue-preview commands hide live claims owned by other agents by
default. A registered agent sees its own live claims first, plus
unclaimed or lease-expired tasks. Tasks are returned with same-owner
claims first, then by `priority DESC`, then `created_at ASC`.
Use `taskline task next --claim` to reserve the single highest-priority
claimable task before doing any work. Plain `taskline task next` is
only a preview and must not be treated as permission to start. Add
repeated `--label` filters to `task next` or `task list --runnable` to
pull from a labeled subset; labels use AND semantics, so
`--label backend --label ui` returns tasks that have both labels.
Matching is case-insensitive, like label deduplication.

**Dependency DAG.** Adding an edge that would close a cycle is
rejected. Self-deps are rejected. Re-adding an existing edge is a
no-op.

## CLI cheat sheet

`-h` on any subcommand prints flags. This is the full agent surface.

### Agent preflight

```bash
taskline status --format json
taskline register --name agent-a  # only when status says registered=false
taskline status --format json
```

Status reports CLI version, server health, checkout-local config directory,
default project, registered agent, and the agent's current live claims across
projects. A configured identity must be accepted by the server; an invalid or
stale token is an error, not an unregistered state. Registration with an
already-valid token is rejected so one agent cannot accidentally replace
another checkout identity.

### Projects

```bash
taskline project create --name demo --description "first project"
taskline project list
```

### Tasks

```bash
# Create (defaults to 'start' state — immediately runnable)
taskline task create --project demo --title "first task" --type feature --priority 1
taskline task create --project demo --title "labeled task" --label backend --label ui

# Create and park in 'pending' (won't show up in `task next`)
taskline task create --project demo --title "later idea" --auto-start=false

# List (filter by state with comma-separated names)
taskline task list --project demo
taskline task list --project demo --state start,dev,test
taskline task list --project demo --mine
taskline task list --project demo --unclaimed
taskline task list --project demo --runnable --label backend
taskline task list --project demo --runnable --mine

# Pick / inspect
taskline task next --project demo            # preview only; does not reserve work
taskline task next --project demo --claim --lease 6h
taskline task next --project demo --claim --label backend
taskline task search --project demo fc7a0732 # short id / full id / text matches
taskline task search --project demo "historical context" --limit 10
taskline task get <id>
taskline task history <id>                  # actor, operation, time, before/after
taskline task context <id>                  # immutable task-start context + recalled memory

# Mutate (PATCH semantics — only pass the flags you want changed)
taskline task update <id> --state test
taskline task update <id> --priority 5 --description "new prose"
taskline task update <id> --label review --label frontend   # replace labels
taskline task update <id> --add-label review --remove-label triage
taskline task update <id> --append-description "new note"
taskline task update <id> --clear-labels                    # remove labels
taskline task update <id> --state done --if-state review
taskline task update <id> --state pending --force            # manual correction
taskline task delete <id>                    # cascades deps + attachments

# Multi-agent ownership
taskline task claim <id> --lease 2h
taskline task heartbeat <id> --lease 6h
taskline task release <id>
taskline task release <id> --force           # manual recovery

# Dependencies
taskline task depend <id> --on <other-id>
taskline task undepend <id> --on <other-id>

# Image attachment (any binary)
taskline task upload <id> --file ./screenshot.png

# Markdown docs (stage deliverables, notes, reports)
taskline task doc create <task-id> --title "Spec" --file ./spec.md
taskline task doc get <doc-id>
taskline task doc update <doc-id> --title "Test Report" --file ./test-report.md
taskline task doc delete <doc-id>

# Link (PR, external design doc, ticket, merged commit — any URL to remember)
taskline task link <task-id> --url https://example.com/pr/42 --label "PR #42"

# Remove a link by its id (links are returned inline on `task get`)
taskline task unlink <link-id>

# Verified engineering memory
taskline capsule list --project demo --query webview
taskline capsule create --project demo --source-task <id> \
  --title "Reusable migration boundary" \
  --summary "Migrate callers before deleting compatibility service" \
  --scope "WebView URL service migrations" \
  --evidence-file ./verified-evidence.md \
  --label webview --fingerprint url-service --producer codex
taskline capsule use <capsule-id> --task <id> --outcome helpful \
  --notes "avoided repeated caller search"
taskline capsule archive <capsule-id>
taskline capsule metrics --project demo
```

Delete returns `{"deleted": true, "id": ...}`; depend returns
`{"task_id": ..., "depends_on": [...]}`. Pipe to `jq` freely.

### Multi-agent claim flow

Run `taskline status --format json` first and confirm the registered agent
identity. Register only when status explicitly reports `registered=false`.
Use `task next --claim` when more than one agent may pull from the same
project. Plain `task next` is a read-only preview and does **not**
reserve work. Never begin implementation from a plain `task next`
result; claim first.

`task next --claim` atomically selects the highest-priority runnable task,
sets `owner`, `claimed_at`, and `lease_expires_at`, and returns the claimed
task. Claimable means runnable and either unclaimed, owned by the registered
agent in this directory, or lease-expired. Same-owner claims are preferred so a
restarted agent can pick up its own unfinished work first.

The default lease is 6h. Use a shorter `--lease` for short tasks. Normal
`task update` commands from a registered directory renew the lease;
`task heartbeat <id>` renews without changing task content. `task release <id>`
gives work back immediately. Expired leases are reclaimed without a background
worker; the next successful claim/update observes the current owner and rejects
stale non-owner writes.

Do not infer your identity from a returned task's `owner` field. That
field says who currently owns the task; your identity is the agent
registered in `.config/taskline/agent.json` under the current working
directory. If a task is claimed by a different live owner, pull a
different task with `task next --claim` instead of trying to act as
that owner.

Use repeated `--label` flags when agents should consume different labeled
subsets inside one project. Example:
`task next --claim --label backend` atomically claims only runnable tasks
tagged `backend`; adding more labels narrows the filter with AND semantics.
The same label filter is available on `task list --runnable` for previews.

### Task docs and links

As you walk a task through the playbook you'll generate artifacts that
belong with it. Use task docs for Markdown content owned by the task
itself, and links for external URLs such as PRs, commits, design tools,
or chat threads. Do not keep stage deliverables only in chat history.

Task docs are first-class Markdown files. They surface inline on
`task get` with `url` fields under `/api/v1/docs/<doc-id>/content`;
fetch full editable content with `taskline task doc get <doc-id>`.
Create or update the stage doc before advancing out of the matching
stage:

- **spec → dev:** `Spec` doc with product design, technical design
  (IDL/API definitions and implementation plan), and test plan/test
  cases. If a Superpowers plan already exists, upload that content as
  the task doc.
- **dev → test:** `Dev Notes` doc summarizing implementation, issues
  encountered, and any divergence from the spec/technical design.
- **test → review:** `Test Report` doc reviewing test cases, module
  tests, real e2e/API/CLI/browser/device checks, agent evaluation,
  pass rate, failures, and whether failures require returning to dev.
- **review → done:** `Review Report` doc covering PR comments, CI
  status, whether the implementation still matches the original design,
  and any justified design updates.

Recommended moments to call it:

- **spec/dev/test/review**: create or update the matching Markdown doc.
- **test → review**: link the PR URL just after `gh pr create` ("PR #N").
- **review → done**: update the Review Report and any merged-commit or
  post-mortem links before changing state. The attached PR link itself is the
  authoritative merge/review/CI evidence.

Docs and links surface inline on `task get` and in the web detail view.
There is no limit on how many docs or links a task can hold; favour
adding too many over too few — they're cheap to remove later.

## Automatic learning notes

During a claimed task, capture a Learning Note without asking the user when:

- a human correction fixes a failed tool, command, workflow, or architecture
  route and can help a future task; or
- the agent recovers from a failed approach and verifies a reusable route.

Run `taskline learning capture <task-id>` immediately. Preserve only minimal
trigger, reusable guidance, scope, labels, fingerprints, and producer. Never
capture secrets, credentials, raw transcripts, guesses, task-only preferences,
or recalled guidance that already existed.

Example: a user supplies a Notion requirement link. Direct reading fails, then
the user explains that `one-flow/notion-to-prd` must normalize it first:

```bash
taskline learning capture <task-id> --project <project> \
  --kind human-correction \
  --trigger "Direct Notion requirement read failed" \
  --guidance "Use one-flow/notion-to-prd before requirement analysis" \
  --scope "Notion requirement analysis" \
  --label notion --label prd --fingerprint notion-to-prd \
  --producer codex
```

During test or wrap-up, list pending notes for the task:

```bash
taskline learning list --task <task-id> --status pending
```

Promote only after commands, tests, artifacts, or merged changes verify the
guidance:

```bash
taskline learning promote <note-id> --evidence-file <file>
```

Reject disproved or non-reusable guidance:

```bash
taskline learning reject <note-id> --reason "<evidence-backed reason>"
```

Leave unverified notes pending. Pending notes remain visible but are never
recalled into another task. Never capture secrets in a Learning Note or its
evidence. Use `--producer codex` by default and `--producer claude-code` when
the recovery came from Claude Code; producer identifies the tool, not a
separate knowledge silo.

## Stage playbook — "work the queue"

When the user says "work the queue" / "do the next task" / "keep
going through the backlog", or explicitly invokes this skill without
more instructions:

1. Run `taskline task next --project <p> --claim --format json`.
2. The CLI emits the bare task object (`id`, `title`, `state`, … as
   top-level fields) on successful claim, or the literal `null` when
   nothing is currently claimable. If you see `null`, report there's
   nothing runnable/claimable and stop. If the returned task has an
   `owner` different from the agent registered in this working
   directory, stop; that is not your task.
3. Read `title`, `description`, any `docs`, and any `images`. Each doc
   includes a raw Markdown `url` under `/api/v1/docs/<doc-id>/content`;
   each image includes a `url` under `/api/v1/images/<image-id>`. Fetch
   and surface them when they are material to the task. When a task
   references a short id, previous work, or historical context, use
   `taskline task search --project <p> "<query>" --format json` to find
   the related task before relying on memory or chat history.
   Immediately run `taskline task context <id> --format json`. This creates
   one immutable task-start snapshot and returns up to five relevant,
   verified exploration capsules from the same project. Read their `scope`
   and `evidence` before using them. Do not substitute recalled knowledge for
   current code verification.
4. Walk the task through the stages below in order. Each stage has the
   same shape: **Trigger** (what just happened) → **Actions** (do
   these now) → **Advance** (literal CLI command to move state) →
   **Skip when** (escape clause).
5. Loop back to step 1 — don't pause to ask the user whether to
   continue.

Higher-order capabilities (brainstorming, planning, code review) are
referenced by what they do, with a Superpowers skill name in
parentheses if your harness has them; drop the parenthetical if not
installed.

### start → spec

- **Trigger:** you just successfully claimed the task from the queue.
- **Actions:**
  1. `git checkout main && git pull`
  2. `git checkout -b feature/<short-kebab-slug>` (slug from the title;
     keep it under ~30 chars).
  3. Confirm `git status` is clean.
- **Advance:** `taskline task update <id> --state spec`
- **Skip when:** the change qualifies as fast-path (see below) — go
  straight to dev.

### spec → dev

- **Trigger:** branch exists, title + description loaded.
- **Actions:**
  1. Clarify the product contract from the task description, project
     docs, and code context: user need, scope, non-goals, UX or
     interaction behavior, and acceptance criteria. Do not ask the user
     for routine design approval; ask only when missing information
     makes safe implementation impossible.
  2. Capture that contract in a `Spec` task doc before advancing. The
     doc must include product design, technical design (IDL/API
     definitions and implementation plan), and test plan/test cases.
     If you already wrote a Superpowers plan, upload that content as the
     doc rather than duplicating it in the task description.
- **Advance:** `taskline task update <id> --state dev`
- **Skip when:** the change is mechanical (rename, formatting,
  one-line config) — go straight to dev.

### Architecture review without user checkpoints

For routine product/technical choices, do not pause for user approval.
High-quality autonomy still needs a second pass: after you identify
2-3 viable approaches, choose the simplest one that fits the product
goal, then run an architecture review before implementation.

Prefer a separate architect-style subagent when your harness supports
subagents. Give it the task title, description, proposed options,
recommended option, and relevant repo constraints; ask it to check for
over-engineering, unclear boundaries, hidden coupling, performance
risks, testability gaps, and violations of the project's philosophy. If
subagents are not available, perform the same review yourself as an
explicit second pass. The final choice should be simple, declarative,
readable, performant enough for the expected workload, and aligned with
existing module boundaries.

Ask the user only when the product intent is genuinely unknowable from
the task, the decision has external/business consequences, credentials
or destructive permissions are missing, or the safe implementation
cannot proceed without information that is not in the repo or taskline
task. In all other cases, record the chosen approach and reason in the
task description or implementation notes, then continue.

### dev → test

- **Trigger:** product spec / acceptance criteria in hand.
- **Actions** (test-first):
  1. Brainstorm the technical approach — list 2-3 implementation options,
     pick one, and name the tradeoff. No human checkpoint. (capability:
     brainstorming — `superpowers:brainstorming`)
  2. Run the architecture review described above, revise the choice if
     it finds a concrete issue, and keep the final plan simple,
     declarative, readable, and aligned with the project goals.
  3. Plan the technical work — architecture boundary, ordered steps, and
     test strategy. (capability: plan writing —
     `superpowers:writing-plans`)
  4. Write or extend failing tests for the new behavior.
  5. Implement until the focused tests pass and the behavior is ready
     for full local verification.
  6. Capture each new reusable recovery or human correction immediately with
     `taskline learning capture`. Do not wait until chat context is lost.
  7. Create or update a `Dev Notes` task doc summarizing the
     implementation, issues encountered, and any divergence from the
     `Spec` doc with the reason.
  8. For each recalled capsule used, record the observed result:
     `taskline capsule use <capsule-id> --task <id> --outcome helpful|rejected|stale`.
     Use `stale` when current code disproves once-valid knowledge. Include a
     short note explaining evidence.
- **Advance:** `taskline task update <id> --state test`
- **Skip when:** never. Implementation must be ready for local
  verification before review begins.

### test → review

- **Trigger:** implementation behavior is complete in the local
  worktree.
- **Actions:**
  1. Review the tests you wrote or touched. Add coverage now if the
     behavior, migration path, CLI surface, or UI state can regress.
  2. Run the full project test suite for whatever you touched.
     For this repo: `( cd server && go test ./... )`,
     `( cd cli && go test ./... )`, `( cd web && pnpm lint && pnpm test && pnpm build )`.
     Run `./scripts/test-skill.sh` when skill docs changed. Lint /
     format as the project requires.
  3. For taskline itself, or any project with an embedded frontend,
     migrations, or runtime startup behavior, verify against the rebuilt
     running binary rather than only isolated tests.
  4. Self code-review for bugs, dead code, boundary issues.
     (capability: code review — `code-review:code-review`)
  5. Fix anything the review or tests surface; re-run the relevant
     tests after each fix.
  6. Run `taskline learning list --task <id> --status pending`. Promote each
     verified candidate with its evidence file, reject disproved guidance with
     a concrete reason, and leave genuinely unverified candidates pending.
  7. Create or update a `Test Report` task doc with reviewed test
     cases, commands/checks run, pass rate, failures, and whether any
     failures require dropping back to `dev`.
  8. Stage and commit. Conventional, minimal messages.
  9. `git push -u origin <branch>`.
  10. `gh pr create` with title, summary, and a test plan.
  11. Attach the PR URL to the task:
     `taskline task link <task-id> --url <pr-url> --label "PR #N"`
     so anyone reading the task later can jump straight to the
     review.
- **Advance:** `taskline task update <id> --state review`
- **Skip when:** never. Tests and a real pushed PR are the gate. The server
  rejects the transition until the PR link has been attached.

### review → done

- **Trigger:** a PR exists for the committed implementation.
- **Actions:**
  1. **Wait for CI** if configured. If it fails, drop the task back to
     `dev`, fix the root cause locally, re-run tests, and push.
  2. **Wait for at least one review** — human or bot
     (`gemini-code-assist`, etc.). Don't merge before any review has
     posted; the whole point of opening a PR is the second pair of
     eyes. Poll with:

     ```bash
     gh pr view <n> --json reviews,reviewDecision,statusCheckRollup
     ```

     Re-check periodically until `reviews` is non-empty.
  3. Read **every** comment surface — one endpoint isn't enough:

     ```bash
     gh api repos/<owner>/<repo>/pulls/<n>/reviews     # bot summaries
     gh api repos/<owner>/<repo>/pulls/<n>/comments    # inline review comments
     gh api repos/<owner>/<repo>/issues/<n>/comments   # top-level PR conversation
     ```

     Address each finding; for real defects, drop the task back to
     `dev`, re-run tests after each batch, and push. If a comment is
     wrong, **reply with reasoning** rather than silently ignoring it.
  4. Merge with `gh pr merge --squash --delete-branch` (or the project's
     required merge style) only after reviews and CI are ready.
  5. Confirm the remote result with
     `gh pr view <n> --json state,mergedAt,statusCheckRollup`.
  6. Create or update a `Review Report` task doc covering PR comments,
     CI status, merge result, and whether the implementation still matches
     the original design. If not, either update the design doc with the
     justified change or drop back to `dev` for rework.
- **Advance:** `taskline task update <id> --state done` *only after*
  (a) CI green or N/A, (b) at least one review posted, and
  (c) every reviewer comment addressed or rebutted, and (d) the PR is merged.
  The server queries GitHub and rejects `done` when merge, review-thread, or CI
  evidence is incomplete.
- **Drop back to dev** with `taskline task update <id> --state dev`
  when review or CI surfaces a real defect. The bidirectional state
  machine exists for exactly this — don't delete-and-recreate.

### done — wrap-up

- **Trigger:** task is `done` after the PR was merged and verified.
- **Actions:**
  1. Run `taskline learning list --task <id> --status pending` again. Verify no
     candidate was silently promoted. Promote only with evidence, reject only
     with reason, and leave unresolved candidates pending.
  2. For durable verified exploration that was not a correction or recovery,
     create one focused capsule with applicability scope, code/module
     fingerprints, and Markdown evidence. Never promote guesses, raw chat,
     secrets, credentials, task-only prose, or conclusions not rechecked
     against code.
  3. `git checkout main && git pull`
  4. Delete the local feature branch (gh's `--delete-branch` may have
     done this already).
- The taskline task is already `done`; this stage is repo hygiene.

## Fast path

A task qualifies as fast-path when **all** of:

- single file changed,
- no behavior visible to other code,
- no test scaffolding or new dependency.

Examples: typo in a comment, raising a log level, bumping a constant.
The product/spec work may collapse, but the delivery gates do not:

```
	start → dev → test → review → done
```

Skip a separate Spec when appropriate and keep stage docs concise, but still
use a branch, commit, real push, PR, CI/review, and merge. Documentation never
substitutes for delivery evidence.

## Gotchas

- **`taskline status` fails for an existing identity** — do not register a new
  name over it. Check `TASKLINE_SERVER` and
  `.config/taskline/agent.json`; repair or intentionally remove the stale local
  identity before registering again.
- **`already registered as ...`** — this checkout already has a valid token.
  Run `taskline status` and continue as that agent; do not rotate its identity.
- **Forgot `--project`?** Export `TASKLINE_PROJECT` once at session
  start. Only `task create`, `task list`, `task search`, and
  `task next` accept `--project` — the rest (`get`, `update`, `delete`, `depend`,
  `upload`) operate on the task id directly and reject the flag with
  "unknown flag".
- **`invalid next state "..."`** — you used a name that isn't in
  `pending/start/spec/dev/test/review/done`. The state `created` was
  renamed to `start`, and `design` was renamed to `spec`; don't
  reintroduce old names.
- **`cannot enter review`** — create and push the branch, open a real GitHub
  PR, attach it with the exact `taskline task link ...` command shown in the
  error, then retry the state update.
- **`cannot enter done`** — follow the listed blocker: resolve every review
  thread, wait for CI, merge the PR, update task docs/links, then retry. Do not
  use `--force`; it cannot bypass delivery evidence.
- **`state entry verification unavailable`** — GitHub could not be queried.
  Configure `TASKLINE_GITHUB_TOKEN`/`GITHUB_TOKEN`/`GH_TOKEN` for the server or
  run `gh auth login` on the server host, then retry.
- **`dependency would create a cycle`** — the edge would loop back.
  Restructure the graph or pick a different anchor.
- **`project name "X" already exists`** — name collision. Reuse the
  existing project (likely what you wanted) or pick a new name.
- **`error: project required`** — neither `--project` nor
  `$TASKLINE_PROJECT` is set.
- **`task next --claim` returned `null`** — nothing is claimable for
  this registered agent. Either the project is empty, every non-done task is
  blocked, every available task is claimed by another live owner, or
  everything left is parked in `pending`. Run
  `taskline task list --project <p> --state pending,start,spec,dev,test,review`
  to see what's stuck and why. Do not automatically move `pending`
  tasks into `start`; promote them only when the task description,
  dependencies, or the user makes clear that they are ready to run.
- **The user said "remind me to X"** — that's a one-off note, not a
  task. Reply directly; don't create a taskline entry.
