import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { Project } from "../lib/api";
import { KnowledgeView } from "./KnowledgeView";

const project: Project = {
  id: "project-1",
  name: "RunEngram",
  description: "",
  created_at: 1785220000000,
  updated_at: 1785220000000,
};

function renderKnowledgeView() {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  render(
    <QueryClientProvider client={queryClient}>
      <KnowledgeView project={project} />
    </QueryClientProvider>
  );
}

describe("KnowledgeView", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("shows pending corrections and promotion metrics", async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.includes("/learning-notes/note-1/promote") && init?.method === "POST") {
        return new Response(
          JSON.stringify({
            id: "note-1",
            project_id: project.id,
            source_task_id: "task-1",
            kind: "human-correction",
            trigger: "Notion requirements need normalization",
            guidance: "Use project/requirement-import before PRD analysis",
            scope: "Notion requirements",
            labels: ["notion"],
            fingerprints: ["notion-to-prd"],
            producer: "codex",
            status: "promoted",
            evidence: "Maintainer verified requirement import output.",
            capsule_id: "capsule-1",
            rejection_reason: "",
            created_at: 1785220000000,
            updated_at: 1785220002000,
            resolved_at: 1785220002000,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.includes("/learning-notes/note-1") && init?.method === "PATCH") {
        return new Response(
          JSON.stringify({
            id: "note-1",
            project_id: project.id,
            source_task_id: "task-1",
            kind: "human-correction",
            trigger: "Notion requirements need normalization",
            guidance: "Use project/requirement-import before PRD analysis",
            scope: "Notion requirements",
            labels: ["notion"],
            fingerprints: ["notion-to-prd"],
            producer: "codex",
            status: "pending",
            evidence: "",
            capsule_id: "",
            rejection_reason: "",
            created_at: 1785220000000,
            updated_at: 1785220001000,
            resolved_at: 0,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.includes("/learning-metrics")) {
        return new Response(
          JSON.stringify({
            capsule_count: 4,
            active_capsule_count: 4,
            learning_note_count: 7,
            pending_note_count: 2,
            promoted_note_count: 4,
            rejected_note_count: 1,
            snapshot_task_count: 3,
            reused_task_count: 2,
            helpful_count: 2,
            rejected_count: 0,
            stale_count: 0,
            helpful_rate: 1,
            promotion_rate: 0.8,
            run_count: 5,
            completed_run_count: 4,
            active_run_count: 1,
            blocked_run_count: 2,
            resumed_run_count: 2,
            run_completion_rate: 0.8,
            recovery_rate: 0.5,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.includes("/learning-notes")) {
        return new Response(
          JSON.stringify({
            learning_notes: [
              {
                id: "note-1",
                project_id: project.id,
                source_task_id: "task-1",
                kind: "human-correction",
                trigger: "Notion link unreadable",
                guidance: "Use project/requirement-import",
                scope: "Notion requirements",
                labels: ["notion"],
                fingerprints: ["notion-to-prd"],
                producer: "codex",
                status: "pending",
                evidence: "",
                capsule_id: "",
                rejection_reason: "",
                created_at: 1785220000000,
                updated_at: 1785220000000,
                resolved_at: 0,
              },
            ],
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      return new Response(JSON.stringify({ capsules: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    renderKnowledgeView();

    await user.click(await screen.findByRole("tab", { name: /Pending review/ }));
    expect(await screen.findByText("Use project/requirement-import")).toBeTruthy();
    expect(screen.getAllByText("Pending review")).toHaveLength(2);
    expect(screen.getAllByText("Verified")).toHaveLength(2);
    expect(screen.getByText(/Human correction/)).toBeTruthy();
    expect(screen.getByText("Agent runs")).toBeTruthy();
    expect(screen.getByText("5")).toBeTruthy();
    expect(screen.getByText("Recovery rate")).toBeTruthy();
    expect(screen.getByText("50%")).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Edit" }));
    const guidance = screen.getByLabelText("Reusable guidance");
    await user.clear(guidance);
    await user.type(guidance, "Use project/requirement-import before PRD analysis");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/learning-notes/note-1"),
      expect.objectContaining({ method: "PATCH" })
    );

    await user.click(screen.getByRole("button", { name: "Validate & enable" }));
    await user.selectOptions(screen.getByLabelText("Memory type"), "project-rule");
    await user.type(
      screen.getByLabelText("Validation evidence"),
      "Maintainer verified requirement import output."
    );
    await user.click(screen.getByRole("button", { name: "Confirm enable" }));
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/learning-notes/note-1/promote"),
      expect.objectContaining({
        method: "POST",
        body: expect.stringContaining('"memory_class":"project-rule"'),
      })
    );
  });
});
