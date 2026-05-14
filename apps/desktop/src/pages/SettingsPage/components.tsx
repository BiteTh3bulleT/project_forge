import type { ReactNode } from "react";

import type { MetricTone } from "./types";

export function RemoteStateChip(props: {
  label: string;
  ok: boolean;
  okText: string;
  offText: string;
}) {
  return (
    <div className="forge-ops-card px-3 py-2 text-[11px] text-forge-mist">
      <div className="forge-ops-label">{props.label}</div>
      <div
        className={
          props.ok
            ? "mt-1 font-semibold text-forge-ash"
            : "mt-1 font-semibold text-forge-emberSoft"
        }
      >
        {props.ok ? props.okText : props.offText}
      </div>
    </div>
  );
}

export function MetricTile(props: {
  label: string;
  value: string;
  detail: string;
  tone: MetricTone;
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 truncate text-xs text-forge-mist/65">
        {props.detail}
      </div>
    </div>
  );
}

export function MiniEmpty(props: { title: string; detail: string }) {
  return (
    <div className="mt-2 rounded border border-dashed border-white/10 bg-black/20 p-3 text-xs">
      <div className="font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 leading-5 text-forge-mist/70">{props.detail}</div>
    </div>
  );
}

function statusClass(tone: MetricTone) {
  if (tone === "ok") return "forge-ops-status forge-ops-status--ok";
  if (tone === "warn") return "forge-ops-status forge-ops-status--warn";
  if (tone === "bad") return "forge-ops-status forge-ops-status--bad";
  return "forge-ops-status forge-ops-status--muted";
}

export function StatusDot(props: { ok: boolean; label: string }) {
  return (
    <div className="inline-flex items-center gap-2 text-[11px]">
      <span
        className={
          props.ok
            ? "h-2 w-2 rounded-full bg-forge-electric"
            : "h-2 w-2 rounded-full bg-forge-emberSoft"
        }
      />
      <span className={props.ok ? "text-forge-ash" : "text-forge-emberSoft"}>
        {props.label}
      </span>
    </div>
  );
}

export function StatusRow(props: {
  label: string;
  value: string;
  tone?: "normal" | "warn";
}) {
  return (
    <div className="flex items-start justify-between gap-4 border-b border-forge-platinum/5 pb-1">
      <span className="text-forge-mist/75">{props.label}</span>
      <span
        className={
          props.tone === "warn"
            ? "text-right text-forge-emberSoft"
            : "text-right text-forge-ash"
        }
      >
        {props.value}
      </span>
    </div>
  );
}

export function MetricRow(props: { label: string; value: string }) {
  return (
    <div className="flex flex-wrap items-start justify-between gap-2 rounded border border-forge-platinum/10 bg-black/25 px-3 py-2 text-[11px] text-forge-mist">
      <span className="text-forge-mist/65">{props.label}</span>
      <span className="text-right text-forge-ash">{props.value}</span>
    </div>
  );
}

export function Panel(props: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          {props.subtitle ? (
            <div className="mt-1 text-xs text-forge-mist/65">
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
