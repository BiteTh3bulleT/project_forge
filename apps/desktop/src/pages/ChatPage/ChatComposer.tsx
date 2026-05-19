import type {
  Dispatch,
  KeyboardEvent,
  RefObject,
  SetStateAction,
} from "react";
import { Link } from "react-router-dom";

import type { ChatAttachment, ModelRuntimeModel } from "../../lib/api";

type InspectorMode = "thinking" | "code" | "files" | "terminal" | "browser";

export function ChatComposer(props: {
  pendingAttachments: ChatAttachment[];
  onPendingAttachmentsChange: Dispatch<SetStateAction<ChatAttachment[]>>;
  onInspectorModeChange: (mode: InspectorMode) => void;
  onSelectedAttachmentIdChange: (id: number) => void;
  onDraftChange: Dispatch<SetStateAction<string>>;
  textareaRef: RefObject<HTMLTextAreaElement>;
  fileInputRef: RefObject<HTMLInputElement>;
  busy: boolean;
  uploading: boolean;
  draft: string;
  onComposerKeyDown: (event: KeyboardEvent<HTMLTextAreaElement>) => void;
  onUploadSelectedFiles: (list: FileList | null) => void;
  onSend: () => void;
  requestAssistant: boolean;
  onRequestAssistantChange: (value: boolean) => void;
  assistantModeSummary: string;
  chatModelMessage: string;
  chatModelSummary: string;
  onRefreshChatModels: () => void;
  showAdvanced: boolean;
  onShowAdvancedChange: Dispatch<SetStateAction<boolean>>;
  selectedChatModelId: string;
  onSelectedChatModelIdChange: (id: string) => void;
  chatModelLoadState: "idle" | "loading" | "ready" | "unavailable" | "error";
  chatModels: ModelRuntimeModel[];
  assistantDryRun: boolean;
  onAssistantDryRunChange: (value: boolean) => void;
  streamAssistant: boolean;
  onStreamAssistantChange: (value: boolean) => void;
  blockingAssistant: boolean;
  onBlockingAssistantChange: (value: boolean) => void;
}) {
  return (
    <footer className="forge-chat-footer border-t border-forge-platinum/10 bg-forge-black/95">
      <div className="forge-chat-content-width mx-auto w-full">
        <div
          className="forge-chat-routing-strip mb-2 rounded-lg border border-forge-platinum/10 bg-black/30 p-2"
          role="status"
          aria-live="polite"
        >
          <div className="flex flex-wrap items-center justify-between gap-2">
            <div className="min-w-0 flex-1">
              <div className="text-[10px] uppercase tracking-[0.16em] text-forge-mist/60">
                Assistant Routing
              </div>
              <div className="mt-0.5 truncate text-xs text-forge-mist/80">
                {props.assistantModeSummary} ·{" "}
                {props.chatModelMessage || props.chatModelSummary}
              </div>
            </div>
            <div className="flex min-w-0 flex-wrap items-center gap-2">
              <label className="inline-flex min-w-0 items-center gap-2 rounded-lg border border-forge-platinum/10 bg-black/20 px-2.5 py-1.5 text-xs text-forge-mist">
                <input
                  aria-label="Use assistant"
                  type="checkbox"
                  checked={props.requestAssistant}
                  onChange={(e) =>
                    props.onRequestAssistantChange(e.target.checked)
                  }
                />
                Use assistant
              </label>
              <button
                type="button"
                onClick={props.onRefreshChatModels}
                aria-label="Refresh chat models"
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
                onClick={() => props.onShowAdvancedChange((v) => !v)}
                aria-expanded={props.showAdvanced}
                className="forge-chat-action-btn border-forge-platinum/10 bg-transparent py-1 px-2.5 text-[11px]"
              >
                {props.showAdvanced ? "Hide advanced" : "Advanced"}
              </button>
            </div>
          </div>

          {props.showAdvanced ? (
            <div className="mt-3 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto]">
              <label className="block min-w-0">
                <span className="text-[11px] font-medium uppercase tracking-wide text-forge-mist">
                  Chat model
                </span>
                <select
                  aria-label="Chat runtime model"
                  className="forge-input mt-1 h-10 w-full py-1 text-sm"
                  value={props.selectedChatModelId}
                  onChange={(e) =>
                    props.onSelectedChatModelIdChange(e.target.value)
                  }
                  disabled={props.chatModelLoadState === "loading"}
                >
                  <option value="">
                    Auto (runtime default / adapter fallback)
                  </option>
                  {props.selectedChatModelId &&
                  !props.chatModels.some(
                    (model) => model.id === props.selectedChatModelId,
                  ) ? (
                    <option value={props.selectedChatModelId}>
                      Saved: {props.selectedChatModelId} (not in current runtime
                      list)
                    </option>
                  ) : null}
                  {props.chatModels.map((model) => (
                    <option key={model.id} value={model.id}>
                      {model.displayName?.trim() || model.id}
                    </option>
                  ))}
                </select>
              </label>
              <button
                type="button"
                onClick={() => props.onSelectedChatModelIdChange("")}
                disabled={!props.selectedChatModelId}
                className="rounded-lg border border-forge-platinum/10 bg-black/25 px-3 py-2 text-[11px] text-forge-mist transition hover:border-forge-platinum/20 hover:text-forge-ash disabled:opacity-40"
              >
                Use auto
              </button>
            </div>
          ) : null}
        </div>

        {props.showAdvanced ? (
          <div className="mb-3 grid gap-2 md:grid-cols-3">
            <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
              <input
                aria-label="Assistant dry run"
                type="checkbox"
                checked={props.assistantDryRun}
                disabled={!props.requestAssistant}
                onChange={(e) =>
                  props.onAssistantDryRunChange(e.target.checked)
                }
              />
              Dry-run
            </label>
            <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
              <input
                aria-label="Stream assistant response"
                type="checkbox"
                checked={props.streamAssistant}
                disabled={
                  !props.requestAssistant ||
                  props.assistantDryRun ||
                  props.blockingAssistant
                }
                onChange={(e) => {
                  props.onStreamAssistantChange(e.target.checked);
                  if (e.target.checked) props.onBlockingAssistantChange(false);
                }}
              />
              Stream response
            </label>
            <label className="inline-flex items-center gap-2 rounded border border-forge-platinum/10 bg-black/25 px-2.5 py-2 text-xs text-forge-mist">
              <input
                aria-label="Block on assistant response"
                type="checkbox"
                checked={props.blockingAssistant}
                disabled={!props.requestAssistant || props.assistantDryRun}
                onChange={(e) => {
                  props.onBlockingAssistantChange(e.target.checked);
                  if (e.target.checked) props.onStreamAssistantChange(false);
                }}
              />
              Block request
            </label>
          </div>
        ) : null}

        <div className="forge-chat-composer-shell">
          {props.pendingAttachments.length > 0 ? (
            <div className="mb-3">
              <div className="mb-2 text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
                Pending attachments
              </div>
              <div className="flex flex-wrap gap-2">
                {props.pendingAttachments.map((item) => (
                  <span
                    key={item.artifactId}
                    className="inline-flex items-center gap-2 rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist"
                  >
                    <button
                      type="button"
                      onClick={() => {
                        props.onInspectorModeChange("files");
                        props.onSelectedAttachmentIdChange(item.artifactId);
                      }}
                      className="truncate hover:text-forge-ash"
                    >
                      {item.fileName}
                    </button>
                    <button
                      type="button"
                      onClick={() =>
                        props.onPendingAttachmentsChange((prev) =>
                          prev.filter(
                            (entry) => entry.artifactId !== item.artifactId,
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
              aria-label="Prepare terminal command"
              onClick={() => {
                props.onInspectorModeChange("terminal");
                props.onDraftChange((prev) => prev || "run ");
                window.setTimeout(() => props.textareaRef.current?.focus(), 0);
              }}
              className="rounded-lg border border-forge-platinum/10 bg-black/35 px-3 py-1.5 text-[11px] font-semibold text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
            >
              Terminal
            </button>
            <button
              type="button"
              aria-label="Prepare web search"
              onClick={() => {
                props.onInspectorModeChange("browser");
                props.onDraftChange((prev) => prev || "search the web for ");
                window.setTimeout(() => props.textareaRef.current?.focus(), 0);
              }}
              className="rounded-lg border border-forge-platinum/10 bg-black/35 px-3 py-1.5 text-[11px] font-semibold text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
            >
              Web search
            </button>
            <button
              type="button"
              aria-label="Prepare browser command"
              onClick={() => {
                props.onInspectorModeChange("browser");
                props.onDraftChange((prev) => prev || "open browser https://");
                window.setTimeout(() => props.textareaRef.current?.focus(), 0);
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
                ref={props.textareaRef}
                rows={2}
                className="forge-chat-composer"
                placeholder="Message FORGE"
                value={props.draft}
                onChange={(e) => props.onDraftChange(e.target.value)}
                onKeyDown={props.onComposerKeyDown}
                disabled={props.busy}
              />
            </div>
            <input
              ref={props.fileInputRef}
              type="file"
              aria-label="Attach chat files"
              className="hidden"
              multiple
              onChange={(e) => props.onUploadSelectedFiles(e.target.files)}
              disabled={props.busy || props.uploading}
            />
            <div className="forge-chat-compose-actions">
              <button
                type="button"
                onClick={() => props.fileInputRef.current?.click()}
                disabled={props.busy || props.uploading}
                aria-label="Attach files"
                className="forge-chat-action-btn bg-black/35"
              >
                {props.uploading ? "Uploading…" : "Attach"}
              </button>
              <button
                type="button"
                onClick={props.onSend}
                aria-label="Send chat message"
                disabled={
                  props.busy ||
                  (!props.draft.trim() && props.pendingAttachments.length === 0)
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
              {props.requestAssistant
                ? `Assistant mode: ${props.assistantModeSummary}`
                : "No assistant reply requested"}
            </span>
          </div>
        </div>
      </div>
    </footer>
  );
}
