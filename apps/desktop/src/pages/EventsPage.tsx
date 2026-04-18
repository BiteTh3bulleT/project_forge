import type { ForgeEvent } from "@forge/shared";
import { GhostButton, Panel } from "@forge/ui";
import { useEffect, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";

export function EventsPage() {
  const [events, setEvents] = useState<ForgeEvent[]>([]);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    try {
      const res = await api.events(200);
      setEvents(res.events);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
    const id = window.setInterval(() => void refresh(), 4000);
    return () => window.clearInterval(id);
  }, []);

  return (
    <div className="space-y-6">
      <Panel title="Events" subtitle="Structured audit log stored locally. This is the spine future automation will hang from." actions={<GhostButton onClick={() => void refresh()}>Refresh</GhostButton>}>
        {err ? <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
      </Panel>

      <div className="space-y-2">
        {events.length === 0 ? (
          <div className="text-sm text-forge-mist">No events yet (or core offline).</div>
        ) : (
          events.map((ev) => (
            <div key={ev.id} className="rounded-lg border border-white/10 bg-forge-iron/40 p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="text-sm font-semibold text-forge-ash">{ev.type}</div>
                <div className="text-[11px] text-forge-mist">{formatTime(ev.createdAtMs)}</div>
              </div>
              <pre className="mt-3 max-h-48 overflow-auto whitespace-pre-wrap break-words font-mono text-[11px] leading-relaxed text-forge-mist">
                {JSON.stringify(ev.payload, null, 2)}
              </pre>
            </div>
          ))
        )}
      </div>
    </div>
  );
}
