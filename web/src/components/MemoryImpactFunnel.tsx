import { ArrowRight, CircleCheckBig, Lightbulb, Radar } from "lucide-react";

import type { LearningMetrics } from "../lib/api";
import { useI18n } from "../lib/i18n";

type ImpactMetrics = Pick<
  LearningMetrics,
  | "recalled_task_count"
  | "recalled_memory_count"
  | "applied_task_count"
  | "helpful_task_count"
  | "ignored_count"
  | "unconfirmed_count"
  | "recall_coverage_rate"
  | "application_rate"
  | "confirmation_rate"
>;

function percentage(value: number): string {
  return `${Math.round(value * 100)}%`;
}

export function MemoryImpactFunnel({ metrics }: { metrics?: ImpactMetrics }) {
  const { locale, t } = useI18n();
  const recalled = metrics?.recalled_task_count ?? 0;
  const applied = metrics?.applied_task_count ?? 0;
  const helpful = metrics?.helpful_task_count ?? 0;

  if (recalled === 0) {
    return (
      <section className="panel p-5" aria-label={t("knowledge.impactTitle")}>
        <p className="eyebrow">{t("knowledge.impactEyebrow")}</p>
        <h2 className="mt-1 text-xl font-bold">{t("knowledge.impactTitle")}</h2>
        <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
          {t("knowledge.impactEmpty")}
        </p>
      </section>
    );
  }

  const taskText = (count: number, singular: string, plural: string) =>
    locale === "zh-CN"
      ? `${count} ${count === 1 ? singular : plural}`
      : `${count} ${count === 1 ? singular : plural}`;
  const stages = [
    {
      icon: Radar,
      value: taskText(
        recalled,
        t("knowledge.impactRecalledOne"),
        t("knowledge.impactRecalledMany")
      ),
      rate: percentage(metrics?.recall_coverage_rate ?? 0),
      hint: t("knowledge.impactRecalledHint"),
    },
    {
      icon: Lightbulb,
      value: taskText(
        applied,
        t("knowledge.impactAppliedOne"),
        t("knowledge.impactAppliedMany")
      ),
      rate: percentage(metrics?.application_rate ?? 0),
      hint: t("knowledge.impactAppliedHint"),
    },
    {
      icon: CircleCheckBig,
      value: taskText(
        helpful,
        t("knowledge.impactHelpfulOne"),
        t("knowledge.impactHelpfulMany")
      ),
      rate: percentage(metrics?.confirmation_rate ?? 0),
      hint: t("knowledge.impactHelpfulHint"),
    },
  ];

  return (
    <section className="panel p-5" aria-label={t("knowledge.impactTitle")}>
      <div className="mb-4">
        <p className="eyebrow">{t("knowledge.impactEyebrow")}</p>
        <h2 className="mt-1 text-xl font-bold">{t("knowledge.impactTitle")}</h2>
        <p className="mt-1 text-sm text-[var(--tl-ink-muted)]">
          {t("knowledge.impactSubtitle")}
        </p>
      </div>
      <div className="grid items-stretch gap-3 lg:grid-cols-[1fr_auto_1fr_auto_1fr]">
        {stages.map(({ icon: Icon, value, rate, hint }, index) => (
          <div className="contents" key={value}>
            <article className="rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
              <div className="flex items-center justify-between gap-3">
                <Icon size={20} className="text-[var(--tl-primary)]" />
                <strong className="text-xl">{rate}</strong>
              </div>
              <p className="mt-3 font-bold">{value}</p>
              <p className="mt-1 text-sm leading-5 text-[var(--tl-ink-muted)]">{hint}</p>
            </article>
            {index < stages.length - 1 ? (
              <ArrowRight
                aria-hidden
                className="mx-auto hidden self-center text-[var(--tl-ink-muted)] lg:block"
                size={20}
              />
            ) : null}
          </div>
        ))}
      </div>
      <div className="mt-3 flex flex-wrap gap-x-5 gap-y-1 text-sm text-[var(--tl-ink-muted)]">
        <span>
          {metrics?.unconfirmed_count ?? 0} {t("knowledge.impactAwaiting")}
        </span>
        <span>
          {metrics?.ignored_count ?? 0} {t("knowledge.impactIgnored")}
        </span>
        <span>
          {metrics?.recalled_memory_count ?? 0} {t("knowledge.impactRecallEvents")}
        </span>
      </div>
    </section>
  );
}
