import { useEffect, useState } from "react";
import { Moon, PanelLeftClose, PanelLeftOpen, Search, Sun } from "lucide-react";
import { useQueryState } from "nuqs";
import { Sidebar } from "./components/Sidebar";
import { KanbanBoard } from "./components/KanbanBoard";
import { GraphView } from "./components/GraphView";
import { KnowledgeView } from "./components/KnowledgeView";
import { ActionConsole } from "./components/ActionConsole";
import { WorkspaceNav, type WorkspaceView } from "./components/WorkspaceNav";
import { CreateTaskButton } from "./components/CreateTaskButton";
import { TaskEditor } from "./components/TaskEditor";
import { TaskSearchDialog } from "./components/TaskSearchDialog";
import { useProjects, useTasks } from "./hooks/queries";
import type { Project, Task } from "./lib/api";
import { I18nProvider, useI18n } from "./lib/i18n";
import { useTheme } from "./lib/theme";

type View = WorkspaceView;

export default function App() {
  return (
    <I18nProvider>
      <TasklineApp />
    </I18nProvider>
  );
}

function TasklineApp() {
  const { t } = useI18n();
  const [sidebarOpen, setSidebarOpen] = useState(true);
  // ?project=<name|id> survives page reload and back/forward; nuqs
  // keeps the URL and state in lockstep without a router dep.
  // history: "replace" so picking a project doesn't pollute the back
  // stack — users browser-back to leave the app, not to step through
  // every sidebar selection. nuqs defaults to "push".
  const [projectKey, setProjectKey] = useQueryState("project", {
    history: "replace",
  });
  const [viewKey, setViewKey] = useQueryState("view", {
    history: "replace",
  });
  const compactShell = useMediaQuery("(max-width: 639px)");
  const view = parseViewKey(viewKey);
  const projects = useProjects();
  const project: Project | null =
    projects.data?.find(
      (p) => p.name === projectKey || p.id === projectKey
    ) ?? null;
  const projectId = project?.id ?? null;
  const hasProject = projectId !== null;

  // Prefer the human-readable name in the URL; the resolver above also
  // accepts an id, so older saved links keep working.
  const selectProject = (p: Project) => {
    void setProjectKey(p.name);
    if (compactShell) setSidebarOpen(false);
  };
  const selectView = (next: View) => {
    void setViewKey(next === "action" ? null : next);
  };

  useEffect(() => {
    if (!hasProject) {
      setSidebarOpen(true);
      return;
    }
    setSidebarOpen(!compactShell);
  }, [compactShell, hasProject, projectId]);

  const sidebar = (
    <Sidebar
      selectedId={project?.id ?? null}
      onSelect={selectProject}
      className={
        compactShell && hasProject
          ? "h-full w-72 max-w-[82vw] p-4 shadow-[var(--tl-shadow-lift)]"
          : "h-full w-64 p-4"
      }
    />
  );
  const showDesktopSidebar = sidebarOpen || !hasProject;
  const showMobileSidebar = compactShell && hasProject && sidebarOpen;
  const SidebarIcon = sidebarOpen ? PanelLeftClose : PanelLeftOpen;

  return (
    <div className="taskline-theme relative h-screen w-screen overflow-hidden flex bg-[var(--tl-bg)] text-[var(--tl-ink)]">
      {!compactShell && (
        <div
          aria-hidden={showDesktopSidebar ? undefined : true}
          className={
            "shrink-0 overflow-hidden transition-[width] duration-300 ease-out " +
            (showDesktopSidebar ? "w-64" : "w-0 pointer-events-none")
          }
        >
          {sidebar}
        </div>
      )}
      {hasProject && !compactShell && (
        <button
          type="button"
          aria-expanded={sidebarOpen}
          aria-label={sidebarOpen ? t("sidebar.collapse") : t("sidebar.expand")}
          title={sidebarOpen ? t("sidebar.collapse") : t("sidebar.expand")}
          onClick={() => setSidebarOpen((open) => !open)}
          className={
            "absolute top-4 z-30 hidden h-8 w-8 items-center justify-center rounded-r-md border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] text-[var(--tl-ink-muted)] shadow-[var(--tl-shadow-paper)] transition-[left,background-color,border-color,box-shadow,color] duration-300 ease-out hover:border-[var(--tl-outline-strong)] hover:bg-[var(--tl-bg-quiet)] hover:text-[var(--tl-ink)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)] sm:inline-flex " +
            (sidebarOpen ? "left-64" : "left-0")
          }
        >
          <SidebarIcon size={16} aria-hidden="true" />
        </button>
      )}
      {showMobileSidebar && (
        <div className="fixed inset-0 z-50 flex sm:hidden">
          <button
            type="button"
            aria-label={t("sidebar.close")}
            className="absolute inset-0 bg-[rgba(37,34,29,0.34)]"
            onClick={() => setSidebarOpen(false)}
          />
          <div className="relative z-10 h-full">{sidebar}</div>
        </div>
      )}
      <main
        data-visual-style="wabi-sabi"
        className="min-w-0 flex-1 flex flex-col overflow-hidden bg-[var(--tl-bg)]"
      >
        {project ? (
          <ProjectWorkspace
            key={project.id}
            project={project}
            view={view}
            onViewChange={selectView}
            sidebarOpen={sidebarOpen}
            showHeaderSidebarToggle={compactShell}
            onToggleSidebar={() => setSidebarOpen((open) => !open)}
          />
        ) : (
          <Welcome
            unresolved={!!projectKey && projects.isSuccess && !project}
            keyValue={projectKey}
            loading={projects.isLoading}
          />
        )}
      </main>
    </div>
  );
}

function ProjectWorkspace({
  project,
  view,
  onViewChange,
  sidebarOpen,
  showHeaderSidebarToggle,
  onToggleSidebar,
}: {
  project: Project;
  view: View;
  onViewChange: (next: View) => void;
  sidebarOpen: boolean;
  showHeaderSidebarToggle: boolean;
  onToggleSidebar: () => void;
}) {
  const { locale, setLocale, t } = useI18n();
  const { theme, toggleTheme } = useTheme();
  const [searchOpen, setSearchOpen] = useState(false);
  const [editingTask, setEditingTask] = useState<Task | null>(null);
  const tasksQ = useTasks(project.id);
  const tasks = tasksQ.data ?? [];
  const SidebarIcon = sidebarOpen ? PanelLeftClose : PanelLeftOpen;

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const cmd = event.metaKey || event.ctrlKey;
      if (cmd && event.key.toLowerCase() === "p") {
        event.preventDefault();
        setSearchOpen(true);
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  return (
    <>
      <header
        className={
          "flex shrink-0 flex-wrap items-center justify-between gap-2 border-b border-[var(--tl-outline)] bg-[var(--tl-surface)] py-3 pr-6 shadow-[0_1px_0_rgba(255,255,255,0.55)] max-sm:px-3 max-sm:py-2 sm:gap-4 " +
          (showHeaderSidebarToggle ? "pl-6" : "pl-10")
        }
      >
        <div className="flex min-w-0 flex-1 items-center gap-3 max-sm:basis-full">
          {showHeaderSidebarToggle && (
            <button
              type="button"
              aria-expanded={sidebarOpen}
              aria-label={sidebarOpen ? t("sidebar.collapse") : t("sidebar.expand")}
              title={sidebarOpen ? t("sidebar.collapse") : t("sidebar.expand")}
              onClick={onToggleSidebar}
              className="inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-md border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] text-[var(--tl-ink-muted)] shadow-[var(--tl-shadow-paper)] transition hover:border-[var(--tl-outline-strong)] hover:bg-[var(--tl-bg-quiet)] hover:text-[var(--tl-ink)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)]"
            >
              <SidebarIcon size={16} aria-hidden="true" />
            </button>
          )}
          <div className="min-w-0">
            <h2 className="truncate text-lg font-bold leading-tight text-[var(--tl-ink)]">
              {project.name}
            </h2>
            {project.description && (
              <p className="mt-0.5 truncate text-xs text-[var(--tl-ink-muted)]">{project.description}</p>
            )}
          </div>
        </div>
        <div className="flex shrink-0 flex-wrap items-stretch justify-end gap-1.5">
          <WorkspaceNav view={view} onChange={onViewChange} />
          <button
            type="button"
            aria-label={
              locale === "zh-CN"
                ? t("locale.switchToEnglish")
                : t("locale.switchToChinese")
            }
            title={
              locale === "zh-CN"
                ? t("locale.switchToEnglish")
                : t("locale.switchToChinese")
            }
            onClick={() => setLocale(locale === "zh-CN" ? "en-US" : "zh-CN")}
            className="inline-flex h-8 min-w-8 items-center justify-center rounded-md border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] px-2 text-xs font-medium text-[var(--tl-ink-muted)] shadow-[var(--tl-shadow-paper)] transition hover:border-[var(--tl-outline-strong)] hover:bg-[var(--tl-bg-quiet)] hover:text-[var(--tl-ink)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)]"
          >
            {locale === "zh-CN" ? "EN" : "中文"}
          </button>
          <button
            type="button"
            aria-label={theme === "dracula" ? t("theme.switchToPaper") : t("theme.switchToDracula")}
            title={theme === "dracula" ? t("theme.switchToPaper") : t("theme.switchToDracula")}
            onClick={toggleTheme}
            className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] text-[var(--tl-ink-muted)] shadow-[var(--tl-shadow-paper)] transition hover:border-[var(--tl-outline-strong)] hover:text-[var(--tl-ink)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)]"
          >
            {theme === "dracula" ? <Sun size={16} /> : <Moon size={16} />}
          </button>
          <button
            type="button"
            aria-label={t("actions.searchTasks")}
            title={t("actions.searchTasks")}
            onClick={() => setSearchOpen(true)}
            className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-[var(--tl-outline)] bg-[var(--tl-surface-raised)] text-[var(--tl-ink-muted)] shadow-[var(--tl-shadow-paper)] transition hover:border-[var(--tl-outline-strong)] hover:bg-[var(--tl-bg-quiet)] hover:text-[var(--tl-ink)] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-[var(--tl-focus)]"
          >
            <Search size={16} aria-hidden="true" />
          </button>
          <CreateTaskButton project={project} allTasks={tasks} />
        </div>
      </header>
      <section className="relative flex-1 overflow-hidden bg-[var(--tl-bg)]">
        <div className="box-border h-full">
          {view === "action" && (
            <ActionConsole
              project={project}
              tasks={tasks}
              loading={tasksQ.isLoading}
              error={tasksQ.error}
              onNavigate={onViewChange}
            />
          )}
          {view === "kanban" && <KanbanBoard project={project} />}
          {view === "graph" && <GraphView project={project} />}
          {view === "knowledge" && <KnowledgeView project={project} />}
        </div>
      </section>
      {searchOpen && (
        <TaskSearchDialog
          project={project}
          onClose={() => setSearchOpen(false)}
          onSelect={(task) => {
            setSearchOpen(false);
            setEditingTask(task);
          }}
        />
      )}
      {editingTask && (
        <TaskEditor
          project={project}
          task={editingTask}
          allTasks={tasks}
          onClose={() => setEditingTask(null)}
        />
      )}
    </>
  );
}

function parseViewKey(value: string | null): View {
  return value === "kanban" || value === "graph" || value === "knowledge"
    ? value
    : "action";
}

function useMediaQuery(query: string) {
  const [matches, setMatches] = useState(() =>
    typeof window !== "undefined" && typeof window.matchMedia === "function"
      ? window.matchMedia(query).matches
      : false
  );

  useEffect(() => {
    if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
      setMatches(false);
      return;
    }
    const media = window.matchMedia(query);
    setMatches(media.matches);
    const onChange = (event: MediaQueryListEvent) => setMatches(event.matches);
    media.addEventListener("change", onChange);
    return () => media.removeEventListener("change", onChange);
  }, [query]);

  return matches;
}

function Welcome({
  unresolved,
  keyValue,
  loading,
}: {
  unresolved: boolean;
  keyValue: string | null;
  loading: boolean;
}) {
  const { t } = useI18n();
  return (
    <div className="flex-1 overflow-y-auto bg-[var(--tl-bg)] p-6 text-[var(--tl-ink-muted)]">
      <div className="mx-auto flex min-h-full max-w-4xl items-center justify-center">
      <div className="panel w-full p-7 lg:p-10">
        <p className="eyebrow">INTERNAL ALPHA</p>
        <h2 className="mt-3 text-3xl font-bold text-[var(--tl-ink)]">RunEngram</h2>
        <p className="mt-3 max-w-2xl text-lg leading-8">{t("welcome.purpose")}</p>
        {unresolved && keyValue && (
          <p className="mt-4 rounded-lg bg-[var(--tl-ochre-soft)] p-3 text-sm text-[var(--tl-ochre)]">
            {t("welcome.noProjectPrefix")} <code className="font-mono">{keyValue}</code>{" "}
            {t("welcome.noProjectSuffix")}
          </p>
        )}
        <div className="mt-7 grid gap-4 md:grid-cols-3">
          {[t("welcome.stepProject"), t("welcome.stepInstall"), t("welcome.stepRun")].map((step, index) => (
            <div key={step} className="rounded-lg border border-[var(--tl-outline)] bg-[var(--tl-bg-quiet)] p-4">
              <span className="text-xs font-bold text-[var(--tl-primary)]">0{index + 1}</span>
              <p className="mt-2 text-sm leading-6 text-[var(--tl-ink)]">{step}</p>
            </div>
          ))}
        </div>
        <div className="mt-6 flex flex-wrap gap-4 text-sm font-semibold">
          <a className="text-[var(--tl-primary)] hover:underline" href="https://github.com/MoonTseng/RunEngram#readme" target="_blank" rel="noreferrer">README</a>
          <a className="text-[var(--tl-primary)] hover:underline" href="https://github.com/MoonTseng/RunEngram/blob/main/%E4%BD%BF%E7%94%A8%E8%AF%B4%E6%98%8E.md" target="_blank" rel="noreferrer">{t("welcome.guide")}</a>
          <a className="text-[var(--tl-primary)] hover:underline" href="https://github.com/MoonTseng/RunEngram/issues" target="_blank" rel="noreferrer">{t("welcome.feedback")}</a>
        </div>
        <p className="mt-7 text-sm text-[var(--tl-ink-faint)]">
          {loading ? t("sidebar.loading") : t("welcome.localBoundary")}
        </p>
      </div>
      </div>
    </div>
  );
}
