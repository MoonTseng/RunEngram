# RunEngram Dashboard UX Refresh

**Date:** 2026-07-28  
**Status:** Approved for implementation planning  
**Scope:** Embedded web UI, documentation screenshots, and frontend API wrappers  
**Out of scope:** New persistence models, hosted services, team permissions, and changes to the agent execution protocol

## Summary

RunEngram will replace its board-first entry screen with a state-aware action console. The UI will default to a Dracula-based dark theme and retain the current warm paper theme as an option.

New users will see a concise internal-tool introduction and setup path. Active users will see the next action, current Agent activity, blockers, and memory already recalled into the task's immutable context. The engineering-memory view will show why each item matched, why it is trusted, where it came from, and whether its evidence remains fresh.

The refresh reorganizes existing capabilities. It does not add a database, background worker, knowledge-graph engine, or second source of truth.

## Problem

The current UI exposes useful capabilities but makes users assemble the product model themselves:

- The board shows task state but does not lead users to the next action.
- A first-time user cannot tell what RunEngram does or how it works with Codex.
- Engineering memory appears as a list of cards. Users must inspect each card to understand relevance and trust.
- The light Wabi-sabi theme has character, but the tool lacks a focused developer-oriented dark mode.
- Metrics occupy space without always explaining what action they support.
- Terms such as Capsule, fingerprint, and promotion describe internals rather than user goals.

The result feels like a visual task tracker. RunEngram's main value—turning verified execution experience into context for later Agent runs—remains secondary.

## Goals

1. Let a returning user find the next useful action within ten seconds.
2. Let a first-time user explain RunEngram's purpose after reading one screen.
3. Put recalled engineering memory next to the task it can help.
4. Show provenance, evidence, scope, freshness, and observed usefulness before asking users to trust memory.
5. Make the interface comfortable for long developer sessions.
6. Preserve existing Kanban, dependency graph, search, task editing, and learning workflows.
7. Keep RunEngram honest about its internal Alpha status and current limits.

## Non-goals

- Build a public SaaS landing page.
- Replace Codex, Claude Code, Cursor, GitHub, or existing engineering SOPs.
- Add project portfolio management, team permissions, chat, or notifications.
- Add passive transcript collection.
- Add editable knowledge graphs.
- Mutate an existing Context Snapshot.
- Estimate causal time savings without pilot evidence.
- Depend on another project's UI implementation.

## Research Synthesis

The design borrows interaction principles, not code or visual identity:

| Reference | Useful principle | RunEngram application |
| --- | --- | --- |
| [Plane](https://github.com/makeplane/plane) | Work items, modules, and cycles keep actions close to project context. | Keep task actions visible and move the complete board to a secondary view. |
| [Huly](https://github.com/hcengineering/platform) | A persistent workspace shell makes related tools feel connected. | Use one left navigation for action console, board, graph, memory, and review queue. |
| [AppFlowy](https://github.com/AppFlowy-IO/AppFlowy) | Comfortable spacing and readable content make dense knowledge usable. | Use a list-detail memory layout with restrained density. |
| [Langfuse](https://github.com/langfuse/langfuse) | Traces and metrics become useful when tied to inspectable evidence. | Pair learning metrics with source tasks, evidence, and observed reuse. |
| [Mem0](https://github.com/mem0ai/mem0) | Agent memory needs scoped retrieval and explicit management. | Keep memory project-scoped and expose search, status, and usage. |
| [Graphiti](https://github.com/getzep/graphiti) | Provenance and temporal validity matter when knowledge changes. | Show source, creation date, code fingerprint freshness, and invalidation state. |
| [Dracula](https://github.com/dracula/dracula-theme) | A stable semantic palette supports developer tools and WCAG AA contrast. | Map Dracula colors to RunEngram semantic design tokens. |

RunEngram will not depend on `dracula-ui`. That repository is archived and no longer maintained. The implementation will use RunEngram's existing Tailwind and CSS-variable stack with semantic tokens derived from the open Dracula palette.

## Product Positioning

RunEngram remains an internal engineering tool in early Alpha. The first-run screen must say:

> RunEngram is an engineering Agent task and experience tool. It sends structured work to coding agents, records execution and verification evidence, and gives verified experience to later tasks.

The screen will display:

- `INTERNAL ALPHA`;
- local server health and storage location;
- current capabilities and limits;
- three setup steps;
- direct links to the README, operating guide, and issue reporting.

The screen will not use marketing claims, large promotional gradients, customer logos, or maturity signals that the current tool cannot support.

## Information Architecture

Each project workspace uses one persistent navigation:

1. **Action Console** — default returning-user view.
2. **Task Board** — complete seven-state workflow.
3. **Dependencies** — existing React Flow graph.
4. **Engineering Memory** — recalled, verified, pending, and stale experience.
5. **Verification Queue** — pending learning notes that need promotion or rejection.

Project selection remains in the left shell. Locale, theme, search, help, and task creation remain available from the header.

The web URL keeps `project=<name|id>`. The new default view uses `view=action`; existing saved `view=kanban`, `view=graph`, and `view=knowledge` links continue to work.

## State-aware Entry Screen

The entry screen depends on available project and task data:

| Condition | Screen | Primary action |
| --- | --- | --- |
| No projects | Internal-tool introduction | Create project workspace |
| Project exists, no tasks | Setup checklist | Create first task and copy Agent prompt |
| Runnable task exists, no live claim | Ready state | Copy prompt or open task |
| Live claim exists | Action console | Continue Agent execution |
| Recent task completed, no next task | Outcome state | Review evidence and learning candidates |

The current local-first product assumes one operator. The action console
therefore selects a live project claim when one exists. Future team identity
can refine this selector without changing the screen structure.

If several live claims exist, the console selects the task with the latest
lease expiry, then priority and creation order. A task switcher exposes the
other live claims.

### First-run setup

The setup checklist contains three steps:

1. Create or select a RunEngram project workspace.
2. Install the Agent Skill and verify the local service with documented commands.
3. Create a task and copy the generated `taskline-management` prompt into the coding Agent opened in the target source repository.

The browser does not claim that it can discover or bind a local source path. Repository association still comes from the CLI environment and current working directory.

## Action Console

The action console uses three levels of information.

### Level 1: Next action

The largest card shows:

- task title, type, priority, owner, and labels;
- current workflow state;
- live claim age and producer;
- satisfied and blocking dependencies;
- concise next action;
- Agent prompt or continuation action.

If no task can run, the card explains the blocker and links to the blocking task. It never presents a disabled button without a reason.

### Level 2: Recalled memory

The adjacent panel shows up to three Exploration Capsules already present in the task's immutable Context Snapshot. Each item shows:

- title;
- match reason;
- observed helpful ratio when available;
- freshness status;
- link to evidence and source task.

The panel does not query unrelated memory and imply that the Agent received it. It distinguishes memory frozen into the Context Snapshot from memory found later through manual search.

### Level 3: Project overview

Below the focus area:

- a compact workflow summary links to the full board;
- learning indicators show recalls, helpful feedback, pending review, and stale items;
- recent evidence or learning warnings appear only when they require action.

Management metrics remain secondary. The first viewport prioritizes execution.

## Engineering Memory

The engineering-memory view replaces the current equal-card grid with a list-detail layout.

### Navigation

Tabs:

- Current task;
- Verified;
- Pending review;
- Needs revalidation.

A memory relationship graph remains roadmap work. The first release favors
clear provenance and filters over another visualization.

### Memory list

Items are ranked by task relevance and observed usefulness. Each row shows:

- human-readable kind;
- guidance title;
- match reason;
- scope and labels;
- source task and date;
- observed helpful count;
- freshness state.

### Detail panel

Selecting an item shows:

- summary and intended use;
- reason it is trusted;
- verification evidence;
- provenance timeline;
- scope and fingerprints;
- reuse history and feedback;
- links to source task and related evidence.

The detail panel offers **Copy guidance**, **Open source task**, and **Record feedback**. It does not offer **Add to current context**, because Context Snapshots are immutable.

### User-facing language

Primary UI labels translate internal terms:

| Internal term | Primary UI label |
| --- | --- |
| Exploration Capsule | Verified experience |
| Learning Note | Experience candidate |
| Promotion | Mark as verified |
| Rejection | Reject candidate |
| Fingerprint mismatch | Needs revalidation |
| Suggested Capsules | Recalled experience |

Technical names may remain in API output, CLI output, and expandable details.

## Theme System

### Default

The app defaults to `dracula` when no stored preference exists. Users can switch to `paper`. The browser saves the explicit choice in `localStorage`.

The app sets `data-theme="dracula|paper"` on the root shell. Components consume semantic variables and never reference theme-specific colors directly.

### Dracula semantic tokens

| Token | Value | Use |
| --- | --- | --- |
| `--tl-bg` | `#282a36` | Workspace background |
| `--tl-bg-quiet` | `#21222c` | Sidebar and quiet regions |
| `--tl-surface` | `#30323f` | Cards and panels |
| `--tl-surface-raised` | `#44475a` | Controls and selected regions |
| `--tl-ink` | `#f8f8f2` | Primary text |
| `--tl-ink-muted` | `#a7a9ba` | Secondary text |
| `--tl-ink-faint` | `#9aa2c7` | Metadata |
| `--tl-outline` | `#44475a` | Borders and separators |
| `--tl-indigo` | `#6272a4` | Decorative accents, never small text |
| `--tl-primary` | `#bd93f9` | Primary action and selection |
| `--tl-info` | `#8be9fd` | Active execution |
| `--tl-success` | `#50fa7b` | Verified and healthy |
| `--tl-warning` | `#f1fa8c` | Pending review |
| `--tl-danger` | `#ff5555` | Failed or invalid |
| `--tl-priority` | `#ffb86c` | High priority |

The paper theme retains the current warm palette after contrast and token-name cleanup.

### Visual rules

- Use 14–16 px body text; metadata must remain at least 12 px in the real UI.
- Limit cards to one shadow level and one border level.
- Use color to reinforce labels, never as the only state indicator.
- Keep motion under 200 ms and respect `prefers-reduced-motion`.
- Maintain visible keyboard focus.
- Meet WCAG 2.1 AA contrast for text and controls.

## Frontend Architecture

### New units

- `ThemeProvider` — resolves, applies, persists, and toggles theme.
- `WorkspaceNavigation` — owns project and view navigation.
- `WelcomeView` — internal-tool introduction and first-run setup.
- `ActionConsole` — renders next-action, recalled-memory, workflow, and learning panels.
- `useActionConsole(projectId)` — composes cached project queries.
- `useTaskContext(taskId, enabled)` — reads context only when a live claim permits it.
- `selectFocusTask(tasks)` — deterministic focus-task selection.

### Refactoring

`KnowledgeView` currently owns queries, metrics, filters, candidates, capsules, and evidence rendering in one component. Split it into:

- `MemoryMetrics`;
- `MemoryTabs`;
- `MemoryList`;
- `MemoryDetail`;
- `CandidateList`.

Each unit receives data through explicit props. Query orchestration remains at view level.

### Existing APIs

The first implementation uses:

- `GET /api/v1/projects`;
- `GET /api/v1/projects/:project/tasks`;
- `GET /api/v1/projects/:project/tasks/next`;
- `GET /api/v1/tasks/:id/context`;
- `GET /api/v1/projects/:project/capsules`;
- `GET /api/v1/projects/:project/learning-notes`;
- `GET /api/v1/projects/:project/learning-metrics`.

The web client needs a typed `ContextSnapshot` shape and `getTaskContext` wrapper. The server already exposes this endpoint and model.

Do not add a dashboard aggregation endpoint in this phase. TanStack Query already caches the required local requests, and the local SQLite service makes their cost small. Pilot data may justify aggregation later.

## Data Flow

### Focus task

1. Load project tasks.
2. Select the most recently renewed task with a live claim.
3. Otherwise preview the server's next runnable task.
4. Otherwise show blocked, empty, or completed state.

The server owns runnable ordering. The client uses lease expiry, priority, and
creation time only to choose among live claims. UI tests lock that ordering.

### Recalled memory

1. Enable the context query only for a valid live claim.
2. Read or create the task's immutable Context Snapshot through the existing endpoint.
3. Render `suggested_capsules` as recalled experience.
4. Record helpful, rejected, or stale feedback through the existing usage endpoint.

Manual memory searches do not alter the frozen snapshot.

### Theme

1. Read `runengram.theme` from `localStorage`.
2. Use `dracula` when the key is absent or invalid.
3. Apply `data-theme` before rendering interactive content.
4. Persist explicit changes.

## Error and Empty States

| Condition | UI response |
| --- | --- |
| API unavailable | Show service status, expected URL, start command, and retry |
| No project | Explain project workspace and offer creation |
| No tasks | Show setup checklist and first-task form |
| No runnable task | List blocking dependencies or explain that all work is complete |
| Claim expired | Mark execution as interrupted and offer documented recovery |
| Context unavailable | Hide recalled-memory claims and explain why context could not load |
| No recalled memory | Explain that the current task has no verified matching experience |
| Learning metrics fail | Keep task actions usable and show a local retry inside learning panel |
| Theme storage unavailable | Use Dracula for the session without blocking the UI |

An error in learning data must not disable task execution.

## Accessibility and Responsive Behavior

- Desktop uses persistent navigation, focus area, and side detail panels.
- Narrow screens collapse navigation into a drawer.
- Action-console panels stack in execution order: next action, recalled memory, workflow, learning.
- Memory detail opens as a full-width panel or dialog on narrow screens.
- Every icon-only control has a label and tooltip.
- Keyboard users can navigate tabs, memory rows, actions, and dialogs.
- Scroll containers remain explicit; no content depends on hover.

## Testing

### Automated

- Theme default, persistence, invalid-value fallback, and both token sets.
- Entry-screen resolver for every lifecycle condition.
- Deterministic focus-task selector.
- Context query disabled without a live claim.
- Action-console loading, empty, blocked, expired, and error states.
- Memory tabs, list-detail selection, evidence disclosure, and feedback actions.
- Existing Kanban, graph, task editor, search, and i18n tests.
- Server, CLI, and Skill suites remain green.

### Manual

- Dracula and paper at desktop and narrow widths.
- Chinese and English copy.
- Keyboard-only navigation and visible focus.
- Contrast inspection against WCAG AA.
- First-time teammate can explain the tool within ten seconds.
- First-time teammate can create and start a task within three minutes after the server is running.
- Active task shows only memory frozen into its Context Snapshot.

## Documentation

Implementation includes:

- new Chinese and English screenshots captured from the real embedded UI;
- replacement of the current light-theme screenshots in `README.md` and `README.zh-CN.md`;
- updated description of the action console, first-run states, theme switch, and memory list-detail view;
- a short “What should I do first?” section;
- clear `INTERNAL ALPHA` or `early alpha` wording;
- no claim that RunEngram passively reads transcripts or autonomously changes team rules.

## Rollout

1. Add theme tokens, provider, persistence, and switch.
2. Add workspace navigation and state-aware welcome screens.
3. Add action console and typed task-context query.
4. Refactor engineering memory into list-detail view.
5. Validate empty, error, narrow-screen, and bilingual states.
6. Capture real screenshots and update both READMEs.
7. Run one-person and teammate pilot; record confusion points before adding more features.

## Acceptance Criteria

- Dracula is the default theme for a new browser.
- Users can switch to the paper theme, and the choice survives reload.
- New users see purpose, scope, and setup before an empty board.
- Returning users see the next action before project metrics.
- Recalled experience comes only from the active task's Context Snapshot.
- Memory details expose reason, evidence, provenance, scope, freshness, and feedback.
- Existing board, graph, task editing, search, and localization remain available.
- No database migration or external service is added.
- Automated frontend, server, CLI, and Skill tests pass.
- README English and Chinese screenshots show the shipped UI.
