import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { ExplorationCapsule, MemoryImpact } from "../lib/api";
import { TaskMemoryImpactPanel } from "./TaskMemoryImpactPanel";

const capsule: ExplorationCapsule = {
  id: "capsule-1",
  project_id: "project-1",
  source_task_id: "task-old",
  memory_class: "project-rule",
  trigger: "Every task",
  title: "Do not run Gradle",
  summary: "Use static inspection unless developer requests a build.",
  scope: "CamScanner",
  evidence: "Verified policy.",
  labels: [],
  fingerprints: [],
  producer: "codex",
  status: "active",
  validation: "verified",
  confidence: 0.9,
  use_count: 2,
  helpful_count: 2,
  rejected_count: 0,
  relations: [],
  created_at: 1,
  updated_at: 1,
};

const impact: MemoryImpact = {
  id: "impact-1",
  project_id: "project-1",
  task_id: "task-1",
  capsule_id: capsule.id,
  state: "recalled",
  recall_source: "task-context",
  context_revision: "rev-1",
  recall_score: 0.95,
  recall_reasons: ["project-rule"],
  stage: "",
  notes: "",
  evidence: [],
  actor: "",
  created_at: 1,
  updated_at: 42,
  resolved_at: 0,
};

describe("TaskMemoryImpactPanel", () => {
  afterEach(cleanup);

  it("shows why memory was recalled and current outcome", () => {
    render(
      <TaskMemoryImpactPanel
        impacts={[impact]}
        capsules={[capsule]}
        onUpdate={vi.fn()}
      />
    );

    expect(screen.getByText("Do not run Gradle")).toBeTruthy();
    expect(screen.getByText("project-rule")).toBeTruthy();
    expect(screen.getByText("Recalled")).toBeTruthy();
  });

  it("records helpful outcome with typed evidence", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn().mockResolvedValue({ ...impact, state: "helpful" });
    render(
      <TaskMemoryImpactPanel
        impacts={[impact]}
        capsules={[capsule]}
        onUpdate={onUpdate}
      />
    );

    await user.click(screen.getByRole("button", { name: "Confirm helpful" }));
    await user.type(screen.getByLabelText("Stage"), "test");
    await user.type(
      screen.getByLabelText("What changed?"),
      "Skipped Gradle and completed static verification."
    );
    await user.selectOptions(screen.getByLabelText("Evidence type"), "task-doc");
    await user.type(screen.getByLabelText("Evidence reference"), "doc:test-report");
    await user.type(
      screen.getByLabelText("Observed result"),
      "Test report confirms no Gradle command was executed."
    );
    await user.click(screen.getByRole("button", { name: "Save result" }));

    expect(onUpdate).toHaveBeenCalledWith(
      impact.id,
      expect.objectContaining({
        state: "helpful",
        stage: "test",
        expected_updated_at: 42,
        evidence: [
          {
            kind: "task-doc",
            ref: "doc:test-report",
            summary: "Test report confirms no Gradle command was executed.",
          },
        ],
      })
    );
  });

  it("preserves draft when save fails", async () => {
    const user = userEvent.setup();
    const onUpdate = vi.fn().mockRejectedValue(new Error("conflict"));
    render(
      <TaskMemoryImpactPanel
        impacts={[impact]}
        capsules={[capsule]}
        onUpdate={onUpdate}
      />
    );

    await user.click(screen.getByRole("button", { name: "Mark applied" }));
    const notes = screen.getByLabelText("What changed?");
    await user.type(notes, "Used static inspection.");
    await user.click(screen.getByRole("button", { name: "Save result" }));

    expect(await screen.findByText("Error: conflict")).toBeTruthy();
    expect((notes as HTMLTextAreaElement).value).toBe("Used static inspection.");
  });
});
