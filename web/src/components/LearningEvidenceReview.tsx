import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { FileText, History, Link2, RotateCcw } from "lucide-react";

import {
  getTask,
  listTaskEvents,
  type LearningNote,
  type MemoryClass,
  type Task,
  type TaskEvent,
} from "../lib/api";
import { useI18n, type TranslationKey } from "../lib/i18n";

export type EvidenceType =
  | "command-test"
  | "code-config"
  | "reviewed-document"
  | "reproduction-fix"
  | "project-convention";

const EVIDENCE_TYPES: EvidenceType[] = [
  "command-test",
  "code-config",
  "reviewed-document",
  "reproduction-fix",
  "project-convention",
];

const EVIDENCE_COPY_KEYS: Record<
  EvidenceType,
  { label: TranslationKey; instruction: TranslationKey; example: TranslationKey }
> = {
  "command-test": {
    label: "knowledge.evidenceType.commandTest",
    instruction: "knowledge.evidenceInstruction.commandTest",
    example: "knowledge.evidenceExample.commandTest",
  },
  "code-config": {
    label: "knowledge.evidenceType.codeConfig",
    instruction: "knowledge.evidenceInstruction.codeConfig",
    example: "knowledge.evidenceExample.codeConfig",
  },
  "reviewed-document": {
    label: "knowledge.evidenceType.reviewedDocument",
    instruction: "knowledge.evidenceInstruction.reviewedDocument",
    example: "knowledge.evidenceExample.reviewedDocument",
  },
  "reproduction-fix": {
    label: "knowledge.evidenceType.reproductionFix",
    instruction: "knowledge.evidenceInstruction.reproductionFix",
    example: "knowledge.evidenceExample.reproductionFix",
  },
  "project-convention": {
    label: "knowledge.evidenceType.projectConvention",
    instruction: "knowledge.evidenceInstruction.projectConvention",
    example: "knowledge.evidenceExample.projectConvention",
  },
};

interface SourceMaterial {
  task: Task;
  events: TaskEvent[];
}

interface EvidenceReference {
  id: string;
  label: string;
  checked: string;
  kind: "document" | "link" | "event";
}

export function hasEvidenceResult(value: string): boolean {
  const trimmed = value.trim();
  if (!trimmed) return false;
  const marker = /(?:^|\n)(?:Result|观察结果)\s*[：:]\s*([^\n]*)/i.exec(value);
  return marker ? marker[1].trim().length >= 4 : trimmed.length >= 12;
}

export function appendEvidenceReference(
  current: string,
  checked: string,
  checkedLabel = "Checked",
  resultLabel = "Result"
): string {
  const block = `${checkedLabel}: ${checked}\n${resultLabel}: `;
  return current.trim() ? `${current.trim()}\n\n${block}` : block;
}

function referencesFromMaterial(
  material: SourceMaterial | undefined,
  labels: { document: string; link: string; event: string }
): EvidenceReference[] {
  if (!material) return [];
  const docs: EvidenceReference[] = (material.task.docs ?? []).map((doc) => ({
    id: `doc-${doc.id}`,
    kind: "document",
    label: doc.title,
    checked: `${labels.document} "${doc.title}"${doc.url ? ` (${doc.url})` : ""}`,
  }));
  const links: EvidenceReference[] = (material.task.links ?? []).map((link) => {
    const label = link.label || link.url;
    return {
      id: `link-${link.id}`,
      kind: "link",
      label,
      checked: `${labels.link} "${label}" (${link.url})`,
    };
  });
  const events: EvidenceReference[] = material.events.map((event) => ({
    id: `event-${event.id}`,
    kind: "event",
    label: event.summary,
    checked: `${labels.event} "${event.summary}"`,
  }));
  return [...docs, ...links, ...events];
}

function ReferenceIcon({ kind }: { kind: EvidenceReference["kind"] }) {
  if (kind === "document") return <FileText aria-hidden="true" size={15} />;
  if (kind === "link") return <Link2 aria-hidden="true" size={15} />;
  return <History aria-hidden="true" size={15} />;
}

export function LearningEvidenceReview({
  note,
  onPromote,
  onCancel,
}: {
  note: LearningNote;
  onPromote: (evidence: string, memoryClass: MemoryClass) => Promise<void>;
  onCancel: () => void;
}) {
  const { t } = useI18n();
  const [memoryClass, setMemoryClass] = useState<MemoryClass>("experience");
  const [evidenceType, setEvidenceType] = useState<EvidenceType>("command-test");
  const [evidence, setEvidence] = useState("");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const material = useQuery({
    queryKey: ["learning-evidence-material", note.source_task_id],
    queryFn: async (): Promise<SourceMaterial> => {
      const [task, events] = await Promise.all([
        getTask(note.source_task_id),
        listTaskEvents(note.source_task_id),
      ]);
      return { task, events: events.slice(-5).reverse() };
    },
  });
  const references = referencesFromMaterial(material.data, {
    document: t("knowledge.evidenceDocument"),
    link: t("knowledge.evidenceLink"),
    event: t("knowledge.evidenceEvent"),
  });
  const selectedCopy = EVIDENCE_COPY_KEYS[evidenceType];
  const evidenceDescriptionID = `learning-evidence-help-${note.id}`;

  return (
    <form
      className="mt-4 space-y-4 rounded-xl border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4"
      onSubmit={async (event) => {
        event.preventDefault();
        if (!hasEvidenceResult(evidence)) return;
        setSaving(true);
        setError("");
        try {
          await onPromote(evidence.trim(), memoryClass);
        } catch (reviewError) {
          setError(String(reviewError));
        } finally {
          setSaving(false);
        }
      }}
    >
      <div>
        <h3 className="text-base font-bold">{t("knowledge.evidenceReviewTitle")}</h3>
        <p className="mt-1 text-sm leading-6 text-[var(--tl-ink-muted)]">
          {t("knowledge.evidenceReviewPurpose")}
        </p>
      </div>

      <label className="block text-sm font-semibold">
        {t("knowledge.memoryClass")}
        <select
          aria-label={t("knowledge.memoryClass")}
          value={memoryClass}
          onChange={(event) => setMemoryClass(event.target.value as MemoryClass)}
          className="mt-1 h-10 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 font-normal"
        >
          <option value="experience">{t("knowledge.scopedExperience")}</option>
          <option value="project-rule">{t("knowledge.projectRule")}</option>
        </select>
        <span className="mt-1 block font-normal leading-5 text-[var(--tl-ink-muted)]">
          {memoryClass === "project-rule"
            ? t("knowledge.projectRuleEffect")
            : t("knowledge.scopedExperienceEffect")}
        </span>
      </label>

      <fieldset>
        <legend className="text-sm font-semibold">{t("knowledge.evidenceType")}</legend>
        <div className="mt-2 flex flex-wrap gap-2">
          {EVIDENCE_TYPES.map((type) => (
            <button
              key={type}
              type="button"
              role="radio"
              aria-checked={evidenceType === type}
              className={
                "rounded-full border px-3 py-2 text-sm font-semibold transition " +
                (evidenceType === type
                  ? "border-[var(--tl-primary)] bg-[var(--tl-primary)] text-[var(--tl-primary-contrast)]"
                  : "border-[var(--tl-outline)] bg-[var(--tl-surface)] text-[var(--tl-ink-muted)] hover:text-[var(--tl-ink)]")
              }
              onClick={() => setEvidenceType(type)}
            >
              {t(EVIDENCE_COPY_KEYS[type].label)}
            </button>
          ))}
        </div>
        <div className="mt-3 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3">
          <p className="text-sm leading-6">{t(selectedCopy.instruction)}</p>
          <p className="mt-1 text-xs leading-5 text-[var(--tl-ink-muted)]">
            {t(selectedCopy.example)}
          </p>
        </div>
      </fieldset>

      <section aria-labelledby={`learning-material-${note.id}`}>
        <div className="flex items-center justify-between gap-3">
          <h4 id={`learning-material-${note.id}`} className="text-sm font-semibold">
            {t("knowledge.evidenceAvailable")}
          </h4>
          {material.isError && (
            <button
              type="button"
              className="inline-flex items-center gap-1 text-xs font-semibold text-[var(--tl-primary)] hover:underline"
              onClick={() => void material.refetch()}
            >
              <RotateCcw aria-hidden="true" size={13} />
              {t("knowledge.evidenceRetry")}
            </button>
          )}
        </div>
        {material.isLoading ? (
          <p className="mt-2 text-sm text-[var(--tl-ink-muted)]">
            {t("knowledge.evidenceLoading")}
          </p>
        ) : material.isError ? (
          <p className="mt-2 text-sm leading-6 text-[var(--tl-rust)]">
            {t("knowledge.evidenceUnavailable")}
          </p>
        ) : references.length === 0 ? (
          <p className="mt-2 text-sm text-[var(--tl-ink-muted)]">
            {t("knowledge.evidenceEmpty")}
          </p>
        ) : (
          <div className="mt-2 flex flex-wrap gap-2">
            {references.map((reference) => (
              <button
                key={reference.id}
                type="button"
                className="inline-flex max-w-full items-center gap-2 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 py-2 text-left text-sm hover:border-[var(--tl-primary)]"
                onClick={() =>
                  setEvidence((current) =>
                    appendEvidenceReference(
                      current,
                      reference.checked,
                      t("knowledge.evidenceCheckedLabel"),
                      t("knowledge.evidenceResultLabel")
                    )
                  )
                }
              >
                <ReferenceIcon kind={reference.kind} />
                <span className="truncate">{reference.label}</span>
              </button>
            ))}
          </div>
        )}
      </section>

      <label className="block text-sm font-semibold">
        {t("knowledge.evidenceQuestion")}
        <textarea
          aria-label={t("knowledge.evidenceQuestion")}
          aria-describedby={evidenceDescriptionID}
          required
          value={evidence}
          placeholder={t("knowledge.evidenceTemplate")}
          onChange={(event) => setEvidence(event.target.value)}
          className="mt-1 min-h-32 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3 font-normal leading-6"
        />
      </label>
      <div
        id={evidenceDescriptionID}
        className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3 text-xs leading-5 text-[var(--tl-ink-muted)]"
      >
        <p>✓ {t("knowledge.evidenceChecklistSource")}</p>
        <p>✓ {t("knowledge.evidenceChecklistResult")}</p>
        <p>✓ {t("knowledge.evidenceChecklistScope")}</p>
        {!hasEvidenceResult(evidence) && (
          <p className="mt-1 font-semibold text-[var(--tl-rust)]">
            {t("knowledge.evidenceRequired")}
          </p>
        )}
      </div>

      {error && <p className="text-sm text-[var(--tl-rust)]">{error}</p>}
      <div className="flex flex-wrap gap-2">
        <button
          type="submit"
          className="primary-button"
          disabled={saving || !hasEvidenceResult(evidence)}
        >
          {saving ? t("actions.saving") : t("knowledge.verifyAndReuse")}
        </button>
        <button type="button" className="secondary-button" onClick={onCancel}>
          {t("actions.cancel")}
        </button>
      </div>
    </form>
  );
}
