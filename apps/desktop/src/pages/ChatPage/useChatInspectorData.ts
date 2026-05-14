import { useMemo } from "react";

import type { ChatAttachment, ChatMessage, ChatThreadDetail } from "../../lib/api";
import { parseMessageParts } from "./messageContent";
import { readAttachments } from "./messageMetadata";
import {
  compactThinkingDetail,
  formatThinkingStage,
  type ChatThinkingEvent,
} from "./thinking";
import {
  isBrowserTool,
  isTerminalTool,
  numberField,
  readToolEntries,
  readToolGatewayActivity,
  type ChatToolEntry,
} from "./toolGateway";

export type MessageAttachmentItem = {
  messageId: number;
  createdAtMs: number;
  role: string;
  attachment: ChatAttachment;
};

export type MessageCodeSnippet = {
  key: string;
  messageId: number;
  createdAtMs: number;
  lang: string;
  code: string;
};

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === "object" && !Array.isArray(v)) {
    return v as Record<string, unknown>;
  }
  return null;
}

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

export function useChatInspectorData(active: ChatThreadDetail | null) {
  const messageAttachments = useMemo<MessageAttachmentItem[]>(() => {
    const out: MessageAttachmentItem[] = [];
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

  return {
    messageAttachments,
    assistantCodeSnippets,
    terminalEntries,
    browserEntries,
    thinkingEntries,
  };
}
