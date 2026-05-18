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

describe("ChatPage cross-session memory recall", () => {
  beforeEach(() => {
    apiMocks.reset();
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
    fireEvent.click(screen.getByRole("button", { name: "Send" }));

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
});
