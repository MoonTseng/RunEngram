# Memory Impact Receipts Design

## Problem

RunEngram already recalls verified memory into later task contexts, but it does
not make the effect visible.

Current CamScanner data proves the gap:

- five tasks have immutable context snapshots;
- recent tasks recalled the rule that Codex must not run Gradle;
- their test reports explicitly state that Gradle was not run because of the
  project rule;
- `reused_task_count` remains zero because no `capsule_usages` receipt was
  submitted.

The dashboard therefore answers "how many memories exist" and "did Agent runs
finish", but not:

- which memory reached a task;
- whether the Agent applied or ignored it;
- what behavior changed;
- which evidence supports that conclusion;
- whether repeated use improved delivery quality or reduced recovery work.

Users experience a board with memory attached, not a learning system.

## Goal

Make the path from memory recall to observed effect visible and auditable:

```text
verified memory
  → recalled into task
  → applied or ignored by Agent
  → result supported by task evidence
  → helpful, rejected, stale, or still unconfirmed
```

RunEngram must distinguish facts from inference. Absence of a Gradle command is
not proof that memory caused the behavior. An impact becomes confirmed only
when an Agent or developer records the decision and links reviewable evidence.

## Non-goals

- No OS-wide command spying.
- No mandatory GitHub, GitLab, pull request, or CI integration.
- No LLM inside the server.
- No fabricated time-saved estimate.
- No requirement to use a specific engineering SOP.
- No task-completion gate when a memory receipt is missing.
- No replacement for current `capsule_usages` confidence calculations.

## Considered approaches

### A. Infer impact from task documents

Scan task reports for phrases such as "did not run Gradle" and mark matching
memory helpful.

Advantages:

- fast to add;
- can show historical results immediately.

Rejected because:

- wording varies across languages and Agents;
- correlation is not causation;
- a missing command can be accidental;
- false helpful results would make memory confidence unreliable.

### B. Explicit impact receipts with evidence

Persist recall automatically. Require the Agent to report applied, ignored, or
final outcome with an evidence reference. Missing reports remain visible as
unconfirmed.

Advantages:

- auditable;
- Agent-neutral;
- preserves current confidence model;
- supports task, memory, and project views;
- can backfill historical recall without inventing outcomes.

This is the selected approach.

### C. Full tool and command telemetry

Capture every tool call and infer policy compliance from execution traces.

Advantages:

- strongest low-level observability.

Rejected for this iteration because:

- couples RunEngram to each Agent runtime;
- adds privacy, storage, and performance costs;
- still cannot prove why an Agent chose not to execute a command.

## Domain model

Add `memory_impacts` as a persisted resource separate from
`capsule_usages`.

One row represents one memory in one task:

| Field | Purpose |
| --- | --- |
| `id` | stable receipt ID |
| `project_id` | project filter without joining through task |
| `task_id` | affected task |
| `capsule_id` | recalled memory |
| `state` | `recalled`, `applied`, `ignored`, `helpful`, `rejected`, `stale`, `unconfirmed` |
| `recall_source` | `task-start`, `dynamic`, or `historical-backfill` |
| `context_revision` | Context Pack revision that exposed the memory |
| `recall_score` | matching score when available |
| `recall_reasons_json` | label, fingerprint, trigger, scope, relation, and warning reasons |
| `stage` | run stage where Agent made the decision |
| `notes` | short explanation |
| `evidence_json` | task doc, link, event, checkpoint, command, file, or free-text references |
| `actor` | Agent or developer recording the decision |
| `created_at` | first recall |
| `updated_at` | latest decision |
| `resolved_at` | final outcome time; zero while unresolved |

Constraint:

```text
UNIQUE(task_id, capsule_id)
```

`project_id` keeps the normal project foreign key and cascade. `task_id` and
`capsule_id` deliberately have no foreign key so impact history survives task
or memory deletion, matching task-event retention.

State meanings:

- `recalled`: memory was present in task context; no decision yet.
- `applied`: Agent chose to follow it; result not evaluated yet.
- `ignored`: Agent reviewed it and found it irrelevant. No confidence change.
- `helpful`: application had reviewable positive evidence.
- `rejected`: current task showed the guidance was not useful or was wrong.
- `stale`: current code or tooling disproved once-valid memory.
- `unconfirmed`: task finished without an explicit decision.

`helpful`, `rejected`, and `stale` remain mirrored into existing
`capsule_usages`, preserving confidence, trust, dispute, and stale behavior.
`recalled`, `applied`, `ignored`, and `unconfirmed` do not affect memory
confidence or `use_count`.

Allowed progression:

```text
recalled → applied | ignored | helpful | rejected | stale | unconfirmed
applied → helpful | rejected | stale
unconfirmed → applied | ignored | helpful | rejected | stale
ignored → applied | helpful | rejected | stale
```

An Agent cannot overwrite a final outcome. A developer may correct one final
outcome to another with optimistic concurrency and a new evidence record.

## Recall flow

### Task-start context

After `GetOrCreateTaskContext` creates the immutable snapshot:

1. upsert one `memory_impacts` row for every project rule and suggested
   capsule;
2. save match explanation and `context_revision`;
3. keep an existing later state if the call is retried.

Snapshot creation must not fail because impact persistence failed after the
snapshot transaction has committed. Service returns the persistence error and
retry remains idempotent.

### Dynamic recall

After `RecallTaskMemory` returns a new Context Pack:

1. upsert newly recalled memories;
2. mark `recall_source=dynamic` for new rows;
3. merge newer recall reasons and revision without resetting applied or final
   states.

### Historical backfill

An idempotent service reconciliation reads existing context snapshots and
creates `historical-backfill` receipts in `recalled` state. It runs before
project impact metrics or impact history are returned, then performs only
missing-row inserts. Schema migration only creates tables and indexes; it does
not parse JSON.

It does not infer applied or helpful from task documents. Existing
`capsule_usages` rows are imported as their recorded final outcomes.

This makes old recall visible immediately while preserving truth.

## Agent receipt flow

Keep `runengram capsule use` as the compatibility command.

Supported outcomes:

| CLI outcome | Impact state | Confidence effect |
| --- | --- | --- |
| `used` | `applied` | none |
| `ignored` | `ignored` | none |
| `helpful` | `helpful` | existing positive effect |
| `rejected` | `rejected` | existing negative effect |
| `stale` | `stale` | existing stale behavior |

Add optional flags:

```text
--stage <stage>
--evidence-kind task-doc|task-link|task-event|checkpoint|command|file|text
--evidence-ref <id-or-reference>
--notes <observed-effect>
```

Rules:

- `used` and final outcomes require notes.
- `helpful`, `rejected`, and `stale` require evidence.
- receipt task and capsule must belong to the same project.
- existing claim rules stay unchanged.
- repeated updates are idempotent for the task/capsule pair.
- a final outcome can be corrected by a developer; history survives through
  task events.

At run completion, unresolved `recalled` rows become `unconfirmed`.
Task completion remains allowed. The missing decision becomes visible instead
of silently counting as zero reuse.

Skill contract:

1. read all recalled memory IDs at task start and dynamic recall;
2. emit `used` when a memory changes a plan, command, test, or validation
   choice;
3. emit `ignored` when it is reviewed but irrelevant;
4. before completion, update applied memory to `helpful`, `rejected`, or
   `stale` with evidence;
5. never mark helpful solely because a forbidden command is absent.

## API

Add:

```text
GET /api/v1/projects/:project/memory-impacts
GET /api/v1/tasks/:id/memory-impacts
GET /api/v1/capsules/:id/memory-impacts
```

Project endpoint accepts:

```text
state
task_id
capsule_id
limit
```

Existing `POST /api/v1/capsules/:id/usage` accepts the new fields and updates
both impact and legacy usage when required.

Extend learning metrics:

| Metric | Meaning |
| --- | --- |
| `recalled_task_count` | tasks receiving at least one memory |
| `recalled_memory_count` | unique task-memory recall pairs |
| `applied_task_count` | tasks with applied or final receipts |
| `helpful_task_count` | tasks with at least one helpful receipt |
| `ignored_count` | reviewed but irrelevant memories |
| `unconfirmed_count` | recalled memories lacking a decision at task finish |
| `recall_coverage_rate` | recalled tasks / tasks with context snapshots |
| `application_rate` | applied or final pairs / recalled pairs |
| `confirmation_rate` | resolved pairs / recalled pairs |

Keep existing fields for API compatibility. Relabel current
`reused_task_count` in the UI as "confirmed application tasks"; do not present
it as recall coverage.

## Web experience

### Project metrics

Replace the current flat memory counters with a visible funnel:

```text
Tasks with recall → Confirmed application → Helpful result
                 ↘ Ignored
                 ↘ Unconfirmed
```

Primary cards:

- recall coverage;
- confirmed application tasks;
- helpful tasks;
- unconfirmed decisions.

Reliability cards such as run completion and recovery remain, but appear in a
separate "Run reliability" group. They must not imply efficiency.

### Memory card and detail

Each verified memory displays:

- recalled task count;
- applied task count;
- helpful/rejected/stale counts;
- last affected task and time.

Memory detail adds "Impact history" with task title, state, notes, evidence,
stage, actor, and timestamp.

### Task impact panel

Action Console and completed-task detail show:

- memories recalled for the task;
- match reason;
- applied/ignored/final state;
- evidence;
- unresolved decision count.

Example:

```text
Applied: Do not run Gradle
Effect: used static checks and device logs instead
Evidence: Test Report — "No Gradle build or test was run"
```

Unconfirmed rows provide quick actions for a developer to mark helpful,
ignored, rejected, or stale without opening the global memory page.

### Copy

Use explicit labels:

- "recalled" means supplied to Agent;
- "applied" means Agent says it changed execution;
- "helpful" means result has evidence;
- "unconfirmed" means RunEngram does not know.

Never use "saved time" without measured baseline data.

All copy supports English and Simplified Chinese. Dracula remains default;
light theme stays supported.

## Efficiency interpretation

This iteration proves behavioral influence, not counterfactual time savings.

RunEngram may state:

- memory was recalled;
- Agent explicitly applied it;
- evidence shows changed execution;
- later task marked the result helpful;
- repeated tasks required fewer corrections or recoveries.

RunEngram may not state:

- exact hours saved;
- a task would have failed without memory;
- absence of a command proves memory was applied.

Later comparative metrics can group comparable work types and compare cycle
time, correction count, failed route count, and review returns after enough
samples exist.

## Error handling

- Recall impact write fails: return error; retry upsert remains safe.
- Typed evidence reference missing: reject final receipt with a clear
  validation error. Free-text evidence remains valid when `kind=text`.
- Task or capsule deleted: retain impact history without task FK, matching task
  event retention behavior.
- Concurrent receipt edits: use `updated_at` optimistic concurrency for web
  developer changes.
- Dynamic recall repeats an existing memory: merge recall metadata; never
  regress applied or final state.
- Completion finalization fails: task completion succeeds; background-free
  retry occurs on next task/metrics read through idempotent reconciliation.

## Testing

### Store

- unique task/capsule upsert;
- state progression without regression;
- historical task deletion preserves impact;
- project/task/capsule filters;
- metrics count recall, application, helpful, ignored, and unconfirmed
  correctly.

### Service

- task-start recall creates receipts;
- dynamic recall adds new receipts;
- repeated recall is idempotent;
- `used` does not change confidence;
- helpful/rejected/stale mirror legacy usage;
- completion marks unresolved recall unconfirmed;
- historical backfill creates recall only;
- cross-project receipt rejected.

### CLI and API

- `ignored` accepted;
- evidence flags encoded;
- old `capsule use --outcome helpful --notes ...` remains compatible;
- new list endpoints filter correctly;
- JSON output remains stable for Agents.

### Web

- funnel separates recall from application;
- memory impact history shows source task and evidence;
- task panel distinguishes applied, ignored, and unconfirmed;
- quick resolution preserves evidence draft on failure;
- English, Simplified Chinese, Dracula, and light themes render correctly.

### Browser smoke

Seed one memory and two tasks:

1. first task recalls then applies and confirms helpful;
2. second task recalls but completes without decision.

Verify project funnel, memory history, task panels, and unconfirmed quick
action against a real rebuilt server.

## Acceptance criteria

- Existing context snapshots appear as recalled history after upgrade.
- A new task automatically records every memory supplied to its context.
- Agent can explicitly record applied, ignored, helpful, rejected, or stale.
- Dashboard no longer reports zero merely because recall and usage were
  conflated.
- Memory detail explains which tasks were affected and links evidence.
- Task detail explains how memory changed execution or states that impact is
  unconfirmed.
- Confidence changes only from confirmed helpful, rejected, or stale outcomes.
- No current API, CLI, skill, or persisted usage workflow breaks.
- Full Go, CLI, web, skill, migration, and browser smoke tests pass.
