import { useQuery } from "@tanstack/react-query";
import { Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import {
  getLearningMetrics,
  listCapsules,
  listLearningNotes,
  promoteLearningNote,
  rejectLearningNote,
  updateLearningNote,
  type CapsuleStatus,
  type ExplorationCapsule,
  type Project,
  type UpdateLearningNoteInput,
} from "../lib/api";
import { useI18n } from "../lib/i18n";
import { MemoryCandidates } from "./MemoryCandidates";
import { MemoryDetail } from "./MemoryDetail";
import { MemoryList } from "./MemoryList";
import { MemoryMetrics } from "./MemoryMetrics";

type MemoryTab = "verified" | "pending" | "stale";

export function KnowledgeView({ project }: { project: Project }) {
  const { t } = useI18n();
  const [tab, setTab] = useState<MemoryTab>("verified");
  const [query, setQuery] = useState("");
  const status: CapsuleStatus = tab === "stale" ? "stale" : "active";
  const capsules = useQuery({
    queryKey: ["capsules", project.id, query, status],
    queryFn: () => listCapsules(project.id, query, status),
    enabled: tab !== "pending",
  });
  const notes = useQuery({
    queryKey: ["learning-notes", project.id, "pending"],
    queryFn: () => listLearningNotes(project.id, { status: "pending", limit: 100 }),
    enabled: tab === "pending",
  });
  const metrics = useQuery({
    queryKey: ["learning-metrics", project.id],
    queryFn: () => getLearningMetrics(project.id),
  });
  const ordered = useMemo(
    () =>
      [...(capsules.data ?? [])].sort((left, right) => {
        const leftFeedback = left.helpful_count + left.rejected_count;
        const rightFeedback = right.helpful_count + right.rejected_count;
        const leftRate = leftFeedback ? left.helpful_count / leftFeedback : 0;
        const rightRate = rightFeedback ? right.helpful_count / rightFeedback : 0;
        return rightRate - leftRate || right.use_count - left.use_count || right.updated_at - left.updated_at;
      }),
    [capsules.data]
  );
  const [selected, setSelected] = useState<ExplorationCapsule | null>(null);

  useEffect(() => {
    setSelected((current) => {
      if (current) {
        const refreshed = ordered.find((capsule) => capsule.id === current.id);
        if (refreshed) return refreshed;
      }
      return ordered[0] ?? null;
    });
  }, [ordered]);

  const tabs: { id: MemoryTab; label: string; count: number }[] = [
    { id: "verified", label: t("knowledge.verified"), count: metrics.data?.active_capsule_count ?? 0 },
    { id: "pending", label: t("knowledge.pendingReview"), count: metrics.data?.pending_note_count ?? 0 },
    { id: "stale", label: t("knowledge.needsRevalidation"), count: metrics.data?.stale_count ?? 0 },
  ];

  return (
    <div className="h-full overflow-y-auto p-5 lg:p-7">
      <div className="mx-auto flex max-w-[1440px] flex-col gap-5">
        <header>
          <p className="eyebrow">ENGINEERING MEMORY</p>
          <h1 className="mt-1 text-3xl font-bold tracking-tight">{t("knowledge.title")}</h1>
          <p className="mt-2 max-w-4xl text-base leading-7 text-[var(--tl-ink-muted)]">{t("knowledge.subtitle")}</p>
        </header>

        <MemoryMetrics metrics={metrics.data} />

        <section className="panel p-2">
          <div className="flex gap-1 overflow-x-auto" role="tablist" aria-label={t("knowledge.memoryViews")}>
            {tabs.map((item) => (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={tab === item.id}
                onClick={() => {
                  setTab(item.id);
                  setSelected(null);
                }}
                className={
                  "inline-flex min-h-10 shrink-0 items-center gap-2 rounded-lg px-4 text-sm font-bold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)] " +
                  (tab === item.id
                    ? "bg-[var(--tl-primary)] text-[var(--tl-primary-contrast)]"
                    : "text-[var(--tl-ink-muted)] hover:bg-[var(--tl-bg-quiet)] hover:text-[var(--tl-ink)]")
                }
              >
                {item.label}
                <span className="rounded-full bg-[color-mix(in_srgb,var(--tl-bg)_35%,transparent)] px-2 py-0.5 text-xs">{item.count}</span>
              </button>
            ))}
          </div>
        </section>

        {tab === "pending" ? (
          <section>
            {notes.isLoading ? (
              <Loading text={t("knowledge.candidatesLoading")} />
            ) : notes.isError ? (
              <Loading text={String(notes.error)} error />
            ) : (
              <MemoryCandidates
                notes={notes.data ?? []}
                onUpdate={async (noteID: string, input: UpdateLearningNoteInput) => {
                  await updateLearningNote(noteID, input);
                  await notes.refetch();
                }}
                onPromote={async (noteID, evidence, memoryClass) => {
                  await promoteLearningNote(noteID, evidence, memoryClass);
                  await Promise.all([notes.refetch(), metrics.refetch(), capsules.refetch()]);
                }}
                onReject={async (noteID, reason) => {
                  await rejectLearningNote(noteID, reason);
                  await Promise.all([notes.refetch(), metrics.refetch()]);
                }}
              />
            )}
          </section>
        ) : (
          <>
            <label className="relative block">
              <Search className="absolute left-3 top-3 text-[var(--tl-ink-muted)]" size={18} />
              <input
                aria-label={t("knowledge.search")}
                value={query}
                onChange={(event) => setQuery(event.target.value)}
                placeholder={t("knowledge.search")}
                className="h-11 w-full rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-surface)] pl-10 pr-3 text-[15px] outline-none focus:border-[var(--tl-primary)]"
              />
            </label>
            {capsules.isLoading ? (
              <Loading text={t("knowledge.loading")} />
            ) : capsules.isError ? (
              <Loading text={String(capsules.error)} error />
            ) : (
              <div className="grid items-start gap-5 lg:grid-cols-[minmax(320px,0.85fr)_minmax(0,1.25fr)]">
                <MemoryList capsules={ordered} selectedID={selected?.id ?? null} onSelect={setSelected} />
                <MemoryDetail
                  capsule={selected}
                  capsules={ordered}
                  onUpdated={async (updated) => {
                    setSelected(updated);
                    await Promise.all([capsules.refetch(), metrics.refetch()]);
                  }}
                  onRelationsChanged={async () => {
                    await Promise.all([capsules.refetch(), metrics.refetch()]);
                  }}
                />
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}

function Loading({ text, error = false }: { text: string; error?: boolean }) {
  return (
    <div className={"panel p-10 text-center text-sm " + (error ? "text-[var(--tl-rust)]" : "text-[var(--tl-ink-muted)]")}>
      {text}
    </div>
  );
}
