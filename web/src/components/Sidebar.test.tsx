import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import type { Project } from "../lib/api";
import { Sidebar } from "./Sidebar";

const queryMocks = vi.hoisted(() => ({
  useProjects: vi.fn(),
  useCreateProject: vi.fn(),
}));

vi.mock("../hooks/queries", () => queryMocks);

const project: Project = {
  id: "project-1",
  name: "taskline",
  description: "",
  created_at: 1780051741142,
  updated_at: 1780051741142,
};

function renderSidebar() {
  queryMocks.useProjects.mockReturnValue({
    data: [project],
    isLoading: false,
    error: null,
  });
  queryMocks.useCreateProject.mockReturnValue({
    mutateAsync: vi.fn(),
    isPending: false,
    error: null,
  });

  render(<Sidebar selectedId={project.id} onSelect={vi.fn()} />);
}

describe("Sidebar", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    cleanup();
  });

  it("does not render the old bottom usage hint", () => {
    renderSidebar();

    expect(screen.queryByText(/polling every 10s/i)).toBeNull();
    expect(screen.queryByText(/drag cards between columns/i)).toBeNull();
  });
});
