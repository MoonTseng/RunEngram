import { CheckCircle2, Clock3, FileCheck2, UserRound } from "lucide-react";

import type { MemoryImpact, MemoryImpactState } from "../lib/api";
import { useI18n } from "../lib/i18n";

const stateKeys: Record<
  MemoryImpactState,
  | "knowledge.impactStateRecalled"
  | "knowledge.impactStateApplied"
  | "knowledge.impactStateIgnored"
  | "knowledge.impactStateHelpful"
  | "knowledge.impactStateRejected"
  | "knowledge.impactStateStale"
  | "knowledge.impactStateUnconfirmed"
> = {
  recalled: "knowledge.impactStateRecalled",
  applied: "knowledge.impactStateApplied",
  ignored: "knowledge.impactStateIgnored",
  helpful: "knowledge.impactStateHelpful",
  rejected: "knowledge.impactStateRejected",
  stale: "knowledge.impactStateStale",
  unconfirmed: "knowledge.impactStateUnconfirmed",
};

export function MemoryImpactHistory({
  impacts,
  loading = false,
  error = "",
}: {
  impacts: MemoryImpact[];
  loading?: boolean;
  error?: string;
}) {
  const { locale, t } = useI18n();
  if (loading) {
    return <p className="text-sm text-[var(--tl-ink-muted)]">{t("knowledge.impactHistoryLoading")}</p>;
  }
  if (error) {
    return <p className="text-sm text-[var(--tl-danger)]">{error}</p>;
  }
  if (impacts.length === 0) {
    return <p className="text-sm text-[var(--tl-ink-muted)]">{t("knowledge.impactHistoryEmpty")}</p>;
  }
  return (
    <div className="grid gap-3">
      {impacts.map((impact) => (
        <article
          key={impact.id}
          className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4"
        >
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div>
              <p className="text-xs font-bold uppercase tracking-wide text-[var(--tl-ink-muted)]">
                {t("knowledge.impactTask")}
              </p>
              <p className="mt-1 font-mono text-sm font-bold">{impact.task_id}</p>
            </div>
            <span className="rounded-full border border-[var(--tl-outline)] px-2.5 py-1 text-xs font-bold">
              {t(stateKeys[impact.state])}
            </span>
          </div>

          <div className="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-xs text-[var(--tl-ink-muted)]">
            {impact.stage ? <span>{t("knowledge.impactStage")}: {impact.stage}</span> : null}
            <span className="inline-flex items-center gap-1">
              <UserRound size={13} /> {impact.actor || t("knowledge.impactSystem")}
            </span>
            <span className="inline-flex items-center gap-1">
              <Clock3 size={13} />
              {new Intl.DateTimeFormat(locale, {
                dateStyle: "medium",
                timeStyle: "short",
              }).format(new Date(impact.updated_at))}
            </span>
          </div>

          {impact.recall_reasons.length > 0 ? (
            <div className="mt-3 flex flex-wrap gap-1.5">
              {impact.recall_reasons.map((reason) => (
                <span
                  key={reason}
                  className="rounded-md bg-[var(--tl-surface)] px-2 py-1 font-mono text-xs"
                >
                  {reason}
                </span>
              ))}
            </div>
          ) : null}

          {impact.notes ? <p className="mt-3 text-sm leading-6">{impact.notes}</p> : null}
          {impact.state === "unconfirmed" ? (
            <p className="mt-3 text-sm leading-6 text-[var(--tl-warning)]">
              {t("knowledge.impactUnconfirmedHint")}
            </p>
          ) : null}

          {impact.evidence.length > 0 ? (
            <div className="mt-3 grid gap-2">
              {impact.evidence.map((evidence, index) => (
                <div
                  key={`${evidence.kind}-${evidence.ref}-${index}`}
                  className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3 text-sm"
                >
                  <div className="flex items-center gap-2 text-xs font-bold text-[var(--tl-ink-muted)]">
                    {impact.state === "helpful" ? <CheckCircle2 size={14} /> : <FileCheck2 size={14} />}
                    {evidence.kind}
                  </div>
                  {evidence.ref ? <p className="mt-1 font-mono text-xs">{evidence.ref}</p> : null}
                  {evidence.summary ? <p className="mt-1 leading-5">{evidence.summary}</p> : null}
                </div>
              ))}
            </div>
          ) : null}
        </article>
      ))}
    </div>
  );
}
