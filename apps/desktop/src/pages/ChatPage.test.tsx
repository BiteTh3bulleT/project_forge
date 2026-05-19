import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ChatPage } from "./ChatPage";

type StoredMessage = {
  id: number;
  threadId: number;
  role: string;
  content: string;
  createdAtMs: number;
  metadata: Record<string, unknown>;
};

type StoredThread = {
  id: number;
  title: string;
  createdAtMs: number;
  updatedAtMs: number;
  messages: StoredMessage[];
};

const apiMocks = vi.hoisted(() => {
  let nextThreadId = 1;
  let nextMessageId = 1;
  let clock = 1_800_000_000_000;
  let threads: StoredThread[] = [];
  let observations: Array<{ summary: string; rawContent: string }> = [];

  function now() {
    clock += 1_000;
    return clock;
  }

  function summarizeThread(thread: StoredThread) {
    const { messages: _messages, ...summary } = thread;
    return summary;
  }

  function findThread(id: number) {
    const thread = threads.find((item) => item.id === id);
    if (!thread) throw new Error(`thread ${id} not found`);
    return thread;
  }

  function appendMessage(
    thread: StoredThread,
    role: string,
    content: string,
    metadata: Record<string, unknown> = {},
  ) {
    const message = {
      id: nextMessageId++,
      threadId: thread.id,
      role,
      content,
      createdAtMs: now(),
      metadata,
    };
    thread.messages.push(message);
    thread.updatedAtMs = message.createdAtMs;
    return message;
  }

  return {
    reset() {
      nextThreadId = 1;
      nextMessageId = 1;
      clock = 1_800_000_000_000;
      threads = [];
      observations = [];
    },
    createObservation: vi.fn(async (body: Record<string, unknown>) => {
      const observation = {
        id: observations.length + 1,
        summary: String(body.summary ?? ""),
        rawContent: String(body.rawContent ?? ""),
      };
      observations.push(observation);
      return { observation };
    }),
    listThreads: vi.fn(async () => ({ threads: threads.map(summarizeThread) })),
    createThread: vi.fn(async (body: { title?: string }) => {
      const createdAtMs = now();
      const thread = {
        id: nextThreadId++,
        title: body.title?.trim() || "New chat",
        createdAtMs,
        updatedAtMs: createdAtMs,
        messages: [],
      };
      threads.unshift(thread);
      return { thread: summarizeThread(thread) };
    }),
    getThread: vi.fn(async (id: number) => {
      const thread = findThread(id);
      return { ...summarizeThread(thread), messages: [...thread.messages] };
    }),
    postMessage: vi.fn(
      async (
        id: number,
        body: {
          content: string;
          requestAssistant?: boolean;
          assistantDryRun?: boolean;
        },
      ) => {
        const thread = findThread(id);
        const userMessage = appendMessage(thread, "user", body.content);
        let assistantMessage: StoredMessage | null = null;
        if (body.requestAssistant !== false && !body.assistantDryRun) {
          const memoryText = observations
            .map((item) => item.summary || item.rawContent)
            .filter(Boolean)
            .join(" ");
          assistantMessage = appendMessage(
            thread,
            "assistant",
            memoryText
              ? `I found this remembered note: ${memoryText}`
              : "I do not have a remembered note for that request.",
            { replyToUserMessageId: userMessage.id },
          );
        }
        return { userMessage, assistantMessage };
      },
    ),
    updateThread: vi.fn(),
    deleteThread: vi.fn(),
    uploadAttachment: vi.fn(),
    assistantStream: vi.fn(),
    queueJob: vi.fn(),
    jobTemplates: vi.fn(async () => ({ templates: [] })),
    listModels: vi.fn(async () => ({ models: [] })),
  };
});

const desktopMocks = vi.hoisted(() => ({
  isTauriDesktop: vi.fn(() => false),
  launchOperatorApp: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    chat: {
      threads: {
        list: apiMocks.listThreads,
        create: apiMocks.createThread,
        get: apiMocks.getThread,
        update: apiMocks.updateThread,
        delete: apiMocks.deleteThread,
        postMessage: apiMocks.postMessage,
        uploadAttachment: apiMocks.uploadAttachment,
        assistantStream: apiMocks.assistantStream,
        queueJob: apiMocks.queueJob,
      },
    },
    jobs: {
      templates: apiMocks.jobTemplates,
    },
    memory: {
      createObservation: apiMocks.createObservation,
    },
    modelRuntime: {
      list: apiMocks.listModels,
    },
  },
}));

vi.mock("../lib/desktop", () => ({
  isTauriDesktop: desktopMocks.isTauriDesktop,
  launchOperatorApp: desktopMocks.launchOperatorApp,
}));

describe("ChatPage cross-session memory recall", () => {
  beforeEach(() => {
    apiMocks.reset();
    desktopMocks.isTauriDesktop.mockReturnValue(false);
    desktopMocks.launchOperatorApp.mockReset();
    desktopMocks.launchOperatorApp.mockResolvedValue({
      appId: "files",
      label: "Files",
      executable: "pcmanfm",
      launched: true,
      message: "Files launch requested",
    });
    vi.clearAllMocks();
    window.localStorage.clear();
  });

  it("asks from the remounted chat surface and recalls an API-committed memory observation", async () => {
    await apiMocks.createObservation({
      type: "decision",
      summary: "Cross-session recall marker: remember the basalt notebook.",
      rawContent: "The basalt notebook belongs in reopened chat memory.",
      originKind: "test",
      originId: "desktop-cross-session-memory",
    });

    const firstSession = render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    fireEvent.click(await screen.findByRole("button", { name: "New chat" }));
    expect(await screen.findByText("Ready for the first message")).toBeTruthy();
    expect(
      screen.getByRole("log", { name: "Chat messages" }),
    ).toBeTruthy();

    firstSession.unmount();

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    const composer = await screen.findByLabelText("Chat message");
    fireEvent.change(composer, {
      target: { value: "What should you remember after reopen?" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Send chat message" }));

    await screen.findByText(/What should you remember after reopen\?/);
    expect(
      await screen.findByText(/remember the basalt notebook/i),
    ).toBeTruthy();
    await waitFor(() => {
      expect(apiMocks.postMessage).toHaveBeenCalledWith(
        1,
        expect.objectContaining({
          content: "What should you remember after reopen?",
          requestAssistant: true,
          stream: true,
        }),
      );
    });
  });

  it("unlocks the composer after stopping a streaming assistant reply", async () => {
    const thread = await apiMocks.createThread({ title: "Streaming chat" });
    apiMocks.postMessage.mockResolvedValueOnce({
      userMessage: {
        id: 1,
        threadId: thread.thread.id,
        role: "user",
        content: "stream this",
        createdAtMs: 1_800_000_001_000,
        metadata: {},
      },
      assistantMessage: null,
      assistantPending: true,
      userMessageId: 1,
      stream: true,
    } as unknown as Awaited<ReturnType<typeof apiMocks.postMessage>>);
    apiMocks.assistantStream.mockImplementationOnce(
      async (
        _threadId: number,
        _userMessageId: number,
        _onEvent: unknown,
        signal?: AbortSignal,
      ) =>
        new Promise<void>((_resolve, reject) => {
          signal?.addEventListener("abort", () => {
            reject(new DOMException("Aborted", "AbortError"));
          });
        }),
    );

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    const composer = await screen.findByLabelText("Chat message");
    fireEvent.change(composer, { target: { value: "stream this" } });
    fireEvent.click(
      screen.getByRole("button", { name: "Send chat message" }),
    );

    fireEvent.click(await screen.findByRole("button", { name: "Stop" }));

    await waitFor(() => {
      expect(
        (screen.getByLabelText("Chat message") as HTMLTextAreaElement)
          .disabled,
      ).toBe(false);
    });
  });

  it("renders provider reasoning stream events only in the Reasoning inspector tab", async () => {
    const thread = await apiMocks.createThread({ title: "Reasoning stream" });
    apiMocks.postMessage.mockResolvedValueOnce({
      userMessage: {
        id: 1,
        threadId: thread.thread.id,
        role: "user",
        content: "stream with reasoning",
        createdAtMs: 1_800_000_001_000,
        metadata: {},
      },
      assistantMessage: null,
      assistantPending: true,
      userMessageId: 1,
      stream: true,
    } as unknown as Awaited<ReturnType<typeof apiMocks.postMessage>>);
    apiMocks.assistantStream.mockImplementationOnce(
      async (
        _threadId: number,
        _userMessageId: number,
        onEvent: unknown,
      ) => {
        const emit = onEvent as (event: { event: string; data: string }) => void;
        emit({
          event: "reasoning",
          data: JSON.stringify({
            text: "Provider private chain summary should stay isolated.",
          }),
        });
        emit({
          event: "token",
          data: JSON.stringify({ text: "Final answer only." }),
        });
        emit({
          event: "done",
          data: JSON.stringify({
            assistantMessage: {
              id: 2,
              threadId: thread.thread.id,
              role: "assistant",
              content: "Final answer only.",
              createdAtMs: 1_800_000_002_000,
              metadata: { replyToUserMessageId: 1 },
            },
          }),
        });
      },
    );

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    const composer = await screen.findByLabelText("Chat message");
    fireEvent.change(composer, {
      target: { value: "stream with reasoning" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Send chat message" }),
    );

    expect(await screen.findByText("Final answer only.")).toBeTruthy();
    const chatLog = screen.getByRole("log", { name: "Chat messages" });
    expect(chatLog.textContent ?? "").not.toContain(
      "Provider private chain summary should stay isolated.",
    );

    fireEvent.click(screen.getByRole("button", { name: "Reasoning" }));

    expect(
      await screen.findByText(
        "Provider private chain summary should stay isolated.",
      ),
    ).toBeTruthy();
  });

  it("shows an empty provider reasoning state when no reasoning has streamed", async () => {
    await apiMocks.createThread({ title: "No reasoning yet" });

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    await waitFor(() => {
      expect(apiMocks.listModels).toHaveBeenCalled();
    });

    fireEvent.click(await screen.findByRole("button", { name: "Reasoning" }));

    expect(
      screen.getByText("No provider reasoning has streamed in this thread yet."),
    ).toBeTruthy();
  });

  it("launches the local Files operator app for a file explorer chat request", async () => {
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    await apiMocks.createThread({ title: "Local desktop launch" });

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    const composer = await screen.findByLabelText("Chat message");
    fireEvent.change(composer, {
      target: { value: "Can you open file expolorer for me please?" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Send chat message" }),
    );

    await waitFor(() => {
      expect(desktopMocks.launchOperatorApp).toHaveBeenCalledWith("files");
    });
    expect(apiMocks.postMessage).toHaveBeenCalledWith(
      1,
      expect.objectContaining({
        content: "Can you open file expolorer for me please?",
        requestAssistant: false,
      }),
    );
  });

  it("retries the previous local Files operator app request", async () => {
    desktopMocks.isTauriDesktop.mockReturnValue(true);
    await apiMocks.createThread({ title: "Local desktop retry" });

    render(
      <MemoryRouter initialEntries={["/chat"]}>
        <ChatPage />
      </MemoryRouter>,
    );

    const composer = await screen.findByLabelText("Chat message");
    fireEvent.change(composer, {
      target: { value: "Open file explorer please" },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Send chat message" }),
    );

    await waitFor(() => {
      expect(desktopMocks.launchOperatorApp).toHaveBeenCalledTimes(1);
    });

    fireEvent.change(screen.getByLabelText("Chat message"), {
      target: { value: "Try again." },
    });
    fireEvent.click(
      screen.getByRole("button", { name: "Send chat message" }),
    );

    await waitFor(() => {
      expect(desktopMocks.launchOperatorApp).toHaveBeenCalledTimes(2);
    });
    expect(desktopMocks.launchOperatorApp).toHaveBeenLastCalledWith("files");
    expect(apiMocks.postMessage).toHaveBeenLastCalledWith(
      1,
      expect.objectContaining({
        content: "Try again.",
        requestAssistant: false,
      }),
    );
  });
});
