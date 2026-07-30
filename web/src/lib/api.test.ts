import { afterEach, describe, expect, it, vi } from "vitest";
import {
  createTask,
  createTaskDoc,
  deleteTaskDoc,
  deleteTaskImage,
  getTask,
  getTaskDoc,
  getTaskContext,
  listLearningNotes,
  listCapsuleMemoryImpacts,
  listTaskMemoryImpacts,
  listTaskEvents,
  searchTasks,
  STATE_LABELS,
  STATES,
  taskDocContentURL,
  taskImageURL,
  updateTask,
  updateMemoryImpact,
  updateTaskDoc,
  uploadTaskImage,
  type TaskDoc,
  type TaskImage,
  type UpdateMemoryImpactInput,
} from "./api";

describe("memory impact helpers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads capsule and task history with encoded ids", async () => {
    const fetchMock = vi.fn().mockImplementation(async () =>
      new Response("[]", {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await listCapsuleMemoryImpacts("capsule/one");
    await listTaskMemoryImpacts("task/one");

    expect(fetchMock.mock.calls.map(([url]) => url)).toEqual([
      "/api/v1/capsules/capsule%2Fone/memory-impacts?limit=100",
      "/api/v1/tasks/task%2Fone/memory-impacts?limit=100",
    ]);
  });

  it("updates a receipt with optimistic concurrency and evidence", async () => {
    const input: UpdateMemoryImpactInput = {
      state: "helpful" as const,
      stage: "test",
      notes: "Rule prevented unsupported build.",
      evidence: [
        {
          kind: "task-doc",
          ref: "doc:test-report",
          summary: "Report confirms no Gradle command ran.",
        },
      ],
      expected_updated_at: 42,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ id: "impact-1", ...input }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await updateMemoryImpact("impact/one", input);

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/memory-impacts/impact%2Fone",
      expect.objectContaining({ method: "PATCH", body: JSON.stringify(input) })
    );
  });
});

describe("task states", () => {
  it("includes the local test stage between dev and review", () => {
    expect(STATES).toEqual([
      "pending",
      "start",
      "spec",
      "dev",
      "test",
      "review",
      "done",
    ]);
    expect(STATE_LABELS.test).toBe("Test");
  });
});

describe("task labels helpers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends labels when creating and updating tasks", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: "task-1", labels: ["backend", "ui"] }), {
          status: 201,
          headers: { "Content-Type": "application/json" },
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ id: "task-1", labels: ["review"] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        })
      );
    vi.stubGlobal("fetch", fetchMock);

    await createTask("taskline", {
      title: "Labeled task",
      type: "feature",
      priority: 0,
      labels: ["backend", "ui"],
    });
    await updateTask("task-1", { labels: ["review"] });

    expect(fetchMock.mock.calls[0][1]?.body).toBe(
      JSON.stringify({
        title: "Labeled task",
        type: "feature",
        priority: 0,
        labels: ["backend", "ui"],
      })
    );
    expect(fetchMock.mock.calls[1][1]?.body).toBe(
      JSON.stringify({ labels: ["review"] })
    );
  });
});

describe("searchTasks", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("searches a project with encoded query and limit", async () => {
    const found = {
      id: "fc7a0732-0000-4000-8000-000000000000",
      project_id: "project-1",
      title: "Found task",
      description: "",
      type: "feature",
      state: "start",
      priority: 0,
      labels: [],
      created_at: 1780051741142,
      updated_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ tasks: [found] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await searchTasks("taskline", "fc7a0732 hooks", 7);

    expect(result).toEqual([found]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects/taskline/tasks/search?q=fc7a0732+hooks&limit=7",
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("listTaskEvents", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads task history and identifies the Web client", async () => {
    const event = {
      id: "event-1",
      task_id: "task/one",
      actor: "web",
      action: "updated",
      summary: "Updated title",
      details: {},
      created_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ events: [event] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(listTaskEvents("task/one")).resolves.toEqual([event]);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone/events",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({ "X-Taskline-Client": "web" }),
      })
    );
  });
});

describe("getTask", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads an attached source task with an encoded task id", async () => {
    const task = {
      id: "task/one",
      project_id: "project-1",
      title: "Verify dependency placement",
      description: "",
      type: "refactor",
      state: "done",
      priority: 1,
      labels: [],
      docs: [],
      links: [],
      created_at: 1780051741142,
      updated_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(task), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getTask("task/one")).resolves.toEqual(task);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone",
      expect.objectContaining({
        method: "GET",
        headers: expect.objectContaining({ "X-Taskline-Client": "web" }),
      })
    );
  });
});

describe("getTaskContext", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads the immutable context snapshot with an encoded task id", async () => {
    const snapshot = {
      id: "snapshot-1",
      task_id: "task/one",
      project_id: "project-1",
      task: { id: "task/one" },
      suggested_capsules: [],
      created_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(snapshot), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(getTaskContext("task/one")).resolves.toEqual(snapshot);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone/context",
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("listLearningNotes", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("loads filtered candidates for a project", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ learning_notes: [] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    await listLearningNotes("project-1", { status: "pending", limit: 100 });

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/projects/project-1/learning-notes?status=pending&limit=100",
      expect.objectContaining({ method: "GET" })
    );
  });
});

describe("uploadTaskImage", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("posts the file as multipart form data", async () => {
    const uploaded: TaskImage = {
      id: "image-1",
      task_id: "task/one",
      filename: "diagram.png",
      mime_type: "image/png",
      size_bytes: 7,
      uploaded_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(uploaded), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const file = new File(["pngbits"], "diagram.png", { type: "image/png" });
    const result = await uploadTaskImage("task/one", file);

    expect(result).toEqual(uploaded);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone/images",
      expect.objectContaining({ method: "POST", body: expect.any(FormData) })
    );
    const init = fetchMock.mock.calls[0][1] as RequestInit;
    expect(init.headers).toEqual({ "X-Taskline-Client": "web" });
    expect((init.body as FormData).get("file")).toBe(file);
  });
});

describe("task image content helpers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("builds an encoded URL for image previews", () => {
    expect(taskImageURL("image/one")).toBe("/api/v1/images/image%2Fone");
  });

  it("deletes an image by id", async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    await deleteTaskImage("image/one");

    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/images/image%2Fone",
      expect.objectContaining({ method: "DELETE" })
    );
  });
});

describe("task docs helpers", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("creates a markdown doc with JSON content", async () => {
    const created: TaskDoc = {
      id: "doc-1",
      task_id: "task/one",
      title: "Spec",
      url: "/api/v1/docs/doc-1/content",
      content: "# Spec",
      created_at: 1780051741142,
      updated_at: 1780051741142,
    };
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify(created), {
        status: 201,
        headers: { "Content-Type": "application/json" },
      })
    );
    vi.stubGlobal("fetch", fetchMock);

    const result = await createTaskDoc("task/one", { title: "Spec", content: "# Spec" });

    expect(result).toEqual(created);
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/tasks/task%2Fone/docs",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ title: "Spec", content: "# Spec" }),
      })
    );
  });

  it("gets, updates, deletes, and builds raw content URLs for docs", async () => {
    const doc: TaskDoc = {
      id: "doc/one",
      task_id: "task-1",
      title: "Test report",
      url: "/api/v1/docs/doc%2Fone/content",
      content: "# Tests",
      created_at: 1780051741142,
      updated_at: 1780051741143,
    };
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(JSON.stringify(doc), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        })
      )
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ...doc, title: "Updated" }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        })
      )
      .mockResolvedValueOnce(new Response(null, { status: 204 }));
    vi.stubGlobal("fetch", fetchMock);

    expect(taskDocContentURL("doc/one")).toBe("/api/v1/docs/doc%2Fone/content");
    expect(await getTaskDoc("doc/one")).toEqual(doc);
    await updateTaskDoc("doc/one", { title: "Updated", content: "# Tests" });
    await deleteTaskDoc("doc/one");

    expect(fetchMock.mock.calls.map(([url, init]) => [url, init?.method])).toEqual([
      ["/api/v1/docs/doc%2Fone", "GET"],
      ["/api/v1/docs/doc%2Fone", "PATCH"],
      ["/api/v1/docs/doc%2Fone", "DELETE"],
    ]);
  });
});
