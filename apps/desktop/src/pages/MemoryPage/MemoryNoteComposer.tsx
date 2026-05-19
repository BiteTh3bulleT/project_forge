import { useState } from "react";

import { Panel } from "./shared";

export function MemoryNoteComposer() {
  const [summary, setSummary] = useState("");
  const [rawContent, setRawContent] = useState("");
  const [tagsText, setTagsText] = useState("");

  return (
    <Panel
      title="Memory Note Intake"
      subtitle="Legacy direct memory writes are disabled; existing observations remain inspectable as historical evidence."
      actions={
        <span
          className="forge-ops-status forge-ops-status--muted"
          title="Legacy mutation endpoint returns HTTP 410"
        >
          read-only
        </span>
      }
    >
      <div className="mb-3 rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs leading-5 text-forge-mist">
        Legacy memory note writes are retired. New canonical memory must enter
        through Courthouse admission review and Control Lane semantic syscalls.
      </div>
      <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]">
        <label className="block min-w-0 text-xs font-semibold text-forge-mist">
          Summary
          <input
            className="forge-input mt-1"
            disabled
            value={summary}
            onChange={(event) => setSummary(event.target.value)}
            placeholder="Decision, preference, or fact to remember"
          />
        </label>
        <label className="block min-w-0 text-xs font-semibold text-forge-mist">
          Tags
          <input
            className="forge-input mt-1"
            disabled
            value={tagsText}
            onChange={(event) => setTagsText(event.target.value)}
            placeholder="comma, separated, tags"
          />
        </label>
        <label className="block min-w-0 text-xs font-semibold text-forge-mist lg:col-span-2">
          Note
          <textarea
            className="forge-input mt-1 min-h-[96px] resize-y text-sm leading-6"
            disabled
            value={rawContent}
            onChange={(event) => setRawContent(event.target.value)}
            placeholder="Write the durable note content here."
          />
        </label>
      </div>
    </Panel>
  );
}
