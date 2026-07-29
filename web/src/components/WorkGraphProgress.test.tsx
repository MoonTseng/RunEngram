import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import type { RunWorkGraph } from "../lib/api";
import { WorkGraphProgress } from "./WorkGraphProgress";

const graph: RunWorkGraph = {
  run_id: "run-1",
  template: "cs-one-flow",
  version: 1,
  completed_node_count: 1,
  verified_node_count: 1,
  artifact_count: 2,
  open_interrupt_count: 1,
  progress_percent: 12,
  nodes: [
    {
      id: "node-1",
      run_id: "run-1",
      key: "requirement-analysis",
      title: "需求分析",
      capability: "prd-analysis",
      kind: "agent-loop",
      position: 0,
      depends_on: [],
      status: "completed",
      attempt: 1,
      summary: "需求边界已确认",
      next_step: "",
      artifact_ids: ["doc-1", "doc-2"],
      evidence: "产品确认",
      input_fingerprint: "prd-v1",
      started_at: 1,
      completed_at: 2,
      updated_at: 2,
    },
    {
      id: "node-2",
      run_id: "run-1",
      key: "technical-design",
      title: "技术方案",
      capability: "technical-design",
      kind: "agent-loop",
      position: 1,
      depends_on: ["requirement-analysis"],
      status: "waiting",
      attempt: 1,
      summary: "",
      next_step: "确认迁移边界",
      artifact_ids: [],
      evidence: "",
      input_fingerprint: "",
      started_at: 3,
      completed_at: 0,
      updated_at: 3,
    },
  ],
  interrupts: [
    {
      id: "interrupt-1",
      run_id: "run-1",
      node_key: "technical-design",
      kind: "approval",
      prompt: "确认迁移边界？",
      options: ["确认", "调整"],
      status: "pending",
      response: "",
      requested_by: "codex",
      responded_by: "",
      created_at: 3,
      resolved_at: 0,
    },
  ],
};

describe("WorkGraphProgress", () => {
  it("shows stage receipts and resolves a human interrupt", () => {
    const onResolve = vi.fn();
    render(
      <WorkGraphProgress
        graph={graph}
        locale="zh-CN"
        recalledCount={3}
        artifacts={[
          { id: "doc-1", title: "需求说明", url: "/api/v1/docs/doc-1/content" },
          { id: "doc-2", title: "验收清单", url: "/api/v1/docs/doc-2/content" },
        ]}
        resolving={false}
        onResolve={onResolve}
      />
    );

    expect(screen.getByText("One-flow 研发闭环")).toBeTruthy();
    expect(screen.getByText("需求边界已确认")).toBeTruthy();
    expect(screen.getByText("产品确认")).toBeTruthy();
    expect(screen.getByRole("link", { name: "需求说明" })).toBeTruthy();
    expect(screen.getByText("确认迁移边界？")).toBeTruthy();
    expect(screen.getByText("条经验已召回")).toBeTruthy();

    fireEvent.click(screen.getByRole("button", { name: "确认" }));
    expect(onResolve).toHaveBeenCalledWith("interrupt-1", "确认", false);

    fireEvent.click(screen.getByRole("button", { name: "退回修改" }));
    expect(onResolve).toHaveBeenCalledWith("interrupt-1", "退回修改", true);
  });

  it("renders legacy null collection fields without blanking the page", () => {
    const legacyGraph = {
      ...graph,
      nodes: [{ ...graph.nodes[0], artifact_ids: null }],
      interrupts: [{ ...graph.interrupts[0], options: null }],
    } as unknown as RunWorkGraph;

    const legacyView = render(
      <WorkGraphProgress
        graph={legacyGraph}
        locale="zh-CN"
        recalledCount={0}
        artifacts={[]}
        resolving={false}
        onResolve={vi.fn()}
      />
    );

    expect(legacyView.container.textContent).toContain("One-flow 研发闭环");
    expect(
      legacyView.container.querySelector(
        'input[placeholder="输入决定或补充信息"]'
      )
    ).toBeTruthy();
  });
});
