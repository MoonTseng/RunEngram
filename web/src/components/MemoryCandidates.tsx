import type { LearningNote } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function MemoryCandidates({ notes }: { notes: LearningNote[] }) {
  const { t } = useI18n();
  if (notes.length === 0) {
    return <p className="py-10 text-center text-[var(--tl-ink-muted)]">{t("knowledge.noCandidates")}</p>;
  }
  return (
    <div className="grid gap-4 lg:grid-cols-2">
      {notes.map((note) => (
        <article key={note.id} className="panel p-5">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="text-xs font-bold uppercase tracking-[0.13em] text-[var(--tl-primary)]">
                {note.kind === "human-correction" ? t("knowledge.humanCorrection") : t("knowledge.agentRecovery")}
              </p>
              <h2 className="mt-2 text-lg font-bold leading-7">{note.guidance}</h2>
            </div>
            <span className="status-chip">{note.status}</span>
          </div>
          <p className="mt-4 text-sm leading-6"><strong>{t("knowledge.trigger")}:</strong> {note.trigger}</p>
          {note.scope && <p className="mt-2 text-sm text-[var(--tl-ink-muted)]"><strong>{t("knowledge.scope")}:</strong> {note.scope}</p>}
          {note.evidence && (
            <details className="mt-4 border-t border-[var(--tl-outline)] pt-3">
              <summary className="cursor-pointer font-semibold text-[var(--tl-primary)]">{t("knowledge.evidence")}</summary>
              <pre className="mt-3 max-h-56 overflow-auto whitespace-pre-wrap rounded-lg bg-[var(--tl-bg-quiet)] p-3 text-sm leading-6">{note.evidence}</pre>
            </details>
          )}
          <p className="mt-4 break-all text-xs text-[var(--tl-ink-faint)]">{t("knowledge.sourceTask")}: {note.source_task_id}</p>
        </article>
      ))}
    </div>
  );
}
