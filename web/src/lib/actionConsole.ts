import type { Task } from "./api";

export type ActionFocus =
  | { kind: "active"; task: Task }
  | { kind: "ready"; task: Task }
  | { kind: "blocked"; task: Task; blockers: Task[] }
  | { kind: "outcome"; task: Task }
  | { kind: "empty" };

function byPriorityThenAge(left: Task, right: Task) {
  return left.priority - right.priority || left.created_at - right.created_at;
}

function blockersFor(task: Task, tasks: Task[]): Task[] {
  const dependencyIDs = new Set(task.depends_on ?? []);
  return tasks.filter(
    (candidate) => dependencyIDs.has(candidate.id) && candidate.state !== "done"
  );
}

export function selectActionFocus(
  tasks: Task[],
  now = Date.now()
): ActionFocus {
  const active = tasks
    .filter(
      (task) =>
        task.state !== "done" &&
        task.state !== "pending" &&
        !!task.owner &&
        (task.lease_expires_at ?? 0) > now
    )
    .sort(
      (left, right) =>
        (right.lease_expires_at ?? 0) - (left.lease_expires_at ?? 0) ||
        byPriorityThenAge(left, right)
    )[0];
  if (active) return { kind: "active", task: active };

  const candidates = tasks
    .filter((task) => task.state !== "done" && task.state !== "pending")
    .sort(byPriorityThenAge);
  const ready = candidates.find((task) => blockersFor(task, tasks).length === 0);
  if (ready) return { kind: "ready", task: ready };

  const blocked = candidates[0];
  if (blocked) {
    return {
      kind: "blocked",
      task: blocked,
      blockers: blockersFor(blocked, tasks),
    };
  }

  const outcome = tasks
    .filter((task) => task.state === "done")
    .sort(
      (left, right) =>
        (right.completed_at ?? right.updated_at) -
        (left.completed_at ?? left.updated_at)
    )[0];
  return outcome ? { kind: "outcome", task: outcome } : { kind: "empty" };
}

export function taskPrompt(projectName: string, focus: ActionFocus): string {
  if (focus.kind === "empty") {
    return `Use taskline-management to create the first RunEngram task for project ${projectName}.`;
  }
  const verb = focus.kind === "active" ? "continue" : "run";
  return `Use taskline-management to ${verb} task ${focus.task.id} in project ${projectName}.`;
}
