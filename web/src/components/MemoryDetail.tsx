import { Clipboard, ExternalLink, ShieldCheck } from "lucide-react";
import { useState } from "react";
import type { ExplorationCapsule } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function MemoryDetail({ capsule }: { capsule: ExplorationCapsule | null }) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  if (!capsule) {
    return (
      <div className="panel flex min-h-80 items-center justify-center p-8 text-center text-sm leading-6 text-[var(--tl-ink-muted)]">
        {t("knowledge.selectMemory")}
      </div>
    );
  }
  const copy = async () => {
    await navigator.clipboard?.writeText(`${capsule.title}\n\n${capsule.summary}\n\n${capsule.evidence}`);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1_500);
  };
  return (
    <article className="panel sticky top-0 max-h-[calc(100vh-12rem)] overflow-y-auto p-5 lg:p-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="eyebrow">{capsule.status === "stale" ? t("knowledge.needsRevalidation") : t("knowledge.verifiedExperience")}</p>
          <h2 className="mt-2 text-2xl font-bold leading-tight">{capsule.title}</h2>
        </div>
        <ShieldCheck size={22} className="shrink-0 text-[var(--tl-moss)]" />
      </div>
      <p className="mt-4 text-[15px] leading-7 text-[var(--tl-ink-muted)]">{capsule.summary}</p>

      <section className="mt-6 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
        <h3 className="font-bold">{t("knowledge.whyTrusted")}</h3>
        <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
          {t("knowledge.trustExplanation")}
        </p>
      </section>

      <dl className="mt-5 grid gap-4 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-[var(--tl-ink-faint)]">{t("knowledge.scope")}</dt>
          <dd className="mt-1 font-semibold">{capsule.scope || "—"}</dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">{t("knowledge.sourceTask")}</dt>
          <dd className="mt-1 break-all font-mono text-xs">{capsule.source_task_id || "—"}</dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">{t("knowledge.producer")}</dt>
          <dd className="mt-1 font-semibold">{capsule.producer || "codex"}</dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">{t("knowledge.observedReuse")}</dt>
          <dd className="mt-1 font-semibold">{capsule.use_count} / {capsule.helpful_count} {t("knowledge.helpful")}</dd>
        </div>
      </dl>

      <div className="mt-5 flex flex-wrap gap-2">
        {[...capsule.labels, ...capsule.fingerprints].map((value) => (
          <span key={value} className="rounded-md border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] px-2 py-1 text-xs text-[var(--tl-ink-muted)]">
            {value}
          </span>
        ))}
      </div>

      <section className="mt-6">
        <h3 className="font-bold">{t("knowledge.evidence")}</h3>
        <pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap rounded-lg bg-[var(--tl-bg-quiet)] p-4 text-sm leading-6">{capsule.evidence || "—"}</pre>
      </section>

      <div className="mt-5 flex flex-wrap gap-2">
        <button type="button" className="primary-button" onClick={copy}>
          <Clipboard size={16} /> {copied ? t("action.copied") : t("knowledge.copyGuidance")}
        </button>
        <button type="button" className="secondary-button" disabled title={t("knowledge.sourceTaskHint")}>
          <ExternalLink size={16} /> {t("knowledge.openSource")}
        </button>
      </div>
    </article>
  );
}
