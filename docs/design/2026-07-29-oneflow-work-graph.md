# One-flow Work Graph

## Decision

Extend the existing Agent Run instead of adding a second orchestration runtime.

- RunEngram owns task state, durable node receipts, evidence, interrupts, and
  verified learning.
- one-flow owns the CamScanner requirement-development SOP.
- Codex, Claude Code, Pi, or another coding agent keeps its autonomous loop
  inside each work node.
- SQLite remains the only runtime dependency.

## User cases

1. Start a normal task with a single Agent loop.
2. Start a CamScanner requirement with the `cs-one-flow` template.
3. Resume after a new Codex session without repeating completed stages.
4. See which stage is active, what it produced, and why it passed.
5. Pause at requirement/final gates and resume after a structured response.
6. Keep completed parallelizable work reusable in a later Graph version.
7. Capture only verified corrections and recoveries as project memory.

## Work Graph Lite

An `AgentRun` optionally selects a versioned workflow template. The first
template is `cs-one-flow`:

```text
requirement-analysis
  → technical-design
  → task-planning
  → implementation
  → refactor
  → verification
  → code-review
  → final-gate
```

Every node stores:

- key, title, capability, kind, dependencies, and order;
- status and attempt;
- input fingerprint;
- summary and next action;
- artifact references and verification evidence;
- start, completion, and update timestamps.

Human questions are typed interrupts attached to a node. They survive process
restart and remain visible until answered or rejected.

## Layering

| Layer | Responsibility |
| --- | --- |
| Capability | SQLite receipts, events, artifacts, interrupts, agent adapters |
| Framework | versioned Work Graph template, dependency validation, resume |
| Strategy | `single-loop`, `cs-one-flow`, future bug/docs/research templates |

## Product surface

- CLI starts a run with `--workflow cs-one-flow`.
- CLI records node receipts and structured interrupts.
- `GET /tasks/:id/resume` returns the current Work Graph.
- Action Console shows stage progress, current action, evidence, artifacts,
  pending decisions, recalled memory, and honest efficiency counters.

No estimated “time saved” is shown. Only observed facts are displayed:
completed reusable stages, verified stages, linked artifacts, resumed runs,
recalled memories, and unresolved decisions.

## Evolution

P0 keeps a linear built-in template. Future migrations may add graph
definitions, fan-out/fan-in, partial-success replay, and capability routing
without changing the Agent-facing node receipt protocol.
