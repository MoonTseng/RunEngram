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
import { useTaskContext } from "../hooks/queries";
import type { LearningMetrics, Project, Task } from "../lib/api";
import { getLearningMetrics } from "../lib/api";
import { selectActionFocus, taskPrompt } from "../lib/actionConsole";
import { useI18n } from "../lib/i18n";
import { useQuery } from "@tanstack/react-query";

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
  const { stateLabel, t, typeLabel } = useI18n();
  const [copied, setCopied] = useState(false);
  const focus = useMemo(() => selectActionFocus(tasks), [tasks]);
  const context = useTaskContext(focus.kind === "active" ? focus.task.id : null);
  const recalled = context.data?.suggested_capsules ?? [];
  const metrics = useQuery<LearningMetrics>({
    queryKey: ["learning-metrics", project.id],
    queryFn: () => getLearningMetrics(project.id),
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
                  ? t("action.continueHint")
                  : focus.kind === "outcome"
                    ? t("action.reviewHint")
                    : focus.kind === "blocked"
                      ? t("action.blockedHint")
                      : t("action.startHint")}
              </p>
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
            ) : context.isLoading ? (
              <MemoryEmpty text={t("action.contextLoading")} />
            ) : context.isError ? (
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
                        <h3 className="font-bold leading-6">{capsule.title}</h3>
                        <ShieldCheck size={16} className="shrink-0 text-[var(--tl-moss)]" />
                      </div>
                      <p className="mt-2 line-clamp-3 text-sm leading-6 text-[var(--tl-ink-muted)]">
                        {capsule.summary}
                      </p>
                      <p className="mt-3 text-xs text-[var(--tl-ink-faint)]">
                        {capsule.scope || t("action.projectScope")} · {capsule.use_count} {t("knowledge.uses")}
                        {helpful === null ? "" : ` · ${helpful}% ${t("action.helpful")}`}
                      </p>
                    </article>
                  );
                })}
              </div>
            )}
            <button type="button" className="mt-4 text-sm font-semibold text-[var(--tl-primary)] hover:underline" onClick={() => onNavigate("knowledge")}>
              {t("action.openMemory")} →
            </button>
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
