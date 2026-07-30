# Learning Evidence Guidance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make learning-candidate verification explain why evidence matters, show what to enter, surface reusable source-task references, and preserve the existing promotion API.

**Architecture:** Extract evidence review from `MemoryCandidates` into a focused React component. Component lazily reads existing task detail and history endpoints, converts documents/links/events into optional references, and stores final evidence as existing plain text. No backend endpoint, schema, or CLI contract changes.

**Tech Stack:** React 19, TypeScript, TanStack Query, Testing Library, Vitest, Tailwind CSS, existing RunEngram REST API and i18n map.

---

## File structure

- Create `web/src/components/LearningEvidenceReview.tsx`: evidence types, writing
  guidance, source-task references, form state, and promotion submission.
- Create `web/src/components/LearningEvidenceReview.test.tsx`: focused interaction,
  loading, reference insertion, validation, and error-preservation tests.
- Modify `web/src/components/MemoryCandidates.tsx`: remove inline review form and
  delegate to `LearningEvidenceReview`.
- Modify `web/src/lib/api.ts`: add typed `getTask(taskId)` client for existing
  task-detail route.
- Modify `web/src/lib/api.test.ts`: verify task IDs are encoded and Web client
  header remains attached.
- Modify `web/src/lib/i18n.tsx`: English and Simplified Chinese review copy.
- Modify `web/src/components/KnowledgeView.test.tsx`: update end-to-end component
  expectations and mock source-task requests.

### Task 1: Add source-task API client

**Files:**
- Modify: `web/src/lib/api.test.ts`
- Modify: `web/src/lib/api.ts`

- [ ] **Step 1: Write failing API client test**

Add `getTask` to imports and add:

```ts
describe("getTask", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads an attached source task with an encoded task id", async () => {
    const task = {
      id: "task/one",
      project_id: "project-1",
      title: "Verify dependency placement",
      description: "",
      type: "refactor",
      state: "done",
      priority: 1,
      labels: [],
      docs: [],
      links: [],
      created_at: 1780051741142,
      updated_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(task), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getTask("task/one")).resolves.toEqual(task);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({ "X-Taskline-Client": "web" }),
      })
    );
  });
});
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd web && pnpm test -- api.test.ts
```

Expected: FAIL because `getTask` is not exported.

- [ ] **Step 3: Add minimal API function**

Add beside `listTasks`:

```ts
export async function getTask(taskId: string): Promise<Task> {
  return request<Task>("GET", `/api/v1/tasks/${encodeURIComponent(taskId)}`);
}
```

- [ ] **Step 4: Run API test**

Run:

```bash
cd web && pnpm test -- api.test.ts
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add web/src/lib/api.ts web/src/lib/api.test.ts
git commit -m "feat(web): load source tasks for memory review"
```

### Task 2: Build guided evidence review component

**Files:**
- Create: `web/src/components/LearningEvidenceReview.tsx`
- Create: `web/src/components/LearningEvidenceReview.test.tsx`

- [ ] **Step 1: Write focused interaction tests**

Create tests with a `QueryClientProvider` and a pending `LearningNote`. Mock
`GET /api/v1/tasks/task-1` with one document and one link, and
`GET /api/v1/tasks/task-1/events` with one event.

Required test cases:

```ts
it("explains why evidence is required and changes examples without losing text", async () => {
  renderReview();
  expect(screen.getByRole("heading", { name: "Confirm whether this experience is reliable" })).toBeTruthy();
  expect(screen.getByText(/given to later Agents/)).toBeTruthy();

  const evidence = screen.getByLabelText("What did you verify?");
  await userEvent.type(evidence, "Checked project convention.");
  await userEvent.click(screen.getByRole("radio", { name: "Code or configuration" }));

  expect(evidence).toHaveValue("Checked project convention.");
  expect(screen.getByText(/file path/)).toBeTruthy();
});

it("inserts a task reference but requires an observed result", async () => {
  renderReview();
  await userEvent.click(await screen.findByRole("button", { name: /Spec.md/ }));

  const evidence = screen.getByLabelText("What did you verify?");
  expect(evidence).toHaveValue(expect.stringContaining('Checked: document "Spec.md"'));
  expect(screen.getByRole("button", { name: "Verify and use in later tasks" })).toBeDisabled();

  await userEvent.type(evidence, "the reviewed requirement matches implementation scope");
  expect(screen.getByRole("button", { name: "Verify and use in later tasks" })).toBeEnabled();
});

it("preserves evidence when promotion fails", async () => {
  const onPromote = vi.fn().mockRejectedValue(new Error("promotion failed"));
  renderReview({ onPromote });

  const evidence = screen.getByLabelText("What did you verify?");
  await userEvent.type(evidence, "Two neighboring modules use the same dependency group.");
  await userEvent.click(screen.getByRole("button", { name: "Verify and use in later tasks" }));

  expect(await screen.findByText("Error: promotion failed")).toBeTruthy();
  expect(evidence).toHaveValue("Two neighboring modules use the same dependency group.");
});
```

Also cover source-task load failure with a visible retry button and working
manual textarea.

- [ ] **Step 2: Run component tests and verify failure**

Run:

```bash
cd web && pnpm test -- LearningEvidenceReview.test.tsx
```

Expected: FAIL because component does not exist.

- [ ] **Step 3: Implement evidence types and draft validation**

Create UI-only types and helpers:

```ts
export type EvidenceType =
  | "command-test"
  | "code-config"
  | "reviewed-document"
  | "reproduction-fix"
  | "project-convention";

const EVIDENCE_TYPES: EvidenceType[] = [
  "command-test",
  "code-config",
  "reviewed-document",
  "reproduction-fix",
  "project-convention",
];

function hasEvidenceResult(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;
  const marker = /(?:^|\n)Result:\s*(.+)/i.exec(trimmed);
  return marker ? marker[1].trim().length >= 4 : trimmed.length >= 12;
}

function appendReference(current: string, checked: string): string {
  const block = `Checked: ${checked}\nResult: `;
  return current.trim() ? `${current.trim()}\n\n${block}` : block;
}
```

`hasEvidenceResult` keeps manually written prose compatible while preventing a
selected reference with an empty `Result:` line from becoming proof.

- [ ] **Step 4: Implement lazy source material loading**

Use existing API calls only:

```ts
const material = useQuery({
  queryKey: ["learning-evidence-material", note.source_task_id],
  queryFn: async () => {
    const [task, events] = await Promise.all([
      getTask(note.source_task_id),
      listTaskEvents(note.source_task_id),
    ]);
    return { task, events: events.slice(-5).reverse() };
  },
});
```

Render:

- task documents as buttons inserting `document "<title>"` plus URL when present;
- task links as buttons inserting `link "<label-or-url>" (<url>)`;
- recent events as buttons inserting `task event "<summary>"`;
- loading text;
- failure text plus `Retry` calling `material.refetch()`;
- empty state when no references exist.

Reference buttons must only fill `Checked:` and leave `Result:` empty.

- [ ] **Step 5: Implement guided form**

Component contract:

```ts
export function LearningEvidenceReview({
  note,
  onPromote,
  onCancel,
}: {
  note: LearningNote;
  onPromote: (evidence: string, memoryClass: MemoryClass) => Promise<void>;
  onCancel: () => void;
}) {
  // local memoryClass, evidenceType, evidence, saving, and error state
}
```

Render order:

1. heading and purpose;
2. memory type selector and current-scope explanation;
3. labelled evidence-type radio group;
4. type-specific instruction and example;
5. source-task material;
6. textarea labelled "What did you verify?" with checklist in
   `aria-describedby`;
7. primary action "Verify and use in later tasks" disabled while saving or
   `hasEvidenceResult(evidence)` is false;
8. cancel action.

Submit calls `onPromote(evidence.trim(), memoryClass)`. Catch errors without
clearing any state. On success, parent removes component.

- [ ] **Step 6: Run focused tests**

Run:

```bash
cd web && pnpm test -- LearningEvidenceReview.test.tsx
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add web/src/components/LearningEvidenceReview.tsx web/src/components/LearningEvidenceReview.test.tsx
git commit -m "feat(web): guide learning evidence review"
```

### Task 3: Integrate review and localize copy

**Files:**
- Modify: `web/src/components/MemoryCandidates.tsx`
- Modify: `web/src/lib/i18n.tsx`
- Modify: `web/src/components/KnowledgeView.test.tsx`

- [ ] **Step 1: Update KnowledgeView integration test first**

Extend fetch mock:

```ts
if (url.endsWith("/api/v1/tasks/task-1") && init?.method === "GET") {
  return new Response(JSON.stringify({
    id: "task-1",
    project_id: project.id,
    title: "Import Notion requirement",
    description: "",
    type: "feature",
    state: "done",
    priority: 1,
    labels: [],
    docs: [{ id: "doc-1", task_id: "task-1", title: "PRD.md", created_at: 1, updated_at: 1 }],
    links: [],
    created_at: 1,
    updated_at: 1,
  }), { status: 200, headers: { "Content-Type": "application/json" } });
}
if (url.endsWith("/api/v1/tasks/task-1/events") && init?.method === "GET") {
  return new Response(JSON.stringify({ events: [] }), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}
```

Change interaction assertions:

```ts
await user.click(screen.getByRole("button", { name: "Validate & enable" }));
expect(screen.getByRole("heading", {
  name: "Confirm whether this experience is reliable",
})).toBeTruthy();
await user.selectOptions(screen.getByLabelText("Memory type"), "project-rule");
await user.type(
  screen.getByLabelText("What did you verify?"),
  "Maintainer reviewed requirement import output and confirmed matching scope."
);
await user.click(
  screen.getByRole("button", { name: "Verify and use in later tasks" })
);
```

- [ ] **Step 2: Run integration test and verify failure**

Run:

```bash
cd web && pnpm test -- KnowledgeView.test.tsx
```

Expected: FAIL because old inline form remains.

- [ ] **Step 3: Replace inline form**

Import `LearningEvidenceReview`. Remove inline `memoryClass`, `evidence`, and
review-form saving logic from `MemoryCandidates`.

Replace current `reviewingID === note.id` form with:

```tsx
{reviewingID === note.id && (
  <LearningEvidenceReview
    note={note}
    onPromote={async (nextEvidence, nextMemoryClass) => {
      await onPromote(note.id, nextEvidence, nextMemoryClass);
      setReviewingID(null);
    }}
    onCancel={() => {
      setReviewingID(null);
      setError("");
    }}
  />
)}
```

Keep edit and reject flows unchanged.

- [ ] **Step 4: Add complete bilingual copy**

Add matching keys to `enUS`:

```ts
"knowledge.evidenceReviewTitle": "Confirm whether this experience is reliable",
"knowledge.evidenceReviewPurpose": "Verified experience may be given to later Agents. Add one fact another person can recheck.",
"knowledge.evidenceType": "What kind of evidence did you check?",
"knowledge.evidenceType.commandTest": "Command or test",
"knowledge.evidenceType.codeConfig": "Code or configuration",
"knowledge.evidenceType.reviewedDocument": "Reviewed document",
"knowledge.evidenceType.reproductionFix": "Reproduction and fix",
"knowledge.evidenceType.projectConvention": "Existing project convention",
"knowledge.evidenceInstruction.commandTest": "Record the command, relevant result, and environment when it matters.",
"knowledge.evidenceInstruction.codeConfig": "Name the file path, symbol or value, and the structure you observed.",
"knowledge.evidenceInstruction.reviewedDocument": "Name the document or link, its reviewed state, and the conclusion that supports this memory.",
"knowledge.evidenceInstruction.reproductionFix": "Record the behavior before the change, the change, and the behavior after it.",
"knowledge.evidenceInstruction.projectConvention": "Name at least two existing examples or one governing project rule.",
"knowledge.evidenceExample.commandTest": "Example: ./gradlew :service:compileDebugKotlin passed; the dependency resolved from the existing service group.",
"knowledge.evidenceExample.codeConfig": "Example: settings.gradle includes :service:foo; neighboring service modules use the same group.",
"knowledge.evidenceExample.reviewedDocument": "Example: Architecture review module-boundaries.md approved this dependency placement.",
"knowledge.evidenceExample.reproductionFix": "Example: Moving the dependency to the service group removed the clean-sync resolution failure.",
"knowledge.evidenceExample.projectConvention": "Example: service:a and service:b are both declared in the existing service dependency group.",
"knowledge.evidenceAvailable": "Evidence available from this task",
"knowledge.evidenceLoading": "Loading task material...",
"knowledge.evidenceUnavailable": "Source task material is unavailable. You can still enter evidence manually.",
"knowledge.evidenceRetry": "Retry",
"knowledge.evidenceEmpty": "No attached documents, links, or recent task events.",
"knowledge.evidenceDocument": "Document",
"knowledge.evidenceLink": "Link",
"knowledge.evidenceEvent": "Task event",
"knowledge.evidenceQuestion": "What did you verify?",
"knowledge.evidenceTemplate": "Checked:\\nResult:\\nScope or environment (optional):",
"knowledge.evidenceChecklistSource": "Name a concrete command, file, document, behavior, or convention.",
"knowledge.evidenceChecklistResult": "Record the observed result, not only an opinion.",
"knowledge.evidenceChecklistScope": "Include enough scope to know when it applies.",
"knowledge.evidenceRequired": "Add the observed result before enabling this memory.",
"knowledge.verifyAndReuse": "Verify and use in later tasks",
"knowledge.projectRuleEffect": "Project rule: included in every task context for this project.",
"knowledge.scopedExperienceEffect": "Scoped experience: recalled only when a later task matches this scope.",
```

Add matching keys to `zhCN`:

```ts
"knowledge.evidenceReviewTitle": "确认这条经验是否可靠",
"knowledge.evidenceReviewPurpose": "通过验证的经验会提供给后续 Agent。请补充一个别人可以复查的事实。",
"knowledge.evidenceType": "你核对的是哪类证据？",
"knowledge.evidenceType.commandTest": "命令或测试",
"knowledge.evidenceType.codeConfig": "代码或配置",
"knowledge.evidenceType.reviewedDocument": "已审核文档",
"knowledge.evidenceType.reproductionFix": "问题复现与修复",
"knowledge.evidenceType.projectConvention": "项目已有约定",
"knowledge.evidenceInstruction.commandTest": "填写执行命令、关键结果，以及必要的运行环境。",
"knowledge.evidenceInstruction.codeConfig": "填写文件路径、符号或配置值，以及你观察到的代码结构。",
"knowledge.evidenceInstruction.reviewedDocument": "填写文档或链接、审核状态，以及支持这条经验的结论。",
"knowledge.evidenceInstruction.reproductionFix": "填写修改前现象、所做修改和修改后结果。",
"knowledge.evidenceInstruction.projectConvention": "填写至少两个已有示例，或一条明确的项目规则。",
"knowledge.evidenceExample.commandTest": "示例：./gradlew :service:compileDebugKotlin 执行通过，依赖从现有 service 分组正确解析。",
"knowledge.evidenceExample.codeConfig": "示例：settings.gradle 已包含 :service:foo，相邻 service 模块采用相同分组。",
"knowledge.evidenceExample.reviewedDocument": "示例：架构评审文档 module-boundaries.md 已确认该依赖位置。",
"knowledge.evidenceExample.reproductionFix": "示例：依赖移入 service 分组后，干净同步不再出现解析失败。",
"knowledge.evidenceExample.projectConvention": "示例：service:a 和 service:b 均声明在现有 service 依赖分组。",
"knowledge.evidenceAvailable": "当前任务中可参考的材料",
"knowledge.evidenceLoading": "正在读取任务材料…",
"knowledge.evidenceUnavailable": "无法读取来源任务材料，你仍可手动填写验证事实。",
"knowledge.evidenceRetry": "重试",
"knowledge.evidenceEmpty": "当前任务没有关联文档、链接或近期操作记录。",
"knowledge.evidenceDocument": "文档",
"knowledge.evidenceLink": "链接",
"knowledge.evidenceEvent": "任务记录",
"knowledge.evidenceQuestion": "你验证了什么？",
"knowledge.evidenceTemplate": "核对对象：\\n观察结果：\\n适用范围或环境（可选）：",
"knowledge.evidenceChecklistSource": "写明具体命令、文件、文档、现象或项目约定。",
"knowledge.evidenceChecklistResult": "记录实际观察结果，不只写主观判断。",
"knowledge.evidenceChecklistScope": "补充足够范围，让后续任务知道何时适用。",
"knowledge.evidenceRequired": "请补充观察结果后再启用这条经验。",
"knowledge.verifyAndReuse": "确认有效并用于后续任务",
"knowledge.projectRuleEffect": "项目规则：进入该项目每个任务的上下文。",
"knowledge.scopedExperienceEffect": "场景经验：仅在后续任务范围匹配时召回。",
```

Use design-spec wording. Keep English and Chinese maps key-for-key identical so
`TranslationKey` remains type-safe.

- [ ] **Step 5: Run integration and i18n tests**

Run:

```bash
cd web && pnpm test -- KnowledgeView.test.tsx i18n.test.tsx
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add web/src/components/MemoryCandidates.tsx web/src/components/KnowledgeView.test.tsx web/src/lib/i18n.tsx
git commit -m "feat(web): integrate evidence guidance"
```

### Task 4: Validate full change and local service

**Files:**
- Modify only if validation exposes a defect.

- [ ] **Step 1: Run frontend quality gates**

```bash
cd web && pnpm lint && pnpm test && pnpm build
```

Expected: lint clean, all tests pass, Vite build writes embedded assets.

- [ ] **Step 2: Run repository quality gates**

```bash
(cd server && go test ./...)
(cd cli && go test ./...)
./scripts/test-skill.sh
./scripts/test-start-local-env.sh
./scripts/test-package-release.sh
```

Expected: all pass.

- [ ] **Step 3: Rebuild and restart**

```bash
./scripts/start-local.sh
curl --fail --silent http://127.0.0.1:8787/api/v1/health
```

Expected: health response reports healthy service and updated embedded web UI.

- [ ] **Step 4: Manual browser smoke test**

Open `http://127.0.0.1:8787/?view=knowledge`, select pending review, and verify:

- purpose is visible without scrolling past form start;
- Dracula colors retain readable contrast;
- source references load;
- selecting reference requires adding a result;
- promotion error retains draft;
- light theme remains readable.
