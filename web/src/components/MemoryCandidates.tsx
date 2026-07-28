import { useState } from "react";
import type { LearningNote } from "../lib/api";
import type { UpdateLearningNoteInput } from "../lib/api";
import { useI18n } from "../lib/i18n";

export function MemoryCandidates({
  notes,
  onUpdate,
}: {
  notes: LearningNote[];
  onUpdate: (noteId: string, input: UpdateLearningNoteInput) => Promise<void>;
}) {
  const { t } = useI18n();
  const [editingID, setEditingID] = useState<string | null>(null);
  const [draft, setDraft] = useState<UpdateLearningNoteInput>({
    trigger: "",
    guidance: "",
    scope: "",
  });
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
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
          {editingID === note.id ? (
            <form
              className="mt-4 space-y-3 border-t border-[var(--tl-outline)] pt-4"
              onSubmit={async (event) => {
                event.preventDefault();
                setSaving(true);
                setError("");
                try {
                  await onUpdate(note.id, draft);
                  setEditingID(null);
                } catch (updateError) {
                  setError(String(updateError));
                } finally {
                  setSaving(false);
                }
              }}
            >
              <label className="block text-sm font-semibold">
                {t("knowledge.trigger")}
                <textarea
                  aria-label={t("knowledge.trigger")}
                  value={draft.trigger}
                  onChange={(event) => setDraft((current) => ({ ...current, trigger: event.target.value }))}
                  className="mt-1 min-h-20 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-3 font-normal outline-none focus:border-[var(--tl-primary)]"
                />
              </label>
              <label className="block text-sm font-semibold">
                {t("knowledge.guidance")}
                <textarea
                  aria-label={t("knowledge.guidance")}
                  value={draft.guidance}
                  onChange={(event) => setDraft((current) => ({ ...current, guidance: event.target.value }))}
                  className="mt-1 min-h-24 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-3 font-normal outline-none focus:border-[var(--tl-primary)]"
                />
              </label>
              <label className="block text-sm font-semibold">
                {t("knowledge.scope")}
                <input
                  aria-label={t("knowledge.scope")}
                  value={draft.scope}
                  onChange={(event) => setDraft((current) => ({ ...current, scope: event.target.value }))}
                  className="mt-1 h-10 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] px-3 font-normal outline-none focus:border-[var(--tl-primary)]"
                />
              </label>
              {error && <p className="text-sm text-[var(--tl-rust)]">{error}</p>}
              <div className="flex gap-2">
                <button type="submit" className="primary-button" disabled={saving}>
                  {saving ? t("actions.saving") : t("knowledge.saveChanges")}
                </button>
                <button
                  type="button"
                  className="secondary-button"
                  onClick={() => {
                    setEditingID(null);
                    setError("");
                  }}
                >
                  {t("actions.cancel")}
                </button>
              </div>
            </form>
          ) : (
            <>
              <p className="mt-4 text-sm leading-6"><strong>{t("knowledge.trigger")}:</strong> {note.trigger}</p>
              {note.scope && <p className="mt-2 text-sm text-[var(--tl-ink-muted)]"><strong>{t("knowledge.scope")}:</strong> {note.scope}</p>}
              <button
                type="button"
                className="mt-4 text-sm font-semibold text-[var(--tl-primary)] hover:underline"
                onClick={() => {
                  setDraft({ trigger: note.trigger, guidance: note.guidance, scope: note.scope });
                  setEditingID(note.id);
                  setError("");
                }}
              >
                {t("knowledge.edit")}
              </button>
            </>
          )}
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
