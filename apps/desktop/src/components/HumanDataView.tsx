import type { ReactNode } from "react";

type HumanDataViewProps = {
  value: unknown;
  empty?: string;
  maxDepth?: number;
  compact?: boolean;
};

export function humanizeKey(key: string) {
  return key
    .replace(/[_-]+/g, " ")
    .replace(/([a-z0-9])([A-Z])/g, "$1 $2")
    .replace(/\s+/g, " ")
    .trim()
    .replace(/^./, (char) => char.toUpperCase());
}

export function summarizeHumanValue(value: unknown): string {
  if (value == null) return "None";
  if (typeof value === "string") return value.trim() || "None";
  if (typeof value === "number") return Number.isFinite(value) ? String(value) : "Invalid number";
  if (typeof value === "boolean") return value ? "Yes" : "No";
  if (Array.isArray(value)) return value.length === 0 ? "No items" : `${value.length} item${value.length === 1 ? "" : "s"}`;
  if (typeof value === "object") {
    const count = Object.keys(value as Record<string, unknown>).length;
    return count === 0 ? "No details" : `${count} field${count === 1 ? "" : "s"}`;
  }
  return String(value);
}

export function HumanDataView({ value, empty = "No details recorded.", maxDepth = 3, compact = false }: HumanDataViewProps) {
  if (isEmpty(value)) {
    return <div className="text-xs text-forge-mist/75">{empty}</div>;
  }

  return <div className={compact ? "space-y-1" : "space-y-2"}>{renderValue(value, 0, maxDepth, compact)}</div>;
}

function renderValue(value: unknown, depth: number, maxDepth: number, compact: boolean): ReactNode {
  if (isPrimitive(value)) {
    return <span className="break-words text-forge-ash">{summarizeHumanValue(value)}</span>;
  }

  if (depth >= maxDepth) {
    return <span className="text-forge-ash">{summarizeHumanValue(value)}</span>;
  }

  if (Array.isArray(value)) {
    if (value.length === 0) return <span className="text-forge-mist/75">No items</span>;
    return (
      <div className={compact ? "space-y-1" : "space-y-2"}>
        {value.map((item, index) => (
          <div key={index} className="rounded border border-white/10 bg-black/20 px-2 py-1.5">
            <div className="mb-1 text-[10px] font-semibold uppercase tracking-[0.12em] text-forge-mist/60">Item {index + 1}</div>
            {renderValue(item, depth + 1, maxDepth, compact)}
          </div>
        ))}
      </div>
    );
  }

  if (typeof value === "object" && value !== null) {
    const entries = Object.entries(value as Record<string, unknown>).filter(([, item]) => !isEmpty(item));
    if (entries.length === 0) return <span className="text-forge-mist/75">No details</span>;

    return (
      <div className={compact ? "grid gap-1" : "grid gap-2"}>
        {entries.map(([key, item]) => (
          <div key={key} className="rounded border border-white/10 bg-black/20 px-2.5 py-2">
            <div className="text-[10px] font-semibold uppercase tracking-[0.12em] text-forge-mist/65">{humanizeKey(key)}</div>
            <div className="mt-1 text-xs leading-relaxed text-forge-mist">{renderValue(item, depth + 1, maxDepth, compact)}</div>
          </div>
        ))}
      </div>
    );
  }

  return <span className="break-words text-forge-ash">{summarizeHumanValue(value)}</span>;
}

function isPrimitive(value: unknown) {
  return value == null || typeof value === "string" || typeof value === "number" || typeof value === "boolean";
}

function isEmpty(value: unknown) {
  if (value == null) return true;
  if (typeof value === "string") return value.trim() === "";
  if (Array.isArray(value)) return value.length === 0;
  if (typeof value === "object") return Object.keys(value as Record<string, unknown>).length === 0;
  return false;
}
