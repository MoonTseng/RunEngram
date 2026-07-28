import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
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
    const fetchMock = vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input);
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
                guidance: "Use one-flow/notion-to-prd",
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

    expect(await screen.findByText("Use one-flow/notion-to-prd")).toBeTruthy();
    expect(screen.getAllByText("Pending candidates")).toHaveLength(2);
    expect(screen.getByText("Promotion rate")).toBeTruthy();
    expect(screen.getByText("80%")).toBeTruthy();
    expect(screen.getByText(/Human correction/)).toBeTruthy();
  });
});
