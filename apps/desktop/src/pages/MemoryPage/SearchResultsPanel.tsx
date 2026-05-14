import type { SearchHit } from "@forge/shared";

import { formatTime } from "../../lib/format";
import { EmptyState, Panel } from "./shared";

export function SearchResultsPanel(props: {
  hits: SearchHit[];
  navigateToChunk: (chunkId: number) => void;
}) {
  return (
    <Panel
      title="Chunk Search Results"
      subtitle="Raw retrieval candidates from indexed source chunks."
    >
      {props.hits.length === 0 ? (
        <EmptyState
          title="No chunk hits"
          detail="The current search query returned no indexed chunks. Try a broader term or clear the query."
        />
      ) : (
        <div className="space-y-2">
          {props.hits.slice(0, 40).map((h) => (
            <button
              key={h.chunkId}
              type="button"
              onClick={() => props.navigateToChunk(h.chunkId)}
              className="w-full rounded border border-forge-platinum/10 bg-black/20 p-3 text-left hover:border-forge-ember/35"
            >
              <div className="break-all text-xs font-semibold text-forge-ash">
                {h.relPath || h.absPath}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                chunk {h.chunkIndex} · score {h.score.toFixed(3)} ·{" "}
                {formatTime(Math.floor(h.mtimeNs / 1_000_000))}
              </div>
              <div className="mt-1 text-xs leading-5 text-forge-mist whitespace-pre-wrap">
                {h.snippet || h.content.slice(0, 220)}
              </div>
            </button>
          ))}
        </div>
      )}
    </Panel>
  );
}
