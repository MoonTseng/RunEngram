// Thin REST wrapper for taskline-server. Mirrors the canonical Project /
// Task shapes from server/api/model/model.go — keep them in sync.

export type TaskState =
  | "pending"
  | "start"
  | "spec"
  | "dev"
  | "test"
  | "review"
  | "done";

export const STATES: TaskState[] = [
  "pending",
  "start",
  "spec",
  "dev",
  "test",
  "review",
  "done",
];

export const STATE_LABELS: Record<TaskState, string> = {
  pending: "Pending",
  start: "Start",
  spec: "Spec",
  dev: "Dev",
  test: "Test",
  review: "Review",
  done: "Done",
};

export type TaskType = "feature" | "bug" | "docs";

export interface Project {
  id: string;
  name: string;
  description: string;
  created_at: number;
  updated_at: number;
}

export interface Agent {
  id: string;
  name: string;
  created_at: number;
  updated_at: number;
}

export interface ActiveClaim {
  id: string;
  title: string;
  claimed_at: number;
  claimed_for_ms: number;
  lease_expires_at: number;
}

export interface ServerStatus {
  ok: boolean;
  server_time: number;
  agent?: Agent;
  active_tasks: ActiveClaim[];
}

export interface Task {
  id: string;
  project_id: string;
  title: string;
  description: string;
  type: TaskType;
  state: TaskState;
  priority: number;
  labels: string[];
  owner?: string;
  claimed_at?: number;
  lease_expires_at?: number;
  completed_at?: number;
  depends_on?: string[];
  images?: TaskImage[];
  docs?: TaskDoc[];
  links?: TaskLink[];
  learning_notes?: LearningNote[];
  created_at: number;
  updated_at: number;
}

export interface TaskEvent {
  id: string;
  task_id: string;
  actor: string;
  action: string;
  summary: string;
  details: Record<string, unknown>;
  created_at: number;
}

export type CapsuleStatus = "active" | "stale" | "archived";
export type MemoryClass = "experience" | "project-rule";
export type MemoryValidation = "verified" | "trusted" | "disputed" | "stale";
export type MemoryRelationType =
  | "derived-from"
  | "validated-by"
  | "applies-to"
  | "supersedes"
  | "conflicts-with"
  | "caused-by";
export type MemoryRelationTargetKind = "capsule" | "task" | "artifact" | "scope";
export type MemoryRelationDirection = "outgoing" | "incoming";

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

export interface UpdateLearningNoteInput {
  trigger: string;
  guidance: string;
  scope: string;
}

export interface ExplorationCapsule {
  id: string;
  project_id: string;
  source_task_id: string;
  memory_class: MemoryClass;
  trigger: string;
  title: string;
  summary: string;
  scope: string;
  evidence: string;
  labels: string[];
  fingerprints: string[];
  producer: string;
  status: CapsuleStatus;
  validation: MemoryValidation;
  confidence: number;
  use_count: number;
  helpful_count: number;
  rejected_count: number;
  relations: MemoryRelation[];
  created_at: number;
  updated_at: number;
}

export interface MemoryRelation {
  id: string;
  project_id: string;
  source_capsule_id: string;
  type: MemoryRelationType;
  target_kind: MemoryRelationTargetKind;
  target_ref: string;
  note: string;
  direction: MemoryRelationDirection;
  created_at: number;
}

export interface MemoryRecallReason {
  code: string;
  value?: string;
}

export interface MemoryRecallExplanation {
  capsule_id: string;
  score: number;
  reasons: MemoryRecallReason[];
  warnings: string[];
}

export interface ContextSnapshot {
  id: string;
  task_id: string;
  project_id: string;
  task: Task;
  project_rules: ExplorationCapsule[];
  suggested_capsules: ExplorationCapsule[];
  context_revision: string;
  explanations: MemoryRecallExplanation[];
  created_at: number;
}

export interface UpdateCapsuleInput {
  memory_class?: MemoryClass;
  trigger?: string;
  title?: string;
  summary?: string;
  scope?: string;
  evidence?: string;
  labels?: string[];
  fingerprints?: string[];
  status?: CapsuleStatus;
  expected_updated_at: number;
}

export interface CreateMemoryRelationInput {
  type: MemoryRelationType;
  target_kind: MemoryRelationTargetKind;
  target_ref: string;
  note?: string;
}

export interface AgentRun {
  id: string;
  task_id: string;
  project_id: string;
  agent_name: string;
  agent_tool: "codex" | "claude-code" | "pi" | "other";
  workflow_template: string;
  workflow_version: number;
  status: "running" | "blocked" | "completed" | "failed";
  summary: string;
  next_step: string;
  started_at: number;
  updated_at: number;
  completed_at: number;
}

export interface TaskResumeContext {
  snapshot: ContextSnapshot;
  latest_run?: AgentRun;
  recent_events: TaskEvent[];
  work_graph?: RunWorkGraph;
}

export type RunNodeStatus =
  | "pending"
  | "ready"
  | "running"
  | "waiting"
  | "completed"
  | "failed"
  | "skipped";

export interface RunNode {
  id: string;
  run_id: string;
  key: string;
  title: string;
  capability: string;
  kind: string;
  position: number;
  depends_on: string[];
  status: RunNodeStatus;
  attempt: number;
  summary: string;
  next_step: string;
  artifact_ids: string[];
  evidence: string;
  input_fingerprint: string;
  started_at: number;
  completed_at: number;
  updated_at: number;
}

export interface RunInterrupt {
  id: string;
  run_id: string;
  node_key: string;
  kind: "approval" | "question" | "choice" | "conflict";
  prompt: string;
  options: string[];
  status: "pending" | "answered" | "rejected";
  response: string;
  requested_by: string;
  responded_by: string;
  created_at: number;
  resolved_at: number;
}

export interface RunWorkGraph {
  run_id: string;
  template: string;
  version: number;
  nodes: RunNode[];
  interrupts: RunInterrupt[];
  completed_node_count: number;
  verified_node_count: number;
  artifact_count: number;
  open_interrupt_count: number;
  progress_percent: number;
}

export interface LearningMetrics {
  capsule_count: number;
  active_capsule_count: number;
  learning_note_count: number;
  pending_note_count: number;
  promoted_note_count: number;
  rejected_note_count: number;
  snapshot_task_count: number;
  reused_task_count: number;
  helpful_count: number;
  rejected_count: number;
  stale_count: number;
  helpful_rate: number;
  promotion_rate: number;
  run_count: number;
  completed_run_count: number;
  active_run_count: number;
  blocked_run_count: number;
  resumed_run_count: number;
  run_completion_rate: number;
  recovery_rate: number;
}

export interface TaskImage {
  id: string;
  task_id: string;
  filename: string;
  mime_type: string;
  size_bytes: number;
  url?: string;
  uploaded_at: number;
}

export interface TaskDoc {
  id: string;
  task_id: string;
  title: string;
  url?: string;
  content?: string;
  created_at: number;
  updated_at: number;
}

export interface TaskLink {
  id: string;
  task_id: string;
  url: string;
  label: string;
  created_at: number;
}

class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
  }
}

async function request<T>(
  method: string,
  path: string,
  body?: unknown
): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: {
      "X-Taskline-Client": "web",
      ...(body ? { "Content-Type": "application/json" } : {}),
    },
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw await readApiError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

async function readApiError(res: Response): Promise<ApiError> {
  let msg = res.statusText;
  try {
    const j = (await res.json()) as { error?: string };
    if (j?.error) msg = j.error;
  } catch {
    // body wasn't JSON; keep statusText
  }
  return new ApiError(res.status, msg);
}

// ─── Projects ──────────────────────────────────────────────────────────

export async function listProjects(): Promise<Project[]> {
  const r = await request<{ projects: Project[] }>("GET", "/api/v1/projects");
  return r.projects ?? [];
}

export async function createProject(
  name: string,
  description: string
): Promise<Project> {
  return request<Project>("POST", "/api/v1/projects", { name, description });
}

export async function deleteProject(projectIdOrName: string): Promise<void> {
  await request(
    "DELETE",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}`
  );
}

// ─── Tasks ─────────────────────────────────────────────────────────────

export async function listTasks(projectIdOrName: string): Promise<Task[]> {
  const r = await request<{ tasks: Task[] }>(
    "GET",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/tasks`
  );
  return r.tasks ?? [];
}

export async function getTaskContext(taskId: string): Promise<ContextSnapshot> {
  return request<ContextSnapshot>(
    "GET",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/context`
  );
}

export async function getTaskResumeContext(taskId: string): Promise<TaskResumeContext> {
  return request<TaskResumeContext>(
    "GET",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/resume`
  );
}

export async function resolveRunInterrupt(
  interruptId: string,
  response: string,
  reject = false
): Promise<RunInterrupt> {
  return request<RunInterrupt>(
    "PATCH",
    `/api/v1/interrupts/${encodeURIComponent(interruptId)}`,
    { response, reject }
  );
}

export async function listCapsules(
  projectIdOrName: string,
  query = "",
  status: CapsuleStatus | "" = "active"
): Promise<ExplorationCapsule[]> {
  const params = new URLSearchParams();
  if (query) params.set("q", query);
  if (status) params.set("status", status);
  const suffix = params.size > 0 ? `?${params.toString()}` : "";
  const response = await request<{ capsules: ExplorationCapsule[] }>(
    "GET",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/capsules${suffix}`
  );
  return response.capsules ?? [];
}

export async function updateCapsule(
  capsuleId: string,
  input: UpdateCapsuleInput
): Promise<ExplorationCapsule> {
  return request<ExplorationCapsule>(
    "PATCH",
    `/api/v1/capsules/${encodeURIComponent(capsuleId)}`,
    input
  );
}

export async function createMemoryRelation(
  capsuleId: string,
  input: CreateMemoryRelationInput
): Promise<MemoryRelation> {
  return request<MemoryRelation>(
    "POST",
    `/api/v1/capsules/${encodeURIComponent(capsuleId)}/relations`,
    input
  );
}

export async function deleteMemoryRelation(relationId: string): Promise<void> {
  await request(
    "DELETE",
    `/api/v1/memory-relations/${encodeURIComponent(relationId)}`
  );
}

export async function getLearningMetrics(
  projectIdOrName: string
): Promise<LearningMetrics> {
  return request<LearningMetrics>(
    "GET",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/learning-metrics`
  );
}

export async function listLearningNotes(
  projectIdOrName: string,
  filters: { status?: LearningNoteStatus | ""; limit?: number } = {}
): Promise<LearningNote[]> {
  const params = new URLSearchParams();
  if (filters.status) params.set("status", filters.status);
  if (filters.limit && filters.limit > 0) {
    params.set("limit", String(filters.limit));
  }
  const suffix = params.size > 0 ? `?${params.toString()}` : "";
  const response = await request<{ learning_notes: LearningNote[] }>(
    "GET",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/learning-notes${suffix}`
  );
  return response.learning_notes ?? [];
}

export async function updateLearningNote(
  noteId: string,
  input: UpdateLearningNoteInput
): Promise<LearningNote> {
  return request<LearningNote>(
    "PATCH",
    `/api/v1/learning-notes/${encodeURIComponent(noteId)}`,
    input
  );
}

export async function promoteLearningNote(
  noteId: string,
  evidence: string,
  memoryClass: MemoryClass
): Promise<LearningNote> {
  return request<LearningNote>(
    "POST",
    `/api/v1/learning-notes/${encodeURIComponent(noteId)}/promote`,
    { evidence, memory_class: memoryClass }
  );
}

export async function rejectLearningNote(
  noteId: string,
  reason: string
): Promise<LearningNote> {
  return request<LearningNote>(
    "POST",
    `/api/v1/learning-notes/${encodeURIComponent(noteId)}/reject`,
    { reason }
  );
}

export async function searchTasks(
  projectIdOrName: string,
  query: string,
  limit = 20
): Promise<Task[]> {
  const params = new URLSearchParams({ q: query });
  if (limit > 0) params.set("limit", String(limit));
  const r = await request<{ tasks: Task[] }>(
    "GET",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/tasks/search?${params.toString()}`
  );
  return r.tasks ?? [];
}

export async function createTask(
  projectIdOrName: string,
  input: {
    title: string;
    description?: string;
    type: TaskType;
    priority: number;
    labels?: string[];
    auto_start?: boolean;
  }
): Promise<Task> {
  return request<Task>(
    "POST",
    `/api/v1/projects/${encodeURIComponent(projectIdOrName)}/tasks`,
    input
  );
}

export async function listTaskEvents(taskId: string): Promise<TaskEvent[]> {
  const response = await request<{ events: TaskEvent[] }>(
    "GET",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/events`
  );
  return response.events ?? [];
}

export async function updateTask(
  id: string,
  patch: Partial<
    Pick<Task, "title" | "description" | "type" | "state" | "priority" | "labels">
  > & { force?: boolean }
): Promise<Task> {
  return request<Task>("PATCH", `/api/v1/tasks/${encodeURIComponent(id)}`, patch);
}

export async function deleteTask(id: string): Promise<void> {
  await request<unknown>("DELETE", `/api/v1/tasks/${encodeURIComponent(id)}`);
}

export async function uploadTaskImage(
  taskId: string,
  file: File
): Promise<TaskImage> {
  const body = new FormData();
  body.append("file", file);
  const res = await fetch(
    `/api/v1/tasks/${encodeURIComponent(taskId)}/images`,
    {
      method: "POST",
      headers: { "X-Taskline-Client": "web" },
      body,
    }
  );
  if (!res.ok) throw await readApiError(res);
  return (await res.json()) as TaskImage;
}

export function taskImageURL(imageId: string): string {
  return `/api/v1/images/${encodeURIComponent(imageId)}`;
}

export async function deleteTaskImage(imageId: string): Promise<void> {
  await request<unknown>("DELETE", taskImageURL(imageId));
}

export async function createTaskDoc(
  taskId: string,
  input: { title: string; content: string }
): Promise<TaskDoc> {
  return request<TaskDoc>(
    "POST",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/docs`,
    input
  );
}

export async function getTaskDoc(docId: string): Promise<TaskDoc> {
  return request<TaskDoc>("GET", `/api/v1/docs/${encodeURIComponent(docId)}`);
}

export async function updateTaskDoc(
  docId: string,
  patch: { title?: string; content?: string }
): Promise<TaskDoc> {
  return request<TaskDoc>(
    "PATCH",
    `/api/v1/docs/${encodeURIComponent(docId)}`,
    patch
  );
}

export function taskDocContentURL(docId: string): string {
  return `/api/v1/docs/${encodeURIComponent(docId)}/content`;
}

export async function deleteTaskDoc(docId: string): Promise<void> {
  await request<unknown>("DELETE", `/api/v1/docs/${encodeURIComponent(docId)}`);
}

export async function addDependency(
  taskId: string,
  dependsOn: string
): Promise<void> {
  await request<unknown>(
    "POST",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/deps`,
    { depends_on: dependsOn }
  );
}

export async function deleteDependency(
  taskId: string,
  dependsOn: string
): Promise<void> {
  await request<unknown>(
    "DELETE",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/deps/${encodeURIComponent(dependsOn)}`
  );
}

export async function addLink(
  taskId: string,
  url: string,
  label: string
): Promise<TaskLink> {
  return request<TaskLink>(
    "POST",
    `/api/v1/tasks/${encodeURIComponent(taskId)}/links`,
    { url, label }
  );
}

export async function deleteLink(linkId: string): Promise<void> {
  await request<unknown>(
    "DELETE",
    `/api/v1/links/${encodeURIComponent(linkId)}`
  );
}

export { ApiError };
