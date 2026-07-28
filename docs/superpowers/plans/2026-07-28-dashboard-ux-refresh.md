# RunEngram Dashboard UX Refresh and Team Plugin Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Make RunEngram's value visible in daily work through an action-first dashboard and trustworthy engineering-memory UI, then package the full local workflow as a versioned Codex plugin for team installation.

**Architecture:** Keep Go server, SQLite store, embedded React UI, REST contract, and existing CLI unchanged at their boundaries. Add frontend-only selectors and API wrappers for action-console data. Package existing management skill plus setup/doctor workflows under a repo marketplace. GitHub releases provide prebuilt server and CLI artifacts; plugin setup installs those artifacts into user-local directories and starts the local service.

**Tech Stack:** React 19, TypeScript, TanStack Query, Tailwind CSS 4, Vitest, Go 1.25, Cobra, shell scripts, GitHub Actions, Codex plugin manifest/marketplace.

---

## Task 1: Theme foundation

**Files:**

- Create: `web/src/lib/theme.tsx`
- Create: `web/src/lib/theme.test.tsx`
- Modify: `web/src/main.tsx`
- Modify: `web/src/index.css`

- [ ] Write tests proving missing preference resolves to `dracula`, stored `paper` resolves to `paper`, switching persists `runengram.theme`, and provider sets `data-theme`.
- [ ] Run `cd web && pnpm test -- theme.test.tsx`; confirm failure because provider does not exist.
- [ ] Implement:

```ts
export type Theme = "dracula" | "paper";
export const THEME_STORAGE_KEY = "runengram.theme";

export function resolveTheme(value: string | null): Theme {
  return value === "paper" ? "paper" : "dracula";
}
```

- [ ] Add `ThemeProvider` and `useTheme`, wrap app in provider.
- [ ] Move current palette under `[data-theme="paper"]`; put Dracula values on `:root` and `[data-theme="dracula"]`.
- [ ] Replace light-only body texture, shadows, form inset, scrollbar, overlays, and React Flow control colors with semantic theme-safe tokens.
- [ ] Run focused tests and `pnpm lint`.
- [ ] Commit: `feat(web): add Dracula-first theme system`

## Task 2: Action-first workspace shell

**Files:**

- Create: `web/src/components/WorkspaceNav.tsx`
- Create: `web/src/components/WorkspaceNav.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/App.test.tsx`
- Modify: `web/src/lib/i18n.tsx`

- [ ] Write tests proving default view is `action`, saved `kanban|graph|knowledge` URLs remain valid, navigation exposes Action Console/Task Board/Dependencies/Engineering Memory, and theme toggle changes accessible label.
- [ ] Run `cd web && pnpm test -- App.test.tsx WorkspaceNav.test.tsx`; confirm new assertions fail.
- [ ] Extend view model:

```ts
type View = "action" | "kanban" | "graph" | "knowledge";

function parseViewKey(value: string | null): View {
  return value === "kanban" || value === "graph" || value === "knowledge"
    ? value
    : "action";
}
```

- [ ] Persist explicit URLs as `view=action|kanban|graph|knowledge`; retain old missing-view links as action default.
- [ ] Replace compact segmented toggle with readable workspace navigation. Keep narrow-screen layout accessible.
- [ ] Add theme toggle beside locale, search, help, and create-task actions.
- [ ] Add matching `en-US` and `zh-CN` strings.
- [ ] Run focused tests and lint.
- [ ] Commit: `feat(web): add action-first workspace navigation`

## Task 3: State-aware internal Alpha entry

**Files:**

- Create: `web/src/components/WorkspaceWelcome.tsx`
- Create: `web/src/components/WorkspaceWelcome.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/components/Sidebar.tsx`
- Modify: `web/src/lib/i18n.tsx`

- [ ] Write tests for no-project introduction, unresolved project warning, selected empty-project checklist, and links to README/operating guide/issues.
- [ ] Run focused tests; confirm failure.
- [ ] Implement internal-tool positioning:

```text
RunEngram sends structured engineering work to coding agents,
records verification evidence, and recalls verified experience
for later tasks.
```

- [ ] Show `INTERNAL ALPHA`, local-only boundary, existing capability list, current limitations, and three-step setup. Avoid SaaS marketing language.
- [ ] For empty project, show create-task action and exact `taskline-management` starter prompt.
- [ ] Keep project creation in sidebar; make relation between project workspace and source repository explicit.
- [ ] Run focused tests and lint.
- [ ] Commit: `feat(web): clarify first-run and empty workspace`

## Task 4: Context snapshot frontend contract

**Files:**

- Modify: `web/src/lib/api.ts`
- Modify: `web/src/lib/api.test.ts`
- Modify: `web/src/hooks/queries.ts`
- Modify: `web/src/hooks/queries.test.tsx`

- [ ] Add failing tests for encoded `GET /api/v1/tasks/:id/context`, `ContextSnapshot` shape, and query enablement only when a selected task exists.
- [ ] Run focused tests; confirm failure.
- [ ] Mirror server contract:

```ts
export interface ContextSnapshot {
  id: string;
  task_id: string;
  project_id: string;
  task: Task;
  suggested_capsules: ExplorationCapsule[];
  created_at: number;
}
```

- [ ] Add `getTaskContext(taskId)` and `useTaskContext(taskId)`.
- [ ] Preserve current HTTP client headers and error mapping.
- [ ] Run focused tests and lint.
- [ ] Commit: `feat(web): expose immutable task context snapshots`

## Task 5: Action Console selectors and UI

**Files:**

- Create: `web/src/lib/actionConsole.ts`
- Create: `web/src/lib/actionConsole.test.ts`
- Create: `web/src/components/ActionConsole.tsx`
- Create: `web/src/components/ActionConsole.test.tsx`
- Modify: `web/src/App.tsx`
- Modify: `web/src/lib/i18n.tsx`

- [ ] Write selector tests for latest live claim, runnable start task, dependency blocker, completed outcome, and deterministic tie-breaking.
- [ ] Write component tests for loading, error, empty, blocked, ready, active, and outcome states.
- [ ] Run focused tests; confirm failure.
- [ ] Implement pure selector:

```ts
export type ActionFocus =
  | { kind: "active"; task: Task }
  | { kind: "ready"; task: Task }
  | { kind: "blocked"; task: Task; blockers: Task[] }
  | { kind: "outcome"; task: Task }
  | { kind: "empty" };
```

- [ ] Build Level 1 focus card: title, type, priority, owner, labels, state, claim age, dependencies, reasoned next action, and starter/continuation prompt.
- [ ] Build Level 2 recalled-experience panel from immutable Context Snapshot only. Show source, scope, observed helpful ratio, status, and evidence expansion. Explain when context is unavailable before claim.
- [ ] Build Level 3 workflow/learning overview with links to board and memory.
- [ ] Do not claim automatic agent launch from browser.
- [ ] Run focused tests and lint.
- [ ] Commit: `feat(web): add state-aware action console`

## Task 6: Engineering Memory list-detail experience

**Files:**

- Create: `web/src/components/MemoryMetrics.tsx`
- Create: `web/src/components/MemoryList.tsx`
- Create: `web/src/components/MemoryDetail.tsx`
- Create: `web/src/components/MemoryCandidates.tsx`
- Modify: `web/src/components/KnowledgeView.tsx`
- Modify: `web/src/components/KnowledgeView.test.tsx`
- Modify: `web/src/lib/i18n.tsx`

- [ ] Expand tests for Current task/Verified/Pending review/Needs revalidation tabs, selected detail, provenance, evidence, freshness, source task, usage feedback, and copy guidance.
- [ ] Run `cd web && pnpm test -- KnowledgeView.test.tsx`; confirm failure.
- [ ] Split current 291-line component into query coordinator plus focused presentational components.
- [ ] Translate internal vocabulary:

```text
Exploration Capsule -> Verified experience
Learning Note -> Experience candidate
stale -> Needs revalidation
promoted -> Verified
```

- [ ] Use list-detail desktop layout and stacked mobile layout. Keep filters, loading/error/empty states, and pending candidates.
- [ ] Rank active memory by observed helpful ratio, use count, then freshness without changing backend ordering contract.
- [ ] Show trust as evidence and provenance, not a generated score.
- [ ] Run focused tests and lint.
- [ ] Commit: `feat(web): make engineering memory inspectable`

## Task 7: Accessibility, responsive polish, full frontend validation

**Files:**

- Modify: `web/src/App.tsx`
- Modify: `web/src/components/ActionConsole.tsx`
- Modify: `web/src/components/WorkspaceNav.tsx`
- Modify: `web/src/components/KnowledgeView.tsx`
- Modify: `web/src/index.css`
- Modify: related frontend tests

- [ ] Add visible focus states, active navigation semantics, minimum control sizes, readable base/secondary type, keyboard navigation, and narrow-screen overflow behavior.
- [ ] Ensure each board column remains independently vertically scrollable.
- [ ] Remove hardcoded red/light-only colors touched by new screens.
- [ ] Run `cd web && pnpm lint && pnpm test && pnpm build`.
- [ ] Commit: `fix(web): polish responsive and accessible workspace`

## Task 8: Team Codex plugin and marketplace

**Files:**

- Create: `.agents/plugins/marketplace.json`
- Create: `plugins/runengram/.codex-plugin/plugin.json`
- Create: `plugins/runengram/skills/taskline-management/SKILL.md`
- Create: `plugins/runengram/skills/runengram-setup/SKILL.md`
- Create: `plugins/runengram/skills/runengram-doctor/SKILL.md`
- Create: `plugins/runengram/scripts/install-runengram.sh`
- Create: `plugins/runengram/scripts/runengram-service.sh`
- Create: `scripts/sync-plugin-skill.sh`
- Modify: `scripts/test-skill.sh`

- [ ] Scaffold repo/team plugin and marketplace with `plugin-creator`; keep manifest name `runengram`, strict semver, real GitHub metadata, bundled skills, scripts, and assets only.
- [ ] Copy canonical management skill through `scripts/sync-plugin-skill.sh`; test byte equality to prevent drift.
- [ ] Write setup skill that checks supported OS/architecture, downloads selected/latest GitHub release, verifies SHA-256 manifest, installs into `~/.local/share/runengram`, links CLI into `~/.local/bin`, starts service, waits for health, and opens local UI.
- [ ] Write doctor skill that verifies binary versions, service health, CLI identity, writable data/docs directories, and emits exact repair commands.
- [ ] Write service wrapper supporting `start|stop|restart|status|open` with explicit PID/log/data paths. Never kill broad process patterns.
- [ ] Document security boundary: plugin installation itself runs no hidden post-install script; first setup invocation performs visible user-local installation.
- [ ] Run plugin validator:

```bash
python3 /Users/yue_zeng/.codex/skills/.system/plugin-creator/scripts/validate_plugin.py plugins/runengram
```

- [ ] Run `./scripts/test-skill.sh` and shell syntax checks.
- [ ] Commit: `feat(plugin): package RunEngram for Codex teams`

## Task 9: Versioned release artifacts

**Files:**

- Create: `scripts/package-release.sh`
- Create: `scripts/package-release.test.sh`
- Create: `.github/workflows/release.yml`
- Modify: `scripts/build.sh`
- Modify: `cli/cmd/version.go`

- [ ] Write shell test proving archive names, required files, executable bits, and checksum manifest.
- [ ] Run test; confirm failure before script exists.
- [ ] Package embedded-web server, CLI, service wrapper, license, and quick-start for `darwin/arm64`, `darwin/amd64`, `linux/arm64`, and `linux/amd64`.
- [ ] Generate `SHA256SUMS`; installer must reject missing or mismatched checksum.
- [ ] Add tag-triggered and manual GitHub Actions release workflow. Build web once per job, cross-compile with `CGO_ENABLED=0`, upload archives/checksums, and attach to GitHub Release.
- [ ] Keep release version injected through existing CLI version variable rather than editing source per release.
- [ ] Run package test and local Darwin package smoke test.
- [ ] Commit: `ci: publish RunEngram team release bundles`

## Task 10: README positioning, comparison, installation, and screenshots

**Files:**

- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `PRODUCT.md`
- Modify: `使用说明.md`
- Create: `docs/assets/runengram-action-console.png`
- Create: `docs/assets/runengram-memory.png`
- Create: `docs/assets/runengram-action-console-light.png`

- [ ] Start rebuilt embedded server with isolated temporary data and deterministic demo fixtures.
- [ ] Capture real Dracula Action Console, Engineering Memory, and paper-theme screenshots at README-safe viewport.
- [ ] Put screenshots near top of both READMEs; keep English/Chinese language switch links working.
- [ ] State innovation precisely:
  - execution state + immutable task context + verified evidence + reusable engineering memory in one local loop;
  - memory is scoped, inspectable, freshness-aware, and feedback-measured;
  - agent/SOP agnostic contract with Codex-first plugin;
  - local single-binary operation and Markdown/SQLite ownership.
- [ ] Add evidence-based comparison:

| Capability | RunEngram | GitHub Copilot Memory | Claude Code Memory | OpenHands | LinearB |
| --- | --- | --- | --- | --- | --- |
| Multi-agent task workflow | Yes | No | No | Agent execution | Analytics |
| Immutable per-run context | Yes | No | No | Conversation state | No |
| Evidence-backed reusable experience | Yes | Citation-validated facts | Editable local notes | No | Delivery evidence |
| Freshness/revalidation | Fingerprint state | Citation validation | Manual | No | Metric windows |
| Learning feedback metrics | Helpful/rejected reuse | No | No | Evaluation-oriented | AI adoption/productivity |
| Local-first source ownership | SQLite + Markdown | Hosted | Machine-local files | Local/remote | Hosted |

- [ ] Mark comparison date and link official product documentation. Separate “available now” from roadmap.
- [ ] Add team plugin flow:

```bash
codex plugin marketplace add MoonTseng/RunEngram --ref main
codex plugin marketplace upgrade runengram
codex plugin add runengram@runengram
```

- [ ] Explain new thread requirement after plugin update and first-use `runengram-setup`.
- [ ] Run Markdown link checks available in repo and `./scripts/test-skill.sh`.
- [ ] Commit: `docs: explain RunEngram value and team rollout`

## Task 11: Final verification and remote delivery

- [ ] Run `cd server && go test ./...`.
- [ ] Run `cd cli && go test ./...`.
- [ ] Run `cd web && pnpm lint && pnpm test && pnpm build`.
- [ ] Run `./scripts/test-skill.sh`.
- [ ] Run plugin validator and release packaging tests.
- [ ] Rebuild/start local server; manually verify Action Console, board scrolling, dependency graph, memory detail, Dracula default, paper switch, locale switch, search, task editor, and browser reload persistence.
- [ ] Review `git diff --check`, `git status`, commit history, secrets scan, placeholder scan, and README claims against implemented code.
- [ ] Push `codex/dashboard-ux-refresh` to `origin`.

