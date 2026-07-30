# RunEngram Public Release Design

## Decision

The project is published as **RunEngram** in
`MoonTseng/RunEngram`.

Product description:

> Verified engineering memory for coding agents.

Chinese description:

> 面向 AI 编程智能体的可验证研发记忆系统。

Tagline:

> Make every agent run improve the next.

## Positioning

RunEngram is not another visual task board and does not replace a coding agent
or an engineering SOP. It connects structured work, agent execution, evidence,
and reusable project context.

The product loop is:

```text
work context -> agent run -> verified evidence -> learning -> next run
```

The current repository ships the local task execution kernel. Context
snapshots, verified exploration capsules, learning promotion, and learning-lift
metrics are the next product layer. Public documentation must distinguish
working features from roadmap items.

## Naming migration

RunEngram is the public product name. Canonical binaries are
`runengram-server`, `runengram`, and `runengram-service`. Release archives keep
`taskline-server` and `taskline` compatibility symlinks. Go module paths,
`TASKLINE_*` environment keys, and `.config/taskline` remain stable so existing
data and automation do not require an atomic migration.

## Documentation

- `README.md` is the default English landing page.
- `README.zh-CN.md` is the complete Simplified Chinese landing page.
- Both pages share structure and product claims but use natural language rather
  than line-by-line translation.
- The web UI continues to support English and Simplified Chinese.
- Architecture internals may remain English-first; user workflows and pilot
  documentation should have Chinese coverage.

## Public release safety

The public branch must not contain:

- company repository names or internal workflow identifiers;
- private IP addresses, SSH commands, passwords, tokens, or internal domains;
- personal runtime paths or databases;
- task data from private projects.

Legacy local history stays local. GitHub `main` starts from a sanitized source
snapshot so removed internal pilot details do not remain reachable in public
Git history.

## README structure

The landing page leads with:

1. one-sentence outcome;
2. concrete developer pain;
3. current capabilities;
4. honest product direction;
5. copyable quick start;
6. architecture and contribution links;
7. license.

This keeps the first screen useful to a new developer while retaining enough
detail for technical evaluation.
