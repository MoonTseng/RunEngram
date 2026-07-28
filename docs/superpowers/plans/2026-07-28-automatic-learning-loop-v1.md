# Automatic Learning Loop v1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Capture durable agent corrections as pending Learning Notes, promote verified notes into reusable Exploration Capsules, recall them in later tasks, and expose learning progress through CLI and Web.

**Architecture:** Add one `learning_notes` resource inside the existing `handler → service → store` stack. Agent skills detect corrections and call authenticated CLI commands; service rules require live task ownership and verification evidence; store promotion atomically creates one capsule and resolves one note. Existing capsule recall and usage feedback stay authoritative.

**Tech Stack:** Go 1.24, Hertz, `modernc.org/sqlite`, Cobra, React 19, TypeScript, TanStack Query, Vitest, Testing Library, Markdown agent skills.

---

## File map

### Server

- Create `server/migrations/0015_learning_notes.sql`: public migration copy.
- Create `server/internal/store/schema/0015_learning_notes.sql`: embedded migration.
- Modify `server/api/model/model.go`: Learning Note enums, model, task attachment, and metrics.
- Modify `server/internal/store/store.go`: migration registration, CRUD, task attachment, metrics, and atomic promotion.
- Modify `server/internal/store/store_test.go`: migration, persistence, and idempotent transaction tests.
- Modify `server/internal/service/learning.go`: capture, list, promote, reject, claim validation, and input limits.
- Modify `server/internal/service/learning_test.go`: behavior tests at service seam.
- Modify `server/api/handler/handler.go`: REST routes, request shapes, authentication, and actor propagation.
- Modify `server/tests/e2e_test.go`: authenticated HTTP and recall loop.
- Modify `server/api/model/http_contract_test.go`: canonical Learning Note shape.

### CLI

- Modify `cli/client/client.go`: duplicated Learning Note shapes and REST methods.
- Create `cli/cmd/learning.go`: `capture`, `list`, `promote`, and `reject`.
- Create `cli/cmd/learning_test.go`: command registration, flags, and output behavior.
- Modify `cli/client/client_test.go`: token use, paths, payloads, and server errors.
- Modify `cli/client/http_contract_test.go`: canonical Learning Note shape.

### Web

- Modify `web/src/lib/api.ts`: Learning Note types, metrics, and list request.
- Modify `web/src/lib/api.test.ts`: JSON contract and query tests.
- Modify `web/src/lib/i18n.tsx`: English and Simplified Chinese copy.
- Modify `web/src/components/KnowledgeView.tsx`: candidate metrics, filters, and cards.
- Create `web/src/components/KnowledgeView.test.tsx`: loading, empty, and grouped candidate rendering.
- Modify `web/src/lib/http-contract.test.ts`: canonical Learning Note shape.

### Agent protocol and documentation

- Modify `skills/taskline-management/SKILL.md`: automatic correction capture and evidence resolution.
- Modify `scripts/test-skill.sh`: enforce new protocol language and commands.
- Modify `README.md` and `README.zh-CN.md`: explain automatic learning loop and examples.
- Modify `PRODUCT.md`: mark Learning Notes and verified promotion as shipped.
- Modify `ARCHITECTURE.md`: schema, transaction, trust boundary, and agent-driven automation.
- Add `testdata/http_contract/learning_note.json`.
- Modify `testdata/http_contract/task_full.json`: empty `learning_notes` task attachment.
- Modify server, CLI, and Web HTTP contract tests to read the new fixture.

## Task 1: Persist pending Learning Notes

**Files:**
- Create: `server/migrations/0015_learning_notes.sql`
- Create: `server/internal/store/schema/0015_learning_notes.sql`
- Modify: `server/api/model/model.go`
- Modify: `server/internal/store/store.go`
- Test: `server/internal/store/store_test.go`

- [ ] **Step 1: Write the failing migration and persistence test**

Add `TestLearningNotePersistenceAndTaskAttachment`:

```go
func TestLearningNotePersistenceAndTaskAttachment(t *testing.T) {
	ctx := context.Background()
	st := newStore(t)
	project := createProject(t, st, "learning-notes")
	task := createTask(t, st, project.ID, "Read Notion PRD")

	note := &model.LearningNote{
		ProjectID: project.ID,
		SourceTaskID: task.ID,
		Kind: model.LearningNoteHumanCorrection,
		Trigger: "Notion link was not readable",
		Guidance: "Use one-flow/notion-to-prd",
		Scope: "Notion requirement analysis",
		Labels: []string{"notion", "prd"},
		Fingerprints: []string{"notion-to-prd"},
		Producer: "codex",
		Status: model.LearningNotePending,
	}
	require.NoError(t, st.CreateLearningNote(ctx, note))

	got, err := st.GetLearningNote(ctx, note.ID)
	require.NoError(t, err)
	require.Equal(t, model.LearningNotePending, got.Status)
	require.Equal(t, []string{"notion", "prd"}, got.Labels)

	withDetails, err := st.GetTask(ctx, task.ID)
	require.NoError(t, err)
	require.Len(t, withDetails.LearningNotes, 1)
	require.Equal(t, note.ID, withDetails.LearningNotes[0].ID)
}
```

- [ ] **Step 2: Run the focused test and verify red**

Run:

```bash
( cd server && go test ./internal/store -run TestLearningNotePersistenceAndTaskAttachment -count=1 )
```

Expected: compile failure because `model.LearningNote` and store methods do not exist.

- [ ] **Step 3: Add model types**

Add to `server/api/model/model.go`:

```go
type LearningNoteKind string

const (
	LearningNoteHumanCorrection LearningNoteKind = "human-correction"
	LearningNoteAgentRecovery LearningNoteKind = "agent-recovery"
)

func (k LearningNoteKind) Valid() bool {
	return k == LearningNoteHumanCorrection || k == LearningNoteAgentRecovery
}

type LearningNoteStatus string

const (
	LearningNotePending LearningNoteStatus = "pending"
	LearningNotePromoted LearningNoteStatus = "promoted"
	LearningNoteRejected LearningNoteStatus = "rejected"
)

func (s LearningNoteStatus) Valid() bool {
	return s == LearningNotePending || s == LearningNotePromoted || s == LearningNoteRejected
}

type LearningNote struct {
	ID string `json:"id"`
	ProjectID string `json:"project_id"`
	SourceTaskID string `json:"source_task_id"`
	Kind LearningNoteKind `json:"kind"`
	Trigger string `json:"trigger"`
	Guidance string `json:"guidance"`
	Scope string `json:"scope"`
	Labels []string `json:"labels"`
	Fingerprints []string `json:"fingerprints"`
	Producer string `json:"producer"`
	Status LearningNoteStatus `json:"status"`
	Evidence string `json:"evidence"`
	CapsuleID string `json:"capsule_id"`
	RejectionReason string `json:"rejection_reason"`
	CreatedAt int64 `json:"created_at"`
	UpdatedAt int64 `json:"updated_at"`
	ResolvedAt int64 `json:"resolved_at"`
}
```

Add `LearningNotes []LearningNote` to `model.Task`.

- [ ] **Step 4: Add migration**

Write identical content to both migration files:

```sql
CREATE TABLE learning_notes (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    source_task_id TEXT NOT NULL,
    kind TEXT NOT NULL CHECK (kind IN ('human-correction','agent-recovery')),
    trigger TEXT NOT NULL,
    guidance TEXT NOT NULL,
    scope TEXT NOT NULL DEFAULT '',
    labels TEXT NOT NULL DEFAULT '[]',
    fingerprints TEXT NOT NULL DEFAULT '[]',
    producer TEXT NOT NULL DEFAULT 'codex',
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending','promoted','rejected')),
    evidence TEXT NOT NULL DEFAULT '',
    capsule_id TEXT NOT NULL DEFAULT '',
    rejection_reason TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    resolved_at INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX idx_learning_notes_project_status
    ON learning_notes(project_id, status, updated_at DESC);
CREATE INDEX idx_learning_notes_task
    ON learning_notes(source_task_id, created_at DESC);
```

Embed `schema/0015_learning_notes.sql` as `schemaLearningNotes` and register migration version 15.

- [ ] **Step 5: Add store CRUD and task attachment**

Add:

```go
type LearningNoteFilter struct {
	ProjectID string
	TaskID string
	Status model.LearningNoteStatus
	Limit int
}

func (s *Store) CreateLearningNote(ctx context.Context, note *model.LearningNote) error
func (s *Store) GetLearningNote(ctx context.Context, id string) (*model.LearningNote, error)
func (s *Store) ListLearningNotes(ctx context.Context, filter LearningNoteFilter) ([]model.LearningNote, error)
func (s *Store) attachLearningNotes(ctx context.Context, task *model.Task) error
func (s *Store) attachLearningNotesForTasks(ctx context.Context, tasks []model.Task) error
```

`CreateLearningNote` must resolve the source task, reject cross-project input,
normalize labels and fingerprints with `normalizeLabels`, assign `pending`,
and set millisecond timestamps. `attachTaskDetails` and batch task attachment
must include Learning Notes.

- [ ] **Step 6: Run store tests**

Run:

```bash
( cd server && go test ./internal/store -count=1 )
```

Expected: all store tests pass, including migration version 15.

- [ ] **Step 7: Commit persistence slice**

```bash
git add server/api/model/model.go server/migrations/0015_learning_notes.sql \
  server/internal/store/schema/0015_learning_notes.sql \
  server/internal/store/store.go server/internal/store/store_test.go
git commit -m "feat(server): persist learning candidates"
```

## Task 2: Enforce capture and atomic promotion

**Files:**
- Modify: `server/internal/service/learning.go`
- Modify: `server/internal/service/learning_test.go`
- Modify: `server/internal/store/store.go`
- Modify: `server/api/model/model.go`
- Test: `server/internal/service/learning_test.go`
- Test: `server/internal/store/store_test.go`

- [ ] **Step 1: Write failing service tests**

Add `TestLearningNoteRequiresClaimAndPromotesOnce`:

```go
func TestLearningNoteRequiresClaimAndPromotesOnce(t *testing.T) {
	ctx := context.Background()
	svc := newSvc(t)
	project, _ := svc.CreateProject(ctx, "auto-learning", "")
	task, _ := svc.CreateTask(ctx, project.ID, "Read Notion PRD",
		"Notion requirement", model.TaskTypeFeature, 1, true, []string{"notion"})

	_, err := svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: task.ID, AgentName: "codex",
		Kind: model.LearningNoteHumanCorrection,
		Trigger: "Notion link was not readable",
		Guidance: "Use one-flow/notion-to-prd",
		Scope: "Notion requirements",
		Fingerprints: []string{"notion-to-prd"},
	})
	require.Error(t, err)

	task, _ = svc.ClaimTask(ctx, task.ID, service.ClaimOptions{Owner: "codex"})
	note, err := svc.CaptureLearningNote(ctx, service.CaptureLearningNoteInput{
		ProjectID: project.ID, SourceTaskID: task.ID, AgentName: "codex",
		Kind: model.LearningNoteHumanCorrection,
		Trigger: "Notion link was not readable",
		Guidance: "Use one-flow/notion-to-prd",
		Scope: "Notion requirements",
		Labels: []string{"notion"},
		Fingerprints: []string{"notion-to-prd"},
	})
	require.NoError(t, err)

	first, err := svc.PromoteLearningNote(ctx, note.ID, "codex",
		"notion-to-prd produced the PRD; analysis tests passed")
	require.NoError(t, err)
	second, err := svc.PromoteLearningNote(ctx, note.ID, "codex",
		"same retry")
	require.NoError(t, err)
	require.Equal(t, first.CapsuleID, second.CapsuleID)

	capsules, err := svc.ListCapsules(ctx, service.CapsuleListInput{
		ProjectID: project.ID, Status: model.CapsuleStatusActive,
	})
	require.NoError(t, err)
	require.Len(t, capsules, 1)
}
```

Add rejection tests for blank reason, wrong owner, expired lease, promoted note,
and idempotent repeated rejection.

- [ ] **Step 2: Run service test and verify red**

Run:

```bash
( cd server && go test ./internal/service -run LearningNote -count=1 )
```

Expected: compile failure for missing service inputs and methods.

- [ ] **Step 3: Add service inputs and validation**

Add:

```go
type CaptureLearningNoteInput struct {
	ProjectID string
	SourceTaskID string
	AgentName string
	Kind model.LearningNoteKind
	Trigger string
	Guidance string
	Scope string
	Labels []string
	Fingerprints []string
	Producer string
}

type LearningNoteListInput struct {
	ProjectID string
	TaskID string
	Status model.LearningNoteStatus
	Limit int
}

func (s *Service) CaptureLearningNote(ctx context.Context, input CaptureLearningNoteInput) (*model.LearningNote, error)
func (s *Service) ListLearningNotes(ctx context.Context, input LearningNoteListInput) ([]model.LearningNote, error)
func (s *Service) PromoteLearningNote(ctx context.Context, id, agentName, evidence string) (*model.LearningNote, error)
func (s *Service) RejectLearningNote(ctx context.Context, id, agentName, reason string) (*model.LearningNote, error)
```

Use one helper:

```go
func requireLiveOwner(task *model.Task, agentName string) error {
	if strings.TrimSpace(agentName) == "" {
		return errors.New("agent identity required")
	}
	if task.Owner != agentName || task.LeaseExpiresAt <= time.Now().UnixMilli() {
		return fmt.Errorf("%w: live source-task claim owned by %s required", store.ErrConflict, agentName)
	}
	return nil
}
```

Apply limits: trigger 2,000 runes; guidance 8,000; scope 2,000; evidence
32,000; rejection reason 2,000. Use `utf8.RuneCountInString`.

- [ ] **Step 4: Write failing store transaction test**

Add a store test that calls:

```go
firstNote, firstCapsule, err := st.PromoteLearningNote(ctx, note.ID, "verified")
require.NoError(t, err)
secondNote, secondCapsule, err := st.PromoteLearningNote(ctx, note.ID, "retry")
require.NoError(t, err)
require.Equal(t, firstNote.CapsuleID, secondNote.CapsuleID)
require.Equal(t, firstCapsule.ID, secondCapsule.ID)
```

Assert only one capsule row exists.

- [ ] **Step 5: Implement atomic store resolution**

Add:

```go
func (s *Store) PromoteLearningNote(
	ctx context.Context, id, evidence string,
) (*model.LearningNote, *model.ExplorationCapsule, error)

func (s *Store) RejectLearningNote(
	ctx context.Context, id, reason string,
) (*model.LearningNote, error)
```

Promotion transaction:

1. Read note inside transaction.
2. If `promoted`, return linked capsule.
3. If `rejected`, return `ErrConflict`.
4. Insert one active capsule using note guidance as summary, trigger as title
   context, scope, labels, fingerprints, producer, and supplied evidence.
5. Update note with `promoted`, evidence, capsule ID, and `resolved_at`.
6. Commit.

Use capsule title:

```go
title := strings.TrimSpace(note.Guidance)
if utf8.RuneCountInString(title) > 120 {
	title = string([]rune(title)[:120])
}
```

Rejection performs a compare-and-set update from `pending` to `rejected`;
retries return the rejected row.

- [ ] **Step 6: Record task history and metrics**

After capture, promotion, or rejection, call `recordTaskEvent` with actions:

```text
learning_note_captured
learning_note_promoted
learning_note_rejected
```

Extend `LearningMetrics`:

```go
LearningNoteCount int `json:"learning_note_count"`
PendingNoteCount int `json:"pending_note_count"`
PromotedNoteCount int `json:"promoted_note_count"`
RejectedNoteCount int `json:"rejected_note_count"`
PromotionRate float64 `json:"promotion_rate"`
```

Calculate `promotion_rate = promoted / (promoted + rejected)` when resolved
count is nonzero.

- [ ] **Step 7: Run service and store tests**

Run:

```bash
( cd server && go test ./internal/store ./internal/service -count=1 )
```

Expected: all tests pass.

- [ ] **Step 8: Commit domain slice**

```bash
git add server/api/model/model.go server/internal/store/store.go \
  server/internal/store/store_test.go server/internal/service/learning.go \
  server/internal/service/learning_test.go
git commit -m "feat(server): promote verified learning notes"
```

## Task 3: Expose authenticated HTTP workflow

**Files:**
- Modify: `server/api/handler/handler.go`
- Modify: `server/tests/e2e_test.go`
- Add: `testdata/http_contract/learning_note.json`
- Modify: `testdata/http_contract/task_full.json`
- Modify: `server/api/model/http_contract_test.go`

- [ ] **Step 1: Add failing API end-to-end test**

Add `TestAutomaticLearningNoteLoopAtAPI`:

```go
func TestAutomaticLearningNoteLoopAtAPI(t *testing.T) {
	baseURL := startTestServer(t)
	project := createProject(t, baseURL, "learning-api")
	token := registerAgent(t, baseURL, "codex")
	source := createTask(t, baseURL, project.ID, "Read Notion requirement")
	claimTask(t, baseURL, token, source.ID)

	note := postJSON[model.LearningNote](t, baseURL,
		"/api/v1/projects/"+project.ID+"/learning-notes", token, map[string]any{
			"source_task_id": source.ID,
			"kind": "human-correction",
			"trigger": "Notion link unreadable",
			"guidance": "Use one-flow/notion-to-prd",
			"scope": "Notion requirement analysis",
			"labels": []string{"notion"},
			"fingerprints": []string{"notion-to-prd"},
			"producer": "codex",
		})
	require.Equal(t, model.LearningNotePending, note.Status)

	promoted := postJSON[model.LearningNote](t, baseURL,
		"/api/v1/learning-notes/"+note.ID+"/promote", token, map[string]any{
			"evidence": "notion-to-prd produced normalized PRD",
		})
	require.NotEmpty(t, promoted.CapsuleID)

	target := createAndClaimTask(t, baseURL, token, project.ID,
		"Analyze another Notion PRD", "Use Notion requirement input")
	context := getJSON[model.ContextSnapshot](t, baseURL,
		"/api/v1/tasks/"+target.ID+"/context", token)
	require.Len(t, context.SuggestedCapsules, 1)
	require.Equal(t, promoted.CapsuleID, context.SuggestedCapsules[0].ID)
}
```

Add 401, wrong-owner 409, blank evidence 400, list-by-task, and reject cases.

- [ ] **Step 2: Run e2e test and verify red**

Run:

```bash
( cd server && go test ./tests -run TestAutomaticLearningNoteLoopAtAPI -count=1 )
```

Expected: 404 for learning-note route.

- [ ] **Step 3: Register routes and request shapes**

Add routes:

```go
v1.POST("/projects/:project/learning-notes", h.captureLearningNote)
v1.GET("/projects/:project/learning-notes", h.listLearningNotes)
v1.GET("/tasks/:id/learning-notes", h.listTaskLearningNotes)
v1.POST("/learning-notes/:id/promote", h.promoteLearningNote)
v1.POST("/learning-notes/:id/reject", h.rejectLearningNote)
```

Add request types:

```go
type captureLearningNoteReq struct {
	SourceTaskID string `json:"source_task_id"`
	Kind model.LearningNoteKind `json:"kind"`
	Trigger string `json:"trigger"`
	Guidance string `json:"guidance"`
	Scope string `json:"scope"`
	Labels []string `json:"labels"`
	Fingerprints []string `json:"fingerprints"`
	Producer string `json:"producer"`
}
type promoteLearningNoteReq struct { Evidence string `json:"evidence"` }
type rejectLearningNoteReq struct { Reason string `json:"reason"` }
```

Mutation handlers must call `requireAgent`, set request actor with
`service.WithActor(ctx, agent.Name)`, and pass `agent.Name` into service.

- [ ] **Step 4: Add canonical contract fixture**

Create `testdata/http_contract/learning_note.json` with exact stable fields:

```json
{
  "id": "note-1",
  "project_id": "project-1",
  "source_task_id": "task-1",
  "kind": "human-correction",
  "trigger": "Notion link unreadable",
  "guidance": "Use one-flow/notion-to-prd",
  "scope": "Notion requirements",
  "labels": ["notion", "prd"],
  "fingerprints": ["notion-to-prd"],
  "producer": "codex",
  "status": "pending",
  "evidence": "",
  "capsule_id": "",
  "rejection_reason": "",
  "created_at": 1785220000000,
  "updated_at": 1785220000000,
  "resolved_at": 0
}
```

Load it from the server contract test. Update `task_full.json` with
`"learning_notes": []`; CLI and Web add their matching contract assertions in
their own slices.

- [ ] **Step 5: Run full server suite**

Run:

```bash
( cd server && go test ./... -count=1 )
```

Expected: all server tests pass.

- [ ] **Step 6: Commit API slice**

```bash
git add server/api/handler/handler.go server/tests/e2e_test.go \
  server/api/model/http_contract_test.go \
  testdata/http_contract/learning_note.json testdata/http_contract/task_full.json
git commit -m "feat(api): expose automatic learning workflow"
```

## Task 4: Add agent-first CLI commands

**Files:**
- Modify: `cli/client/client.go`
- Create: `cli/cmd/learning.go`
- Create: `cli/cmd/learning_test.go`
- Modify: `cli/client/client_test.go`
- Modify: `cli/client/http_contract_test.go`

- [ ] **Step 1: Write failing client request test**

Test authenticated capture:

```go
func TestLearningClientUsesIdentityAndStablePayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/api/v1/projects/demo/learning-notes", r.URL.Path)
		require.Equal(t, "Bearer token-1", r.Header.Get("Authorization"))
		require.Equal(t, http.MethodPost, r.Method)
		var body client.CaptureLearningNoteInput
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		require.Equal(t, "Use one-flow/notion-to-prd", body.Guidance)
		writeFixture(t, w, "learning_note.json")
	}))
	defer server.Close()

	c := client.New(server.URL, "token-1")
	_, err := c.CaptureLearningNote("demo", client.CaptureLearningNoteInput{
		SourceTaskID: "task-1",
		Kind: "human-correction",
		Trigger: "Notion link unreadable",
		Guidance: "Use one-flow/notion-to-prd",
	})
	require.NoError(t, err)
}
```

- [ ] **Step 2: Run client test and verify red**

Run:

```bash
( cd cli && go test ./client -run TestLearningClientUsesIdentityAndStablePayload -count=1 )
```

Expected: compile failure for missing types and methods.

- [ ] **Step 3: Add duplicated CLI types and REST methods**

Add `LearningNote`, `CaptureLearningNoteInput`, and metric fields with JSON
names matching server. Add `LearningNotes []LearningNote` to the duplicated
CLI `Task` type. Extend `cli/client/http_contract_test.go` with
`learning_note.json`. Add:

```go
func (c *Client) CaptureLearningNote(project string, input CaptureLearningNoteInput) (*LearningNote, error)
func (c *Client) ListLearningNotes(project, taskID, status string, limit int) ([]LearningNote, error)
func (c *Client) PromoteLearningNote(id, evidence string) (*LearningNote, error)
func (c *Client) RejectLearningNote(id, reason string) (*LearningNote, error)
```

All mutation methods use the client bearer token through existing `do`.

- [ ] **Step 4: Write failing command registration test**

```go
func TestLearningCommandsRegistered(t *testing.T) {
	require.NotNil(t, findCommand(rootCmd, "learning capture"))
	require.NotNil(t, findCommand(rootCmd, "learning list"))
	require.NotNil(t, findCommand(rootCmd, "learning promote"))
	require.NotNil(t, findCommand(rootCmd, "learning reject"))
	require.NotNil(t, learningCaptureCmd.Flag("trigger"))
	require.NotNil(t, learningCaptureCmd.Flag("guidance"))
	require.NotNil(t, learningPromoteCmd.Flag("evidence-file"))
}
```

- [ ] **Step 5: Implement `cli/cmd/learning.go`**

Register:

```text
taskline learning capture <task-id>
taskline learning list
taskline learning promote <note-id>
taskline learning reject <note-id>
```

Capture flags:

```text
--project, --kind, --trigger, --guidance, --scope,
--label, --fingerprint, --producer
```

List flags:

```text
--project, --task, --status, --limit
```

Promotion reads `--evidence-file`; rejection requires `--reason`.
Use `requireIdentity()` before mutation commands and `output.Render` for all
output.

- [ ] **Step 6: Extend metrics table**

Print:

```text
Learning notes: 2 pending / 4 promoted / 1 rejected
Promotion rate: 80%
```

Keep existing capsule and helpful-rate lines.

- [ ] **Step 7: Run CLI suite**

Run:

```bash
( cd cli && go test ./... -count=1 )
```

Expected: all CLI tests pass.

- [ ] **Step 8: Commit CLI slice**

```bash
git add cli/client/client.go cli/client/client_test.go \
  cli/client/http_contract_test.go cli/cmd/learning.go \
  cli/cmd/learning_test.go
git commit -m "feat(cli): capture and promote learning notes"
```

## Task 5: Automate agent correction handling

**Files:**
- Modify: `skills/taskline-management/SKILL.md`
- Modify: `scripts/test-skill.sh`

- [ ] **Step 1: Add failing skill smoke checks**

Require these literal command surfaces and safety statements:

```bash
grep -q 'taskline learning capture' skills/taskline-management/SKILL.md
grep -q 'taskline learning promote' skills/taskline-management/SKILL.md
grep -q 'taskline learning reject' skills/taskline-management/SKILL.md
grep -q 'Never capture secrets' skills/taskline-management/SKILL.md
grep -q 'human correction' skills/taskline-management/SKILL.md
```

- [ ] **Step 2: Run skill test and verify red**

Run:

```bash
./scripts/test-skill.sh
```

Expected: failure on missing automatic-learning contract.

- [ ] **Step 3: Add Learning Note protocol**

Add a section with deterministic behavior:

```markdown
## Automatic learning notes

During a claimed task, capture a Learning Note without asking the user when:

- the user corrects a failed tool, command, workflow, or architecture route
  and the correction can help a future task; or
- the agent recovers from a failed approach and verifies a reusable route.

Run `taskline learning capture <task-id>` immediately. Preserve only the
minimal trigger, reusable guidance, scope, labels, fingerprints, and producer.
Never capture secrets, credentials, raw transcripts, guesses, task-only
preferences, or recalled guidance that already existed.

During test or wrap-up, list pending notes for the task. Promote a note only
after commands, tests, artifacts, or merged changes verify it:

`taskline learning promote <note-id> --evidence-file <file>`

Reject disproved guidance:

`taskline learning reject <note-id> --reason "<evidence-backed reason>"`

Leave unverified notes pending. Pending notes are visible but never recalled.
```

Add the Notion example from the design.

- [ ] **Step 4: Integrate stage playbook**

At `dev → test`, capture recoveries. At `test → review`, resolve every pending
note from the task. At `done — wrap-up`, verify no pending note was silently
promoted.

- [ ] **Step 5: Run skill tests**

Run:

```bash
./scripts/test-skill.sh
```

Expected: public and installed skill smoke tests pass.

- [ ] **Step 6: Commit agent protocol**

```bash
git add skills/taskline-management/SKILL.md scripts/test-skill.sh
git commit -m "feat(skill): automate verified learning capture"
```

## Task 6: Show learning candidates in Web

**Files:**
- Modify: `web/src/lib/api.ts`
- Modify: `web/src/lib/api.test.ts`
- Modify: `web/src/lib/i18n.tsx`
- Modify: `web/src/components/KnowledgeView.tsx`
- Create: `web/src/components/KnowledgeView.test.tsx`
- Modify: `web/src/lib/http-contract.test.ts`

- [ ] **Step 1: Write failing API and component tests**

Add API expectation:

```ts
await listLearningNotes("project-1", { status: "pending", limit: 100 });
expect(fetchMock).toHaveBeenCalledWith(
  "/api/v1/projects/project-1/learning-notes?status=pending&limit=100",
  expect.anything()
);
```

Add component behavior:

```tsx
it("shows pending corrections and promotion metrics", async () => {
  mockAPI({
    metrics: {
      pending_note_count: 2,
      promoted_note_count: 4,
      rejected_note_count: 1,
      promotion_rate: 0.8,
    },
    notes: [{
      id: "note-1",
      kind: "human-correction",
      trigger: "Notion link unreadable",
      guidance: "Use one-flow/notion-to-prd",
      scope: "Notion requirements",
      status: "pending",
      producer: "codex",
    }],
  });

  renderKnowledgeView();
  expect(await screen.findByText("Use one-flow/notion-to-prd")).toBeTruthy();
  expect(screen.getByText("2")).toBeTruthy();
  expect(screen.getByText("80%")).toBeTruthy();
});
```

- [ ] **Step 2: Run focused Web tests and verify red**

Run:

```bash
( cd web && pnpm test -- KnowledgeView.test.tsx api.test.ts )
```

Expected: missing type/function/component assertions fail.

- [ ] **Step 3: Add Web API types**

Add:

```ts
export type LearningNoteKind = "human-correction" | "agent-recovery";
export type LearningNoteStatus = "pending" | "promoted" | "rejected";

export interface LearningNote {
  id: string;
  project_id: string;
  source_task_id: string;
  kind: LearningNoteKind;
  trigger: string;
  guidance: string;
  scope: string;
  labels: string[];
  fingerprints: string[];
  producer: string;
  status: LearningNoteStatus;
  evidence: string;
  capsule_id: string;
  rejection_reason: string;
  created_at: number;
  updated_at: number;
  resolved_at: number;
}
```

Add metric fields and `listLearningNotes(project, filters)`.
Add `learning_notes?: LearningNote[]` to the Web `Task` interface. Extend
`web/src/lib/http-contract.test.ts` with `learning_note.json`.

- [ ] **Step 4: Add bilingual copy**

English keys:

```text
Learning candidates
Pending candidates
Promoted candidates
Rejected candidates
Promotion rate
Human correction
Agent recovery
No learning candidates yet
```

Chinese translations:

```text
学习候选
待验证候选
已晋升候选
已拒绝候选
经验晋升率
人工纠正
Agent 自恢复
暂无学习候选
```

- [ ] **Step 5: Extend Knowledge view**

Fetch notes for selected project. Add four candidate metrics. Add a status
filter and candidate cards above active capsules. Each card shows guidance as
heading, trigger, scope, producer, source task, labels, fingerprints, evidence,
and rejection reason when present.

Keep capsule cards and helpful metrics. Do not add Web mutation buttons because
resolution requires authenticated agent evidence.

- [ ] **Step 6: Run Web suite**

Run:

```bash
( cd web && pnpm lint && pnpm test && pnpm build )
```

Expected: lint, all Vitest tests, TypeScript, and Vite build pass.

- [ ] **Step 7: Commit Web slice**

```bash
git add web/src/lib/api.ts web/src/lib/api.test.ts web/src/lib/i18n.tsx \
  web/src/lib/http-contract.test.ts web/src/components/KnowledgeView.tsx \
  web/src/components/KnowledgeView.test.tsx
git commit -m "feat(web): visualize automatic learning progress"
```

## Task 7: Update product and architecture documentation

**Files:**
- Modify: `README.md`
- Modify: `README.zh-CN.md`
- Modify: `PRODUCT.md`
- Modify: `ARCHITECTURE.md`

- [ ] **Step 1: Replace alpha roadmap claims**

Remove statements that evidence-to-memory promotion remains unimplemented.
State exact boundary: agent-driven capture, persisted pending notes, verified
atomic promotion, no passive transcript ingestion.

- [ ] **Step 2: Add end-to-end example**

Document:

```bash
taskline learning capture "$TASK_ID" \
  --project demo \
  --kind human-correction \
  --trigger "Notion requirement could not be read" \
  --guidance "Use one-flow/notion-to-prd for Notion links" \
  --scope "Notion requirement analysis" \
  --label notion --fingerprint notion-to-prd

taskline learning list --project demo --task "$TASK_ID" --status pending
taskline learning promote "$NOTE_ID" --evidence-file ./verified-learning.md
taskline task context "$NEXT_TASK_ID"
```

- [ ] **Step 3: Update architecture schema and trust boundary**

Add `learning_notes` to data model. Explain:

- no FK to task so learning provenance survives task deletion;
- promotion transaction creates capsule once;
- only live task owner captures and resolves;
- active capsules remain sole automatic recall source.

- [ ] **Step 4: Update roadmap**

Mark Learning Note review and evidence promotion complete. Keep team review,
skill/test/rule enforcement, and richer causal metrics on roadmap.

- [ ] **Step 5: Run prose and link checks**

Run:

```bash
rg -n "promotion remains|will promote|manual capsule" README.md README.zh-CN.md PRODUCT.md ARCHITECTURE.md
git diff --check
```

Expected: no stale roadmap claims and no whitespace errors.

- [ ] **Step 6: Commit docs**

```bash
git add README.md README.zh-CN.md PRODUCT.md ARCHITECTURE.md
git commit -m "docs: explain automatic learning loop"
```

## Task 8: Verify rebuilt product end to end

**Files:**
- Modify only when verification reveals a defect.

- [ ] **Step 1: Run all automated suites**

Run:

```bash
( cd server && go test ./... -count=1 )
( cd cli && go test ./... -count=1 )
( cd web && pnpm lint && pnpm test && pnpm build )
./scripts/test-skill.sh
./scripts/build.sh
```

Expected: every command exits 0.

- [ ] **Step 2: Start rebuilt local server**

Run:

```bash
./scripts/start-local.sh
curl -fsS http://127.0.0.1:8787/healthz
```

Expected:

```json
{"ok":true}
```

- [ ] **Step 3: Register isolated smoke identity**

Use a temporary project directory:

```bash
SMOKE_DIR="$(mktemp -d)"
cd "$SMOKE_DIR"
taskline register --name auto-learning-smoke
```

Create project, source task, claim, and capture Notion correction.

- [ ] **Step 4: Verify pending notes never recall**

Create and claim a second Notion task before promotion. Run its context and
assert `suggested_capsules` excludes the pending note.

- [ ] **Step 5: Promote and verify later recall**

Promote with evidence, create a third Notion task, claim it, and run context.
Assert one suggested capsule has the promoted note's `capsule_id`.

- [ ] **Step 6: Verify Knowledge view in browser**

Open `http://127.0.0.1:8787/?project=<smoke-project>&view=knowledge`.
Confirm:

- candidate metrics show one promoted note;
- candidate card shows `Use one-flow/notion-to-prd`;
- active capsule card shows matching scope and evidence;
- English/Chinese switch updates all new copy;
- browser console has no errors.

- [ ] **Step 7: Update task stage documents**

Attach:

- `Dev Notes`: implementation summary and design deviations;
- `Test Report`: commands, pass counts, API/CLI/browser smoke results;
- Learning Note generated from any reusable correction found during this work.

- [ ] **Step 8: Final review and branch delivery**

Run:

```bash
git diff main...HEAD --check
git status --short
git log --oneline main..HEAD
git push -u origin codex/automatic-learning-loop-v1
```

Create a GitHub PR with summary and test plan. Attach PR URL to RunEngram task.
Advance task to `review`. Do not enter `done` until PR review, CI, merge, and
all review threads satisfy existing evidence gates.
