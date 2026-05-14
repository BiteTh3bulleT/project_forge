import type { ReactNode } from "react";

export type MemoryView = "all" | "inspect" | "search" | "maintenance";

export type MaintenanceGate = {
  label: string;
  pass: boolean;
};

export function Panel(props: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel min-w-0">
      <div className="forge-ops-panel__head flex-col items-stretch sm:flex-row sm:items-center">
        <div className="min-w-0">
          <div className="forge-ops-title break-words">{props.title}</div>
          {props.subtitle ? (
            <div className="mt-1 max-w-3xl break-words text-xs leading-5 text-forge-mist/65">
              {props.subtitle}
            </div>
          ) : null}
        </div>
        {props.actions ? (
          <div className="flex flex-wrap items-center gap-2">
            {props.actions}
          </div>
        ) : null}
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}

export function EmptyState(props: {
  title: string;
  detail: string;
  tone?: "muted" | "warn" | "bad";
}) {
  const toneClass =
    props.tone === "bad"
      ? "border-forge-ember/30 bg-forge-ember/10"
      : props.tone === "warn"
        ? "border-forge-amber/30 bg-forge-amber/10"
        : "border-forge-platinum/10 bg-black/20";
  return (
    <div className={["rounded border border-dashed p-4", toneClass].join(" ")}>
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/75">
        {props.detail}
      </div>
    </div>
  );
}

export function MemoryMetric(props: {
  label: string;
  value: string | number;
  detail: string;
  tone: "ok" | "warn" | "bad" | "muted";
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 break-words text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={`forge-ops-status forge-ops-status--${props.tone}`}>
          {props.tone}
        </span>
      </div>
      <div className="mt-3 break-words text-xs leading-5 text-forge-mist/65">
        {props.detail}
      </div>
    </div>
  );
}

export function GateLine(props: {
  prefix: "IF" | "AND";
  label: string;
  pass: boolean;
}) {
  return (
    <div className="flex items-center justify-between gap-3 border-b border-forge-platinum/5 pb-1 last:border-b-0 last:pb-0">
      <div className="min-w-0 break-words font-mono text-[11px] leading-5 text-forge-mist">
        <span className="mr-2 text-forge-mist/60">{props.prefix}</span>
        {props.label}
      </div>
      <div
        className={
          props.pass
            ? "shrink-0 text-[11px] font-semibold text-forge-electric"
            : "shrink-0 text-[11px] font-semibold text-forge-emberSoft"
        }
      >
        {props.pass ? "pass" : "fail"}
      </div>
    </div>
  );
}
