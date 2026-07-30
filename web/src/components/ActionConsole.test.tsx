import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { ExplorationCapsule, MemoryImpact, Project, Task } from "../lib/api";
import { ActionConsole } from "./ActionConsole";

const project: Project = {
  id: "project-1",
  name: "CamScanner",
  description: "",
  created_at: 1,
  updated_at: 1,
};

const task: Task = {
  id: "task-1",
  project_id: project.id,
  title: "Inspect password flow",
  description: "",
  type: "feature",
  state: "dev",
  priority: 1,
  labels: [],
  depends_on: [],
  images: [],
  docs: [],
  links: [],
  owner: "codex",
  claimed_at: 1,
  lease_expires_at: Date.now() + 60_000,
  created_at: 1,
  updated_at: 1,
};

const capsule: ExplorationCapsule = {
  id: "capsule-1",
  project_id: project.id,
  source_task_id: "source-1",
  memory_class: "project-rule",
  trigger: "Every task",
  title: "Do not run Gradle",
  summary: "Use static inspection.",
  scope: "CamScanner",
  evidence: "Verified.",
  labels: [],
  fingerprints: [],
  producer: "codex",
  status: "active",
  validation: "verified",
  confidence: 0.9,
  use_count: 1,
  helpful_count: 1,
  rejected_count: 0,
  relations: [],
  created_at: 1,
  updated_at: 1,
};

const impact: MemoryImpact = {
  id: "impact-1",
  project_id: project.id,
  task_id: task.id,
  capsule_id: capsule.id,
  state: "recalled",
  recall_source: "task-context",
  context_revision: "rev-1",
  recall_score: 1,
  recall_reasons: ["project-rule"],
  stage: "",
  notes: "",
  evidence: [],
  actor: "",
  created_at: 1,
  updated_at: 1,
  resolved_at: 0,
};

describe("ActionConsole memory impact", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("shows recalled memory and its task-level impact receipt", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async (input: RequestInfo | URL) => {
        const url = String(input);
        if (url.endsWith(`/tasks/${task.id}/resume`)) {
          return new Response(
            JSON.stringify({
              task,
              snapshot: {
                id: "snapshot-1",
                task_id: task.id,
                project_id: project.id,
                task,
                project_rules: [capsule],
                suggested_capsules: [],
                context_revision: "rev-1",
                explanations: [],
                created_at: 1,
              },
              latest_run: null,
              work_graph: null,
            }),
            { status: 200, headers: { "Content-Type": "application/json" } }
          );
        }
        if (url.includes("/memory-impacts")) {
          return new Response(JSON.stringify([impact]), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          });
        }
        return new Response(
          JSON.stringify({
            active_capsule_count: 1,
            pending_note_count: 0,
            run_count: 1,
          }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      })
    );
    const queryClient = new QueryClient({
      defaultOptions: { queries: { retry: false } },
    });

    render(
      <QueryClientProvider client={queryClient}>
        <ActionConsole
          project={project}
          tasks={[task]}
          loading={false}
          error={null}
          onNavigate={vi.fn()}
        />
      </QueryClientProvider>
    );

    expect(await screen.findByText("Memory affected this task")).toBeTruthy();
    expect(await screen.findAllByText("Do not run Gradle")).toHaveLength(2);
    expect(screen.getByText("project-rule")).toBeTruthy();
  });
});
