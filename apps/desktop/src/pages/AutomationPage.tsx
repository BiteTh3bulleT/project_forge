import type { AutomationHistory, AutomationRule } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function parseJSONMap(raw: string, field: string): Record<string, unknown> {
  try {
    const parsed = JSON.parse(raw);
    if (!parsed || typeof parsed !== "object" || Array.isArray(parsed)) {
      throw new Error(`${field} must be a JSON object`);
    }
    return parsed as Record<string, unknown>;
  } catch (e) {
    throw new Error(`${field} invalid JSON: ${e instanceof Error ? e.message : String(e)}`);
  }
}

function pretty(v: unknown): string {
  return JSON.stringify(v ?? {}, null, 2);
}

export function AutomationPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [rules, setRules] = useState<AutomationRule[]>([]);
  const [history, setHistory] = useState<AutomationHistory[]>([]);
  const [selectedRuleId, setSelectedRuleId] = useState<number | null>(null);
  const [name, setName] = useState("");
  const [trigger, setTrigger] = useState("");
  const [condition, setCondition] = useState('{\n  "always": true\n}');
  const [action, setAction] = useState('{\n  "type": "create_review"\n}');
  const [scope, setScope] = useState("{}");
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
    setCondition(pretty(rule.condition));
    setAction(pretty(rule.action));
    setScope(pretty(rule.scope));
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
          <JsonInput label="Condition JSON" value={condition} onChange={setCondition} />
          <JsonInput label="Action JSON" value={action} onChange={setAction} />
          <JsonInput label="Scope JSON" value={scope} onChange={setScope} />
        </div>

        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            onClick={async () => {
              try {
                const payload = {
                  id: selectedRuleId ?? undefined,
                  name,
                  trigger,
                  condition: parseJSONMap(condition, "condition"),
                  action: parseJSONMap(action, "action"),
                  scope: parseJSONMap(scope, "scope"),
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
              setCondition('{\n  "always": true\n}');
              setAction('{\n  "type": "create_review"\n}');
              setScope("{}");
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
                    selectedRuleId === r.id ? "border-forge-ember/40 bg-black/30" : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
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

        <Panel title="Automation History" subtitle="Immutable run history with preview/result payload snapshots.">
          {history.length === 0 ? (
            <div className="text-sm text-forge-mist">No history rows yet.</div>
          ) : (
            <div className="space-y-2">
              {history.map((h) => (
                <div key={h.id} className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
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

      <Panel title="Last Run Payload" subtitle="Most recent rule run preview/execution payload.">
        {!runResult ? (
          <div className="text-sm text-forge-mist">Run a rule to inspect payload output.</div>
        ) : (
          <pre className="max-h-[320px] overflow-auto rounded border border-forge-platinum/10 bg-black/30 p-3 text-[11px] text-forge-mist">
            {JSON.stringify(runResult, null, 2)}
          </pre>
        )}
      </Panel>
    </div>
  );
}

function JsonInput(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-forge-mist">{props.label}</label>
      <textarea className="forge-input mt-1 min-h-[130px] font-mono text-[12px]" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
    </div>
  );
}
