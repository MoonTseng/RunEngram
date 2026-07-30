import {
  ArrowRight,
  Bot,
  CheckCircle2,
  Clipboard,
  History,
  ShieldCheck,
  Sparkles,
  TriangleAlert,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useResolveRunInterrupt, useTaskResumeContext } from "../hooks/queries";
import type { LearningMetrics, Project, Task } from "../lib/api";
import {
  getLearningMetrics,
  listTaskMemoryImpacts,
  updateMemoryImpact,
} from "../lib/api";
import { selectActionFocus, taskPrompt } from "../lib/actionConsole";
import { useI18n } from "../lib/i18n";
import { useQuery } from "@tanstack/react-query";
import { TaskMemoryImpactPanel } from "./TaskMemoryImpactPanel";
import { WorkGraphProgress } from "./WorkGraphProgress";

export function ActionConsole({
  project,
  tasks,
  loading,
  error,
  onNavigate,
}: {
  project: Project;
  tasks: Task[];
  loading: boolean;
  error: unknown;
  onNavigate: (view: "kanban" | "knowledge") => void;
}) {
  const { locale, stateLabel, t, typeLabel } = useI18n();
  const [copied, setCopied] = useState(false);
  const focus = useMemo(() => selectActionFocus(tasks), [tasks]);
  const focusTaskID = focus.kind === "empty" ? null : focus.task.id;
  const resume = useTaskResumeContext(
    focus.kind === "active" || focus.kind === "outcome" ? focus.task.id : null
  );
  const context = resume.data?.snapshot;
  const latestRun = resume.data?.latest_run;
  const workGraph = resume.data?.work_graph;
  const projectRules = context?.project_rules ?? [];
  const recalled = [...projectRules, ...(context?.suggested_capsules ?? [])];
  const resolveInterrupt = useResolveRunInterrupt(
    focus.kind === "active" ? focus.task.id : null
  );
  const metrics = useQuery<LearningMetrics>({
    queryKey: ["learning-metrics", project.id],
    queryFn: () => getLearningMetrics(project.id),
  });
  const impacts = useQuery({
    queryKey: ["memory-impacts", "task", focusTaskID],
    queryFn: () => listTaskMemoryImpacts(focusTaskID!),
    enabled: Boolean(focusTaskID),
  });

  if (loading) {
    return <ConsoleState title={t("action.loading")} />;
  }
  if (error) {
    return <ConsoleState title={t("action.loadFailed")} detail={String(error)} />;
  }
  if (focus.kind === "empty") {
    return (
      <ConsoleState
        title={t("action.emptyTitle")}
        detail={t("action.emptyHint")}
        action={
          <button className="primary-button" onClick={() => onNavigate("kanban")}>
            {t("action.openBoard")} <ArrowRight size={16} />
          </button>
        }
      />
    );
  }

  const task = focus.task;
  const captured = (task.learning_notes ?? []).filter((note) => note.status === "pending");
  const prompt = taskPrompt(project.name, focus);
  const status =
    focus.kind === "active"
      ? t("action.active")
      : focus.kind === "ready"
        ? t("action.ready")
        : focus.kind === "blocked"
          ? t("action.blocked")
          : t("action.outcome");
  const statusIcon =
    focus.kind === "active" ? (
      <Bot size={18} />
    ) : focus.kind === "blocked" ? (
      <TriangleAlert size={18} />
    ) : (
      <CheckCircle2 size={18} />
    );

  const copyPrompt = async () => {
    await navigator.clipboard?.writeText(prompt);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1_600);
  };

  return (
    <div className="h-full overflow-y-auto p-5 lg:p-7">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-5">
        <header>
          <p className="eyebrow">{t("action.eyebrow")}</p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">{t("action.title")}</h1>
          <p className="mt-2 text-base text-[var(--tl-ink-muted)]">{t("action.subtitle")}</p>
        </header>

        <div className="grid gap-5 xl:grid-cols-[minmax(0,1.45fr)_minmax(320px,0.8fr)]">
          <section className="panel p-5 lg:p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <span className="status-chip">{statusIcon}{status}</span>
                <h2 className="mt-4 text-2xl font-bold leading-tight">{task.title}</h2>
                <p className="mt-2 text-sm text-[var(--tl-ink-muted)]">
                  {typeLabel(task.type)} · P{task.priority} · {stateLabel(task.state)}
                  {task.owner ? ` · ${task.owner}` : ""}
                </p>
              </div>
              <code className="rounded-md bg-[var(--tl-bg-quiet)] px-2.5 py-1.5 text-xs text-[var(--tl-ink-faint)]">
                {task.id.slice(0, 12)}
              </code>
            </div>

            {task.description && (
              <p className="mt-5 line-clamp-4 text-[15px] leading-7 text-[var(--tl-ink-muted)]">
                {task.description}
              </p>
            )}

            {workGraph && (workGraph.nodes ?? []).length > 0 && (
              <WorkGraphProgress
                graph={workGraph}
                locale={locale}
                recalledCount={recalled.length}
                artifacts={[
                  ...(task.docs ?? []).map((doc) => ({
                    id: doc.id,
                    title: doc.title,
                    url: doc.url,
                  })),
                  ...(task.links ?? []).map((link) => ({
                    id: link.id,
                    title: link.label || link.url,
                    url: link.url,
                  })),
                  ...(task.images ?? []).map((image) => ({
                    id: image.id,
                    title: image.filename,
                    url: image.url,
                  })),
                ]}
                resolving={resolveInterrupt.isPending}
                error={
                  resolveInterrupt.isError
                    ? String(resolveInterrupt.error)
                    : undefined
                }
                onResolve={(interruptId, response, reject) =>
                  resolveInterrupt.mutate({ interruptId, response, reject })
                }
              />
            )}

            {focus.kind === "blocked" && (
              <div className="mt-5 rounded-lg border border-[var(--tl-rust)] bg-[var(--tl-rust-soft)] p-4">
                <p className="font-semibold">{t("action.blockedBy")}</p>
                <ul className="mt-2 space-y-1 text-sm">
                  {focus.blockers.map((blocker) => (
                    <li key={blocker.id}>• {blocker.title}</li>
                  ))}
                </ul>
              </div>
            )}

            <div className="mt-6 rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
              <p className="text-xs font-bold uppercase tracking-[0.15em] text-[var(--tl-primary)]">
                {t("action.next")}
              </p>
              <p className="mt-2 text-[15px] leading-7">
                {focus.kind === "active"
                  ? latestRun?.next_step || t("action.continueHint")
                  : focus.kind === "outcome"
                    ? t("action.reviewHint")
                    : focus.kind === "blocked"
                      ? t("action.blockedHint")
                      : t("action.startHint")}
              </p>
              {focus.kind === "active" && latestRun && (
                <div className="mt-3 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <strong className="text-sm">{t("action.latestCheckpoint")}</strong>
                    <span className="status-chip">
                      {latestRun.agent_tool} · {latestRun.status}
                    </span>
                  </div>
                  <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
                    {latestRun.summary || t("action.continueHint")}
                  </p>
                  {latestRun.next_step && (
                    <p className="mt-2 text-sm">
                      <strong>{t("action.nextStep")}:</strong> {latestRun.next_step}
                    </p>
                  )}
                </div>
              )}
              <code className="mt-3 block overflow-x-auto rounded-lg bg-[var(--tl-surface)] p-3 text-sm leading-6 text-[var(--tl-ink-muted)]">
                {prompt}
              </code>
              <div className="mt-4 flex flex-wrap gap-2">
                <button type="button" className="primary-button" onClick={copyPrompt}>
                  <Clipboard size={16} /> {copied ? t("action.copied") : t("action.copyPrompt")}
                </button>
                <button type="button" className="secondary-button" onClick={() => onNavigate("kanban")}>
                  {t("action.openBoard")} <ArrowRight size={16} />
                </button>
              </div>
            </div>
          </section>

          <section className="panel p-5">
            <div className="flex items-center gap-2">
              <Sparkles size={18} className="text-[var(--tl-primary)]" />
              <h2 className="text-lg font-bold">{t("action.recalled")}</h2>
            </div>
            <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
              {t("action.recalledHint")}
            </p>

            {focus.kind !== "active" ? (
              <MemoryEmpty text={t("action.contextAfterClaim")} />
            ) : resume.isLoading ? (
              <MemoryEmpty text={t("action.contextLoading")} />
            ) : resume.isError ? (
              <MemoryEmpty text={t("action.contextUnavailable")} />
            ) : recalled.length === 0 ? (
              <MemoryEmpty text={t("action.noRecall")} />
            ) : (
              <div className="mt-4 space-y-3">
                {recalled.slice(0, 3).map((capsule) => {
                  const feedback = capsule.helpful_count + capsule.rejected_count;
                  const helpful = feedback > 0
                    ? Math.round((capsule.helpful_count / feedback) * 100)
                    : null;
                  return (
                    <article key={capsule.id} className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
                      <div className="flex items-start justify-between gap-2">
                        <div>
                          <span className="status-chip">
                            {capsule.memory_class === "project-rule"
                              ? t("knowledge.projectRule")
                              : t("knowledge.scopedExperience")}
                          </span>
                          <h3 className="mt-2 font-bold leading-6">{capsule.title}</h3>
                        </div>
                        <ShieldCheck size={16} className="mt-1 shrink-0 text-[var(--tl-moss)]" />
                      </div>
                      <p className="mt-2 line-clamp-3 text-sm leading-6 text-[var(--tl-ink-muted)]">
                        {capsule.summary}
                      </p>
                      <p className="mt-3 text-xs text-[var(--tl-ink-faint)]">
                        {capsule.scope || t("action.projectScope")} · {capsule.use_count} {t("knowledge.uses")}
                        {helpful === null ? "" : ` · ${helpful}% ${t("action.helpful")}`}
                        {` · ${Math.round(capsule.confidence * 100)}% ${t("knowledge.confidence")}`}
                      </p>
                    </article>
                  );
                })}
              </div>
            )}
            <div className="mt-5 border-t border-[var(--tl-outline)] pt-5">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <h3 className="font-bold">{t("action.impactTitle")}</h3>
                <span className="status-chip">
                  {(impacts.data ?? []).filter((impact) =>
                    ["applied", "helpful", "rejected", "stale"].includes(impact.state)
                  ).length}
                  /{impacts.data?.length ?? 0}
                </span>
              </div>
              <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
                {t("action.impactHint")}
              </p>
              {impacts.isLoading ? (
                <MemoryEmpty text={t("knowledge.impactHistoryLoading")} />
              ) : impacts.isError ? (
                <MemoryEmpty text={String(impacts.error)} />
              ) : (
                <TaskMemoryImpactPanel
                  impacts={impacts.data ?? []}
                  capsules={recalled}
                  onUpdate={async (impactID, input) => {
                    await updateMemoryImpact(impactID, input);
                    await Promise.all([impacts.refetch(), metrics.refetch()]);
                  }}
                />
              )}
            </div>
            <button type="button" className="mt-4 text-sm font-semibold text-[var(--tl-primary)] hover:underline" onClick={() => onNavigate("knowledge")}>
              {t("action.openMemory")} →
            </button>

            <div className="mt-5 border-t border-[var(--tl-outline)] pt-5">
              <div className="flex items-center justify-between gap-3">
                <h3 className="font-bold">{t("action.learningReceipt")}</h3>
                <span className="status-chip">{captured.length}</span>
              </div>
              <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
                {t("action.learningReceiptHint")}
              </p>
              {captured.length === 0 ? (
                <MemoryEmpty text={t("action.noLearningCaptured")} />
              ) : (
                <div className="mt-3 space-y-2">
                  {captured.slice(0, 3).map((note) => (
                    <article
                      key={note.id}
                      className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-3"
                    >
                      <p className="text-sm font-bold leading-6">{note.guidance}</p>
                      <p className="mt-1 text-xs leading-5 text-[var(--tl-ink-muted)]">
                        {note.trigger}
                        {note.scope ? ` · ${note.scope}` : ""}
                      </p>
                    </article>
                  ))}
                </div>
              )}
              <p className="mt-3 text-xs leading-5 text-[var(--tl-ink-faint)]">
                {t("action.notCapturedPolicy")}
              </p>
            </div>
          </section>
        </div>

        <section className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <OverviewMetric icon={<History size={17} />} label={t("action.running")} value={tasks.filter((item) => item.owner && item.state !== "done").length} />
          <OverviewMetric icon={<CheckCircle2 size={17} />} label={t("action.completed")} value={tasks.filter((item) => item.state === "done").length} />
          <OverviewMetric icon={<Sparkles size={17} />} label={t("action.verifiedMemory")} value={metrics.data?.active_capsule_count ?? 0} />
          <OverviewMetric icon={<TriangleAlert size={17} />} label={t("action.pendingReview")} value={metrics.data?.pending_note_count ?? 0} />
        </section>
      </div>
    </div>
  );
}

function ConsoleState({
  title,
  detail,
  action,
}: {
  title: string;
  detail?: string;
  action?: React.ReactNode;
}) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <div className="panel max-w-xl p-8 text-center">
        <h1 className="text-2xl font-bold">{title}</h1>
        {detail && <p className="mt-3 leading-7 text-[var(--tl-ink-muted)]">{detail}</p>}
        {action && <div className="mt-5 flex justify-center">{action}</div>}
      </div>
    </div>
  );
}

function MemoryEmpty({ text }: { text: string }) {
  return (
    <div className="mt-4 rounded-lg border border-dashed border-[var(--tl-outline)] p-5 text-sm leading-6 text-[var(--tl-ink-muted)]">
      {text}
    </div>
  );
}

function OverviewMetric({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
}) {
  return (
    <div className="panel flex items-center justify-between p-4">
      <div className="flex items-center gap-2 text-sm text-[var(--tl-ink-muted)]">{icon}{label}</div>
      <strong className="text-xl">{value}</strong>
    </div>
  );
}
