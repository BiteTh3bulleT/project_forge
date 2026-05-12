import type { JobTemplate } from "@forge/shared";
import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type CSSProperties,
  type PointerEvent as ReactPointerEvent,
} from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";

import {
  api,
  type ChatAttachment,
  type ChatMessage,
  type ChatThreadDetail,
  type ChatThreadSummary,
  type ModelRuntimeModel,
} from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";
import {
  describeAssistantMode,
  describeChatModel,
  preferredAutoChatModel,
  readCachedChatModelSelection,
  supportsChatCapability,
  usableChatStatus,
  writeCachedChatModelSelection,
} from "./ChatPage/chatModels";
import { ChatThreadRail } from "./ChatPage/ChatThreadRail";
import {
  CodeBlock,
  parseMessageParts,
} from "./ChatPage/messageContent";
import { readAttachments } from "./ChatPage/messageMetadata";
import {
  AttachmentInspectorCard,
  MessageRow,
} from "./ChatPage/messageSurface";
import {
  CHAT_PANE_LAYOUT_KEY,
  clampNumber,
  readStoredChatPaneLayout,
  type ChatPaneLayout,
} from "./ChatPage/paneLayout";
import {
  compactThinkingDetail,
  formatThinkingStage,
  ThinkingTimeline,
  type ChatThinkingEvent,
} from "./ChatPage/thinking";
import {
  BrowserResultPanel,
  TerminalTranscript,
  isBrowserTool,
  isTerminalTool,
  numberField,
  readToolEntries,
  readToolGatewayActivity,
  type ChatToolEntry,
} from "./ChatPage/toolGateway";

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === "object" && !Array.isArray(v))
    return v as Record<string, unknown>;
  return null;
}

function normalizeMessage(m: ChatMessage): ChatMessage {
  return { ...m, metadata: asRecord(m.metadata) ?? {} };
}

function normalizeThread(d: ChatThreadDetail): ChatThreadDetail {
  return {
    ...d,
    messages: Array.isArray(d.messages)
      ? d.messages.map((m) => normalizeMessage(m))
      : [],
  };
}

function applyPostMessages(
  prev: ChatThreadDetail | null,
  threadId: number,
  res: { userMessage: ChatMessage; assistantMessage: ChatMessage | null },
): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  const next = [...prev.messages, normalizeMessage(res.userMessage)];
  if (res.assistantMessage) next.push(normalizeMessage(res.assistantMessage));
  const updatedAt = Math.max(
    prev.updatedAtMs,
    res.userMessage.createdAtMs,
    res.assistantMessage?.createdAtMs ?? 0,
  );
  return { ...prev, messages: next, updatedAtMs: updatedAt };
}

function applyUserMessageOnly(
  prev: ChatThreadDetail | null,
  threadId: number,
  um: ChatMessage,
): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  return {
    ...prev,
    messages: [...prev.messages, normalizeMessage(um)],
    updatedAtMs: Math.max(prev.updatedAtMs, um.createdAtMs),
  };
}

function appendAssistantMessage(
  prev: ChatThreadDetail | null,
  threadId: number,
  am: ChatMessage,
): ChatThreadDetail | null {
  if (!prev || prev.id !== threadId) return prev;
  return {
    ...prev,
    messages: [...prev.messages, normalizeMessage(am)],
    updatedAtMs: Math.max(prev.updatedAtMs, am.createdAtMs),
  };
}

function findAssistantReply(
  messages: ChatMessage[],
  userMessageId: number,
): ChatMessage | undefined {
  return messages.find((message) => {
    if (message.role !== "assistant") return false;
    const raw = message.metadata?.replyToUserMessageId;
    return (
      raw === userMessageId ||
      raw === String(userMessageId) ||
      Number(raw) === userMessageId
    );
  });
}

function sleep(ms: number) {
  return new Promise<void>((resolve) => setTimeout(resolve, ms));
}

function trimLine(value: string, fallback: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : fallback;
}

const CHAT_STREAM_TOKEN_FLUSH_CHARS = 240;
const CHAT_STREAM_TOKEN_FLUSH_MS = 32;

function extractApiErrorMessage(err: unknown): string {
  const message = err instanceof Error ? err.message : String(err);
  const trimmed = message.trim();
  if (!trimmed.startsWith("{")) return trimmed;
  try {
    const parsed = JSON.parse(trimmed) as { error?: { message?: string } };
    return parsed?.error?.message?.trim() || trimmed;
  } catch {
    return trimmed;
  }
}

type MessageCodeSnippet = {
  key: string;
  messageId: number;
  createdAtMs: number;
  lang: string;
  code: string;
};

function readThinkingEvents(message: ChatMessage): ChatThinkingEvent[] {
  const meta = message.metadata ?? {};
  const activity = readToolGatewayActivity(meta);
  const pipeline = asRecord(meta.toolPipeline);
  const activityStages = activity?.stages;
  const pipelineStages = pipeline?.stages;
  const rawStages: unknown[] = Array.isArray(activityStages)
    ? activityStages
    : Array.isArray(pipelineStages)
      ? pipelineStages
      : [];
  const out: ChatThinkingEvent[] = [];
  for (let i = 0; i < rawStages.length; i++) {
    const row = asRecord(rawStages[i]);
    if (!row) continue;
    const stage =
      typeof row.stage === "string" && row.stage.trim()
        ? row.stage.trim()
        : "stage";
    const at = numberField(row, "atMs") ?? message.createdAtMs;
    out.push({
      key: `${message.id}-think-${i}-${stage}`,
      messageId: message.id,
      at,
      kind: "stage",
      text: formatThinkingStage(stage),
      detail: compactThinkingDetail(row),
      data: row,
    });
  }
  return out;
}

export function ChatPage() {
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const setStatus = useUiStore((s) => s.setStatusLine);

  const [threads, setThreads] = useState<ChatThreadSummary[]>([]);
  const [active, setActive] = useState<ChatThreadDetail | null>(null);
  const [draft, setDraft] = useState("");
  const [threadFilter, setThreadFilter] = useState("");
  const [requestAssistant, setRequestAssistant] = useState(true);
  const [assistantDryRun, setAssistantDryRun] = useState(false);
  const [streamAssistant, setStreamAssistant] = useState(true);
  const [blockingAssistant, setBlockingAssistant] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [showForgeActions, setShowForgeActions] = useState(false);
  const [streamingText, setStreamingText] = useState<string | null>(null);
  const [streamingEvents, setStreamingEvents] = useState<ChatThinkingEvent[]>(
    [],
  );
  const [templates, setTemplates] = useState<JobTemplate[]>([]);
  const [isEditingTitle, setIsEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [pendingAttachments, setPendingAttachments] = useState<
    ChatAttachment[]
  >([]);
  const [inspectorMode, setInspectorMode] = useState<
    "thinking" | "code" | "files" | "terminal" | "browser"
  >("thinking");
  const [selectedSnippetKey, setSelectedSnippetKey] = useState<string>("");
  const [selectedAttachmentId, setSelectedAttachmentId] = useState<number>(0);
  const [chatModels, setChatModels] = useState<ModelRuntimeModel[]>([]);
  const [chatModelLoadState, setChatModelLoadState] = useState<
    "idle" | "loading" | "ready" | "unavailable" | "error"
  >("idle");
  const [chatModelMessage, setChatModelMessage] = useState("");
  const [selectedChatModelId, setSelectedChatModelId] = useState<string>(() =>
    readCachedChatModelSelection(),
  );
  const [paneLayout, setPaneLayout] = useState<ChatPaneLayout>(() =>
    readStoredChatPaneLayout(),
  );

  const [jobForm, setJobForm] = useState<{
    templateId: string;
    title: string;
    userRequest: string;
    objective: string;
  }>({
    templateId: "search_packet",
    title: "Queued from chat",
    userRequest: "",
    objective: "",
  });

  const messagesEndRef = useRef<HTMLDivElement>(null);
  const messagesScrollRef = useRef<HTMLDivElement>(null);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const streamEsRef = useRef<EventSource | null>(null);
  const streamTokenBufferRef = useRef("");
  const streamTokenFlushTimerRef = useRef<ReturnType<
    typeof window.setTimeout
  > | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  const chatPaneStyle = useMemo(
    () =>
      ({
        "--forge-chat-thread-width": `${paneLayout.threadWidth}px`,
        "--forge-chat-inspector-width": `${paneLayout.inspectorWidth}px`,
      }) as CSSProperties,
    [paneLayout.inspectorWidth, paneLayout.threadWidth],
  );

  const inspectorSplitStyle = useMemo(
    () =>
      ({
        gridTemplateRows: `${paneLayout.inspectorListHeight}px 7px minmax(0, 1fr)`,
      }) as CSSProperties,
    [paneLayout.inspectorListHeight],
  );

  useEffect(() => {
    try {
      window.localStorage.setItem(
        CHAT_PANE_LAYOUT_KEY,
        JSON.stringify(paneLayout),
      );
    } catch {
      // Layout persistence is cosmetic; ignore storage failures.
    }
  }, [paneLayout]);

  const startHorizontalPaneResize = useCallback(
    (
      pane: "thread" | "inspector",
      event: ReactPointerEvent<HTMLDivElement>,
    ) => {
      if (event.button !== 0) return;
      event.preventDefault();
      const startX = event.clientX;
      const start = paneLayout;
      const previousCursor = document.body.style.cursor;
      const previousUserSelect = document.body.style.userSelect;
      document.body.style.cursor = "col-resize";
      document.body.style.userSelect = "none";

      const onMove = (moveEvent: PointerEvent) => {
        const dx = moveEvent.clientX - startX;
        setPaneLayout((current) => ({
          ...current,
          threadWidth:
            pane === "thread"
              ? clampNumber(start.threadWidth + dx, 200, 420)
              : current.threadWidth,
          inspectorWidth:
            pane === "inspector"
              ? clampNumber(start.inspectorWidth - dx, 300, 680)
              : current.inspectorWidth,
        }));
      };
      const onUp = () => {
        document.body.style.cursor = previousCursor;
        document.body.style.userSelect = previousUserSelect;
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
      };

      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp, { once: true });
    },
    [paneLayout],
  );

  const startInspectorSplitResize = useCallback(
    (event: ReactPointerEvent<HTMLDivElement>) => {
      if (event.button !== 0) return;
      event.preventDefault();
      const startY = event.clientY;
      const startHeight = paneLayout.inspectorListHeight;
      const previousCursor = document.body.style.cursor;
      const previousUserSelect = document.body.style.userSelect;
      document.body.style.cursor = "row-resize";
      document.body.style.userSelect = "none";

      const onMove = (moveEvent: PointerEvent) => {
        const dy = moveEvent.clientY - startY;
        setPaneLayout((current) => ({
          ...current,
          inspectorListHeight: clampNumber(startHeight + dy, 160, 520),
        }));
      };
      const onUp = () => {
        document.body.style.cursor = previousCursor;
        document.body.style.userSelect = previousUserSelect;
        window.removeEventListener("pointermove", onMove);
        window.removeEventListener("pointerup", onUp);
      };

      window.addEventListener("pointermove", onMove);
      window.addEventListener("pointerup", onUp, { once: true });
    },
    [paneLayout.inspectorListHeight],
  );

  useEffect(() => {
    return () => {
      streamEsRef.current?.close();
      streamEsRef.current = null;
      if (streamTokenFlushTimerRef.current)
        window.clearTimeout(streamTokenFlushTimerRef.current);
      streamTokenFlushTimerRef.current = null;
      streamTokenBufferRef.current = "";
    };
  }, []);

  useEffect(() => {
    if (!textareaRef.current) return;
    textareaRef.current.style.height = "0px";
    const next = Math.max(54, Math.min(180, textareaRef.current.scrollHeight));
    textareaRef.current.style.height = `${next}px`;
  }, [draft]);

  const threadIdParam = params.get("threadId");

  const refreshThreads = useCallback(async () => {
    const res = await api.chat.threads.list(120);
    setThreads(Array.isArray(res?.threads) ? res.threads : []);
  }, []);

  const refreshChatModels = useCallback(async () => {
    setChatModelLoadState("loading");
    setChatModelMessage("");
    try {
      const res = await api.modelRuntime.list();
      const next = (Array.isArray(res.models) ? res.models : [])
        .filter(
          (model) => supportsChatCapability(model) && usableChatStatus(model),
        )
        .sort((a, b) => a.id.localeCompare(b.id));
      setChatModels(next);
      setSelectedChatModelId((prev) =>
        prev && next.some((item) => item.id === prev) ? prev : "",
      );
      setChatModelLoadState("ready");
      if (next.length === 0) {
        setChatModelMessage(
          "No chat-capable runtime models are currently available.",
        );
      }
    } catch (e) {
      const message = extractApiErrorMessage(e);
      setChatModels([]);
      if (
        (e instanceof Error ? e.message : String(e)).includes(
          "MODEL_RUNTIME_UNAVAILABLE",
        )
      ) {
        setChatModelLoadState("unavailable");
        setChatModelMessage(
          "Model runtime is unavailable. Chat will use configured adapter fallback.",
        );
      } else {
        setChatModelLoadState("error");
        setChatModelMessage(
          message || "Model runtime model list could not be loaded.",
        );
      }
    }
  }, []);

  const loadThread = useCallback(
    async (id: number) => {
      const d = await api.chat.threads.get(id);
      const normalized = normalizeThread(d);
      setActive(normalized);
      setTitleDraft(trimLine(normalized.title, `Thread #${normalized.id}`));
      setIsEditingTitle(false);
      setParams({ threadId: String(normalized.id) });
    },
    [setParams],
  );

  useEffect(() => {
    let cancelled = false;
    async function boot() {
      try {
        const [tRes, tpl] = await Promise.all([
          api.chat.threads.list(120),
          api.jobs.templates(),
        ]);
        if (cancelled) return;
        const threadList = Array.isArray(tRes?.threads) ? tRes.threads : [];
        const templateList = Array.isArray(tpl?.templates) ? tpl.templates : [];
        setThreads(threadList);
        setTemplates(templateList);
        setErr(null);

        const fromUrl = threadIdParam ? Number(threadIdParam) : NaN;
        if (Number.isFinite(fromUrl) && fromUrl > 0) {
          const d = await api.chat.threads.get(fromUrl);
          if (!cancelled) {
            const normalized = normalizeThread(d);
            setActive(normalized);
            setTitleDraft(
              trimLine(normalized.title, `Thread #${normalized.id}`),
            );
          }
          return;
        }
        if (threadList.length > 0 && !cancelled) {
          const d = await api.chat.threads.get(threadList[0].id);
          if (!cancelled) {
            const normalized = normalizeThread(d);
            setActive(normalized);
            setTitleDraft(
              trimLine(normalized.title, `Thread #${normalized.id}`),
            );
            setParams({ threadId: String(d.id) });
          }
          return;
        }
        if (!cancelled) {
          setActive(null);
          setTitleDraft("");
        }
      } catch (e) {
        if (!cancelled) {
          setErr(e instanceof Error ? e.message : String(e));
          setActive(null);
          setTitleDraft("");
        }
      }
    }
    void boot();
    return () => {
      cancelled = true;
    };
  }, [threadIdParam, setParams]);

  useEffect(() => {
    writeCachedChatModelSelection(selectedChatModelId);
  }, [selectedChatModelId]);

  useEffect(() => {
    void refreshChatModels();
  }, [refreshChatModels]);

  useEffect(() => {
    const scroller = messagesScrollRef.current;
    if (!scroller) return;
    window.requestAnimationFrame(() => {
      scroller.scrollTop = scroller.scrollHeight;
    });
  }, [active?.id, active?.messages.length, streamingText]);

  const inspectAttachment = useCallback((artifactId: number) => {
    setInspectorMode("files");
    setSelectedAttachmentId(artifactId);
  }, []);

  async function newThread() {
    setBusy(true);
    try {
      const res = await api.chat.threads.create({ title: "New chat" });
      await refreshThreads();
      await loadThread(res.thread.id);
      setStatus(`Thread ${res.thread.id} created.`);
      setErr(null);
      setShowForgeActions(false);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveThreadTitle() {
    if (!active) return;
    const title = titleDraft.trim();
    if (!title) {
      setTitleDraft(trimLine(active.title, `Thread #${active.id}`));
      setIsEditingTitle(false);
      return;
    }
    setBusy(true);
    try {
      const res = await api.chat.threads.update(active.id, { title });
      setActive((prev) =>
        prev
          ? {
              ...prev,
              title: res.thread.title,
              updatedAtMs: res.thread.updatedAtMs,
            }
          : prev,
      );
      setThreads((prev) =>
        prev.map((t) =>
          t.id === res.thread.id
            ? {
                ...t,
                title: res.thread.title,
                updatedAtMs: res.thread.updatedAtMs,
              }
            : t,
        ),
      );
      setIsEditingTitle(false);
      setStatus("Thread renamed.");
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function deleteActiveThread() {
    if (!active) return;
    if (!window.confirm("Delete this chat and all messages?")) return;
    setBusy(true);
    try {
      await api.chat.threads.delete(active.id);
      setActive(null);
      setTitleDraft("");
      setParams({});
      await refreshThreads();
      setStatus("Thread deleted.");
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  function stopStreaming() {
    streamEsRef.current?.close();
    streamEsRef.current = null;
    if (streamTokenFlushTimerRef.current)
      window.clearTimeout(streamTokenFlushTimerRef.current);
    streamTokenFlushTimerRef.current = null;
    streamTokenBufferRef.current = "";
    setStreamingText(null);
    setStreamingEvents([]);
    setStatus("Assistant stream stopped.");
  }

  async function send() {
    if (!active) return;
    const text = draft.trim();
    const hasAttachments = pendingAttachments.length > 0;
    if (!text && !hasAttachments) return;

    setBusy(true);
    setErr(null);
    if (streamTokenFlushTimerRef.current)
      window.clearTimeout(streamTokenFlushTimerRef.current);
    streamTokenFlushTimerRef.current = null;
    streamTokenBufferRef.current = "";
    setStreamingText(null);
    setStreamingEvents([]);
    streamEsRef.current?.close();
    streamEsRef.current = null;

    const useStream =
      requestAssistant &&
      !assistantDryRun &&
      streamAssistant &&
      !blockingAssistant;
    const useSyncBlock =
      requestAssistant && !assistantDryRun && blockingAssistant;
    const useAsyncPoll =
      requestAssistant && !assistantDryRun && !useStream && !useSyncBlock;

    const body: Parameters<typeof api.chat.threads.postMessage>[1] = {
      content: text || "Attached files for context.",
      attachmentArtifactIds: pendingAttachments.map((item) => item.artifactId),
      requestAssistant,
      assistantDryRun,
    };
    const requestedModel = selectedChatModelId.trim();
    const modelListReady =
      chatModelLoadState === "ready" || chatModelLoadState === "unavailable";
    const requestedModelAvailable =
      requestedModel &&
      (!modelListReady ||
        chatModels.some((model) => model.id === requestedModel));
    if (requestedModel && requestedModelAvailable) {
      body.modelId = requestedModel;
    } else {
      if (requestedModel && modelListReady) {
        setSelectedChatModelId("");
      }
      const preferred = preferredAutoChatModel(chatModels);
      if (preferred) {
        body.modelId = preferred.id;
      }
    }
    if (useStream) body.stream = true;
    else if (useSyncBlock) body.syncAssistant = true;
    else if (useAsyncPoll) body.asyncAssistant = true;

    try {
      const res = await api.chat.threads.postMessage(active.id, body);
      const tid = active.id;
      setDraft("");
      setPendingAttachments([]);

      if (res.assistantMessage) {
        setActive((prev) => applyPostMessages(prev, tid, res));
        void refreshThreads();
        setStatus(
          requestAssistant ? "Assistant reply saved." : "Message saved.",
        );
        textareaRef.current?.focus();
        return;
      }

      setActive((prev) => applyUserMessageOnly(prev, tid, res.userMessage));
      void refreshThreads();

      if (
        res.assistantPending &&
        res.stream === true &&
        res.userMessageId != null
      ) {
        setStatus("Streaming assistant reply…");
        await openAssistantStream(tid, res.userMessageId);
        setStatus("Assistant reply saved.");
        textareaRef.current?.focus();
        return;
      }

      if (
        res.assistantPending &&
        res.asyncAssistant &&
        res.userMessageId != null
      ) {
        setStatus("Waiting for assistant reply…");
        await pollForAssistantReply(tid, res.userMessageId);
        setStatus("Assistant reply saved.");
        textareaRef.current?.focus();
        return;
      }

      setStatus(
        requestAssistant
          ? "Message saved (assistant not returned)."
          : "Message saved.",
      );
      textareaRef.current?.focus();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
      setStreamingText(null);
    }
  }

  function openAssistantStream(
    threadId: number,
    userMessageId: number,
  ): Promise<void> {
    return new Promise((resolve, reject) => {
      const url = api.chat.threads.assistantStreamUrl(threadId, userMessageId);
      setStreamingText("");
      setInspectorMode("thinking");
      const es = new EventSource(url);
      streamEsRef.current = es;
      setStreamingEvents([]);
      streamTokenBufferRef.current = "";
      if (streamTokenFlushTimerRef.current)
        window.clearTimeout(streamTokenFlushTimerRef.current);
      streamTokenFlushTimerRef.current = null;
      let firstTokenShown = false;

      const flushTokenBuffer = () => {
        if (streamTokenFlushTimerRef.current) {
          window.clearTimeout(streamTokenFlushTimerRef.current);
          streamTokenFlushTimerRef.current = null;
        }
        if (!streamTokenBufferRef.current) return;
        const chunk = streamTokenBufferRef.current;
        streamTokenBufferRef.current = "";
        setStreamingText((prev) => (prev == null ? chunk : prev + chunk));
      };

      const queueToken = (token: string) => {
        if (!token) return;
        streamTokenBufferRef.current += token;
        if (!firstTokenShown) {
          firstTokenShown = true;
          flushTokenBuffer();
          return;
        }
        if (
          streamTokenBufferRef.current.length >= CHAT_STREAM_TOKEN_FLUSH_CHARS
        ) {
          flushTokenBuffer();
          return;
        }
        if (!streamTokenFlushTimerRef.current) {
          streamTokenFlushTimerRef.current = window.setTimeout(
            flushTokenBuffer,
            CHAT_STREAM_TOKEN_FLUSH_MS,
          );
        }
      };

      const cleanupStream = () => {
        es.close();
        streamEsRef.current = null;
        if (streamTokenFlushTimerRef.current)
          window.clearTimeout(streamTokenFlushTimerRef.current);
        streamTokenFlushTimerRef.current = null;
        streamTokenBufferRef.current = "";
        setStreamingText(null);
        setStreamingEvents([]);
      };

      const recoverSavedAssistantReply = async (): Promise<boolean> => {
        try {
          const d = await api.chat.threads.get(threadId);
          const messages = Array.isArray(d.messages)
            ? d.messages.map((m) => normalizeMessage(m))
            : [];
          if (!findAssistantReply(messages, userMessageId)) return false;
          setActive(normalizeThread(d));
          void refreshThreads();
          return true;
        } catch {
          return false;
        }
      };

      const pushThinkingEvent = (event: ChatThinkingEvent) => {
        setStreamingEvents((prev) => [...prev.slice(-79), event]);
      };

      let settled = false;

      es.addEventListener("token", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as {
            text?: string;
          };
          const token = typeof raw.text === "string" ? raw.text : "";
          queueToken(token);
        } catch {
          /* ignore malformed token payload */
        }
      });

      es.addEventListener("done", (ev) => {
        try {
          if (settled) return;
          settled = true;
          const raw = JSON.parse((ev as MessageEvent).data as string) as {
            assistantMessage?: ChatMessage;
          };
          es.close();
          streamEsRef.current = null;
          flushTokenBuffer();
          setStreamingText(null);
          setStreamingEvents([]);
          const am = raw.assistantMessage;
          if (am) {
            setActive((prev) => appendAssistantMessage(prev, threadId, am));
          }
          void refreshThreads();
          resolve();
        } catch (e) {
          settled = true;
          es.close();
          streamEsRef.current = null;
          flushTokenBuffer();
          setStreamingText(null);
          reject(e);
        }
      });

      const failOrRecover = (error: Error) => {
        if (settled) return;
        settled = true;
        cleanupStream();
        void recoverSavedAssistantReply().then((recovered) => {
          if (recovered) {
            resolve();
            return;
          }
          reject(error);
        });
      };

      es.addEventListener("error", (ev) => {
        const data =
          typeof (ev as MessageEvent).data === "string"
            ? ((ev as MessageEvent).data as string)
            : "";
        if (!data) return;
        try {
          const raw = JSON.parse(data) as { message?: string; error?: string };
          const message =
            raw.message?.trim() ||
            raw.error?.trim() ||
            "Assistant stream failed.";
          failOrRecover(new Error(message));
        } catch {
          failOrRecover(new Error(data));
        }
      });

      es.onerror = () => {
        failOrRecover(new Error("Assistant stream disconnected."));
      };

      es.addEventListener("agent_stage", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<
            string,
            unknown
          >;
          const stage = typeof raw.stage === "string" ? raw.stage : "stage";
          const at = typeof raw.atMs === "number" ? raw.atMs : Date.now();
          pushThinkingEvent({
            at,
            kind: "stage",
            text: formatThinkingStage(stage),
            detail: compactThinkingDetail(raw),
            data: raw,
          });
        } catch {
          /* ignore malformed stage payload */
        }
      });

      es.addEventListener("tool_call", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<
            string,
            unknown
          >;
          const modelName =
            typeof raw.modelName === "string" ? raw.modelName : "tool";
          const at = Date.now();
          pushThinkingEvent({
            at,
            kind: "call",
            text: `Tool call: ${modelName}`,
            detail: compactThinkingDetail(raw),
            data: raw,
          });
        } catch {
          /* ignore malformed tool_call payload */
        }
      });

      es.addEventListener("tool_result", (ev) => {
        try {
          const raw = JSON.parse((ev as MessageEvent).data as string) as Record<
            string,
            unknown
          >;
          const modelName =
            typeof raw.modelName === "string" ? raw.modelName : "tool";
          const state = typeof raw.state === "string" ? raw.state : "unknown";
          const at = Date.now();
          pushThinkingEvent({
            at,
            kind: "result",
            text: `Tool result: ${modelName}`,
            detail: `state: ${state}${compactThinkingDetail(raw) ? ` · ${compactThinkingDetail(raw)}` : ""}`,
            data: raw,
          });
        } catch {
          /* ignore malformed tool_result payload */
        }
      });
    });
  }

  async function pollForAssistantReply(
    threadId: number,
    userMessageId: number,
  ) {
    const deadline = Date.now() + 120_000;
    while (Date.now() < deadline) {
      const d = await api.chat.threads.get(threadId);
      const messages = Array.isArray(d.messages)
        ? d.messages.map((m) => normalizeMessage(m))
        : [];
      if (findAssistantReply(messages, userMessageId)) {
        setActive(normalizeThread(d));
        void refreshThreads();
        return;
      }
      await sleep(400);
    }
    throw new Error("Timed out waiting for assistant reply.");
  }

  function onComposerKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      void send();
    }
  }

  async function uploadSelectedFiles(list: FileList | null) {
    if (!active || !list || list.length === 0) return;
    const files = Array.from(list);
    setUploading(true);
    setErr(null);
    try {
      const uploaded: ChatAttachment[] = [];
      for (const file of files) {
        const res = await api.chat.threads.uploadAttachment(
          active.id,
          file,
          file.name,
        );
        uploaded.push({
          artifactId: res.artifact.id,
          title: res.artifact.title,
          mimeType: res.artifact.mimeType,
          fileName: file.name,
          textPreview: res.previewText || undefined,
        });
      }
      setPendingAttachments((prev) => {
        const next = [...prev];
        for (const item of uploaded) {
          if (!next.some((existing) => existing.artifactId === item.artifactId))
            next.push(item);
        }
        return next;
      });
      setStatus(`${uploaded.length} attachment(s) uploaded.`);
      setInspectorMode("files");
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setUploading(false);
      if (fileInputRef.current) fileInputRef.current.value = "";
    }
  }

  async function queueJob() {
    if (!active) return;
    const templateId = templates.some((t) => t.id === jobForm.templateId)
      ? jobForm.templateId
      : templates[0]?.id;
    if (!templateId) {
      setErr("No job templates available from core.");
      return;
    }
    const lastUserMessage = [...active.messages]
      .reverse()
      .find((m) => m.role === "user");
    const fallbackRequest =
      lastUserMessage?.content || "Job requested from chat";
    setBusy(true);
    try {
      const res = await api.chat.threads.queueJob(active.id, {
        templateId,
        title: jobForm.title.trim() || "Job from chat",
        userRequest: jobForm.userRequest.trim() || fallbackRequest,
        objective:
          jobForm.objective.trim() || "Operator-requested job from chat thread",
        query: jobForm.userRequest.trim() || fallbackRequest,
        initiatingSource: "chat",
      });
      setStatus(`Job ${res.job.id} queued from chat.`);
      setShowForgeActions(false);
      navigate(`/jobs/${res.job.id}`);
      await loadThread(active.id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const filteredThreads = useMemo(() => {
    const query = threadFilter.trim().toLowerCase();
    const ordered = [...threads].sort((a, b) => b.updatedAtMs - a.updatedAtMs);
    if (!query) return ordered;
    return ordered.filter((thread) =>
      `${thread.title} ${thread.id}`.toLowerCase().includes(query),
    );
  }, [threadFilter, threads]);

  const messageAttachments = useMemo(() => {
    const out: Array<{
      messageId: number;
      createdAtMs: number;
      role: string;
      attachment: ChatAttachment;
    }> = [];
    for (const message of active?.messages ?? []) {
      for (const attachment of readAttachments(message.metadata)) {
        out.push({
          messageId: message.id,
          createdAtMs: message.createdAtMs,
          role: message.role,
          attachment,
        });
      }
    }
    const seen = new Set<number>();
    return out
      .sort((a, b) => b.createdAtMs - a.createdAtMs)
      .filter((item) => {
        if (seen.has(item.attachment.artifactId)) return false;
        seen.add(item.attachment.artifactId);
        return true;
      });
  }, [active?.messages]);

  const assistantCodeSnippets = useMemo<MessageCodeSnippet[]>(() => {
    const out: MessageCodeSnippet[] = [];
    for (const message of active?.messages ?? []) {
      if (message.role !== "assistant") continue;
      const parts = parseMessageParts(message.content);
      let index = 0;
      for (const part of parts) {
        if (part.type !== "code") continue;
        out.push({
          key: `${message.id}:${index}`,
          messageId: message.id,
          createdAtMs: message.createdAtMs,
          lang: part.lang || "code",
          code: part.text,
        });
        index += 1;
      }
    }
    return out.sort((a, b) => b.createdAtMs - a.createdAtMs);
  }, [active?.messages]);

  const terminalEntries = useMemo<ChatToolEntry[]>(() => {
    const out: ChatToolEntry[] = [];
    for (const message of active?.messages ?? []) {
      for (const entry of readToolEntries(message)) {
        if (isTerminalTool(entry.tool)) out.push(entry);
      }
    }
    return out.sort((a, b) => b.createdAtMs - a.createdAtMs);
  }, [active?.messages]);

  const browserEntries = useMemo<ChatToolEntry[]>(() => {
    const out: ChatToolEntry[] = [];
    for (const message of active?.messages ?? []) {
      for (const entry of readToolEntries(message)) {
        if (isBrowserTool(entry.tool)) out.push(entry);
      }
    }
    return out.sort((a, b) => b.createdAtMs - a.createdAtMs);
  }, [active?.messages]);

  const thinkingEntries = useMemo<ChatThinkingEvent[]>(() => {
    const out: ChatThinkingEvent[] = [];
    for (const message of active?.messages ?? []) {
      if (message.role !== "assistant") continue;
      out.push(...readThinkingEvents(message));
    }
    return out.sort((a, b) => b.at - a.at);
  }, [active?.messages]);

  useEffect(() => {
    if (!selectedSnippetKey && assistantCodeSnippets.length > 0) {
      setSelectedSnippetKey(assistantCodeSnippets[0].key);
    }
    if (
      selectedSnippetKey &&
      !assistantCodeSnippets.some((item) => item.key === selectedSnippetKey)
    ) {
      setSelectedSnippetKey(assistantCodeSnippets[0]?.key ?? "");
    }
  }, [assistantCodeSnippets, selectedSnippetKey]);

  useEffect(() => {
    if (!selectedAttachmentId && messageAttachments.length > 0) {
      setSelectedAttachmentId(messageAttachments[0].attachment.artifactId);
    }
    if (
      selectedAttachmentId &&
      !messageAttachments.some(
        (item) => item.attachment.artifactId === selectedAttachmentId,
      )
    ) {
      setSelectedAttachmentId(
        messageAttachments[0]?.attachment.artifactId ?? 0,
      );
    }
  }, [messageAttachments, selectedAttachmentId]);

  const selectedChatModel = useMemo(
    () => chatModels.find((model) => model.id === selectedChatModelId) ?? null,
    [chatModels, selectedChatModelId],
  );

  const autoChatModel = useMemo(
    () => preferredAutoChatModel(chatModels),
    [chatModels],
  );

  const assistantModeSummary = useMemo(
    () =>
      describeAssistantMode({
        requestAssistant,
        assistantDryRun,
        streamAssistant,
        blockingAssistant,
      }),
    [assistantDryRun, blockingAssistant, requestAssistant, streamAssistant],
  );

  const chatModelSummary = useMemo(
    () => describeChatModel(selectedChatModelId, chatModels),
    [selectedChatModelId, chatModels],
  );

  return (
    <div
      className={[
        "forge-chat-layout forge-chat-layout--ops grid h-full min-h-0 overflow-hidden bg-forge-black text-forge-ash",
        !active ? "forge-chat-layout--empty" : "",
      ].join(" ")}
      style={chatPaneStyle}
    >
      <ChatThreadRail
        threads={threads}
        filteredThreads={filteredThreads}
        activeThreadId={active?.id ?? null}
        threadFilter={threadFilter}
        busy={busy}
        error={err}
        onNewThread={() => void newThread()}
        onLoadThread={(threadId) => void loadThread(threadId)}
        onThreadFilterChange={setThreadFilter}
      />

      <div
        role="separator"
        aria-label="Resize chat thread rail"
        aria-orientation="vertical"
        className="forge-chat-resizer forge-chat-resizer--left"
        onPointerDown={(event) => startHorizontalPaneResize("thread", event)}
      />

      <section className="flex min-h-0 min-w-0 flex-col overflow-hidden">
        {!active ? (
          <div className="forge-chat-empty-state flex flex-1 items-center justify-center p-6">
            <div className="w-full max-w-2xl rounded-xl border border-dashed border-forge-platinum/15 bg-black/40 px-6 py-7 text-center shadow-[0_24px_70px_rgba(0,0,0,0.35)]">
              <div className="mx-auto mb-4 h-1 w-20 rounded-full bg-forge-ember" />
              <h2 className="text-xl font-semibold text-forge-ash">
                Open a command thread
              </h2>
              <p className="mx-auto mt-2 max-w-lg text-sm leading-6 text-forge-mist/80">
                Create a thread to use the composer, attachments, terminal,
                browser, code, and thinking inspectors from one surface.
              </p>
              <button
                type="button"
                onClick={() => void newThread()}
                disabled={busy}
                className="forge-chat-primary-btn mt-5 px-4 py-2 text-sm"
              >
                New chat
              </button>
            </div>
          </div>
        ) : (
          <>
            <header className="forge-chat-header border-b border-forge-platinum/10 bg-forge-black/90">
              <div className="forge-chat-content-width mx-auto flex w-full flex-col gap-3 py-3 lg:flex-row lg:items-center lg:justify-between">
                <div className="min-w-0 flex-1">
                  {isEditingTitle ? (
                    <div className="flex items-center gap-2">
                      <label htmlFor="thread-title" className="sr-only">
                        Thread title
                      </label>
                      <input
                        id="thread-title"
                        className="forge-input h-9 text-sm"
                        value={titleDraft}
                        onChange={(e) => setTitleDraft(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === "Enter") {
                            e.preventDefault();
                            void saveThreadTitle();
                          }
                          if (e.key === "Escape") {
                            e.preventDefault();
                            setTitleDraft(
                              trimLine(active.title, `Thread #${active.id}`),
                            );
                            setIsEditingTitle(false);
                          }
                        }}
                        autoFocus
                      />
                      <button
                        type="button"
                        onClick={() => void saveThreadTitle()}
                        className="forge-chat-action-btn"
                      >
                        Save
                      </button>
                    </div>
                  ) : (
                    <div className="flex min-w-0 items-center gap-2">
                      <h2 className="truncate text-base font-semibold text-forge-ash">
                        {trimLine(active.title, `Thread #${active.id}`)}
                      </h2>
                      <button
                        type="button"
                        onClick={() => setIsEditingTitle(true)}
                        className="rounded border border-transparent px-2 py-1 text-[11px] text-forge-mist transition hover:border-forge-platinum/10 hover:text-forge-ash"
                      >
                        Rename
                      </button>
                    </div>
                  )}
                  <div className="mt-1 flex flex-wrap items-center gap-2 text-xs text-forge-mist">
                    <span>{active.messages.length} message(s)</span>
                    <span className="text-forge-mist/35">•</span>
                    <span>updated {formatTime(active.updatedAtMs)}</span>
                    <span className="text-forge-mist/35">•</span>
                    <span className="font-mono text-[11px] text-forge-mist/80">
                      thread {active.id}
                    </span>
                  </div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <span className="rounded-full border border-forge-ember/25 bg-forge-ember/10 px-2.5 py-1 text-[10px] uppercase tracking-[0.14em] text-forge-emberSoft">
                      {assistantModeSummary}
                    </span>
                    <span className="rounded-full border border-forge-platinum/10 bg-forge-platinum/5 px-2.5 py-1 text-[10px] text-forge-mist">
                      {selectedChatModel?.displayName?.trim() ||
                        selectedChatModel?.id ||
                        autoChatModel?.displayName?.trim() ||
                        autoChatModel?.id ||
                        "Auto model"}
                    </span>
                  </div>
                </div>

                <div className="flex w-full min-w-0 flex-wrap items-center justify-start gap-2 lg:w-auto lg:justify-end">
                  {streamingText !== null ? (
                    <button
                      type="button"
                      onClick={stopStreaming}
                      className="rounded-lg border border-forge-ember/35 bg-forge-ember/10 px-3 py-1.5 text-xs font-semibold text-forge-emberSoft"
                    >
                      Stop
                    </button>
                  ) : null}
                  <button
                    type="button"
                    onClick={() => setShowForgeActions((v) => !v)}
                    className={[
                      "forge-chat-action-btn",
                      showForgeActions
                        ? "forge-chat-primary-btn"
                        : "bg-transparent",
                    ].join(" ")}
                  >
                    {showForgeActions ? "Hide Forge" : "Forge Actions"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void deleteActiveThread()}
                    disabled={busy}
                    className="forge-chat-action-btn bg-transparent border-forge-platinum/10 px-2.5"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </header>

            {showForgeActions ? (
              <div className="forge-chat-toolbar border-b border-forge-platinum/10 bg-forge-carbon/80">
                <div className="forge-chat-content-width mx-auto grid w-full gap-2 md:grid-cols-2">
                  <label className="block">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                      Template
                    </span>
                    <select
                      aria-label="Job template"
                      className="forge-input mt-1"
                      value={
                        templates.some((t) => t.id === jobForm.templateId)
                          ? jobForm.templateId
                          : (templates[0]?.id ?? "")
                      }
                      onChange={(e) =>
                        setJobForm((f) => ({
                          ...f,
                          templateId: e.target.value,
                        }))
                      }
                    >
                      {templates.length === 0 ? (
                        <option value="">No templates available</option>
                      ) : (
                        templates.map((tpl) => (
                          <option key={tpl.id} value={tpl.id}>
                            {tpl.name} ({tpl.id})
                          </option>
                        ))
                      )}
                    </select>
                  </label>
                  <label className="block">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                      Title
                    </span>
                    <input
                      aria-label="Job title"
                      className="forge-input mt-1"
                      value={jobForm.title}
                      onChange={(e) =>
                        setJobForm((f) => ({ ...f, title: e.target.value }))
                      }
                    />
                  </label>
                  <label className="block md:col-span-2">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                      Request override
                    </span>
                    <input
                      aria-label="Job request"
                      className="forge-input mt-1"
                      value={jobForm.userRequest}
                      onChange={(e) =>
                        setJobForm((f) => ({
                          ...f,
                          userRequest: e.target.value,
                        }))
                      }
                      placeholder="Optional override for packet request"
                    />
                  </label>
                  <label className="block md:col-span-2">
                    <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                      Objective
                    </span>
                    <input
                      aria-label="Job objective"
                      className="forge-input mt-1"
                      value={jobForm.objective}
                      onChange={(e) =>
                        setJobForm((f) => ({ ...f, objective: e.target.value }))
                      }
                    />
                  </label>
                </div>
                <div className="forge-chat-content-width mx-auto mt-3 flex w-full gap-2">
                  <button
                    type="button"
                    onClick={() => void queueJob()}
                    disabled={busy || templates.length === 0}
                    className="forge-chat-action-btn forge-chat-primary-btn"
                  >
                    Queue Job
                  </button>
                </div>
              </div>
            ) : null}

            <div
              ref={messagesScrollRef}
              className="forge-chat-messages forge-chat-scroll min-h-0 flex-1 overflow-y-auto px-5 py-6"
            >
              <div className="forge-chat-message-stack forge-chat-content-width mx-auto flex w-full flex-col gap-5">
                {active.messages.length === 0 ? (
                  <div className="rounded-xl border border-dashed border-forge-platinum/15 bg-black/35 px-5 py-7 text-center shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
                    <div className="mx-auto mb-3 h-1 w-16 rounded-full bg-forge-ember/90" />
                    <div className="text-base font-semibold text-forge-ash">
                      Ready for the first message
                    </div>
                    <div className="mx-auto mt-2 max-w-md text-sm leading-6 text-forge-mist/75">
                      Ask for analysis, attach evidence, or use a quick tool
                      prompt from the composer.
                    </div>
                  </div>
                ) : (
                  active.messages.map((message) => (
                    <MessageRow
                      key={message.id}
                      message={message}
                      onInspectAttachment={inspectAttachment}
                    />
                  ))
                )}

                {streamingText !== null ? (
                  <div className="rounded-xl border border-forge-platinum/10 bg-forge-charcoal p-4">
                    <div className="mb-2 text-[10px] uppercase tracking-wide text-forge-mist/70">
                      Assistant is typing…
                    </div>
                    <div className="text-[15px] leading-7 text-forge-ash whitespace-pre-wrap break-words">
                      {streamingText || "…"}
                    </div>
                    {streamingEvents.length > 0 ? (
                      <div className="mt-3 rounded-lg border border-forge-platinum/10 bg-black/25 p-2">
                        <ThinkingTimeline
                          events={streamingEvents.slice().reverse()}
                          live
                          compact
                        />
                      </div>
                    ) : null}
                  </div>
                ) : null}
                <div ref={messagesEndRef} className="h-px w-full" aria-hidden />
              </div>
            </div>

            <footer className="forge-chat-footer border-t border-forge-platinum/10 bg-forge-black/95">
              <div className="forge-chat-content-width mx-auto w-full">
                <div className="forge-chat-routing-strip mb-2 rounded-lg border border-forge-platinum/10 bg-black/30 p-2">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="min-w-0 flex-1">
                      <div className="text-[10px] uppercase tracking-[0.16em] text-forge-mist/60">
                        Assistant Routing
                      </div>
                      <div className="mt-0.5 truncate text-xs text-forge-mist/80">
                        {assistantModeSummary} ·{" "}
                        {chatModelMessage || chatModelSummary}
                      </div>
                    </div>
                    <div className="flex min-w-0 flex-wrap items-center gap-2">
                      <label className="inline-flex min-w-0 items-center gap-2 rounded-lg border border-forge-platinum/10 bg-black/20 px-2.5 py-1.5 text-xs text-forge-mist">
                        <input
                          aria-label="Use assistant"
                          type="checkbox"
                          checked={requestAssistant}
                          onChange={(e) =>
                            setRequestAssistant(e.target.checked)
                          }
                        />
                        Use assistant
                      </label>
                      <button
                        type="button"
                        onClick={() => void refreshChatModels()}
                        className="min-w-0 rounded-lg border border-forge-platinum/10 bg-forge-platinum/5 px-3 py-1.5 text-[11px] text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
                      >
                        Refresh models
                      </button>
                      <Link
                        to="/models"
                        className="min-w-0 rounded-lg border border-forge-ember/25 bg-forge-ember/10 px-3 py-1.5 text-[11px] text-forge-emberSoft transition hover:border-forge-ember/40 hover:text-forge-ash"
                      >
                        Open Models
                      </Link>
                      <button
                        type="button"
                        onClick={() => setShowAdvanced((v) => !v)}
                        className="forge-chat-action-btn border-forge-platinum/10 bg-transparent py-1 px-2.5 text-[11px]"
                      >
                        {showAdvanced ? "Hide advanced" : "Advanced"}
                      </button>
                    </div>
                  </div>

                  {showAdvanced ? (
                    <div className="mt-3 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]">
                      <label className="block min-w-0">
                        <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                          Chat model
                        </span>
                        <select
                          aria-label="Chat runtime model"
                          className="forge-input mt-1 h-10 w-full py-1 text-sm"
                          value={selectedChatModelId}
                          onChange={(e) =>
                            setSelectedChatModelId(e.target.value)
                          }
                          disabled={chatModelLoadState === "loading"}
                        >
                          <option value="">
                            Auto (runtime default / adapter fallback)
                          </option>
                          {selectedChatModelId &&
                          !chatModels.some(
                            (model) => model.id === selectedChatModelId,
                          ) ? (
                            <option value={selectedChatModelId}>
                              Saved: {selectedChatModelId} (not in current
                              runtime list)
                            </option>
                          ) : null}
                          {chatModels.map((model) => (
                            <option key={model.id} value={model.id}>
                              {model.displayName?.trim() || model.id}
                            </option>
                          ))}
                        </select>
                      </label>
                      <button
                        type="button"
                        onClick={() => setSelectedChatModelId("")}
                        disabled={!selectedChatModelId}
                        className="rounded-lg border border-forge-platinum/10 bg-black/25 px-3 py-2 text-[11px] text-forge-mist transition hover:border-forge-platinum/20 hover:text-forge-ash disabled:opacity-40"
                      >
                        Use auto
                      </button>
                    </div>
                  ) : null}
                </div>

                {showAdvanced ? (
                  <div className="mb-3 grid gap-2 md:grid-cols-3">
                    <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Assistant dry run"
                        type="checkbox"
                        checked={assistantDryRun}
                        disabled={!requestAssistant}
                        onChange={(e) => setAssistantDryRun(e.target.checked)}
                      />
                      Dry-run
                    </label>
                    <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Stream assistant response"
                        type="checkbox"
                        checked={streamAssistant}
                        disabled={
                          !requestAssistant ||
                          assistantDryRun ||
                          blockingAssistant
                        }
                        onChange={(e) => {
                          setStreamAssistant(e.target.checked);
                          if (e.target.checked) setBlockingAssistant(false);
                        }}
                      />
                      Stream response
                    </label>
                    <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
                      <input
                        aria-label="Block on assistant response"
                        type="checkbox"
                        checked={blockingAssistant}
                        disabled={!requestAssistant || assistantDryRun}
                        onChange={(e) => {
                          setBlockingAssistant(e.target.checked);
                          if (e.target.checked) setStreamAssistant(false);
                        }}
                      />
                      Block request
                    </label>
                  </div>
                ) : null}

                <div className="forge-chat-composer-shell">
                  {pendingAttachments.length > 0 ? (
                    <div className="mb-3">
                      <div className="mb-2 text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
                        Pending attachments
                      </div>
                      <div className="flex flex-wrap gap-2">
                        {pendingAttachments.map((item) => (
                          <span
                            key={item.artifactId}
                            className="inline-flex items-center gap-2 rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist"
                          >
                            <button
                              type="button"
                              onClick={() => {
                                setInspectorMode("files");
                                setSelectedAttachmentId(item.artifactId);
                              }}
                              className="truncate hover:text-forge-ash"
                            >
                              {item.fileName}
                            </button>
                            <button
                              type="button"
                              onClick={() =>
                                setPendingAttachments((prev) =>
                                  prev.filter(
                                    (entry) =>
                                      entry.artifactId !== item.artifactId,
                                  ),
                                )
                              }
                              className="text-forge-mist/70 hover:text-forge-ash"
                              aria-label={`Remove ${item.fileName}`}
                            >
                              ×
                            </button>
                          </span>
                        ))}
                      </div>
                    </div>
                  ) : null}

                  <div className="mb-2 flex flex-wrap gap-2">
                    <button
                      type="button"
                      onClick={() => {
                        setInspectorMode("terminal");
                        setDraft((prev) => prev || "run ");
                        window.setTimeout(
                          () => textareaRef.current?.focus(),
                          0,
                        );
                      }}
                      className="rounded-lg border border-forge-platinum/10 bg-black/35 px-3 py-1.5 text-[11px] font-semibold text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
                    >
                      Terminal
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setInspectorMode("browser");
                        setDraft((prev) => prev || "search the web for ");
                        window.setTimeout(
                          () => textareaRef.current?.focus(),
                          0,
                        );
                      }}
                      className="rounded-lg border border-forge-platinum/10 bg-black/35 px-3 py-1.5 text-[11px] font-semibold text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
                    >
                      Web search
                    </button>
                    <button
                      type="button"
                      onClick={() => {
                        setInspectorMode("browser");
                        setDraft((prev) => prev || "open browser https://");
                        window.setTimeout(
                          () => textareaRef.current?.focus(),
                          0,
                        );
                      }}
                      className="rounded-lg border border-forge-platinum/10 bg-black/35 px-3 py-1.5 text-[11px] font-semibold text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
                    >
                      Browser
                    </button>
                  </div>

                  <div className="forge-chat-compose-row">
                    <div className="min-w-0 flex-1">
                      <label htmlFor="chat-composer" className="sr-only">
                        Message
                      </label>
                      <textarea
                        id="chat-composer"
                        aria-label="Chat message"
                        ref={textareaRef}
                        rows={2}
                        className="forge-chat-composer"
                        placeholder="Message FORGE"
                        value={draft}
                        onChange={(e) => setDraft(e.target.value)}
                        onKeyDown={onComposerKeyDown}
                        disabled={busy}
                      />
                    </div>
                    <input
                      ref={fileInputRef}
                      type="file"
                      className="hidden"
                      multiple
                      onChange={(e) => void uploadSelectedFiles(e.target.files)}
                      disabled={busy || uploading}
                    />
                    <div className="forge-chat-compose-actions">
                      <button
                        type="button"
                        onClick={() => fileInputRef.current?.click()}
                        disabled={busy || uploading || !active}
                        className="forge-chat-action-btn bg-black/35"
                      >
                        {uploading ? "Uploading…" : "Attach"}
                      </button>
                      <button
                        type="button"
                        onClick={() => void send()}
                        disabled={
                          busy ||
                          (!draft.trim() && pendingAttachments.length === 0)
                        }
                        className="forge-chat-action-btn forge-chat-primary-btn text-sm"
                      >
                        Send
                      </button>
                    </div>
                  </div>
                  <div className="mt-2 flex flex-wrap items-center justify-between gap-2 text-[11px] text-forge-mist/75">
                    <span>Enter to send · Shift+Enter for newline</span>
                    <span className="hidden min-w-0 break-words sm:inline">
                      {requestAssistant
                        ? `Assistant mode: ${assistantModeSummary}`
                        : "No assistant reply requested"}
                    </span>
                  </div>
                </div>
              </div>
            </footer>
          </>
        )}
      </section>

      {active ? (
        <div
          role="separator"
          aria-label="Resize chat inspector"
          aria-orientation="vertical"
          className="forge-chat-resizer forge-chat-resizer--right"
          onPointerDown={(event) =>
            startHorizontalPaneResize("inspector", event)
          }
        />
      ) : null}

      {active ? (
        <aside className="forge-chat-inspector min-h-0 min-w-0 flex-col overflow-hidden border-l border-forge-platinum/10 bg-forge-black/95 shadow-[-8px_0_40px_rgba(0,0,0,0.28)]">
          <div className="flex flex-col gap-3 border-b border-forge-platinum/10 bg-forge-carbon/75 px-4 py-3">
            <div className="text-sm font-semibold text-forge-ash">
              Inspector
            </div>
            <div className="flex max-w-full items-center gap-2 overflow-x-auto pb-1 text-xs">
              <button
                type="button"
                onClick={() => setInspectorMode("thinking")}
                className={[
                  "shrink-0 rounded border px-2 py-1",
                  inspectorMode === "thinking"
                    ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                    : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
                ].join(" ")}
              >
                Thinking
              </button>
              <button
                type="button"
                onClick={() => setInspectorMode("terminal")}
                className={[
                  "shrink-0 rounded border px-2 py-1",
                  inspectorMode === "terminal"
                    ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                    : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
                ].join(" ")}
              >
                Terminal
              </button>
              <button
                type="button"
                onClick={() => setInspectorMode("browser")}
                className={[
                  "shrink-0 rounded border px-2 py-1",
                  inspectorMode === "browser"
                    ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                    : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
                ].join(" ")}
              >
                Browser
              </button>
              <button
                type="button"
                onClick={() => setInspectorMode("code")}
                className={[
                  "shrink-0 rounded border px-2 py-1",
                  inspectorMode === "code"
                    ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                    : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
                ].join(" ")}
              >
                Code
              </button>
              <button
                type="button"
                onClick={() => setInspectorMode("files")}
                className={[
                  "shrink-0 rounded border px-2 py-1",
                  inspectorMode === "files"
                    ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                    : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
                ].join(" ")}
              >
                Files
              </button>
            </div>
          </div>

          {inspectorMode === "thinking" ? (
            <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
              {streamingText !== null && streamingEvents.length > 0 ? (
                <div className="mb-4">
                  <ThinkingTimeline
                    events={streamingEvents.slice().reverse()}
                    live
                  />
                </div>
              ) : null}
              <ThinkingTimeline
                events={thinkingEntries}
                emptyText={
                  streamingText !== null
                    ? "FORGE has not emitted a visible thinking event yet."
                    : "No FORGE thinking trace in this thread yet."
                }
              />
            </div>
          ) : inspectorMode === "terminal" ? (
            <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
              {terminalEntries.length === 0 ? (
                <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                  No terminal runs in this thread yet.
                </div>
              ) : (
                <div className="space-y-3">
                  {terminalEntries.map((entry) => (
                    <div key={entry.key}>
                      <div className="mb-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/60">
                        message {entry.messageId} ·{" "}
                        {formatTime(entry.createdAtMs)}
                      </div>
                      <TerminalTranscript entry={entry} compact />
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : inspectorMode === "browser" ? (
            <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
              {browserEntries.length === 0 ? (
                <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                  No browser or web-search results in this thread yet.
                </div>
              ) : (
                <div className="space-y-3">
                  {browserEntries.map((entry) => (
                    <div key={entry.key}>
                      <div className="mb-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/60">
                        {entry.tool} · message {entry.messageId} ·{" "}
                        {formatTime(entry.createdAtMs)}
                      </div>
                      <BrowserResultPanel entry={entry} compact />
                    </div>
                  ))}
                </div>
              )}
            </div>
          ) : inspectorMode === "code" ? (
            <div
              className="forge-chat-inspector-split grid min-h-0 flex-1"
              style={inspectorSplitStyle}
            >
              <div className="forge-chat-scroll overflow-y-auto border-b border-forge-platinum/10 p-2">
                {assistantCodeSnippets.length === 0 ? (
                  <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                    No assistant code blocks in this thread yet.
                  </div>
                ) : (
                  assistantCodeSnippets.map((snippet) => (
                    <button
                      key={snippet.key}
                      type="button"
                      onClick={() => setSelectedSnippetKey(snippet.key)}
                      className={[
                        "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                        selectedSnippetKey === snippet.key
                          ? "border-forge-platinum/20 bg-forge-platinum/10"
                          : "border-forge-platinum/10 bg-black/20 hover:border-forge-platinum/20",
                      ].join(" ")}
                    >
                      <div className="text-[11px] font-semibold uppercase tracking-[0.12em] text-forge-ash">
                        {snippet.lang}
                      </div>
                      <div className="mt-1 line-clamp-2 font-mono text-[11px] text-forge-mist">
                        {snippet.code.trim() || "(empty code block)"}
                      </div>
                      <div className="mt-1 text-[10px] text-forge-mist/70">
                        {formatTime(snippet.createdAtMs)}
                      </div>
                    </button>
                  ))
                )}
              </div>
              <div
                role="separator"
                aria-label="Resize code inspector list"
                aria-orientation="horizontal"
                className="forge-chat-row-resizer"
                onPointerDown={startInspectorSplitResize}
              />
              <div className="forge-chat-scroll overflow-y-auto p-3">
                {assistantCodeSnippets.find(
                  (item) => item.key === selectedSnippetKey,
                ) ? (
                  <CodeBlock
                    code={
                      assistantCodeSnippets.find(
                        (item) => item.key === selectedSnippetKey,
                      )?.code ?? ""
                    }
                    lang={
                      assistantCodeSnippets.find(
                        (item) => item.key === selectedSnippetKey,
                      )?.lang ?? "code"
                    }
                  />
                ) : (
                  <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                    Select a code block.
                  </div>
                )}
              </div>
            </div>
          ) : inspectorMode === "files" ? (
            <div
              className="forge-chat-inspector-split grid min-h-0 flex-1"
              style={inspectorSplitStyle}
            >
              <div className="forge-chat-scroll overflow-y-auto border-b border-forge-platinum/10 p-2">
                {messageAttachments.length === 0 ? (
                  <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                    No files attached in this thread.
                  </div>
                ) : (
                  messageAttachments.map((item) => (
                    <button
                      key={item.attachment.artifactId}
                      type="button"
                      onClick={() =>
                        setSelectedAttachmentId(item.attachment.artifactId)
                      }
                      className={[
                        "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                        selectedAttachmentId === item.attachment.artifactId
                          ? "border-forge-platinum/20 bg-forge-platinum/10"
                          : "border-forge-platinum/10 bg-black/20 hover:border-forge-platinum/20",
                      ].join(" ")}
                    >
                      <div className="truncate text-xs font-semibold text-forge-ash">
                        {item.attachment.title}
                      </div>
                      <div className="mt-1 truncate text-[11px] text-forge-mist">
                        {item.attachment.fileName}
                      </div>
                      <div className="mt-1 text-[10px] text-forge-mist/70">
                        {item.attachment.mimeType}
                      </div>
                    </button>
                  ))
                )}
              </div>
              <div
                role="separator"
                aria-label="Resize file inspector list"
                aria-orientation="horizontal"
                className="forge-chat-row-resizer"
                onPointerDown={startInspectorSplitResize}
              />
              <div className="forge-chat-scroll overflow-y-auto p-3">
                {messageAttachments.find(
                  (item) => item.attachment.artifactId === selectedAttachmentId,
                ) ? (
                  <AttachmentInspectorCard
                    attachment={
                      messageAttachments.find(
                        (item) =>
                          item.attachment.artifactId === selectedAttachmentId,
                      )!.attachment
                    }
                  />
                ) : (
                  <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                    Select a file.
                  </div>
                )}
              </div>
            </div>
          ) : null}
        </aside>
      ) : null}
    </div>
  );
}
