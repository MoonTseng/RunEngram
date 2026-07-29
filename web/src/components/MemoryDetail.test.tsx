import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "vitest";

import type { ExplorationCapsule } from "../lib/api";
import { MemoryDetail } from "./MemoryDetail";

const capsule: ExplorationCapsule = {
  id: "capsule-new",
  project_id: "project-1",
  source_task_id: "task-1",
  memory_class: "experience",
  trigger: "Gradle multi-module build",
  title: "Compile modules serially",
  summary: "Disable parallel compilation when validating a migration.",
  scope: "Android modules",
  evidence: "Three modules compiled successfully.",
  labels: ["android"],
  fingerprints: ["gradle"],
  producer: "codex",
  status: "active",
  validation: "verified",
  confidence: 0.8,
  use_count: 2,
  helpful_count: 2,
  rejected_count: 0,
  relations: [
    {
      id: "relation-1",
      project_id: "project-1",
      source_capsule_id: "capsule-new",
      type: "supersedes",
      target_kind: "capsule",
      target_ref: "capsule-old",
      note: "Old command was flaky.",
      direction: "outgoing",
      created_at: 1785220000000,
    },
  ],
  created_at: 1785220000000,
  updated_at: 1785220001000,
};

const oldCapsule: ExplorationCapsule = {
  ...capsule,
  id: "capsule-old",
  title: "Old branch convention",
  relations: [],
  updated_at: 1785210001000,
};

describe("MemoryDetail", () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
  });

  it("edits memory with CAS and manages typed relations", async () => {
    const user = userEvent.setup();
    const onUpdated = vi.fn();
    const onRelationsChanged = vi.fn();
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      if (url.endsWith("/capsules/capsule-new") && init?.method === "PATCH") {
        return new Response(
          JSON.stringify({ ...capsule, title: "Compile modules one by one", updated_at: 1785220002000 }),
          { status: 200, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.endsWith("/capsules/capsule-new/relations") && init?.method === "POST") {
        return new Response(
          JSON.stringify({
            id: "relation-2",
            project_id: "project-1",
            source_capsule_id: "capsule-new",
            type: "applies-to",
            target_kind: "scope",
            target_ref: "billing-module",
            note: "",
            direction: "outgoing",
            created_at: 1785220003000,
          }),
          { status: 201, headers: { "Content-Type": "application/json" } }
        );
      }
      if (url.endsWith("/memory-relations/relation-1") && init?.method === "DELETE") {
        return new Response(null, { status: 204 });
      }
      return new Response("not found", { status: 404 });
    });
    vi.stubGlobal("fetch", fetchMock);

    render(
      <MemoryDetail
        capsule={capsule}
        capsules={[capsule, oldCapsule]}
        onUpdated={onUpdated}
        onRelationsChanged={onRelationsChanged}
      />
    );

    expect(screen.getByText("Memory graph")).toBeTruthy();
    expect(screen.getByText(/capsule-old/)).toBeTruthy();

    await user.selectOptions(screen.getByLabelText("Relation type"), "supersedes");
    expect(screen.getByRole("option", { name: "Old branch convention" })).toBeTruthy();

    await user.click(screen.getByRole("button", { name: "Edit memory" }));
    const title = screen.getByLabelText("Memory title");
    await user.clear(title);
    await user.type(title, "Compile modules one by one");
    await user.click(screen.getByRole("button", { name: "Save changes" }));
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/capsules/capsule-new"),
      expect.objectContaining({
        method: "PATCH",
        body: expect.stringContaining('"expected_updated_at":1785220001000'),
      })
    );
    expect(onUpdated).toHaveBeenCalled();

    await user.selectOptions(screen.getByLabelText("Relation type"), "applies-to");
    await user.selectOptions(screen.getByLabelText("Target type"), "scope");
    await user.type(screen.getByLabelText("Target reference"), "billing-module");
    await user.click(screen.getByRole("button", { name: "Add relation" }));
    expect(onRelationsChanged).toHaveBeenCalled();

    await user.click(screen.getByRole("button", { name: "Remove relation" }));
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("/memory-relations/relation-1"),
      expect.objectContaining({ method: "DELETE" })
    );
  });
});
