import { formatTime } from "../../lib/format";

export type ChatThinkingEvent = {
  key?: string;
  messageId?: number;
  at: number;
  kind: string;
  text: string;
  detail?: string;
  data?: Record<string, unknown>;
};

export function formatThinkingStage(stage: string): string {
  const labels: Record<string, string> = {
    request_received: "Request received",
    tools_skipped: "No tool route needed",
    tools_attached: "Gateway tools attached",
    dry_run: "Dry run selected",
    deterministic_combined_shortcut: "Deterministic file workflow selected",
    deterministic_python_banner_shortcut:
      "Deterministic script workflow selected",
    forge_tool_choice_enforced: "FORGE tool route enforced",
    runtime_primary: "Model runtime selected",
    runtime_fallback: "Runtime fallback selected",
    adapter_mismatch: "Adapter mismatch detected",
    ollama_chat_error: "Ollama chat failed",
    model_raw_response: "Model response received",
    tool_call_check: "Tool call check",
    tool_args: "Tool arguments received",
    backend_dispatch: "Gateway dispatch",
    execution_result: "Gateway result",
    gateway_error: "Gateway error",
    path_precheck_failed: "Path boundary rejected",
    attachment_read_resolved: "Attachment resolved",
    attachment_read_error: "Attachment read failed",
    model_prose_discarded: "Unverified model prose discarded",
    sync_completion_start: "Synchronous completion started",
    sync_completion_done: "Synchronous completion saved",
    stream_downgrade: "Streaming downgraded",
    cached_reply: "Cached reply returned",
  };
  return labels[stage] ?? stage.replaceAll("_", " ");
}

export function compactThinkingDetail(row: Record<string, unknown>): string {
  const parts: string[] = [];
  const keys = [
    "tool",
    "toolId",
    "gatewayTool",
    "laneId",
    "reason",
    "status",
    "state",
    "count",
    "turn",
    "index",
    "paths",
    "error",
    "modelId",
    "backend",
  ];
  for (const key of keys) {
    const raw = row[key];
    if (raw == null || raw === "") continue;
    if (Array.isArray(raw)) {
      if (raw.length === 0) continue;
      parts.push(`${key}: ${raw.map((item) => String(item)).join(", ")}`);
      continue;
    }
    parts.push(`${key}: ${String(raw)}`);
  }
  return parts.slice(0, 5).join(" · ");
}

export function ThinkingTimeline(props: {
  events: ChatThinkingEvent[];
  live?: boolean;
  emptyText?: string;
  compact?: boolean;
}) {
  if (props.events.length === 0) {
    return (
      <div className="rounded border border-dashed border-forge-platinum/10 px-3 py-4 text-xs text-forge-mist">
        {props.emptyText ?? "No FORGE thinking trace is available yet."}
      </div>
    );
  }
  const events = props.compact ? props.events.slice(0, 12) : props.events;
  return (
    <div className="space-y-2">
      {props.live ? (
        <div className="mb-2 flex items-center justify-between rounded-lg border border-forge-platinum/10 bg-black/25 px-3 py-2">
          <span className="text-[10px] uppercase tracking-[0.16em] text-forge-mist/70">
            Live FORGE Thinking
          </span>
          <span className="font-mono text-[10px] text-forge-platinum">
            streaming
          </span>
        </div>
      ) : null}
      {events.map((evt, index) => (
        <div
          key={evt.key ?? `${evt.kind}-${evt.at}-${index}`}
          className="rounded-lg border border-forge-platinum/10 bg-black/25 px-3 py-2"
        >
          <div className="flex items-center justify-between gap-2">
            <div className="min-w-0 truncate text-xs font-semibold text-forge-ash">
              {evt.text}
            </div>
            <div className="shrink-0 font-mono text-[10px] text-forge-mist/65">
              {formatTime(evt.at)}
            </div>
          </div>
          {evt.detail ? (
            <div className="mt-1 break-words font-mono text-[10px] leading-5 text-forge-mist/80">
              {evt.detail}
            </div>
          ) : null}
          {evt.messageId ? (
            <div className="mt-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/45">
              message {evt.messageId}
            </div>
          ) : null}
        </div>
      ))}
    </div>
  );
}
