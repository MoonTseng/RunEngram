# Product

RunEngram is verified engineering memory for coding agents.

For implementation details see [`ARCHITECTURE.md`](./ARCHITECTURE.md). For
repository conventions see [`AGENTS.md`](./AGENTS.md).

## The problem

Coding agents can accelerate one task while leaving the engineering system no
smarter afterward.

A developer explains the same architecture boundaries in a new session.
Another agent searches the same entry points and repeats the same failed
commands. A useful workaround remains inside a transcript. A board records
that work moved from development to done, but none of that verified experience
improves the next task.

This creates four costs:

1. **Repeated context:** requirements, constraints, and decisions must be
   reconstructed for every run.
2. **Repeated exploration:** agents rediscover code paths, dependencies, and
   valid verification commands.
3. **Untrusted memory:** raw transcripts mix facts, guesses, obsolete code, and
   environment-specific accidents.
4. **Invisible value:** teams can measure task counts but not exploration,
   rework, or manual effort saved by AI.

## Product thesis

Every agent run should leave verified, reusable engineering memory.

RunEngram connects four records:

```text
Work -> Run -> Evidence -> Learning
  ^                           |
  +---------------------------+
```

- **Work** captures the desired outcome, scope, dependencies, priority, and
  acceptance criteria.
- **Run** captures who or what executed the work, which context it received,
  and how execution progressed.
- **Evidence** captures code changes, tests, reviews, commands, and observable
  results.
- **Learning** captures only reusable conclusions with provenance, scope,
  freshness, and invalidation rules.

The loop matters more than any individual screen. Kanban is one view over work;
it is not the product.

## Current product boundary

The current alpha ships the **task execution and learning kernel**:

- a canonical seven-state workflow;
- a dependency DAG and server-side runnable ordering;
- atomic claims, leases, heartbeats, and recovery;
- Markdown documents, images, links, and labels;
- append-only task operation history;
- manual review and completion, with optional PR, CI, and document evidence;
- JSON-first CLI and an agent-facing skill;
- an embedded bilingual web UI with an action-first default workspace,
  Kanban, dependency graph, engineering-memory inspection, and Dracula/paper
  themes;
- a native Codex marketplace plugin with checksum-verified local runtime
  installation and upgrade;
- local SQLite and filesystem storage;
- immutable task-start context snapshots;
- tool-neutral Agent runs with normalized events, compact checkpoints,
  interruption recovery, and completion/recovery metrics;
- an optional `cs-one-flow` Work Graph: eight dependency-checked stages,
  per-stage artifacts and evidence, typed human interrupts, and complete
  cross-session restore;
- verified Exploration Capsules with project scope, evidence, fingerprints,
  source task, and producer;
- agent-driven capture of human corrections and successful recovery paths as
  pending learning notes;
- capture of explicit reusable project conventions, visible learning receipts,
  and manual correction of pending candidates before promotion;
- evidence-gated, idempotent promotion from one pending learning note to one
  active Exploration Capsule, plus explicit rejection;
- deterministic same-project recall;
- observed helpful, rejected, and stale reuse outcomes plus honest aggregate
  metrics;
- candidate, pending, promoted, rejected, and promotion-rate visibility in the
  CLI and web Knowledge view.

The current binary and environment-variable names retain `taskline` for
compatibility. RunEngram is the public product name.

The alpha does **not** passively ingest transcripts, autonomously rewrite
skills or tests, or claim causal time-saved estimates. Capture is driven by
the agent skill; promotion requires explicit verification evidence. Automatic
evidence-to-skill/test/rule enforcement and richer learning-lift measurement
remain roadmap capabilities.

## Product principles

### 1. Agent-compatible, human-legible

Agents need stable JSON, deterministic exit codes, explicit state, and cheap
queries. Humans need readable Markdown, visible dependencies, explainable
history, and a small number of useful views. Both operate on the same model.

### 2. Evidence before memory

Transcripts are inputs, not truth. A reusable learning must preserve:

- source run and source task;
- project and module scope;
- supporting verification;
- relevant code or dependency fingerprint;
- expiration or invalidation condition.

If a claim cannot explain why it is trusted and when it becomes stale, it must
not be injected automatically.

### 3. Local-first by default

One service and one SQLite file should be enough for a developer or small team
pilot. No Redis, PostgreSQL, vector database, or cloud account is required.
Project source and private task documents stay under the operator's control.

### 4. Tool and SOP agnostic

RunEngram coordinates work around coding agents; it does not own their internal
reasoning process. Codex, Claude Code, Cursor, custom agents, and team SOPs can
map their local phases onto the same work protocol.

Graph is selective, not mandatory. Short docs, research, and tightly coupled
changes keep one flexible Agent loop. Long requirement work uses a Work Graph
when context decay, independent verification, parallelism, or explicit human
decisions justify the extra structure. A project-specific SOP such as
`cs-sop-one-flow` supplies domain behavior; RunEngram supplies durable state and
receipts.

### 5. Reversible work, append-only evidence

Real work moves backward. Review can reveal a defect; testing can invalidate a
design assumption. Workflow state may move between known stages, while task
history and evidence remain append-only and explain why the move happened.

### 6. Small protocol, extensible learning

The task protocol stays compact:

`pending → start → spec → dev → test → review → done`

Learning assets may grow by type, but they must not require every team to adopt
the same development SOP.

## Why these seven states

- `pending`: recorded but not runnable;
- `start`: clear enough to claim;
- `spec`: requirements, UX, scope, and acceptance criteria;
- `dev`: technical design and implementation;
- `test`: local build, tests, and regression verification;
- `review`: PR, review conversations, and CI;
- `done`: delivery evidence satisfies the configured gate.

A separate `blocked` state is unnecessary when an unfinished dependency
already makes work non-runnable. A task needing human input can retain its
current state, evidence, and explicit dependency without creating a second
source of truth.

## Learning model

The intended learning path has four levels:

1. **Run artifact:** task-specific description, document, command, or result.
2. **Learning note:** candidate reusable insight with source and scope.
3. **Project knowledge:** reviewed context available to future tasks.
4. **Enforced knowledge:** skill, lint rule, test, template, or workflow gate.

The current agent protocol automatically captures two high-signal candidate
types: a human correction that changed execution, and an agent recovery where
a failed path was replaced by a verified successful path. Capture alone never
changes future behavior. Promotion is explicit, requires evidence, and creates
one active project-memory capsule atomically. Repeated evidence increases
confidence; conflicts require review.

## Success metrics

RunEngram succeeds only when developers feel less repeated work. Useful pilot
metrics include:

- time from task claim to first valid implementation change;
- repeated code-search and failed-command count;
- number of context clarification turns;
- reopen or rollback rate after review;
- manual interventions per completed task;
- percentage of runs reusing a verified learning;
- time saved when reused context is fresh;
- stale-learning rejection rate.

Task throughput alone is not enough. Faster output with higher rework is not a
learning improvement.

## Non-goals

- replacing GitHub, GitLab, CI, or code review;
- replacing a coding agent or prescribing one universal SOP;
- storing complete hidden reasoning or treating transcripts as authoritative;
- autonomous team-wide rule changes without review;
- becoming a general-purpose Jira or enterprise portfolio manager;
- requiring a hosted SaaS control plane for local use.

## Roadmap order

1. ✅ Stabilize one-task execution, evidence, recovery, and Codex plugin
   packaging.
2. ✅ Add immutable context snapshots and freshness fingerprints.
3. ✅ Add verified exploration capsules and task-level observed reuse.
4. ✅ Add agent-driven learning-note capture, evidence-gated promotion, and
   candidate metrics.
5. ✅ Add first-class resumable runs, normalized events, visible learning
   receipts, and editable pending candidates.
6. ✅ Add selective One-flow Work Graphs, typed human interrupts, stage
   receipts, and an Action Console that exposes observed completion, evidence,
   artifacts, recalled context, and pending decisions.
7. Add optional human/team review policy and promotion into skills, tests,
   lint rules, templates, and workflow gates.
8. Add richer learning-lift measurement after pilot data exists.
9. Add optional small-team synchronization without weakening local-first use.

Each phase must demonstrate reduced repeated work before the next layer earns
more complexity.
