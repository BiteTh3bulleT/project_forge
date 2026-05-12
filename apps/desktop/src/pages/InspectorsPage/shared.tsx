import type { ReactNode } from "react";
import { Link } from "react-router-dom";

import { HumanDataView } from "../../components/HumanDataView";
import type { ProcessHealthInvocation } from "../../lib/api";

export function parseProcessRuntimeLine(items: ProcessHealthInvocation[]) {
  const parseMs = (value: number | undefined) =>
    value == null || value <= 0 ? "—" : `${value}ms`;
  return items.map((invocation) => (
    <tr
      key={invocation.invocationId}
      className="border-b border-white/10 last:border-b-0"
    >
      <td className="px-2 py-1.5 text-[11px]">{invocation.invocationId}</td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.toolId}</td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.action}</td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.domain}</td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.status}</td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.policyOutcome}</td>
      <td className="px-2 py-1.5 text-[11px]">
        {parseMs(invocation.durationMs)}
      </td>
      <td className="px-2 py-1.5 text-[11px]">{invocation.traceId || "—"}</td>
    </tr>
  ));
}

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
  tone?: "muted" | "bad";
}) {
  const toneClass =
    props.tone === "bad"
      ? "border-forge-ember/30 bg-forge-ember/10"
      : "border-forge-platinum/10 bg-black/20";
  return (
    <div className={["rounded border border-dashed p-4", toneClass].join(" ")}>
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 break-words text-xs leading-5 text-forge-mist/75">
        {props.detail}
      </div>
    </div>
  );
}

export function JsonBlock(props: {
  value: unknown;
  empty?: string;
  maxHeightClass?: string;
}) {
  if (
    props.value == null ||
    (Array.isArray(props.value) && props.value.length === 0) ||
    (typeof props.value === "object" &&
      props.value !== null &&
      Object.keys(props.value as Record<string, unknown>).length === 0)
  ) {
    return (
      <div className="text-xs text-forge-mist/75">
        {props.empty ?? "No recorded evidence."}
      </div>
    );
  }
  return (
    <div
      className={[
        "overflow-auto rounded border border-forge-platinum/10 bg-black/25 p-3 text-[11px] text-forge-mist",
        props.maxHeightClass ?? "max-h-[360px]",
      ].join(" ")}
    >
      <HumanDataView value={props.value} compact />
    </div>
  );
}

export function MetricChip(props: { label: string; value: string | number }) {
  return (
    <div className="min-w-0 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2">
      <div className="forge-ops-label">{props.label}</div>
      <div className="mt-1 break-words text-sm font-semibold text-forge-ash">
        {props.value}
      </div>
    </div>
  );
}

export function SummaryLink(props: { to: string; label: string }) {
  return (
    <Link
      to={props.to}
      className="min-w-0 break-all rounded border border-forge-platinum/10 bg-black/20 px-2.5 py-1 text-[11px] leading-5 text-forge-mist transition hover:border-forge-ember/30 hover:text-forge-ash"
    >
      {props.label}
    </Link>
  );
}

export function OperatorStepCard(props: {
  step: string;
  title: string;
  detail: string;
}) {
  return (
    <div className="forge-ops-card min-w-0 p-3">
      <div className="text-[10px] font-semibold uppercase tracking-[0.12em] text-forge-emberSoft">
        {props.step}
      </div>
      <div className="mt-1 break-words text-sm font-semibold text-forge-ash">
        {props.title}
      </div>
      <div className="mt-1 break-words text-xs leading-5 text-forge-mist/80">
        {props.detail}
      </div>
    </div>
  );
}

export function CountPill(props: { label: string; value: string | number }) {
  return (
    <div className="max-w-full rounded border border-forge-platinum/10 bg-black/20 px-2.5 py-1 text-left text-[11px] leading-5 text-forge-mist">
      <span className="text-forge-mist/65">{props.label}</span>{" "}
      <span className="min-w-0 break-all">{props.value}</span>
    </div>
  );
}

export function InspectorMetric(props: {
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
