import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import type { MemoryImpact } from "../lib/api";
import { MemoryImpactHistory } from "./MemoryImpactHistory";

const impact: MemoryImpact = {
  id: "impact-1",
  project_id: "project-1",
  task_id: "task-42",
  capsule_id: "capsule-1",
  state: "helpful",
  recall_source: "task-context",
  context_revision: "rev-1",
  recall_score: 0.92,
  recall_reasons: ["label:android", "scope:camscanner"],
  stage: "test",
  notes: "Skipped Gradle and finished static verification.",
  evidence: [
    {
      kind: "task-doc",
      ref: "doc:test-report",
      summary: "Report confirms no Gradle command was executed.",
    },
  ],
  actor: "codex",
  created_at: 1780051741142,
  updated_at: 1780051742142,
  resolved_at: 1780051742142,
};

describe("MemoryImpactHistory", () => {
  afterEach(cleanup);

  it("shows task, recall reason, state, actor, and evidence", () => {
    render(<MemoryImpactHistory impacts={[impact]} />);

    expect(screen.getByText("task-42")).toBeTruthy();
    expect(screen.getByText("Helpful")).toBeTruthy();
    expect(screen.getByText("label:android")).toBeTruthy();
    expect(screen.getByText("doc:test-report")).toBeTruthy();
    expect(screen.getByText(/Report confirms no Gradle/)).toBeTruthy();
    expect(screen.getByText(/codex/)).toBeTruthy();
  });

  it("explains an unconfirmed receipt", () => {
    render(
      <MemoryImpactHistory
        impacts={[{ ...impact, state: "unconfirmed", evidence: [], actor: "system" }]}
      />
    );

    expect(
      screen.getByText(
        "Memory was recalled, but this completed task did not record whether it helped."
      )
    ).toBeTruthy();
  });
});
