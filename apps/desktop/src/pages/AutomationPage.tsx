import type { AutomationHistory, AutomationRule } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

export function AutomationPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [history, setHistory] = useState<AutomationHistory[]>([]);
  const [selectedRuleId, setSelectedRuleId] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [trigger, setTrigger] = useState("");
  const [condition, setCondition] = useState("always: yes");
  const [action, setAction] = useState("type: create_review");
  const [scope, setScope] = useState("");
  const [enabled, setEnabled] = useState(true);
  const [dryRunDefault, setDryRunDefault] = useState(true);
  const [runDry, setRunDry] = useState(true);
  const [runResult, setRunResult] = useState<Record<string, unknown> | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [rs, hs] = await Promise.all([
        api.automation.listRules({ limit: 200 }),
        api.automation.history(150),
      ]);
      setRules(rs.rules);
      setHistory(hs.history);
      setErr(null);
      if (selectedRuleId == null && rs.rules.length > 0) {
        selectRule(rs.rules[0]);
      }
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  function selectRule(rule: AutomationRule) {
    setSelectedRuleId(rule.id);
    setName(rule.name);
    setTrigger(rule.trigger);
    setCondition(mapToReadableText(rule.condition));
    setAction(mapToReadableText(rule.action));
    setScope(mapToReadableText(rule.scope));
    setEnabled(rule.enabled);
    setDryRunDefault(rule.dryRunDefault);
  }

  useEffect(() => {
    void load();
  }, []);

  return (
    <div className="space-y-6">
      <Panel
        title="Automation Rules"
        subtitle="Bounded automations with trigger + condition + action contracts, history, and dry-run preview support."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}

        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Rule name</label>
            <input className="forge-input mt-1" value={name} onChange={(e) => setName(e.target.value)} placeholder="Queue review after import" />
          </div>
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Trigger</label>
            <input className="forge-input mt-1" value={trigger} onChange={(e) => setTrigger(e.target.value)} placeholder="import.execution.created" />
          </div>
          <div className="flex items-end gap-3 pb-2">
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={enabled} onChange={(e) => setEnabled(e.target.checked)} />
              Enabled
            </label>
            <label className="flex items-center gap-2 text-xs text-forge-mist">
              <input type="checkbox" checked={dryRunDefault} onChange={(e) => setDryRunDefault(e.target.checked)} />
              Dry-run default
            </label>
          </div>
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-3">
          <ReadableMapInput label="Condition rules" value={condition} onChange={setCondition} />
          <ReadableMapInput label="Action rules" value={action} onChange={setAction} />
          <ReadableMapInput label="Scope rules" value={scope} onChange={setScope} />
        </div>

        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const payload = {
                  id: selectedRuleId ?? undefined,
                  name,
                  trigger,
                  condition: parseReadableMap(condition, "condition"),
                  action: parseReadableMap(action, "action"),
                  scope: parseReadableMap(scope, "scope"),
                  enabled,
                  dryRunDefault,
                };
                const res = await api.automation.saveRule(payload);
                setSelectedRuleId(res.rule.id);
                setStatus(`Automation rule saved: ${res.rule.id}`);
                await load();
              } catch (e) {
                setErr(e instanceof Error ? e.message : String(e));
              }
            }}
          >
            Save Rule
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setSelectedRuleId(null);
              setName("");
              setTrigger("");
              setCondition("always: yes");
              setAction("type: create_review");
              setScope("");
              setEnabled(true);
              setDryRunDefault(true);
            }}
          >
            New Rule Draft
          </GhostButton>
          <GhostButton
            disabled={selectedRuleId == null}
            onClick={async () => {
              if (selectedRuleId == null) return;
              const res = await api.automation.runRule({
                ruleId: selectedRuleId,
                dryRun: runDry,
              });
              setRunResult(res.result as unknown as Record<string, unknown>);
              setStatus(`Automation rule ${selectedRuleId} run: ${res.result.executed ? "executed" : "preview/skipped"}.`);
              await load();
            }}
          >
            Run Selected Rule
          </GhostButton>
          <label className="ml-1 flex items-center gap-2 text-xs text-forge-mist">
            <input type="checkbox" checked={runDry} onChange={(e) => setRunDry(e.target.checked)} />
            run as dry-run
          </label>
        </div>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Rules" subtitle="Persisted automation contracts. Select one to edit or run.">
          {rules.length === 0 ? (
            <div className="text-sm text-forge-mist">No automation rules yet.</div>
          ) : (
            <div className="space-y-2">
              {rules.map((r) => (
                <button
                  key={r.id}
                  type="button"
                  onClick={() => selectRule(r)}
                  className={[
                    "w-full rounded border px-3 py-2 text-left",
                    selectedRuleId === r.id ? "border-forge-ember/40 bg-black/30" : "border-white/10 bg-black/20 hover:border-forge-ember/35",
                  ].join(" ")}
                >
                  <div className="text-sm font-semibold text-forge-ash">{r.name}</div>
                  <div className="mt-1 text-xs text-forge-mist">
                    #{r.id} · {r.trigger} · enabled {String(r.enabled)} · dry-run default {String(r.dryRunDefault)}
                  </div>
                  <div className="mt-1 text-[11px] text-forge-mist">updated {formatTime(r.updatedAtMs)}</div>
                </button>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Automation History" subtitle="Immutable run history with preview and result snapshots.">
          {history.length === 0 ? (
            <div className="text-sm text-forge-mist">No history rows yet.</div>
          ) : (
            <div className="space-y-2">
              {history.map((h) => (
                <div key={h.id} className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    history #{h.id} · rule {h.ruleId ?? "n/a"} · {h.status}
                  </div>
                  <div className="mt-1">
                    trigger {h.trigger} · matched {String(h.matched)} · dry-run {String(h.dryRun)}
                  </div>
                  <div className="mt-1">{h.message}</div>
                  <div className="mt-1 text-[11px]">{formatTime(h.createdAtMs)}</div>
                </div>
              ))}
            </div>
          )}
        </Panel>
      </div>

      <Panel title="Last Run Details" subtitle="Most recent rule run preview or execution result.">
        {!runResult ? (
          <div className="text-sm text-forge-mist">Run a rule to inspect the latest result.</div>
        ) : (
          <div className="max-h-[320px] overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
            <HumanDataView value={runResult} />
          </div>
        )}
      </Panel>
    </div>
  );
}

function ReadableMapInput(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-forge-mist">{props.label}</label>
      <textarea className="forge-input mt-1 min-h-[130px] font-mono text-[12px]" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
      <div className="mt-1 text-[10px] text-forge-mist/65">Use one key: value rule per line.</div>
    </div>
  );
}

function mapToReadableText(value: unknown, prefix = ""): string {
  if (!value || typeof value !== "object" || Array.isArray(value)) return "";
  return Object.entries(value as Record<string, unknown>)
    .flatMap(([key, item]) => {
      const path = prefix ? `${prefix}.${key}` : key;
      if (item && typeof item === "object" && !Array.isArray(item)) return mapToReadableText(item, path).split("\n").filter(Boolean);
      return `${path}: ${formatReadableValue(item)}`;
    })
    .join("\n");
}

function parseReadableMap(raw: string, field: string): Record<string, unknown> {
  const root: Record<string, unknown> = {};
  const lines = raw
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  for (const line of lines) {
    const splitAt = line.indexOf(":");
    if (splitAt < 1) throw new Error(`${field} rule is missing a colon: ${line}`);
    const path = line.slice(0, splitAt).trim().split(".").filter(Boolean);
    let cursor = root;
    for (const segment of path.slice(0, -1)) {
      const next = cursor[segment];
      if (!next || typeof next !== "object" || Array.isArray(next)) cursor[segment] = {};
      cursor = cursor[segment] as Record<string, unknown>;
    }
    cursor[path[path.length - 1]] = parseReadableValue(line.slice(splitAt + 1).trim());
  }
  return root;
}

function formatReadableValue(value: unknown) {
  if (typeof value === "boolean") return value ? "yes" : "no";
  if (Array.isArray(value)) return value.join(", ");
  if (value == null) return "none";
  return String(value);
}

function parseReadableValue(value: string): unknown {
  const normalized = value.toLowerCase();
  if (normalized === "yes" || normalized === "true") return true;
  if (normalized === "no" || normalized === "false") return false;
  if (normalized === "none" || normalized === "null") return null;
  const numberValue = Number(value);
  if (value !== "" && Number.isFinite(numberValue)) return numberValue;
  if (value.includes(",")) return value.split(",").map((part) => part.trim()).filter(Boolean);
  return value;
}
