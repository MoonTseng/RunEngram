import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { cleanup, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { LearningNote, MemoryClass } from "../lib/api";
import { LearningEvidenceReview } from "./LearningEvidenceReview";

const note: LearningNote = {
  id: "note-1",
  project_id: "project-1",
  source_task_id: "task-1",
  kind: "human-correction",
  trigger: "Service dependency placement",
  guidance: "Keep service module dependencies in the existing service group.",
  scope: "Gradle service modules",
  labels: ["gradle"],
  fingerprints: ["service-dependencies"],
  producer: "codex",
  status: "pending",
  evidence: "",
  capsule_id: "",
  rejection_reason: "",
  created_at: 1785220000000,
  updated_at: 1785220000000,
  resolved_at: 0,
};

function successResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function sourceTaskResponse() {
  return {
    id: "task-1",
    project_id: "project-1",
    title: "Move service dependency",
    description: "",
    type: "refactor",
    state: "done",
    priority: 1,
    labels: [],
    docs: [
      {
        id: "doc-1",
        task_id: "task-1",
        title: "Spec.md",
        url: "/api/v1/docs/doc-1/content",
        created_at: 1785220000000,
        updated_at: 1785220000000,
      },
    ],
    links: [
      {
        id: "link-1",
        task_id: "task-1",
        label: "Architecture review",
        url: "https://example.com/review",
        created_at: 1785220000000,
      },
    ],
    created_at: 1785220000000,
    updated_at: 1785220000000,
  };
}

function stubSourceMaterial(options?: { taskFails?: boolean }) {
  const fetchMock = vi.fn((input: RequestInfo | URL) => {
    const url = String(input);
    if (url.endsWith("/api/v1/tasks/task-1")) {
      return Promise.resolve(
        options?.taskFails
          ? new Response(JSON.stringify({ error: "not found" }), {
              status: 404,
              headers: { "Content-Type": "application/json" },
            })
          : successResponse(sourceTaskResponse())
      );
    }
    if (url.endsWith("/api/v1/tasks/task-1/events")) {
      return Promise.resolve(
        successResponse({
          events: [
            {
              id: "event-1",
              task_id: "task-1",
              actor: "codex",
              action: "updated",
              summary: "Verified clean Gradle sync",
              details: {},
              created_at: 1785220001000,
            },
          ],
        })
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({ error: `unexpected ${url}` }), {
        status: 500,
        headers: { "Content-Type": "application/json" },
      })
    );
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}

function renderReview(
  options: {
    onPromote?: (evidence: string, memoryClass: MemoryClass) => Promise<void>;
    taskFails?: boolean;
  } = {}
) {
  stubSourceMaterial({ taskFails: options.taskFails });
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  const onPromote = options.onPromote ?? vi.fn().mockResolvedValue(undefined);
  render(
    <QueryClientProvider client={queryClient}>
      <LearningEvidenceReview note={note} onPromote={onPromote} onCancel={vi.fn()} />
    </QueryClientProvider>
  );
  return { onPromote };
}

describe("LearningEvidenceReview", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("explains why evidence is required and changes examples without losing text", async () => {
    const user = userEvent.setup();
    renderReview();

    expect(
      screen.getByRole("heading", {
        name: "Confirm whether this experience is reliable",
      })
    ).toBeTruthy();
    expect(screen.getByText(/given to later Agents/)).toBeTruthy();

    const evidence = screen.getByLabelText("What did you verify?") as HTMLTextAreaElement;
    await user.type(evidence, "Checked project convention.");
    await user.click(screen.getByRole("radio", { name: "Code or configuration" }));

    expect(evidence.value).toBe("Checked project convention.");
    expect(screen.getByText(/file path/)).toBeTruthy();
  });

  it("inserts a task reference but requires an observed result", async () => {
    const user = userEvent.setup();
    renderReview();

    await user.click(await screen.findByRole("button", { name: /Spec.md/ }));

    const evidence = screen.getByLabelText("What did you verify?") as HTMLTextAreaElement;
    expect(evidence.value).toContain('Checked: document "Spec.md"');
    const submit = screen.getByRole("button", {
      name: "Verify and use in later tasks",
    }) as HTMLButtonElement;
    expect(submit.disabled).toBe(true);

    await user.type(evidence, "the reviewed requirement matches implementation scope");
    expect(submit.disabled).toBe(false);
  });

  it("submits manual evidence and selected memory type", async () => {
    const user = userEvent.setup();
    const { onPromote } = renderReview();

    await user.selectOptions(screen.getByLabelText("Memory type"), "project-rule");
    await user.type(
      screen.getByLabelText("What did you verify?"),
      "Two neighboring modules use the same dependency group."
    );
    await user.click(
      screen.getByRole("button", { name: "Verify and use in later tasks" })
    );

    await waitFor(() =>
      expect(onPromote).toHaveBeenCalledWith(
        "Two neighboring modules use the same dependency group.",
        "project-rule"
      )
    );
  });

  it("preserves evidence when promotion fails", async () => {
    const user = userEvent.setup();
    const onPromote = vi.fn().mockRejectedValue(new Error("promotion failed"));
    renderReview({ onPromote });

    const evidence = screen.getByLabelText("What did you verify?") as HTMLTextAreaElement;
    await user.type(evidence, "Two neighboring modules use the same dependency group.");
    await user.click(
      screen.getByRole("button", { name: "Verify and use in later tasks" })
    );

    expect(await screen.findByText("Error: promotion failed")).toBeTruthy();
    expect(evidence.value).toBe(
      "Two neighboring modules use the same dependency group."
    );
  });

  it("keeps manual review available when source task material fails", async () => {
    const user = userEvent.setup();
    renderReview({ taskFails: true });

    expect(
      await screen.findByText(
        "Source task material is unavailable. You can still enter evidence manually."
      )
    ).toBeTruthy();
    expect(screen.getByRole("button", { name: "Retry" })).toBeTruthy();

    const evidence = screen.getByLabelText("What did you verify?") as HTMLTextAreaElement;
    await user.type(evidence, "Reviewed existing Gradle grouping in build.gradle.");
    expect(evidence.value).toBe("Reviewed existing Gradle grouping in build.gradle.");
  });
});
