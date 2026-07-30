# Workflow Adapters and Work Graphs

## Decision

Extend Agent Run with a durable, workflow-neutral outer graph. Do not add a
second Agent engine.

- Project skills, commands, and human playbooks own domain behavior.
- Codex, Claude Code, Pi, or another Agent owns its autonomous loop inside each
  node.
- RunEngram owns task state, dependency checks, durable receipts, evidence,
  interrupts, recovery, and verified learning.
- SQLite remains the only runtime dependency.

## Adapter contract

A Workflow Adapter is portable JSON:

```json
{
  "template": "content-review",
  "version": 1,
  "nodes": [
    {
      "key": "draft",
      "title": "Draft",
      "capability": "writing",
      "kind": "agent-loop",
      "depends_on": []
    },
    {
      "key": "review",
      "title": "Review",
      "capability": "verification",
      "kind": "evaluator",
      "depends_on": ["draft"]
    }
  ]
}
```

`template` and node keys are lowercase slugs. Definitions contain 1–32 unique
nodes. Allowed node kinds are `agent-loop`, `tool`, `evaluator`, and `human`.
The server rejects missing dependencies, self-dependencies, duplicate keys, and
cycles before creating a run.

Start a custom flow:

```bash
runengram run start <task-id> --agent-tool codex \
  --workflow content-review \
  --workflow-file examples/workflows/content-review.json
```

The built-in `engineering-flow` requires no file:

```bash
runengram run start <task-id> --agent-tool codex \
  --workflow engineering-flow
```

## Durable runtime state

Every node stores:

- key, title, capability, kind, dependencies, and order;
- status and attempt;
- input fingerprint;
- summary and next action;
- task artifact references and verification evidence;
- start, completion, and update timestamps.

Human questions are typed interrupts attached to a node. They survive process
restart and remain visible until answered or rejected. Upstream changes
invalidate downstream receipts. Human gates require a fresh response from an
actor other than the executing Agent.

## Boundary

```text
Project workflow / SOP
          │ maps outputs through Workflow Adapter
          ▼
RunEngram Work Graph
          │ stores state, receipts, evidence, interrupts
          ▼
Codex / Claude Code / Pi loops inside each node
```

RunEngram never interprets hidden reasoning, invokes a project SOP by name, or
claims that one stage model fits every project. A short task can stay on
`single-loop`. A custom graph can represent development, bug investigation,
documentation, research, release, or another repeatable flow.

## Product surface

- CLI starts built-in or custom workflows.
- CLI records node receipts and structured interrupts.
- `GET /tasks/:id/resume` returns the current Work Graph.
- Action Console renders node titles from the adapter and shows progress,
  evidence, artifacts, decisions, and recalled memory.

No estimated “time saved” is shown. Only observed facts are displayed:
completed stages, verified stages, linked artifacts, resumed runs, recalled
memories, and unresolved decisions.
