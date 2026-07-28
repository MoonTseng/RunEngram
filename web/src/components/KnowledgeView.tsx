import { useState, type ReactNode } from "react";
import { useQuery } from "@tanstack/react-query";
import { BookOpen, CheckCircle2, Database, Search, Users } from "lucide-react";

import type { CapsuleStatus, Project } from "../lib/api";
import { getLearningMetrics, listCapsules } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function KnowledgeView({ project }: { project: Project }) {
  const { locale, t } = useI18n();
  const [query, setQuery] = useState("");
  const [status, setStatus] = useState<CapsuleStatus | "">("active");
  const capsules = useQuery({
    queryKey: ["capsules", project.id, query, status],
    queryFn: () => listCapsules(project.id, query, status),
  });
  const metrics = useQuery({
    queryKey: ["learning-metrics", project.id],
    queryFn: () => getLearningMetrics(project.id),
  });
  const chinese = locale === "zh-CN";

  return (
    <div className="h-full overflow-y-auto p-6 max-sm:p-3">
      <div className="mx-auto flex max-w-6xl flex-col gap-5">
        <section className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-5 shadow-[var(--tl-shadow-paper)]">
          <p className="text-sm font-semibold uppercase tracking-[0.16em] text-[var(--tl-primary)]">
            Engineering Memory
          </p>
          <h1 className="mt-2 text-2xl font-bold text-[var(--tl-ink)]">
            {t("knowledge.title")}
          </h1>
          <p className="mt-2 max-w-3xl text-[15px] leading-7 text-[var(--tl-ink-muted)]">
            {t("knowledge.subtitle")}
          </p>
        </section>

        <section className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          <Metric icon={<Database size={18} />} label={t("knowledge.active")} value={metrics.data?.active_capsule_count ?? 0} />
          <Metric icon={<BookOpen size={18} />} label={t("knowledge.snapshots")} value={metrics.data?.snapshot_task_count ?? 0} />
          <Metric icon={<Users size={18} />} label={t("knowledge.reusedTasks")} value={metrics.data?.reused_task_count ?? 0} />
          <Metric
            icon={<CheckCircle2 size={18} />}
            label={t("knowledge.helpfulRate")}
            value={`${Math.round((metrics.data?.helpful_rate ?? 0) * 100)}%`}
          />
        </section>

        <section className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-5">
          <div className="flex flex-wrap gap-3">
            <label className="relative min-w-64 flex-1">
              <Search className="absolute left-3 top-3 text-[var(--tl-ink-muted)]" size={18} />
              <input
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t("knowledge.search")}
                className="h-11 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] pl-10 pr-3 text-[15px] text-[var(--tl-ink)] outline-none focus:border-[var(--tl-primary)]"
              />
            </label>
            <select
              value={status}
              onChange={(event) => setStatus(event.target.value as CapsuleStatus | "")}
              className="h-11 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] px-3 text-[15px]"
            >
              <option value="">{t("knowledge.all")}</option>
              <option value="active">{t("knowledge.active")}</option>
              <option value="stale">{t("knowledge.stale")}</option>
              <option value="archived">{t("knowledge.archived")}</option>
            </select>
          </div>

          {capsules.isLoading ? (
            <p className="py-12 text-center text-[15px] text-[var(--tl-ink-muted)]">{t("knowledge.loading")}</p>
          ) : capsules.isError ? (
            <p className="py-12 text-center text-[15px] text-red-700">{String(capsules.error)}</p>
          ) : capsules.data?.length === 0 ? (
            <div className="py-12 text-center">
              <p className="text-lg font-semibold text-[var(--tl-ink)]">{t("knowledge.empty")}</p>
              <p className="mx-auto mt-2 max-w-2xl text-[15px] leading-7 text-[var(--tl-ink-muted)]">
                {t("knowledge.emptyHint")}
              </p>
            </div>
          ) : (
            <div className="mt-5 grid gap-4 lg:grid-cols-2">
              {capsules.data?.map((capsule) => (
                <article key={capsule.id} className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] p-5">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <h2 className="text-lg font-bold leading-6 text-[var(--tl-ink)]">{capsule.title}</h2>
                      <p className="mt-1 text-sm text-[var(--tl-ink-muted)]">
                        {capsule.producer || "codex"} · {capsule.use_count} {t("knowledge.uses")}
                      </p>
                    </div>
                    <span className="rounded-full bg-[var(--tl-bg-quiet)] px-2.5 py-1 text-xs font-semibold text-[var(--tl-ink-muted)]">
                      {capsule.status}
                    </span>
                  </div>
                  <p className="mt-4 text-[15px] leading-7 text-[var(--tl-ink)]">{capsule.summary}</p>
                  {capsule.scope && (
                    <p className="mt-3 text-sm leading-6 text-[var(--tl-ink-muted)]">
                      <strong>{t("knowledge.scope")}:</strong> {capsule.scope}
                    </p>
                  )}
                  <div className="mt-3 flex flex-wrap gap-1.5">
                    {[...capsule.labels, ...capsule.fingerprints].map((value) => (
                      <span key={value} className="rounded-md border border-[var(--tl-outline)] px-2 py-1 text-xs text-[var(--tl-ink-muted)]">
                        {value}
                      </span>
                    ))}
                  </div>
                  <details className="mt-4 border-t border-[var(--tl-outline)] pt-3">
                    <summary className="cursor-pointer text-sm font-semibold text-[var(--tl-primary)]">
                      {t("knowledge.evidence")}
                    </summary>
                    <pre className="mt-3 max-h-64 overflow-auto whitespace-pre-wrap rounded-lg bg-[var(--tl-bg-quiet)] p-3 text-sm leading-6 text-[var(--tl-ink)]">
                      {capsule.evidence}
                    </pre>
                  </details>
                  <p className="mt-3 text-xs text-[var(--tl-ink-muted)]">
                    {chinese ? "来源任务" : "Source task"}: {capsule.source_task_id || "—"}
                  </p>
                </article>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function Metric({ icon, label, value }: { icon: ReactNode; label: string; value: string | number }) {
  return (
    <div className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-4">
      <div className="flex items-center gap-2 text-sm font-semibold text-[var(--tl-ink-muted)]">{icon}{label}</div>
      <p className="mt-2 text-2xl font-bold text-[var(--tl-ink)]">{value}</p>
    </div>
  );
}
