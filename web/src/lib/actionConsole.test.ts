import { describe, expect, it } from "vitest";
import type { Task } from "./api";
import { selectActionFocus, taskPrompt } from "./actionConsole";

const base: Task = {
  id: "task-1",
  project_id: "project-1",
  title: "Task",
  description: "",
  type: "feature",
  state: "start",
  priority: 1,
  labels: [],
  depends_on: [],
  created_at: 100,
  updated_at: 100,
};

describe("selectActionFocus", () => {
  it("prefers the live claim with the latest lease", () => {
    const result = selectActionFocus(
      [
        { ...base, id: "one", owner: "codex", lease_expires_at: 500 },
        { ...base, id: "two", owner: "codex", lease_expires_at: 900 },
      ],
      200
    );
    expect(result.kind).toBe("active");
    if (result.kind === "active") expect(result.task.id).toBe("two");
  });

  it("returns a ready task before a blocked higher-priority task", () => {
    const result = selectActionFocus([
      { ...base, id: "blocked", priority: 0, depends_on: ["dependency"] },
      { ...base, id: "dependency", state: "pending" },
      { ...base, id: "ready", priority: 2 },
    ]);
    expect(result.kind).toBe("ready");
    if (result.kind === "ready") expect(result.task.id).toBe("ready");
  });

  it("explains blockers when nothing can run", () => {
    const result = selectActionFocus([
      { ...base, id: "blocked", depends_on: ["dependency"] },
      { ...base, id: "dependency", state: "pending" },
    ]);
    expect(result.kind).toBe("blocked");
    if (result.kind === "blocked") expect(result.blockers[0].id).toBe("dependency");
  });

  it("surfaces latest completed outcome when no work is runnable", () => {
    const result = selectActionFocus([
      { ...base, id: "old", state: "done", completed_at: 200 },
      { ...base, id: "new", state: "done", completed_at: 300 },
    ]);
    expect(result.kind).toBe("outcome");
    if (result.kind === "outcome") expect(result.task.id).toBe("new");
  });

  it("builds a continuation prompt for active work", () => {
    const focus = {
      kind: "active" as const,
      task: { ...base, owner: "codex", lease_expires_at: 900 },
    };
    expect(taskPrompt("CamScanner", focus)).toContain("continue task task-1");
  });
});
