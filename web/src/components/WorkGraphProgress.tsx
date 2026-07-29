import {
  CheckCircle2,
  Circle,
  CircleDashed,
  FileCheck2,
  Gauge,
  LoaderCircle,
  PauseCircle,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
} from "lucide-react";
import { useState } from "react";
import type { RunNode, RunNodeStatus, RunWorkGraph } from "../lib/api";
import type { Locale } from "../lib/i18n";

const stageLabels: Record<string, Record<Locale, string>> = {
  "requirement-analysis": { "zh-CN": "需求分析", "en-US": "Requirement" },
  "technical-design": { "zh-CN": "技术方案", "en-US": "Design" },
  "task-planning": { "zh-CN": "任务规划", "en-US": "Plan" },
  implementation: { "zh-CN": "代码实现", "en-US": "Implement" },
  refactor: { "zh-CN": "重构优化", "en-US": "Refactor" },
  verification: { "zh-CN": "测试验证", "en-US": "Verify" },
  "code-review": { "zh-CN": "独立复核", "en-US": "Review" },
  "final-gate": { "zh-CN": "结果确认", "en-US": "Final gate" },
};

const statusLabels: Record<RunNodeStatus, Record<Locale, string>> = {
  pending: { "zh-CN": "等待前序", "en-US": "Pending" },
  ready: { "zh-CN": "可开始", "en-US": "Ready" },
  running: { "zh-CN": "执行中", "en-US": "Running" },
  waiting: { "zh-CN": "等你决策", "en-US": "Needs input" },
  completed: { "zh-CN": "已完成", "en-US": "Completed" },
  failed: { "zh-CN": "需修复", "en-US": "Failed" },
  skipped: { "zh-CN": "已跳过", "en-US": "Skipped" },
};

export function WorkGraphProgress({
  graph,
  locale,
  recalledCount,
  artifacts,
  resolving,
  error,
  onResolve,
}: {
  graph: RunWorkGraph;
  locale: Locale;
  recalledCount: number;
  artifacts: Array<{ id: string; title: string; url?: string }>;
  resolving: boolean;
  error?: string;
  onResolve: (interruptId: string, response: string, reject: boolean) => void;
}) {
  const zh = locale === "zh-CN";
  const [response, setResponse] = useState("");
  const nodes = graph.nodes ?? [];
  const interrupts = graph.interrupts ?? [];
  const current = nodes.find((node) =>
    ["waiting", "running", "ready", "failed"].includes(node.status)
  );
  const interrupt = interrupts[0];

  return (
    <section
      className="mt-6 overflow-hidden rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)]"
      aria-label={zh ? "可恢复工作图" : "Resumable Work Graph"}
    >
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-[var(--tl-outline)] p-4">
        <div>
          <div className="flex items-center gap-2">
            <Gauge size={18} className="text-[var(--tl-primary)]" />
            <h3 className="font-bold">
              {zh ? "可恢复工作图" : "Resumable Work Graph"}
            </h3>
          </div>
          <p className="mt-1 text-sm text-[var(--tl-ink-muted)]">
            {zh
              ? "每个阶段留下结果、证据和恢复点。"
              : "Every stage leaves a result, evidence, and recovery point."}
          </p>
          <p className="mt-1 font-mono text-xs text-[var(--tl-ink-faint)]">
            {graph.template} · v{graph.version}
          </p>
        </div>
        <strong className="text-2xl">{graph.progress_percent}%</strong>
      </div>

      <div className="h-1.5 bg-[var(--tl-outline)]">
        <div
          className="h-full bg-[var(--tl-primary)] transition-[width]"
          style={{ width: `${graph.progress_percent}%` }}
        />
      </div>

      <div className="grid gap-2 p-4 sm:grid-cols-2 xl:grid-cols-4">
        {nodes.map((node) => (
          <StageCard
            key={node.id}
            node={node}
            locale={locale}
            artifacts={artifacts}
          />
        ))}
      </div>

      {current && (
        <div className="mx-4 mb-4 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-4">
          <p className="text-xs font-bold uppercase tracking-[0.14em] text-[var(--tl-primary)]">
            {zh ? "当前动作" : "Current action"}
          </p>
          <div className="mt-2 flex flex-wrap items-center justify-between gap-2">
            <strong>{stageLabels[current.key]?.[locale] ?? current.title}</strong>
            <span className="status-chip">{statusLabels[current.status][locale]}</span>
          </div>
          <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
            {current.next_step ||
              (zh ? "Agent 将从此阶段继续。" : "Agent resumes from this stage.")}
          </p>
        </div>
      )}

      {interrupt && (
        <div className="mx-4 mb-4 rounded-lg border border-[var(--tl-amber)] bg-[var(--tl-amber-soft)] p-4">
          <div className="flex items-center gap-2 font-bold">
            <PauseCircle size={18} />
            {zh ? "需要你的决策" : "Your decision is needed"}
          </div>
          <p className="mt-2 text-[15px] leading-6">{interrupt.prompt}</p>
          {(interrupt.options ?? []).length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-2">
              {(interrupt.options ?? []).map((option, index) => (
                <button
                  key={option}
                  type="button"
                  className={index === 0 ? "primary-button" : "secondary-button"}
                  disabled={resolving}
                  onClick={() => onResolve(interrupt.id, option, false)}
                >
                  {option}
                </button>
              ))}
              {interrupt.kind === "approval" && (
                <button
                  type="button"
                  className="secondary-button"
                  disabled={resolving}
                  onClick={() =>
                    onResolve(
                      interrupt.id,
                      zh ? "退回修改" : "Request changes",
                      true
                    )
                  }
                >
                  {zh ? "退回修改" : "Request changes"}
                </button>
              )}
            </div>
          ) : (
            <div className="mt-3 flex gap-2">
              <input
                className="min-w-0 flex-1 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 py-2 text-sm"
                value={response}
                onChange={(event) => setResponse(event.target.value)}
                placeholder={zh ? "输入决定或补充信息" : "Enter decision or context"}
              />
              <button
                type="button"
                className="primary-button"
                disabled={resolving || !response.trim()}
                onClick={() => onResolve(interrupt.id, response.trim(), false)}
              >
                {zh ? "继续" : "Continue"}
              </button>
            </div>
          )}
          {error && <p className="mt-2 text-sm text-[var(--tl-rust)]">{error}</p>}
        </div>
      )}

      <div className="grid grid-cols-2 gap-2 border-t border-[var(--tl-outline)] p-4 lg:grid-cols-5">
        <Receipt
          icon={<CheckCircle2 size={16} />}
          value={`${graph.completed_node_count}/${nodes.length}`}
          label={zh ? "阶段完成" : "stages done"}
        />
        <Receipt
          icon={<ShieldCheck size={16} />}
          value={String(graph.verified_node_count)}
          label={zh ? "阶段有证据" : "verified stages"}
        />
        <Receipt
          icon={<FileCheck2 size={16} />}
          value={String(graph.artifact_count)}
          label={zh ? "交付物已关联" : "linked artifacts"}
        />
        <Receipt
          icon={<Sparkles size={16} />}
          value={String(recalledCount)}
          label={zh ? "条经验已召回" : "memories recalled"}
        />
        <Receipt
          icon={<PauseCircle size={16} />}
          value={String(graph.open_interrupt_count)}
          label={zh ? "个决策待处理" : "open decisions"}
        />
      </div>
    </section>
  );
}

function StageCard({
  node,
  locale,
  artifacts,
}: {
  node: RunNode;
  locale: Locale;
  artifacts: Array<{ id: string; title: string; url?: string }>;
}) {
  const Icon =
    node.status === "completed" || node.status === "skipped"
      ? CheckCircle2
      : node.status === "running"
        ? LoaderCircle
        : node.status === "waiting"
          ? PauseCircle
          : node.status === "failed"
            ? TriangleAlert
            : node.status === "ready"
              ? Circle
              : CircleDashed;
  const active = ["ready", "running", "waiting", "failed"].includes(node.status);

  return (
    <article
      className={`rounded-lg border p-3 ${
        active
          ? "border-[var(--tl-primary)] bg-[var(--tl-primary-soft)]"
          : "border-[var(--tl-outline)] bg-[var(--tl-surface)]"
      }`}
    >
      <div className="flex items-start justify-between gap-2">
        <span className="flex items-center gap-2 text-sm font-bold">
          <Icon size={16} className={node.status === "running" ? "animate-spin" : ""} />
          {stageLabels[node.key]?.[locale] ?? node.title}
        </span>
        <small className="text-[var(--tl-ink-faint)]">{statusLabels[node.status][locale]}</small>
      </div>
      {node.summary && (
        <p className="mt-2 line-clamp-2 text-xs leading-5 text-[var(--tl-ink-muted)]">
          {node.summary}
        </p>
      )}
      {node.evidence && (
        <p className="mt-2 rounded-md bg-[var(--tl-bg-quiet)] px-2 py-1.5 text-xs leading-5 text-[var(--tl-moss)]">
          <strong>{locale === "zh-CN" ? "通过依据：" : "Evidence: "}</strong>
          {node.evidence}
        </p>
      )}
      {(node.artifact_ids ?? []).length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {(node.artifact_ids ?? []).map((artifactId) => {
            const artifact = artifacts.find((item) => item.id === artifactId);
            if (!artifact) {
              return (
                <span key={artifactId} className="status-chip">
                  {artifactId.slice(0, 8)}
                </span>
              );
            }
            return artifact.url ? (
              <a
                key={artifactId}
                className="status-chip hover:underline"
                href={artifact.url}
                target="_blank"
                rel="noreferrer"
              >
                {artifact.title}
              </a>
            ) : (
              <span key={artifactId} className="status-chip">
                {artifact.title}
              </span>
            );
          })}
        </div>
      )}
    </article>
  );
}

function Receipt({
  icon,
  value,
  label,
}: {
  icon: React.ReactNode;
  value: string;
  label: string;
}) {
  return (
    <div className="rounded-lg bg-[var(--tl-surface)] p-3">
      <div className="flex items-center gap-2 text-[var(--tl-primary)]">
        {icon}
        <strong className="text-lg">{value}</strong>
      </div>
      <p className="mt-1 text-xs text-[var(--tl-ink-muted)]">{label}</p>
    </div>
  );
}
