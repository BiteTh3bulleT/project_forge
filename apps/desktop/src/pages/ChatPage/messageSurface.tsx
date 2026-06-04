import { memo } from "react";
import { Link } from "react-router-dom";

import type { ChatAttachment, ChatMessage } from "../../lib/api";
import { formatTime } from "../../lib/format";
import { RichMessage } from "./messageContent";
import {
  readAttachments,
  readCorrelationId,
  readJobId,
  readTraceId,
} from "./messageMetadata";
import {
  ToolGatewayActivityPanel,
  readToolGatewayActivity,
} from "./toolGateway";

export const MessageRow = memo(function MessageRow(props: {
  message: ChatMessage;
  onInspectAttachment: (artifactId: number) => void;
}) {
  const { message } = props;
  const role = message.role.toLowerCase();
  const isUser = role === "user";
  const isAssistant = role === "assistant";
  const jobId = readJobId(message.metadata);
  const correlationId = readCorrelationId(message.metadata);
  const traceId = readTraceId(message.metadata);
  const attachments = readAttachments(message.metadata);
  const toolActivity = isAssistant
    ? readToolGatewayActivity(message.metadata)
    : null;

  if (role === "system") {
    return (
      <div className="rounded-xl border border-forge-platinum/10 bg-forge-platinum/5 px-4 py-3 text-xs text-forge-mist">
        <div className="font-semibold uppercase tracking-wide text-forge-mist/80">
          System
        </div>
        <div className="mt-2 whitespace-pre-wrap break-words text-forge-ash">
          {message.content}
        </div>
        <div className="mt-2 text-[10px] text-forge-mist/60">
          {formatTime(message.createdAtMs)}
        </div>
      </div>
    );
  }

  return (
    <div
      className={`flex min-w-0 max-w-full ${isUser ? "justify-end" : "justify-start"}`}
    >
      <div
        className={[
          "min-w-0 max-w-full flex-1 md:max-w-[46rem]",
          isUser ? "md:max-w-[42rem]" : "",
        ].join(" ")}
      >
        <div
          className={[
            "forge-message-card min-w-0 overflow-hidden rounded-xl border px-4 py-3",
            isUser
              ? "border-forge-platinum/15 bg-forge-platinum/10 text-forge-ash"
              : "border-forge-platinum/10 bg-forge-carbon text-forge-ash",
          ].join(" ")}
        >
          <div className="mb-2 flex items-center justify-between gap-3">
            <div className="flex flex-wrap items-center gap-2">
              <div className="text-[10px] uppercase tracking-wide text-forge-mist/80">
                {isUser ? "You" : isAssistant ? "FORGE" : role}
              </div>
              {toolActivity ? (
                <span className="rounded-full border border-forge-electric/20 bg-forge-electric/10 px-2 py-0.5 text-[10px] uppercase tracking-[0.14em] text-forge-electric">
                  Gateway activity
                </span>
              ) : null}
            </div>
            <div className="text-[10px] text-forge-mist/65">
              {formatTime(message.createdAtMs)}
            </div>
          </div>
          <RichMessage content={message.content} />
          {toolActivity ? (
            <ToolGatewayActivityPanel activity={toolActivity} />
          ) : null}
          {attachments.length > 0 ? (
            <div className="mt-3 border-t border-forge-platinum/10 pt-2">
              <div className="mb-1 text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
                Attachments
              </div>
              <div className="flex flex-wrap gap-2">
                {attachments.map((attachment) => (
                  <button
                    key={attachment.artifactId}
                    type="button"
                    onClick={() =>
                      props.onInspectAttachment(attachment.artifactId)
                    }
                    className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                  >
                    {attachment.fileName}
                  </button>
                ))}
              </div>
            </div>
          ) : null}
          {jobId || correlationId || traceId ? (
            <div className="mt-3 border-t border-forge-platinum/10 pt-2 text-xs text-forge-mist">
              <div className="mb-2 text-[10px] uppercase tracking-[0.14em] text-forge-mist/65">
                Traceability
              </div>
              <div className="flex flex-wrap gap-2">
                {jobId ? (
                  <Link
                    to={`/jobs/${encodeURIComponent(jobId)}`}
                    className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                  >
                    Job {jobId}
                  </Link>
                ) : null}
                {correlationId ? (
                  <>
                    <Link
                      to={`/audit?correlationId=${encodeURIComponent(correlationId)}`}
                      className="rounded-full border border-forge-electric/20 bg-forge-electric/10 px-2.5 py-1 text-[11px] text-forge-electric transition hover:text-forge-ash"
                    >
                      Audit {correlationId}
                    </Link>
                    <Link
                      to={`/inspectors?correlationId=${encodeURIComponent(correlationId)}`}
                      className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                    >
                      Correlation {correlationId}
                    </Link>
                  </>
                ) : null}
                {traceId ? (
                  <>
                    <Link
                      to={`/audit?traceId=${encodeURIComponent(traceId)}`}
                      className="rounded-full border border-forge-electric/20 bg-forge-electric/10 px-2.5 py-1 text-[11px] text-forge-electric transition hover:text-forge-ash"
                    >
                      Audit {traceId}
                    </Link>
                    <Link
                      to={`/inspectors?traceId=${encodeURIComponent(traceId)}`}
                      className="rounded-full border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist transition hover:text-forge-ash"
                    >
                      Trace {traceId}
                    </Link>
                  </>
                ) : null}
              </div>
            </div>
          ) : null}
        </div>
      </div>
    </div>
  );
});

export function AttachmentInspectorCard(props: { attachment: ChatAttachment }) {
  return (
    <div className="space-y-3 rounded-xl border border-forge-platinum/10 bg-forge-carbon p-3">
      <div>
        <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/70">
          Attachment
        </div>
        <div className="mt-1 text-sm font-semibold text-forge-ash">
          {props.attachment.title}
        </div>
      </div>
      <div className="text-[11px] text-forge-mist">
        <div className="truncate">{props.attachment.fileName}</div>
        <div className="mt-1">{props.attachment.mimeType}</div>
      </div>
      <div className="flex flex-wrap gap-2">
        <Link
          to={`/workbench?artifactId=${encodeURIComponent(String(props.attachment.artifactId))}`}
          className="rounded border border-forge-platinum/15 bg-forge-platinum/5 px-2.5 py-1 text-[11px] text-forge-mist hover:text-forge-ash"
        >
          Open in Workbench
        </Link>
      </div>
      {props.attachment.textPreview ? (
        <pre className="forge-chat-scroll max-h-[420px] overflow-auto whitespace-pre-wrap rounded border border-forge-platinum/10 bg-black/25 p-2 font-mono text-[11px] text-forge-mist">
          {props.attachment.textPreview}
        </pre>
      ) : (
        <div className="text-[11px] text-forge-mist/75">
          No inline preview for this file type.
        </div>
      )}
    </div>
  );
}
