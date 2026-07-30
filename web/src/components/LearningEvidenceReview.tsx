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

const EVIDENCE_COPY: Record<
  EvidenceType,
  { label: string; instruction: string; example: string }
> = {
  "command-test": {
    label: "Command or test",
    instruction:
      "Record the command, relevant result, and environment when it matters.",
    example:
      "Example: ./gradlew :service:compileDebugKotlin passed; the dependency resolved from the existing service group.",
  },
  "code-config": {
    label: "Code or configuration",
    instruction:
      "Name the file path, symbol or value, and the structure you observed.",
    example:
      "Example: settings.gradle includes :service:foo; neighboring service modules use the same group.",
  },
  "reviewed-document": {
    label: "Reviewed document",
    instruction:
      "Name the document or link, its reviewed state, and the conclusion that supports this memory.",
    example:
      "Example: Architecture review module-boundaries.md approved this dependency placement.",
  },
  "reproduction-fix": {
    label: "Reproduction and fix",
    instruction:
      "Record the behavior before the change, the change, and the behavior after it.",
    example:
      "Example: Moving the dependency to the service group removed the clean-sync resolution failure.",
  },
  "project-convention": {
    label: "Existing project convention",
    instruction:
      "Name at least two existing examples or one governing project rule.",
    example:
      "Example: service:a and service:b are both declared in the existing service dependency group.",
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
  const marker = /(?:^|\n)Result:\s*([^\n]*)/i.exec(value);
  return marker ? marker[1].trim().length >= 4 : trimmed.length >= 12;
}

export function appendEvidenceReference(current: string, checked: string): string {
  const block = `Checked: ${checked}\nResult: `;
  return current.trim() ? `${current.trim()}\n\n${block}` : block;
}

function referencesFromMaterial(material?: SourceMaterial): EvidenceReference[] {
  if (!material) return [];
  const docs: EvidenceReference[] = (material.task.docs ?? []).map((doc) => ({
    id: `doc-${doc.id}`,
    kind: "document",
    label: doc.title,
    checked: `document "${doc.title}"${doc.url ? ` (${doc.url})` : ""}`,
  }));
  const links: EvidenceReference[] = (material.task.links ?? []).map((link) => {
    const label = link.label || link.url;
    return {
      id: `link-${link.id}`,
      kind: "link",
      label,
      checked: `link "${label}" (${link.url})`,
    };
  });
  const events: EvidenceReference[] = material.events.map((event) => ({
    id: `event-${event.id}`,
    kind: "event",
    label: event.summary,
    checked: `task event "${event.summary}"`,
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
  const references = referencesFromMaterial(material.data);
  const selectedCopy = EVIDENCE_COPY[evidenceType];
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
        <h3 className="text-base font-bold">
          Confirm whether this experience is reliable
        </h3>
        <p className="mt-1 text-sm leading-6 text-[var(--tl-ink-muted)]">
          Verified experience may be given to later Agents. Add one fact another
          person can recheck.
        </p>
      </div>

      <label className="block text-sm font-semibold">
        Memory type
        <select
          aria-label="Memory type"
          value={memoryClass}
          onChange={(event) => setMemoryClass(event.target.value as MemoryClass)}
          className="mt-1 h-10 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 font-normal"
        >
          <option value="experience">Scoped experience · recall when relevant</option>
          <option value="project-rule">Project rule · always apply</option>
        </select>
        <span className="mt-1 block font-normal leading-5 text-[var(--tl-ink-muted)]">
          {memoryClass === "project-rule"
            ? "Project rule: included in every task context for this project."
            : "Scoped experience: recalled only when a later task matches this scope."}
        </span>
      </label>

      <fieldset>
        <legend className="text-sm font-semibold">
          What kind of evidence did you check?
        </legend>
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
              {EVIDENCE_COPY[type].label}
            </button>
          ))}
        </div>
        <div className="mt-3 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3">
          <p className="text-sm leading-6">{selectedCopy.instruction}</p>
          <p className="mt-1 text-xs leading-5 text-[var(--tl-ink-muted)]">
            {selectedCopy.example}
          </p>
        </div>
      </fieldset>

      <section aria-labelledby={`learning-material-${note.id}`}>
        <div className="flex items-center justify-between gap-3">
          <h4 id={`learning-material-${note.id}`} className="text-sm font-semibold">
            Evidence available from this task
          </h4>
          {material.isError && (
            <button
              type="button"
              className="inline-flex items-center gap-1 text-xs font-semibold text-[var(--tl-primary)] hover:underline"
              onClick={() => void material.refetch()}
            >
              <RotateCcw aria-hidden="true" size={13} />
              Retry
            </button>
          )}
        </div>
        {material.isLoading ? (
          <p className="mt-2 text-sm text-[var(--tl-ink-muted)]">
            Loading task material...
          </p>
        ) : material.isError ? (
          <p className="mt-2 text-sm leading-6 text-[var(--tl-rust)]">
            Source task material is unavailable. You can still enter evidence manually.
          </p>
        ) : references.length === 0 ? (
          <p className="mt-2 text-sm text-[var(--tl-ink-muted)]">
            No attached documents, links, or recent task events.
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
                    appendEvidenceReference(current, reference.checked)
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
        What did you verify?
        <textarea
          aria-label="What did you verify?"
          aria-describedby={evidenceDescriptionID}
          required
          value={evidence}
          placeholder={"Checked:\nResult:\nScope or environment (optional):"}
          onChange={(event) => setEvidence(event.target.value)}
          className="mt-1 min-h-32 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3 font-normal leading-6"
        />
      </label>
      <div
        id={evidenceDescriptionID}
        className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3 text-xs leading-5 text-[var(--tl-ink-muted)]"
      >
        <p>✓ Name a concrete command, file, document, behavior, or convention.</p>
        <p>✓ Record the observed result, not only an opinion.</p>
        <p>✓ Include enough scope to know when it applies.</p>
        {!hasEvidenceResult(evidence) && (
          <p className="mt-1 font-semibold text-[var(--tl-rust)]">
            Add the observed result before enabling this memory.
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
          {saving ? "Saving..." : "Verify and use in later tasks"}
        </button>
        <button type="button" className="secondary-button" onClick={onCancel}>
          Cancel
        </button>
      </div>
    </form>
  );
}
