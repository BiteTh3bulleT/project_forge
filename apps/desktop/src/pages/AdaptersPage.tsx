import type { AdapterInfo } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";

export function AdaptersPage() {
  const [items, setItems] = useState<AdapterInfo[]>([]);
  const [err, setErr] = useState<string | null>(null);
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function refresh() {
    try {
      const res = await api.adapters.list();
      setItems(res.adapters);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  return (
    <div className="forge-ops-board space-y-5">
      <Panel
        title="Adapters"
        subtitle="Bounded workers behind a common request contract: capability, scope, write intent, packet ref, timeout, dry-run, correlation id."
        actions={
          <GhostButton onClick={() => void refresh()}>Refresh</GhostButton>
        }
      >
        {err ? (
          <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
            {err}
          </div>
        ) : null}
      </Panel>

      <div className="grid gap-4 lg:grid-cols-3">
        {items.map((a) => (
          <Panel
            key={a.id}
            title={a.displayName}
            subtitle={`Status: ${a.status}`}
          >
            <div className="text-sm text-forge-mist">{a.detail}</div>
            <div className="mt-2 text-xs text-forge-mist">
              Capabilities: {a.capabilities.join(", ") || "none"}
            </div>
            <div className="mt-3 rounded-md border border-white/10 bg-black/25 p-3 text-[11px] text-forge-mist">
              <div className="text-forge-ash">Configuration</div>
              <div className="mt-2">
                <HumanDataView value={a.config} compact />
              </div>
            </div>
            <div className="mt-4 flex gap-2">
              <PrimaryButton
                onClick={async () => {
                  const cap = a.capabilities.includes("status")
                    ? "status"
                    : (a.capabilities[0] ?? "status");
                  const r = await api.adapters.invoke(a.id, {
                    adapterId: a.id,
                    capability: cap,
                    scope: {
                      allowedPaths: [],
                      forbiddenPaths: [],
                      selectedPaths: [],
                    },
                    writeIntent: false,
                    timeoutMs: 5000,
                    dryRun: false,
                    correlationId: `manual-${Date.now()}`,
                    input: {},
                  });
                  setStatus(`${a.id}: ${r.message}`);
                }}
              >
                Probe
              </PrimaryButton>
            </div>
          </Panel>
        ))}
      </div>
    </div>
  );
}
