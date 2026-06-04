import type { ReactNode } from "react";
import { Link } from "react-router-dom";

type TraceAuditTarget = {
  kind: "correlation" | "trace";
  id: string;
};

function recordFromUnknown(value: unknown): Record<string, unknown> {
  if (value && typeof value === "object" && !Array.isArray(value)) {
    return value as Record<string, unknown>;
  }
  return {};
}

function stringField(record: Record<string, unknown>, keys: string[]) {
  for (const key of keys) {
    const value = record[key];
    if (typeof value === "string" && value.trim()) return value.trim();
  }
  return "";
}

export function traceAuditTargetFrom(value: unknown): TraceAuditTarget | null {
  const record = recordFromUnknown(value);
  const traceId = stringField(record, ["traceId", "trace_id"]);
  if (traceId) return { kind: "trace", id: traceId };

  const correlationId = stringField(record, [
    "correlationId",
    "correlation_id",
  ]);
  if (correlationId) return { kind: "correlation", id: correlationId };

  return null;
}

export function auditJobHref(jobId: string) {
  return `/audit?jobId=${encodeURIComponent(jobId)}`;
}

export function auditTraceHref(target: TraceAuditTarget) {
  const key = target.kind === "trace" ? "traceId" : "correlationId";
  return `/audit?${key}=${encodeURIComponent(target.id)}`;
}

export function AuditJobLink(props: {
  jobId: string;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <Link
      className={props.className ?? "text-forge-emberSoft underline"}
      to={auditJobHref(props.jobId)}
    >
      {props.children ?? `Audit ${props.jobId}`}
    </Link>
  );
}

export function AuditTraceLink(props: {
  target: TraceAuditTarget;
  children?: ReactNode;
  className?: string;
}) {
  return (
    <Link
      className={props.className ?? "text-forge-emberSoft underline"}
      to={auditTraceHref(props.target)}
    >
      {props.children ?? `Audit ${props.target.id}`}
    </Link>
  );
}
