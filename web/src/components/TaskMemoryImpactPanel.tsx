import { useState } from "react";

import type {
  ExplorationCapsule,
  MemoryImpact,
  MemoryImpactEvidenceKind,
  MemoryImpactState,
  UpdateMemoryImpactInput,
} from "../lib/api";
import { useI18n } from "../lib/i18n";

const evidenceKinds: MemoryImpactEvidenceKind[] = [
  "command",
  "task-doc",
  "task-event",
  "link",
  "code-reference",
  "observation",
];

type Draft = {
  impact: MemoryImpact;
  state: MemoryImpactState;
  stage: string;
  notes: string;
  evidenceKind: MemoryImpactEvidenceKind | "";
  evidenceRef: string;
  evidenceSummary: string;
};

export function TaskMemoryImpactPanel({
  impacts,
  capsules,
  onUpdate,
}: {
  impacts: MemoryImpact[];
  capsules: ExplorationCapsule[];
  onUpdate: (impactId: string, input: UpdateMemoryImpactInput) => Promise<unknown>;
}) {
  const { t } = useI18n();
  const [draft, setDraft] = useState<Draft | null>(null);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");

  if (impacts.length === 0) {
    return <p className="mt-3 text-sm text-[var(--tl-ink-muted)]">{t("action.impactEmpty")}</p>;
  }

  const begin = (impact: MemoryImpact, state: MemoryImpactState) => {
    setDraft({
      impact,
      state,
      stage: impact.stage,
      notes: impact.notes,
      evidenceKind: impact.evidence[0]?.kind ?? "",
      evidenceRef: impact.evidence[0]?.ref ?? "",
      evidenceSummary: impact.evidence[0]?.summary ?? "",
    });
    setError("");
  };

  const submit = async () => {
    if (!draft || !draft.notes.trim()) {
      setError(t("action.impactNotesRequired"));
      return;
    }
    const terminal = ["helpful", "rejected", "stale"].includes(draft.state);
    if (
      terminal &&
      (!draft.evidenceKind ||
        (!draft.evidenceRef.trim() && !draft.evidenceSummary.trim()))
    ) {
      setError(t("action.impactEvidenceRequired"));
      return;
    }
    const evidence = draft.evidenceKind
      ? [
          {
            kind: draft.evidenceKind,
            ref: draft.evidenceRef.trim(),
            summary: draft.evidenceSummary.trim(),
          },
        ]
      : [];
    setSaving(true);
    setError("");
    try {
      await onUpdate(draft.impact.id, {
        state: draft.state,
        stage: draft.stage.trim(),
        notes: draft.notes.trim(),
        evidence,
        expected_updated_at: draft.impact.updated_at,
      });
      setDraft(null);
    } catch (saveError) {
      setError(String(saveError));
    } finally {
      setSaving(false);
    }
  };

  return (
    <div className="mt-4 grid gap-3">
      {impacts.map((impact) => {
        const capsule = capsules.find((item) => item.id === impact.capsule_id);
        return (
          <article
            key={impact.id}
            className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4"
          >
            <div className="flex flex-wrap items-start justify-between gap-2">
              <div>
                <h4 className="font-bold">{capsule?.title ?? impact.capsule_id}</h4>
                <p className="mt-1 text-xs text-[var(--tl-ink-muted)]">
                  {t(`knowledge.impactState${impact.state[0].toUpperCase()}${impact.state.slice(1)}` as
                    | "knowledge.impactStateRecalled"
                    | "knowledge.impactStateApplied"
                    | "knowledge.impactStateIgnored"
                    | "knowledge.impactStateHelpful"
                    | "knowledge.impactStateRejected"
                    | "knowledge.impactStateStale"
                    | "knowledge.impactStateUnconfirmed")}
                </p>
              </div>
              <span className="font-mono text-xs text-[var(--tl-ink-faint)]">
                {Math.round(impact.recall_score * 100)}%
              </span>
            </div>
            {impact.recall_reasons.length > 0 ? (
              <div className="mt-2 flex flex-wrap gap-1">
                {impact.recall_reasons.map((reason) => (
                  <span key={reason} className="rounded bg-[var(--tl-surface)] px-2 py-1 font-mono text-xs">
                    {reason}
                  </span>
                ))}
              </div>
            ) : null}
            <div className="mt-3 flex flex-wrap gap-2">
              <button type="button" className="secondary-button" onClick={() => begin(impact, "applied")}>
                {t("action.impactApplied")}
              </button>
              <button type="button" className="secondary-button" onClick={() => begin(impact, "ignored")}>
                {t("action.impactIgnored")}
              </button>
              <button type="button" className="secondary-button" onClick={() => begin(impact, "helpful")}>
                {t("action.impactHelpful")}
              </button>
              <button type="button" className="secondary-button" onClick={() => begin(impact, "rejected")}>
                {t("action.impactRejected")}
              </button>
              <button type="button" className="secondary-button" onClick={() => begin(impact, "stale")}>
                {t("action.impactStale")}
              </button>
            </div>
          </article>
        );
      })}

      {draft ? (
        <div className="rounded-xl border border-[var(--tl-primary)]/50 bg-[var(--tl-surface)] p-4">
          <div className="grid gap-3">
            <label className="grid gap-1 text-sm font-semibold">
              {t("action.impactStage")}
              <input
                aria-label={t("action.impactStage")}
                value={draft.stage}
                onChange={(event) => setDraft({ ...draft, stage: event.target.value })}
                className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg)] px-3"
              />
            </label>
            <label className="grid gap-1 text-sm font-semibold">
              {t("action.impactNotes")}
              <textarea
                aria-label={t("action.impactNotes")}
                value={draft.notes}
                onChange={(event) => setDraft({ ...draft, notes: event.target.value })}
                className="min-h-24 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg)] p-3"
              />
            </label>
            {["helpful", "rejected", "stale"].includes(draft.state) ? (
              <>
                <label className="grid gap-1 text-sm font-semibold">
                  {t("action.impactEvidenceType")}
                  <select
                    aria-label={t("action.impactEvidenceType")}
                    value={draft.evidenceKind}
                    onChange={(event) =>
                      setDraft({
                        ...draft,
                        evidenceKind: event.target.value as MemoryImpactEvidenceKind,
                      })
                    }
                    className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg)] px-3"
                  >
                    <option value="">{t("action.impactSelectEvidence")}</option>
                    {evidenceKinds.map((kind) => (
                      <option key={kind} value={kind}>{kind}</option>
                    ))}
                  </select>
                </label>
                <label className="grid gap-1 text-sm font-semibold">
                  {t("action.impactEvidenceRef")}
                  <input
                    aria-label={t("action.impactEvidenceRef")}
                    value={draft.evidenceRef}
                    onChange={(event) => setDraft({ ...draft, evidenceRef: event.target.value })}
                    className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg)] px-3"
                  />
                </label>
                <label className="grid gap-1 text-sm font-semibold">
                  {t("action.impactEvidenceResult")}
                  <textarea
                    aria-label={t("action.impactEvidenceResult")}
                    value={draft.evidenceSummary}
                    onChange={(event) => setDraft({ ...draft, evidenceSummary: event.target.value })}
                    className="min-h-20 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg)] p-3"
                  />
                </label>
              </>
            ) : null}
          </div>
          {error ? <p className="mt-3 text-sm text-[var(--tl-danger)]">{error}</p> : null}
          <div className="mt-4 flex gap-2">
            <button type="button" className="primary-button" disabled={saving} onClick={submit}>
              {saving ? t("actions.saving") : t("action.impactSave")}
            </button>
            <button type="button" className="secondary-button" disabled={saving} onClick={() => setDraft(null)}>
              {t("actions.cancel")}
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
