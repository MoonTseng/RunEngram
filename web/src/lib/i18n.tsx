import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import type { TaskState, TaskType } from "./api";

export type Locale = "en-US" | "zh-CN";

const STORAGE_KEY = "taskline.locale";

const enUS = {
  "actions.add": "Add",
  "actions.adding": "Adding…",
  "actions.cancel": "Cancel",
  "actions.close": "Close",
  "actions.create": "Create",
  "actions.creating": "Creating…",
  "actions.newTask": "+ New",
  "actions.save": "Save",
  "actions.saving": "Saving…",
  "actions.searchTasks": "Search tasks",
  "card.blocked": "Blocked: depends on other tasks not yet done",
  "card.claimedBy": "Claimed by",
  "card.dependencies": "deps",
  "card.dependenciesDone": "Dependencies are done",
  "card.moreLabels": "more labels",
  "card.open": "Open task",
  "card.owner": "Owner",
  "card.priority": "Priority",
  "card.viewHistory": "View history for",
  "claim.claimed": "Claimed",
  "claim.label": "Claim",
  "claim.leaseExpires": "lease expires",
  "claim.notSet": "not set",
  "claim.unknown": "unknown",
  "claim.worked": "worked",
  "claim.working": "working",
  "dependencies.add": "Add dependency",
  "dependencies.addPlaceholder": "add dependency...",
  "dependencies.deleted": "(deleted)",
  "dependencies.empty": "No dependencies.",
  "dependencies.label": "Depends",
  "editor.close": "Close",
  "editor.createDialog": "Create task in",
  "editor.description": "Description",
  "editor.editDialog": "Edit task",
  "editor.editTitle": "Edit task",
  "editor.loading": "Loading editor…",
  "editor.newTitle": "New task in",
  "editor.openMarkdown": "Open markdown editor",
  "editor.priority": "Priority",
  "editor.state": "State",
  "editor.title": "Title",
  "editor.titleRequired": "Title is required.",
  "editor.type": "Type",
  "images.attachment": "Image attachment",
  "images.closePreview": "Close image preview",
  "images.delete": "Delete image",
  "images.empty": "No images attached.",
  "images.label": "Images",
  "images.notImage": "Selected file is not an image.",
  "images.preview": "Image preview",
  "images.unknown": "unknown",
  "images.upload": "Upload image",
  "images.uploading": "Uploading...",
  "images.view": "View image",
  "history.after": "After",
  "history.before": "Before",
  "history.close": "Close task history",
  "history.contentChanged": "Content changed",
  "history.dialog": "History for",
  "history.empty": "No operations recorded.",
  "history.loading": "Loading history...",
  "history.title": "Task history",
  "markdown.back": "Back to task editor",
  "markdown.description": "Markdown description",
  "markdown.descriptionEditor": "Markdown description editor",
  "markdown.document": "Markdown document",
  "markdown.documentEditor": "Markdown document editor",
  "markdown.documentTitle": "Document title",
  "graph.deleteDependency": "Delete dependency",
  "graph.updateFailed": "Task update failed",
  "labels.add": "Add label",
  "labels.common": "Common labels",
  "labels.label": "Labels",
  "labels.max": "Maximum of 20 labels reached",
  "labels.new": "New label",
  "labels.placeholder": "Type a label and press Enter or comma",
  "labels.remove": "Remove label",
  "labels.showCommon": "Show common labels",
  "links.label": "Links",
  "links.optionalLabel": "label (optional)",
  "links.remove": "Remove link",
  "locale.switchToChinese": "切换为中文",
  "locale.switchToEnglish": "切换为 English",
  "menu.clone": "Clone",
  "menu.copyTaskId": "Copy task ID",
  "menu.delete": "Delete",
  "menu.edit": "Edit",
  "menu.taskActions": "Task actions for",
  "docs.add": "Add doc",
  "docs.createFirst": "Create the task before adding docs.",
  "docs.delete": "Delete doc",
  "docs.empty": "No docs attached.",
  "docs.label": "Docs",
  "docs.open": "Open doc",
  "sidebar.close": "Close sidebar",
  "sidebar.collapse": "Collapse sidebar",
  "sidebar.expand": "Expand sidebar",
  "sidebar.createProject": "Create",
  "sidebar.creatingProject": "Creating…",
  "sidebar.descriptionPlaceholder": "description (optional)",
  "sidebar.empty": "No projects yet.",
  "sidebar.failed": "Failed to load projects:",
  "sidebar.loading": "Loading…",
  "sidebar.namePlaceholder": "project name",
  "sidebar.newProject": "+ New",
  "sidebar.projects": "Projects",
  "search.close": "Close search",
  "search.empty": "No matches",
  "search.searching": "Searching...",
  "sort.created": "Created oldest first",
  "sort.execution": "Next execution order",
  "sort.priority": "Priority high to low",
  "sort.updated": "Recently updated",
  "views.board": "Board view",
  "views.graph": "Graph",
  "views.kanban": "Kanban",
  "welcome.cliSync": "The kanban view auto-refreshes every 10 seconds — changes you make from the CLI in another terminal will appear here.",
  "welcome.noProjectPrefix": "No project matches",
  "welcome.noProjectSuffix": "in the URL. Pick another from the sidebar.",
  "welcome.pickProject": "Pick a project from the sidebar, or create one with",
} as const;

export type TranslationKey = keyof typeof enUS;

const zhCN: Record<TranslationKey, string> = {
  "actions.add": "添加",
  "actions.adding": "添加中…",
  "actions.cancel": "取消",
  "actions.close": "关闭",
  "actions.create": "创建",
  "actions.creating": "创建中…",
  "actions.newTask": "+ 新建任务",
  "actions.save": "保存",
  "actions.saving": "保存中…",
  "actions.searchTasks": "搜索任务",
  "card.blocked": "被阻塞：依赖任务尚未完成",
  "card.claimedBy": "领取人",
  "card.dependencies": "依赖",
  "card.dependenciesDone": "依赖任务均已完成",
  "card.moreLabels": "个更多标签",
  "card.open": "打开任务",
  "card.owner": "负责人",
  "card.priority": "优先级",
  "card.viewHistory": "查看任务历史",
  "claim.claimed": "领取于",
  "claim.label": "任务领取",
  "claim.leaseExpires": "租约到期",
  "claim.notSet": "未设置",
  "claim.unknown": "未知",
  "claim.worked": "已工作",
  "claim.working": "工作中",
  "dependencies.add": "添加依赖任务",
  "dependencies.addPlaceholder": "添加依赖任务…",
  "dependencies.deleted": "（已删除）",
  "dependencies.empty": "暂无依赖任务。",
  "dependencies.label": "依赖任务",
  "editor.close": "关闭",
  "editor.createDialog": "在项目中创建任务",
  "editor.description": "描述",
  "editor.editDialog": "编辑任务",
  "editor.editTitle": "编辑任务",
  "editor.loading": "正在加载编辑器…",
  "editor.newTitle": "新建任务 ·",
  "editor.openMarkdown": "打开 Markdown 编辑器",
  "editor.priority": "优先级",
  "editor.state": "状态",
  "editor.title": "标题",
  "editor.titleRequired": "请填写任务标题。",
  "editor.type": "类型",
  "images.attachment": "图片附件",
  "images.closePreview": "关闭图片预览",
  "images.delete": "删除图片",
  "images.empty": "暂无图片。",
  "images.label": "图片",
  "images.notImage": "所选文件不是图片。",
  "images.preview": "图片预览",
  "images.unknown": "未知",
  "images.upload": "上传图片",
  "images.uploading": "上传中…",
  "images.view": "查看图片",
  "history.after": "变更后",
  "history.before": "变更前",
  "history.close": "关闭任务历史",
  "history.contentChanged": "内容已变更",
  "history.dialog": "任务历史",
  "history.empty": "暂无操作记录。",
  "history.loading": "正在加载历史…",
  "history.title": "任务历史",
  "markdown.back": "返回任务编辑器",
  "markdown.description": "Markdown 描述",
  "markdown.descriptionEditor": "Markdown 描述编辑器",
  "markdown.document": "Markdown 文档",
  "markdown.documentEditor": "Markdown 文档编辑器",
  "markdown.documentTitle": "文档标题",
  "graph.deleteDependency": "删除依赖关系",
  "graph.updateFailed": "任务更新失败",
  "labels.add": "添加标签",
  "labels.common": "常用标签",
  "labels.label": "标签",
  "labels.max": "最多可添加 20 个标签",
  "labels.new": "新标签",
  "labels.placeholder": "输入标签后按 Enter 或逗号",
  "labels.remove": "移除标签",
  "labels.showCommon": "显示常用标签",
  "links.label": "链接",
  "links.optionalLabel": "说明（可选）",
  "links.remove": "移除链接",
  "locale.switchToChinese": "切换为中文",
  "locale.switchToEnglish": "切换为 English",
  "menu.clone": "复制任务",
  "menu.copyTaskId": "复制任务 ID",
  "menu.delete": "删除",
  "menu.edit": "编辑",
  "menu.taskActions": "任务操作",
  "docs.add": "添加文档",
  "docs.createFirst": "请先创建任务，再添加文档。",
  "docs.delete": "删除文档",
  "docs.empty": "暂无文档。",
  "docs.label": "文档",
  "docs.open": "打开文档",
  "sidebar.close": "关闭项目栏",
  "sidebar.collapse": "收起项目栏",
  "sidebar.expand": "展开项目栏",
  "sidebar.createProject": "创建",
  "sidebar.creatingProject": "创建中…",
  "sidebar.descriptionPlaceholder": "项目说明（可选）",
  "sidebar.empty": "暂无项目。",
  "sidebar.failed": "项目加载失败：",
  "sidebar.loading": "加载中…",
  "sidebar.namePlaceholder": "项目名称",
  "sidebar.newProject": "+ 新建项目",
  "sidebar.projects": "项目",
  "search.close": "关闭搜索",
  "search.empty": "没有匹配的任务",
  "search.searching": "搜索中…",
  "sort.created": "按创建时间升序",
  "sort.execution": "按下一步执行顺序",
  "sort.priority": "按优先级从高到低",
  "sort.updated": "最近更新优先",
  "views.board": "任务视图",
  "views.graph": "依赖图",
  "views.kanban": "看板",
  "welcome.cliSync": "看板每 10 秒自动刷新；你在另一个终端通过 CLI 做出的变更也会显示在这里。",
  "welcome.noProjectPrefix": "找不到项目",
  "welcome.noProjectSuffix": "。请从左侧项目栏选择其他项目。",
  "welcome.pickProject": "从左侧项目栏选择项目，或点击",
};

const stateLabels: Record<Locale, Record<TaskState, string>> = {
  "en-US": {
    pending: "Pending",
    start: "Start",
    spec: "Spec",
    dev: "Dev",
    test: "Test",
    review: "Review",
    done: "Done",
  },
  "zh-CN": {
    pending: "待规划",
    start: "待开始",
    spec: "方案设计",
    dev: "开发中",
    test: "测试中",
    review: "评审中",
    done: "已完成",
  },
};

const typeLabels: Record<Locale, Record<TaskType, string>> = {
  "en-US": { feature: "feature", bug: "bug", docs: "docs" },
  "zh-CN": { feature: "功能", bug: "缺陷", docs: "文档任务" },
};

const messages: Record<Locale, Record<TranslationKey, string>> = {
  "en-US": enUS,
  "zh-CN": zhCN,
};

export function resolveLocale(language?: string | null): Locale {
  return language?.toLowerCase().startsWith("zh") ? "zh-CN" : "en-US";
}

function browserLocale(): Locale {
  if (typeof window === "undefined") return "en-US";
  const storage =
    typeof window.localStorage?.getItem === "function" ? window.localStorage : null;
  const stored = storage?.getItem(STORAGE_KEY);
  if (stored === "en-US" || stored === "zh-CN") return stored;
  return resolveLocale(window.navigator.language);
}

export function translate(locale: Locale, key: TranslationKey): string {
  return messages[locale][key];
}

type I18nValue = {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  stateLabel: (state: TaskState) => string;
  t: (key: TranslationKey) => string;
  typeLabel: (type: TaskType) => string;
};

const I18nContext = createContext<I18nValue>({
  locale: "en-US",
  setLocale: () => undefined,
  stateLabel: (state) => stateLabels["en-US"][state],
  t: (key) => enUS[key],
  typeLabel: (type) => typeLabels["en-US"][type],
});

export function I18nProvider({ children }: { children: ReactNode }) {
  const [locale, setLocaleState] = useState<Locale>(browserLocale);

  const setLocale = (next: Locale) => {
    if (typeof window.localStorage?.setItem === "function") {
      window.localStorage.setItem(STORAGE_KEY, next);
    }
    setLocaleState(next);
  };

  useEffect(() => {
    document.documentElement.lang = locale;
  }, [locale]);

  const value = useMemo<I18nValue>(
    () => ({
      locale,
      setLocale,
      stateLabel: (state) => stateLabels[locale][state],
      t: (key) => translate(locale, key),
      typeLabel: (type) => typeLabels[locale][type],
    }),
    [locale]
  );

  return <I18nContext.Provider value={value}>{children}</I18nContext.Provider>;
}

export function useI18n(): I18nValue {
  return useContext(I18nContext);
}
