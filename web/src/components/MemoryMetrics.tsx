import {
  BookOpenCheck,
  Bot,
  CheckCircle2,
  RefreshCw,
  RotateCcw,
  ShieldCheck,
  Sparkles,
  Unplug,
} from "lucide-react";
import type { LearningMetrics } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function MemoryMetrics({ metrics }: { metrics?: LearningMetrics }) {
  const { t } = useI18n();
  const items = [
    { label: t("knowledge.verified"), value: metrics?.active_capsule_count ?? 0, icon: ShieldCheck },
    { label: t("knowledge.reusedTasks"), value: metrics?.reused_task_count ?? 0, icon: Sparkles },
    { label: t("knowledge.pendingReview"), value: metrics?.pending_note_count ?? 0, icon: BookOpenCheck },
    { label: t("knowledge.needsRevalidation"), value: metrics?.stale_count ?? 0, icon: RefreshCw },
    { label: t("knowledge.agentRuns"), value: metrics?.run_count ?? 0, icon: Bot },
    {
      label: t("knowledge.runCompletionRate"),
      value: `${Math.round((metrics?.run_completion_rate ?? 0) * 100)}%`,
      icon: CheckCircle2,
    },
    { label: t("knowledge.resumedRuns"), value: metrics?.resumed_run_count ?? 0, icon: RotateCcw },
    {
      label: t("knowledge.recoveryRate"),
      value: `${Math.round((metrics?.recovery_rate ?? 0) * 100)}%`,
      icon: Unplug,
    },
  ];
  return (
    <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {items.map(({ label, value, icon: Icon }) => (
        <div key={label} className="panel flex items-center justify-between p-4">
          <div className="flex items-center gap-2 text-sm text-[var(--tl-ink-muted)]">
            <Icon size={17} className="text-[var(--tl-primary)]" />
            {label}
          </div>
          <strong className="text-2xl">{value}</strong>
        </div>
      ))}
    </section>
  );
}
