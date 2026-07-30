import { cleanup, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { MemoryImpactFunnel } from "./MemoryImpactFunnel";

describe("MemoryImpactFunnel", () => {
  afterEach(cleanup);

  it("shows recall, application, and confirmed-helpful stages", () => {
    render(
      <MemoryImpactFunnel
        metrics={{
          recalled_task_count: 4,
          recalled_memory_count: 7,
          applied_task_count: 3,
          helpful_task_count: 2,
          ignored_count: 1,
          unconfirmed_count: 1,
          recall_coverage_rate: 1,
          application_rate: 0.75,
          confirmation_rate: 2 / 3,
        }}
      />
    );

    expect(screen.getByText("4 tasks recalled memory")).toBeTruthy();
    expect(screen.getByText("3 tasks applied memory")).toBeTruthy();
    expect(screen.getByText("2 tasks confirmed helpful")).toBeTruthy();
    expect(screen.getByText("1 awaiting confirmation")).toBeTruthy();
    expect(screen.getByText("1 marked not applicable")).toBeTruthy();
  });

  it("explains empty measurements without claiming time savings", () => {
    render(<MemoryImpactFunnel />);

    expect(screen.getByText("No memory impact recorded yet")).toBeTruthy();
    expect(screen.queryByText(/hours saved/i)).toBeNull();
  });
});
