# Learning Evidence Guidance Design

## Problem

RunEngram requires validation evidence before a learning candidate becomes
reusable memory. The current review panel only shows a textarea named
"Validation evidence" and a short placeholder. A reviewer cannot tell:

- why evidence matters;
- which facts qualify as evidence;
- where to find those facts;
- what will happen after confirmation.

This creates two bad outcomes: useful candidates remain pending, or reviewers
enter vague claims that later Agents cannot verify.

## Goal

Make memory review understandable on first use and completable without learning
RunEngram internals.

The panel must tell the reviewer:

1. **Purpose:** only verified experience is injected into later Agent context.
2. **Action:** choose the kind of fact used to verify the candidate, then record
   what was checked and what result was observed.
3. **Outcome:** confirmation makes the memory eligible for later tasks according
   to its selected scope.

RunEngram must never invent evidence or treat an unverified suggestion as proof.

## Non-goals

- No automatic promotion.
- No LLM-generated evidence.
- No requirement for GitHub, pull requests, or CI.
- No new persistence schema in this iteration.
- No hard-coded dependency on a specific engineering SOP.

## Review flow

### 1. Explain the decision

Replace the current generic hint with:

> **Confirm whether this experience is reliable**
>
> Verified experience may be given to later Agents. Add one fact another person
> can recheck: a command and result, a reviewed file or document, a reproduced
> behavior, or an existing project convention.

Chinese copy:

> **确认这条经验是否可靠**
>
> 通过验证的经验会提供给后续 Agent。请补充一个别人可以复查的事实，例如命令与结果、已审核文件或文档、复现现象，或项目中已有的约定。

### 2. Choose evidence type

Provide five choices. Each choice changes the instructions and example shown
above the textarea.

| Type | Reviewer records | Example |
| --- | --- | --- |
| Command or test | command, relevant result, environment when needed | `./gradlew :service:compileDebugKotlin` passed; dependency resolved from the existing service group |
| Code or configuration | file path, relevant symbol/value, observed structure | `settings.gradle` includes `:service:foo`; neighboring service modules are declared in the same Gradle group |
| Reviewed document | document name/link, reviewer or approved state, relevant conclusion | Architecture review `module-boundaries.md` approved this dependency placement |
| Reproduction and fix | before behavior, change, after behavior | Moving the dependency to the service group removed the resolution failure on a clean sync |
| Existing project convention | at least two existing examples or a governing rule | `service:a` and `service:b` are both declared in the existing service dependency group |

Changing type does not overwrite text already entered.

### 3. Surface existing task material

When the review panel opens, lazily load the source task and its history through
existing task APIs. Show a compact "Evidence available from this task" section:

- attached documents;
- attached links;
- recent task events whose summaries may help locate a test, verification, or
  reviewed result.

These items are references, not proof by themselves. Selecting one inserts its
title/reference into the draft and leaves the reviewer responsible for adding
the checked result. If task material cannot be loaded, manual entry remains
available and the draft is preserved.

No bulk preloading: source-task data loads only for the candidate currently
being reviewed.

### 4. Give a concrete writing template

The textarea label becomes "What did you verify?" / "你验证了什么？".

Template:

```text
Checked:
Result:
Scope or environment (optional):
```

The UI shows a three-item checklist:

- names a concrete command, file, document, behavior, or convention;
- records the observed result, not only an opinion;
- contains enough scope to know when the conclusion applies.

Checklist is guidance, not a brittle semantic gate. Existing server validation
continues to require non-empty evidence.

### 5. State the outcome in the action

Rename the primary action:

- Chinese: `确认有效并用于后续任务`
- English: `Verify and use in later tasks`

Below the memory-type selector, explain:

- project rule: enters every task context for this project;
- scoped experience: recalled only when a later task matches its scope.

## Component and data changes

### Frontend

- Extract the review form from `MemoryCandidates` into a focused
  `LearningEvidenceReview` component.
- Add an `EvidenceType` UI-only union and per-type localized guidance.
- Add `getTask(taskId)` to the web API client, using existing
  `GET /api/v1/tasks/:id`.
- Reuse `listTaskEvents(taskId)` for recent source-task history.
- Keep final evidence as plain text so CLI, API, and stored memory remain
  backward compatible.

### Backend

No endpoint or schema change. Existing task detail, task history, and learning
promotion endpoints provide required data.

### Internationalization

All labels, explanations, examples, empty states, and errors receive English
and Simplified Chinese entries. Examples remain project-neutral.

## Error handling

- Source task missing or deleted: show "Source task material is unavailable";
  manual evidence still works.
- Task material request fails: show retry action; preserve memory type and
  evidence draft.
- Promotion fails: keep panel open and preserve all inputs.
- Empty evidence: disable submit and show why evidence is required.
- Switching candidates: reset evidence draft only after reviewer explicitly
  opens another candidate.

## Accessibility

- Evidence-type controls use a labelled radio group.
- Guidance text is connected to the textarea with `aria-describedby`.
- Loading, error, and selection states do not rely on color alone.
- Keyboard users can choose a source reference and submit without pointer input.

## Tests

Component tests cover:

1. purpose and next action are visible when review opens;
2. evidence-type selection changes example without deleting the draft;
3. task documents, links, and recent events appear as optional references;
4. selecting a reference inserts it without claiming a successful result;
5. empty evidence cannot be promoted;
6. promotion error preserves the draft;
7. Chinese and English copy render through existing i18n system.

Existing server and CLI learning-promotion tests remain unchanged because the
wire contract does not change.

## Acceptance criteria

- A first-time reviewer can explain why evidence is required from the panel
  alone.
- Panel gives a concrete next action and at least one relevant example.
- Reviewer can reuse source-task references without RunEngram fabricating a
  result.
- Existing manual evidence and API promotion flows remain compatible.
- UI works in default Dracula and optional light themes.
