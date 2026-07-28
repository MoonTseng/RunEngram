import { ChevronRight, RefreshCw, ShieldCheck } from "lucide-react";
import type { ExplorationCapsule } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function MemoryList({
  capsules,
  selectedID,
  onSelect,
}: {
  capsules: ExplorationCapsule[];
  selectedID: string | null;
  onSelect: (capsule: ExplorationCapsule) => void;
}) {
  const { t } = useI18n();
  if (capsules.length === 0) {
    return (
      <div className="rounded-xl border border-dashed border-[var(--tl-outline)] p-8 text-center">
        <p className="font-semibold">{t("knowledge.empty")}</p>
        <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">{t("knowledge.emptyHint")}</p>
      </div>
    );
  }
  return (
    <div className="space-y-2">
      {capsules.map((capsule) => {
        const feedback = capsule.helpful_count + capsule.rejected_count;
        const helpful = feedback > 0
          ? Math.round((capsule.helpful_count / feedback) * 100)
          : null;
        return (
          <button
            key={capsule.id}
            type="button"
            aria-pressed={selectedID === capsule.id}
            onClick={() => onSelect(capsule)}
            className={
              "w-full rounded-xl border p-4 text-left transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)] " +
              (selectedID === capsule.id
                ? "border-[var(--tl-primary)] bg-[var(--tl-surface-raised)] shadow-[var(--tl-shadow-card)]"
                : "border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] hover:border-[var(--tl-outline-strong)]")
            }
          >
            <div className="flex items-start gap-3">
              {capsule.status === "stale" ? (
                <RefreshCw size={17} className="mt-1 shrink-0 text-[var(--tl-ochre)]" />
              ) : (
                <ShieldCheck size={17} className="mt-1 shrink-0 text-[var(--tl-moss)]" />
              )}
              <div className="min-w-0 flex-1">
                <h3 className="font-bold leading-6">{capsule.title}</h3>
                <p className="mt-1 line-clamp-2 text-sm leading-6 text-[var(--tl-ink-muted)]">{capsule.summary}</p>
                <p className="mt-2 text-xs text-[var(--tl-ink-faint)]">
                  {capsule.producer || "codex"} · {capsule.use_count} {t("knowledge.uses")}
                  {helpful === null ? "" : ` · ${helpful}% ${t("knowledge.helpful")}`}
                </p>
              </div>
              <ChevronRight size={17} className="mt-1 shrink-0 text-[var(--tl-ink-faint)]" />
            </div>
          </button>
        );
      })}
    </div>
  );
}
