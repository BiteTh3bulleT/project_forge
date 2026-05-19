import type { CSSProperties, PointerEvent as ReactPointerEvent } from "react";

import { formatTime } from "../../lib/format";
import { CodeBlock } from "./messageContent";
import { AttachmentInspectorCard } from "./messageSurface";
import {
  ThinkingTimeline,
  type ChatThinkingEvent,
} from "./thinking";
import {
  BrowserResultPanel,
  TerminalTranscript,
  type ChatToolEntry,
} from "./toolGateway";
import type {
  MessageAttachmentItem,
  MessageCodeSnippet,
} from "./useChatInspectorData";

type InspectorMode =
  | "thinking"
  | "reasoning"
  | "code"
  | "files"
  | "terminal"
  | "browser";

export function ChatInspector(props: {
  inspectorMode: InspectorMode;
  onInspectorModeChange: (mode: InspectorMode) => void;
  streamingText: string | null;
  streamingEvents: ChatThinkingEvent[];
  reasoningText: string;
  reasoningStreaming: boolean;
  thinkingEntries: ChatThinkingEvent[];
  terminalEntries: ChatToolEntry[];
  browserEntries: ChatToolEntry[];
  assistantCodeSnippets: MessageCodeSnippet[];
  selectedSnippetKey: string;
  onSelectedSnippetKeyChange: (key: string) => void;
  messageAttachments: MessageAttachmentItem[];
  selectedAttachmentId: number;
  onSelectedAttachmentIdChange: (id: number) => void;
  inspectorSplitStyle: CSSProperties;
  onStartInspectorSplitResize: (
    event: ReactPointerEvent<HTMLDivElement>,
  ) => void;
}) {
  const selectedSnippet = props.assistantCodeSnippets.find(
    (item) => item.key === props.selectedSnippetKey,
  );
  const selectedAttachment = props.messageAttachments.find(
    (item) => item.attachment.artifactId === props.selectedAttachmentId,
  );

  return (
    <aside className="forge-chat-inspector min-h-0 min-w-0 flex-col overflow-hidden border-l border-forge-platinum/10 bg-forge-black/95 shadow-[-8px_0_40px_rgba(0,0,0,0.28)]">
      <div className="flex flex-col gap-3 border-b border-forge-platinum/10 bg-forge-carbon/75 px-4 py-3">
        <div className="text-sm font-semibold text-forge-ash">Inspector</div>
        <div className="flex max-w-full items-center gap-2 overflow-x-auto pb-1 text-xs">
          {(
            [
              ["thinking", "Thinking"],
              ["reasoning", "Reasoning"],
              ["terminal", "Terminal"],
              ["browser", "Browser"],
              ["code", "Code"],
              ["files", "Files"],
            ] satisfies Array<[InspectorMode, string]>
          ).map(([mode, label]) => (
            <button
              key={mode}
              type="button"
              onClick={() => props.onInspectorModeChange(mode)}
              className={[
                "shrink-0 rounded border px-2 py-1",
                props.inspectorMode === mode
                  ? "border-forge-ember/45 bg-forge-ember/10 text-forge-emberSoft"
                  : "border-forge-platinum/10 text-forge-mist hover:border-forge-ember/30",
              ].join(" ")}
            >
              {label}
            </button>
          ))}
        </div>
      </div>

      {props.inspectorMode === "thinking" ? (
        <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
          {props.streamingText !== null && props.streamingEvents.length > 0 ? (
            <div className="mb-4">
              <ThinkingTimeline
                events={props.streamingEvents.slice().reverse()}
                live
              />
            </div>
          ) : null}
          <ThinkingTimeline
            events={props.thinkingEntries}
            emptyText={
              props.streamingText !== null
                ? "FORGE has not emitted a visible thinking event yet."
                : "No FORGE thinking trace in this thread yet."
            }
          />
        </div>
      ) : props.inspectorMode === "reasoning" ? (
        <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
          {props.reasoningText.trim() ? (
            <div className="rounded-lg border border-forge-platinum/10 bg-black/25 p-3">
              <div className="mb-2 text-[10px] uppercase tracking-[0.14em] text-forge-mist/60">
                Provider reasoning stream
              </div>
              <div className="whitespace-pre-wrap break-words font-mono text-xs leading-6 text-forge-ash">
                {props.reasoningText}
              </div>
            </div>
          ) : (
            <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
              {props.reasoningStreaming
                ? "Waiting for provider reasoning stream events."
                : "No provider reasoning has streamed in this thread yet."}
            </div>
          )}
        </div>
      ) : props.inspectorMode === "terminal" ? (
        <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
          {props.terminalEntries.length === 0 ? (
            <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
              No terminal runs in this thread yet.
            </div>
          ) : (
            <div className="space-y-3">
              {props.terminalEntries.map((entry) => (
                <div key={entry.key}>
                  <div className="mb-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/60">
                    message {entry.messageId} · {formatTime(entry.createdAtMs)}
                  </div>
                  <TerminalTranscript entry={entry} compact />
                </div>
              ))}
            </div>
          )}
        </div>
      ) : props.inspectorMode === "browser" ? (
        <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-3">
          {props.browserEntries.length === 0 ? (
            <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
              No browser or web-search results in this thread yet.
            </div>
          ) : (
            <div className="space-y-3">
              {props.browserEntries.map((entry) => (
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
      ) : props.inspectorMode === "code" ? (
        <div
          className="forge-chat-inspector-split grid min-h-0 flex-1"
          style={props.inspectorSplitStyle}
        >
          <div className="forge-chat-scroll overflow-y-auto border-b border-forge-platinum/10 p-2">
            {props.assistantCodeSnippets.length === 0 ? (
              <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                No assistant code blocks in this thread yet.
              </div>
            ) : (
              props.assistantCodeSnippets.map((snippet) => (
                <button
                  key={snippet.key}
                  type="button"
                  onClick={() => props.onSelectedSnippetKeyChange(snippet.key)}
                  className={[
                    "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                    props.selectedSnippetKey === snippet.key
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
            onPointerDown={props.onStartInspectorSplitResize}
          />
          <div className="forge-chat-scroll overflow-y-auto p-3">
            {selectedSnippet ? (
              <CodeBlock
                code={selectedSnippet.code}
                lang={selectedSnippet.lang}
              />
            ) : (
              <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                Select a code block.
              </div>
            )}
          </div>
        </div>
      ) : props.inspectorMode === "files" ? (
        <div
          className="forge-chat-inspector-split grid min-h-0 flex-1"
          style={props.inspectorSplitStyle}
        >
          <div className="forge-chat-scroll overflow-y-auto border-b border-forge-platinum/10 p-2">
            {props.messageAttachments.length === 0 ? (
              <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
                No files attached in this thread.
              </div>
            ) : (
              props.messageAttachments.map((item) => (
                <button
                  key={item.attachment.artifactId}
                  type="button"
                  onClick={() =>
                    props.onSelectedAttachmentIdChange(
                      item.attachment.artifactId,
                    )
                  }
                  className={[
                    "mb-2 w-full rounded-lg border px-3 py-2 text-left",
                    props.selectedAttachmentId === item.attachment.artifactId
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
            onPointerDown={props.onStartInspectorSplitResize}
          />
          <div className="forge-chat-scroll overflow-y-auto p-3">
            {selectedAttachment ? (
              <AttachmentInspectorCard
                attachment={selectedAttachment.attachment}
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
  );
}
