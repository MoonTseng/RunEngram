import {
  Clipboard,
  ExternalLink,
  Link2,
  Pencil,
  ShieldCheck,
  Trash2,
  X,
} from "lucide-react";
import { useEffect, useState } from "react";
import {
  createMemoryRelation,
  deleteMemoryRelation,
  updateCapsule,
  type ExplorationCapsule,
  type MemoryRelationTargetKind,
  type MemoryRelationType,
} from "../lib/api";
import { useI18n } from "../lib/i18n";

interface MemoryDetailProps {
  capsule: ExplorationCapsule | null;
  capsules?: ExplorationCapsule[];
  onUpdated?: (capsule: ExplorationCapsule) => void;
  onRelationsChanged?: () => void;
}

const relationTypes: MemoryRelationType[] = [
  "derived-from",
  "validated-by",
  "applies-to",
  "supersedes",
  "conflicts-with",
  "caused-by",
];

const targetKinds: MemoryRelationTargetKind[] = [
  "capsule",
  "task",
  "artifact",
  "scope",
];

export function MemoryDetail({
  capsule,
  capsules = [],
  onUpdated,
  onRelationsChanged,
}: MemoryDetailProps) {
  const { t } = useI18n();
  const [copied, setCopied] = useState(false);
  const [editing, setEditing] = useState(false);
  const [title, setTitle] = useState("");
  const [summary, setSummary] = useState("");
  const [trigger, setTrigger] = useState("");
  const [scope, setScope] = useState("");
  const [evidence, setEvidence] = useState("");
  const [relationType, setRelationType] =
    useState<MemoryRelationType>("validated-by");
  const [targetKind, setTargetKind] =
    useState<MemoryRelationTargetKind>("artifact");
  const [targetRef, setTargetRef] = useState("");
  const [relationNote, setRelationNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    setTitle(capsule?.title ?? "");
    setSummary(capsule?.summary ?? "");
    setTrigger(capsule?.trigger ?? "");
    setScope(capsule?.scope ?? "");
    setEvidence(capsule?.evidence ?? "");
    setEditing(false);
    setError("");
  }, [
    capsule?.evidence,
    capsule?.id,
    capsule?.scope,
    capsule?.summary,
    capsule?.title,
    capsule?.trigger,
    capsule?.updated_at,
  ]);

  if (!capsule) {
    return (
      <div className="panel flex min-h-80 items-center justify-center p-8 text-center text-sm leading-6 text-[var(--tl-ink-muted)]">
        {t("knowledge.selectMemory")}
      </div>
    );
  }

  const copy = async () => {
    await navigator.clipboard?.writeText(
      `${capsule.title}\n\n${capsule.summary}\n\n${capsule.evidence}`
    );
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1_500);
  };

  const save = async () => {
    setBusy(true);
    setError("");
    try {
      const updated = await updateCapsule(capsule.id, {
        title,
        summary,
        trigger,
        scope,
        evidence,
        expected_updated_at: capsule.updated_at,
      });
      onUpdated?.(updated);
      setEditing(false);
    } catch (saveError) {
      setError(
        String(saveError).includes("conflict")
          ? t("knowledge.concurrentConflict")
          : String(saveError)
      );
    } finally {
      setBusy(false);
    }
  };

  const addRelation = async () => {
    const trimmedTarget = targetRef.trim();
    if (!trimmedTarget) {
      setError(t("knowledge.targetRequired"));
      return;
    }
    setBusy(true);
    setError("");
    try {
      await createMemoryRelation(capsule.id, {
        type: relationType,
        target_kind: targetKind,
        target_ref: trimmedTarget,
        note: relationNote.trim(),
      });
      setTargetRef("");
      setRelationNote("");
      onRelationsChanged?.();
    } catch (relationError) {
      setError(String(relationError));
    } finally {
      setBusy(false);
    }
  };

  const removeRelation = async (relationId: string) => {
    setBusy(true);
    setError("");
    try {
      await deleteMemoryRelation(relationId);
      onRelationsChanged?.();
    } catch (relationError) {
      setError(String(relationError));
    } finally {
      setBusy(false);
    }
  };

  const selectRelationType = (value: MemoryRelationType) => {
    setRelationType(value);
    setTargetRef("");
    if (value === "applies-to") setTargetKind("scope");
    if (value === "supersedes" || value === "conflicts-with") {
      setTargetKind("capsule");
    }
  };

  return (
    <article className="panel sticky top-0 max-h-[calc(100vh-12rem)] overflow-y-auto p-5 lg:p-6">
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="eyebrow">
            {capsule.status === "stale"
              ? t("knowledge.needsRevalidation")
              : t("knowledge.verifiedExperience")}
          </p>
          <h2 className="mt-2 text-2xl font-bold leading-tight">
            {capsule.title}
          </h2>
        </div>
        <ShieldCheck
          size={22}
          className="shrink-0 text-[var(--tl-moss)]"
        />
      </div>

      {editing ? (
        <section className="mt-5 grid gap-3 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
          <label className="grid gap-1 text-sm font-semibold">
            {t("knowledge.memoryTitle")}
            <input
              aria-label={t("knowledge.memoryTitle")}
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3"
            />
          </label>
          <label className="grid gap-1 text-sm font-semibold">
            {t("knowledge.summary")}
            <textarea
              aria-label={t("knowledge.summary")}
              value={summary}
              onChange={(event) => setSummary(event.target.value)}
              className="min-h-24 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3"
            />
          </label>
          <label className="grid gap-1 text-sm font-semibold">
            {t("knowledge.trigger")}
            <textarea
              aria-label={t("knowledge.trigger")}
              value={trigger}
              onChange={(event) => setTrigger(event.target.value)}
              className="min-h-20 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3"
            />
          </label>
          <label className="grid gap-1 text-sm font-semibold">
            {t("knowledge.scope")}
            <input
              aria-label={t("knowledge.scope")}
              value={scope}
              onChange={(event) => setScope(event.target.value)}
              className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3"
            />
          </label>
          <label className="grid gap-1 text-sm font-semibold">
            {t("knowledge.evidence")}
            <textarea
              aria-label={t("knowledge.evidence")}
              value={evidence}
              onChange={(event) => setEvidence(event.target.value)}
              className="min-h-28 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] p-3"
            />
          </label>
          <div className="flex flex-wrap gap-2">
            <button
              type="button"
              className="primary-button"
              disabled={busy}
              onClick={save}
            >
              {t("action.saveChanges")}
            </button>
            <button
              type="button"
              className="secondary-button"
              onClick={() => setEditing(false)}
            >
              <X size={16} /> {t("action.cancel")}
            </button>
          </div>
        </section>
      ) : (
        <>
          <p className="mt-4 text-[15px] leading-7 text-[var(--tl-ink-muted)]">
            {capsule.summary}
          </p>
          {capsule.trigger && (
            <p className="mt-3 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-3 text-sm leading-6">
              <strong>{t("knowledge.trigger")}:</strong> {capsule.trigger}
            </p>
          )}
        </>
      )}

      <section className="mt-6 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
        <h3 className="font-bold">{t("knowledge.whyTrusted")}</h3>
        <p className="mt-2 text-sm leading-6 text-[var(--tl-ink-muted)]">
          {t("knowledge.trustExplanation")}
        </p>
      </section>

      <dl className="mt-5 grid gap-4 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.scope")}
          </dt>
          <dd className="mt-1 font-semibold">{capsule.scope || "—"}</dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.sourceTask")}
          </dt>
          <dd className="mt-1 break-all font-mono text-xs">
            {capsule.source_task_id || "—"}
          </dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.producer")}
          </dt>
          <dd className="mt-1 font-semibold">
            {capsule.producer || "codex"}
          </dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.observedReuse")}
          </dt>
          <dd className="mt-1 font-semibold">
            {capsule.use_count} / {capsule.helpful_count}{" "}
            {t("knowledge.helpful")}
          </dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.memoryClass")}
          </dt>
          <dd className="mt-1 font-semibold">
            {capsule.memory_class === "project-rule"
              ? t("knowledge.projectRule")
              : t("knowledge.scopedExperience")}
          </dd>
        </div>
        <div>
          <dt className="text-[var(--tl-ink-faint)]">
            {t("knowledge.confidence")}
          </dt>
          <dd className="mt-1 font-semibold">
            {Math.round(capsule.confidence * 100)}%
          </dd>
        </div>
      </dl>

      <div className="mt-5 flex flex-wrap gap-2">
        {[...capsule.labels, ...capsule.fingerprints].map((value) => (
          <span
            key={value}
            className="rounded-md border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] px-2 py-1 text-xs text-[var(--tl-ink-muted)]"
          >
            {value}
          </span>
        ))}
      </div>

      <section className="mt-6">
        <h3 className="font-bold">{t("knowledge.evidence")}</h3>
        <pre className="mt-2 max-h-72 overflow-auto whitespace-pre-wrap rounded-lg bg-[var(--tl-bg-quiet)] p-4 text-sm leading-6">
          {capsule.evidence || "—"}
        </pre>
      </section>

      <section className="mt-6 rounded-lg border border-[var(--tl-outline)] p-4">
        <div className="flex items-center gap-2">
          <Link2 size={18} className="text-[var(--tl-primary)]" />
          <h3 className="font-bold">{t("knowledge.memoryGraph")}</h3>
        </div>
        <p className="mt-1 text-sm leading-6 text-[var(--tl-ink-muted)]">
          {t("knowledge.memoryGraphHint")}
        </p>

        <div className="mt-3 grid gap-2">
          {(capsule.relations ?? []).length === 0 ? (
            <p className="text-sm text-[var(--tl-ink-faint)]">
              {t("knowledge.noRelations")}
            </p>
          ) : (
            capsule.relations.map((relation) => {
              const target =
                relation.direction === "incoming"
                  ? relation.source_capsule_id
                  : relation.target_ref;
              return (
                <div
                  key={`${relation.id}-${relation.direction}`}
                  className="flex items-start justify-between gap-3 rounded-lg bg-[var(--tl-bg-quiet)] p-3"
                >
                  <div className="min-w-0 text-sm">
                    <p className="font-semibold">
                      {relation.direction === "incoming" ? "←" : "→"}{" "}
                      {relation.type}
                    </p>
                    <p className="mt-1 break-all font-mono text-xs text-[var(--tl-ink-muted)]">
                      {relation.target_kind}:{target}
                    </p>
                    {relation.note && (
                      <p className="mt-1 text-[var(--tl-ink-muted)]">
                        {relation.note}
                      </p>
                    )}
                  </div>
                  <button
                    type="button"
                    className="icon-button"
                    disabled={busy}
                    aria-label={t("knowledge.removeRelation")}
                    title={t("knowledge.removeRelation")}
                    onClick={() => removeRelation(relation.id)}
                  >
                    <Trash2 size={15} />
                  </button>
                </div>
              );
            })
          )}
        </div>

        <div className="mt-4 grid gap-2 sm:grid-cols-2">
          <label className="grid gap-1 text-xs font-semibold">
            {t("knowledge.relationType")}
            <select
              aria-label={t("knowledge.relationType")}
              value={relationType}
              onChange={(event) =>
                selectRelationType(event.target.value as MemoryRelationType)
              }
              className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-2 text-sm"
            >
              {relationTypes.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-xs font-semibold">
            {t("knowledge.targetType")}
            <select
              aria-label={t("knowledge.targetType")}
              value={targetKind}
              disabled={
                relationType === "applies-to" ||
                relationType === "supersedes" ||
                relationType === "conflicts-with"
              }
              onChange={(event) => {
                setTargetRef("");
                setTargetKind(
                  event.target.value as MemoryRelationTargetKind
                );
              }}
              className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-2 text-sm"
            >
              {targetKinds.map((value) => (
                <option key={value} value={value}>
                  {value}
                </option>
              ))}
            </select>
          </label>
          <label className="grid gap-1 text-xs font-semibold sm:col-span-2">
            {t("knowledge.targetReference")}
            {targetKind === "capsule" ? (
              <select
                aria-label={t("knowledge.targetReference")}
                value={targetRef}
                onChange={(event) => setTargetRef(event.target.value)}
                className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-2 text-sm"
              >
                <option value="">{t("knowledge.selectMemoryTarget")}</option>
                {capsules
                  .filter((item) => item.id !== capsule.id)
                  .map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.title}
                    </option>
                  ))}
              </select>
            ) : (
              <input
                aria-label={t("knowledge.targetReference")}
                value={targetRef}
                onChange={(event) => setTargetRef(event.target.value)}
                placeholder={t("knowledge.targetReferenceHint")}
                className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 text-sm"
              />
            )}
          </label>
          <label className="grid gap-1 text-xs font-semibold sm:col-span-2">
            {t("knowledge.relationNote")}
            <input
              aria-label={t("knowledge.relationNote")}
              value={relationNote}
              onChange={(event) => setRelationNote(event.target.value)}
              className="h-10 rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] px-3 text-sm"
            />
          </label>
          <button
            type="button"
            className="secondary-button sm:col-span-2"
            disabled={busy}
            onClick={addRelation}
          >
            <Link2 size={16} /> {t("knowledge.addRelation")}
          </button>
        </div>
      </section>

      {error && (
        <p className="mt-4 rounded-lg border border-[var(--tl-danger)]/40 bg-[var(--tl-danger)]/10 p-3 text-sm text-[var(--tl-danger)]">
          {error}
        </p>
      )}

      <div className="mt-5 flex flex-wrap gap-2">
        <button
          type="button"
          className="primary-button"
          onClick={() => setEditing(true)}
        >
          <Pencil size={16} /> {t("knowledge.editMemory")}
        </button>
        <button type="button" className="secondary-button" onClick={copy}>
          <Clipboard size={16} />{" "}
          {copied ? t("action.copied") : t("knowledge.copyGuidance")}
        </button>
        <button
          type="button"
          className="secondary-button"
          disabled
          title={t("knowledge.sourceTaskHint")}
        >
          <ExternalLink size={16} /> {t("knowledge.openSource")}
        </button>
      </div>
    </article>
  );
}
