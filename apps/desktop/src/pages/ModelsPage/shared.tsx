import { summarizeHumanValue } from "../../components/HumanDataView";
import type {
  ModelRuntimeModel,
  ModelRuntimeUsageSummary,
} from "../../lib/api";

const CONTROL_ACTOR = "operator";
const CONTROL_SOURCE = "desktop";
const MODEL_MANAGEMENT_CAPABILITY = "model.management";
export const CHAT_MODEL_SELECTION_CACHE_KEY =
  "forge.chat.requestedModelId.v1";

export function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export function normalizeStatus(status?: string) {
  return (status ?? "unknown").trim().toLowerCase();
}

export function badgeClass(status?: string) {
  switch (normalizeStatus(status)) {
    case "loaded":
    case "verified":
    case "available":
    case "ok":
    case "healthy":
      return "forge-ops-status forge-ops-status--ok";
    case "loading":
    case "unloading":
    case "imported":
      return "forge-ops-status forge-ops-status--warn";
    case "disabled":
    case "archived":
      return "forge-ops-status forge-ops-status--muted";
    case "error":
    case "unavailable":
      return "forge-ops-status forge-ops-status--bad";
    default:
      return "forge-ops-status forge-ops-status--muted";
  }
}

export function summarizeList(values?: string[]) {
  if (!Array.isArray(values) || values.length === 0) return "none";
  return values.join(", ");
}

export function summarizeValue(value: unknown) {
  const summary = summarizeHumanValue(value);
  return summary === "None" ? "—" : summary;
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
    <div className={cx("rounded border border-dashed p-4", toneClass)}>
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/75">
        {props.detail}
      </div>
    </div>
  );
}

export function supportsChatCapability(model: ModelRuntimeModel) {
  const capabilities = Array.isArray(model.capabilities)
    ? model.capabilities
    : [];
  if (capabilities.length === 0) return true;
  return capabilities.some((capability) => {
    const normalized = String(capability).trim().toLowerCase();
    return normalized === "chat" || normalized === "completion";
  });
}

export function usableChatStatus(model: ModelRuntimeModel) {
  const status = normalizeStatus(model.status);
  return (
    status !== "disabled" &&
    status !== "archived" &&
    status !== "unavailable" &&
    status !== "error"
  );
}

export function readCachedChatModelSelection() {
  if (typeof window === "undefined") return "";
  try {
    return (
      window.localStorage.getItem(CHAT_MODEL_SELECTION_CACHE_KEY) ?? ""
    ).trim();
  } catch {
    return "";
  }
}

export function writeCachedChatModelSelection(value: string) {
  if (typeof window === "undefined") return;
  try {
    const trimmed = value.trim();
    if (!trimmed) {
      window.localStorage.removeItem(CHAT_MODEL_SELECTION_CACHE_KEY);
      return;
    }
    window.localStorage.setItem(CHAT_MODEL_SELECTION_CACHE_KEY, trimmed);
  } catch {
    return;
  }
}

export function modelManagementRequest(
  metadata?: Record<string, unknown>,
  approvalId?: string,
) {
  const request = {
    actor: CONTROL_ACTOR,
    source: CONTROL_SOURCE,
    capabilityId: MODEL_MANAGEMENT_CAPABILITY,
    metadata,
  };
  const normalizedApprovalId = approvalId?.trim();
  if (!normalizedApprovalId) return request;
  return {
    ...request,
    approvalId: normalizedApprovalId,
  };
}

export type ModelGovernanceDecision = {
  requiresApproval?: boolean;
  approved?: boolean;
  dryRun?: boolean;
  approvalRequestId?: number;
  operation?: string;
  reason?: string;
};

export function modelGovernanceDecision(
  payload: unknown,
): ModelGovernanceDecision | null {
  if (!payload || typeof payload !== "object") return null;
  const governance = (payload as { governance?: unknown }).governance;
  if (!governance || typeof governance !== "object") return null;
  return governance as ModelGovernanceDecision;
}

export function modelGovernanceMessage(payload: unknown, label: string) {
  const decision = modelGovernanceDecision(payload);
  if (!decision) return null;
  const approvalId =
    typeof decision.approvalRequestId === "number"
      ? ` #${decision.approvalRequestId}`
      : "";
  if (decision.requiresApproval && !decision.approved) {
    return `${label} requires approval${approvalId}.`;
  }
  if (decision.dryRun) {
    return `${label} dry run completed.`;
  }
  return null;
}

export function emptyModelRuntimeUsage(): ModelRuntimeUsageSummary {
  return {
    registered: 0,
    imported: 0,
    verified: 0,
    available: 0,
    disabled: 0,
    archived: 0,
    loaded: 0,
    queueDepth: 0,
    running: 0,
    completed: 0,
    backends: {},
  };
}

export function Metric(props: {
  label: string;
  value: string | number;
  hint: string;
  tone?: "ok" | "warn" | "bad" | "muted";
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 break-words text-xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={opsToneClass(props.tone ?? "muted")}>
          {props.tone ?? "muted"}
        </span>
      </div>
      <div className="mt-3 break-words text-xs leading-5 text-forge-mist/65">
        {props.hint}
      </div>
    </div>
  );
}

export function StateBox(props: { title: string; rows: Array<[string, string]> }) {
  return (
    <div className="rounded border border-forge-platinum/10 bg-black/25 p-3">
      <div className="forge-ops-title text-sm">{props.title}</div>
      <div className="mt-2 grid gap-2">
        {props.rows.map(([label, value]) => (
          <div
            key={label}
            className="flex items-start justify-between gap-3 text-[11px] text-forge-mist"
          >
            <span className="shrink-0">{label}</span>
            <span className="min-w-0 break-words text-right text-forge-ash">
              {value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

function opsToneClass(tone: "ok" | "warn" | "bad" | "muted") {
  if (tone === "ok") return "forge-ops-status forge-ops-status--ok";
  if (tone === "warn") return "forge-ops-status forge-ops-status--warn";
  if (tone === "bad") return "forge-ops-status forge-ops-status--bad";
  return "forge-ops-status forge-ops-status--muted";
}
