import { base, fetchWithTimeout, j } from "./client";
import type {
  ChatMessage,
  ChatThreadDetail,
  ChatThreadSummary,
  ForgeArtifact,
} from "./types";
import type { JobRecord } from "@forge/shared";

export type AssistantStreamEvent = {
  event: string;
  data: string;
};

async function readSSEStream(
  res: Response,
  onEvent: (event: AssistantStreamEvent) => void,
) {
  if (!res.body) {
    throw new Error("Assistant stream did not return a readable body.");
  }
  const reader = res.body.getReader();
  const decoder = new TextDecoder();
  let buffer = "";

  const emitBlock = (block: string) => {
    let event = "message";
    const data: string[] = [];
    for (const line of block.split(/\r?\n/)) {
      if (line.startsWith("event:")) {
        event = line.slice("event:".length).trim();
      } else if (line.startsWith("data:")) {
        data.push(line.slice("data:".length).trimStart());
      }
    }
    if (data.length > 0) {
      onEvent({ event, data: data.join("\n") });
    }
  };

  for (;;) {
    const { value, done } = await reader.read();
    if (done) break;
    buffer += decoder.decode(value, { stream: true });
    const blocks = buffer.split(/\r?\n\r?\n/);
    buffer = blocks.pop() ?? "";
    for (const block of blocks) {
      emitBlock(block);
    }
  }
  buffer += decoder.decode();
  if (buffer.trim()) {
    emitBlock(buffer);
  }
}

export const chatApi = {
  threads: {
    list: (limit = 80) =>
      j<{ threads: ChatThreadSummary[] }>(
        `/api/chat/threads?limit=${encodeURIComponent(String(limit))}`,
      ),
    create: (body: { title?: string; dossierId?: number }) =>
      j<{ thread: ChatThreadSummary }>("/api/chat/threads", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    get: (id: number) =>
      j<ChatThreadDetail>(
        `/api/chat/threads/${encodeURIComponent(String(id))}`,
      ),
    update: (id: number, body: { title: string }) =>
      j<{ thread: ChatThreadSummary }>(
        `/api/chat/threads/${encodeURIComponent(String(id))}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    delete: (id: number) =>
      j<void>(`/api/chat/threads/${encodeURIComponent(String(id))}`, {
        method: "DELETE",
      }),
    postMessage: (
      id: number,
      body: {
        content: string;
        modelId?: string;
        attachmentArtifactIds?: number[];
        requestAssistant?: boolean;
        assistantDryRun?: boolean;
        stream?: boolean;
        asyncAssistant?: boolean;
        syncAssistant?: boolean;
      },
    ) =>
      j<{
        userMessage: ChatMessage;
        assistantMessage: ChatMessage | null;
        assistantPending?: boolean;
        userMessageId?: number;
        stream?: boolean;
        asyncAssistant?: boolean;
      }>(`/api/chat/threads/${encodeURIComponent(String(id))}/messages`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    uploadAttachment: async (id: number, file: File, title?: string) => {
      const fd = new FormData();
      fd.append("file", file);
      if (title && title.trim()) fd.append("title", title.trim());
      const res = await fetchWithTimeout(
        `${base()}/api/chat/threads/${encodeURIComponent(String(id))}/attachments`,
        {
          method: "POST",
          body: fd,
        },
      );
      if (!res.ok) {
        const t = await res.text().catch(() => "");
        throw new Error(t || `${res.status} ${res.statusText}`);
      }
      return (await res.json()) as {
        artifact: ForgeArtifact;
        bytes: number;
        previewText: string;
      };
    },
    assistantStream: async (
      threadId: number,
      userMessageId: number,
      onEvent: (event: AssistantStreamEvent) => void,
      signal?: AbortSignal,
    ) => {
      const res = await fetchWithTimeout(
        `${base()}/api/chat/threads/${encodeURIComponent(String(threadId))}/assistant-stream?userMessageId=${encodeURIComponent(String(userMessageId))}`,
        {
          headers: { Accept: "text/event-stream" },
          signal,
        },
      );
      if (!res.ok) {
        const t = await res.text().catch(() => "");
        throw new Error(t || `${res.status} ${res.statusText}`);
      }
      await readSSEStream(res, onEvent);
    },
    queueJob: (id: number, body: Record<string, unknown>) =>
      j<{ job: JobRecord }>(
        `/api/chat/threads/${encodeURIComponent(String(id))}/jobs`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
  },
};
