import { BrainCircuit, GitFork, LayoutDashboard, ListTodo } from "lucide-react";
import { useI18n } from "../lib/i18n";

export type WorkspaceView = "action" | "kanban" | "graph" | "knowledge";

export function WorkspaceNav({
  view,
  onChange,
}: {
  view: WorkspaceView;
  onChange: (view: WorkspaceView) => void;
}) {
  const { t } = useI18n();
  const items = [
    { id: "action" as const, label: t("views.action"), icon: LayoutDashboard },
    { id: "kanban" as const, label: t("views.kanban"), icon: ListTodo },
    { id: "graph" as const, label: t("views.graph"), icon: GitFork },
    { id: "knowledge" as const, label: t("views.knowledge"), icon: BrainCircuit },
  ];

  return (
    <nav
      aria-label={t("views.workspace")}
      className="flex min-w-0 items-center gap-1 overflow-x-auto rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-1"
    >
      {items.map(({ id, label, icon: Icon }) => (
        <button
          key={id}
          type="button"
          aria-current={view === id ? "page" : undefined}
          onClick={() => onChange(id)}
          className={
            "inline-flex h-9 shrink-0 items-center gap-1.5 rounded-md px-3 text-sm font-semibold transition focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)] " +
            (view === id
              ? "bg-[var(--tl-primary)] text-[var(--tl-primary-contrast)] shadow-[var(--tl-shadow-card)]"
              : "text-[var(--tl-ink-muted)] hover:bg-[var(--tl-surface-raised)] hover:text-[var(--tl-ink)]")
          }
        >
          <Icon size={15} aria-hidden="true" />
          <span>{label}</span>
        </button>
      ))}
    </nav>
  );
}
