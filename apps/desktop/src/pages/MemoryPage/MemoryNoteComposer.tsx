import { GhostButton } from "@forge/ui";
import { useState } from "react";

import { api } from "../../lib/api";
import { Panel } from "./shared";

export function MemoryNoteComposer(props: {
  dossierId: string;
  loadObservations: () => Promise<void>;
  setSelectedObsId: (id: number) => void;
  setErr: (error: string | null) => void;
  setStatus: (status: string) => void;
  setStatusLine: (status: string) => void;
}) {
  const [summary, setSummary] = useState("");
  const [rawContent, setRawContent] = useState("");
  const [tagsText, setTagsText] = useState("");
  const [busy, setBusy] = useState(false);

  const summaryValue = summary.trim();
  const rawValue = rawContent.trim();
  const canCreate = summaryValue.length > 0 && rawValue.length > 0 && !busy;

  return (
    <Panel
      title="Write Memory Note"
      subtitle="Commit an operator note as a governed memory observation."
      actions={
        <GhostButton
          disabled={!canCreate}
          onClick={async () => {
            if (!canCreate) return;
            setBusy(true);
            try {
              const parsedDossierId = props.dossierId.trim()
                ? Number(props.dossierId.trim())
                : undefined;
              const res = await api.memory.createObservation({
                type: "note",
                summary: summaryValue,
                rawContent: rawValue,
                tags: tagsText
                  .split(",")
                  .map((tag) => tag.trim())
                  .filter(Boolean),
                dossierId: Number.isFinite(parsedDossierId)
                  ? parsedDossierId
                  : undefined,
                confidence: 0.8,
                verificationState: "operator_recorded",
                originKind: "operator_note",
                originId: `desktop:${Date.now()}`,
                observedAtMs: Date.now(),
              });
              setSummary("");
              setRawContent("");
              setTagsText("");
              await props.loadObservations();
              props.setSelectedObsId(res.observation.id);
              props.setErr(null);
              const status = `Memory note recorded as observation #${res.observation.id}.`;
              props.setStatus(status);
              props.setStatusLine(status);
            } catch (error) {
              const message = error instanceof Error ? error.message : String(error);
              props.setErr(message);
              props.setStatusLine(`Memory note failed: ${message}`);
            } finally {
              setBusy(false);
            }
          }}
        >
          {busy ? "Recording" : "Record note"}
        </GhostButton>
      }
    >
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <label className="block min-w-0 text-xs font-semibold text-forge-mist">
          Summary
          <input
            className="forge-input mt-1"
            value={summary}
            onChange={(event) => setSummary(event.target.value)}
            placeholder="Decision, preference, or fact to remember"
          />
        </label>
        <label className="block min-w-0 text-xs font-semibold text-forge-mist">
          Tags
          <input
            className="forge-input mt-1"
            value={tagsText}
            onChange={(event) => setTagsText(event.target.value)}
            placeholder="comma, separated, tags"
          />
        </label>
        <label className="block min-w-0 text-xs font-semibold text-forge-mist lg:col-span-2">
          Note
          <textarea
            className="forge-input mt-1 min-h-[96px] resize-y text-sm leading-6"
            value={rawContent}
            onChange={(event) => setRawContent(event.target.value)}
            placeholder="Write the durable note content here."
          />
        </label>
      </div>
    </Panel>
  );
}
