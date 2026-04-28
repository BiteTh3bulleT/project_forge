import type { SearchHit } from "@forge/shared";
import { Panel } from "@forge/ui";
import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";

export function MemoryDetailPage() {
  const { id } = useParams();
  const [chunk, setChunk] = useState<SearchHit | null>(null);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      const n = Number(id);
      if (!Number.isFinite(n)) {
        setErr("Invalid chunk id.");
        return;
      }
      try {
        const c = await api.chunk(n);
        if (cancelled) return;
        setChunk(c);
        setErr(null);
      } catch (e) {
        if (cancelled) return;
        setChunk(null);
        setErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [id]);

  if (err) {
    return (
      <Panel title="Chunk detail" subtitle="Could not load this chunk.">
        <div className="text-sm text-forge-emberSoft">{err}</div>
        <div className="mt-4">
          <Link to="/memory" className="forge-btn forge-btn--ghost inline-flex">
            Back to Memory
          </Link>
        </div>
      </Panel>
    );
  }

  if (!chunk) {
    return (
      <Panel title="Chunk detail" subtitle="Loading…">
        <div className="text-sm text-forge-mist">Pulling text and file metadata from the core.</div>
      </Panel>
    );
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Chunk detail"
        subtitle="Full chunk text for inspection. This is the durable unit stored in SQLite today; vectors can attach later without breaking the file/chunk model."
        actions={
          <Link className="forge-btn forge-btn--ghost inline-flex" to="/memory">
            Back
          </Link>
        }
      >
        <div className="space-y-2 text-sm text-forge-mist">
          <div>
            <span className="text-forge-ash">Path:</span> <span className="font-mono text-xs text-forge-ash">{chunk.absPath}</span>
          </div>
          <div>
            <span className="text-forge-ash">Relative:</span> <span className="font-mono text-xs text-forge-ash">{chunk.relPath}</span>
          </div>
          <div>
            <span className="text-forge-ash">Modified:</span> {formatTime(Math.floor(chunk.mtimeNs / 1_000_000))}
          </div>
          <div>
            <span className="text-forge-ash">Chunk:</span> #{chunk.chunkIndex} · id {chunk.chunkId}
          </div>
        </div>
      </Panel>

      <Panel title="Text" subtitle={`${chunk.contentLength} bytes (UTF-8 length may differ from byte length for non-ASCII).`}>
        <pre className="max-h-[560px] overflow-auto whitespace-pre-wrap rounded-md border border-forge-platinum/10 bg-black/30 p-4 font-mono text-xs leading-relaxed text-forge-ash">
          {chunk.content}
        </pre>
      </Panel>
    </div>
  );
}
