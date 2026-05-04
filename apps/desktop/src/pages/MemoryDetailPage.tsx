import type { SearchHit } from "@forge/shared";
import { useEffect, useState, type ReactNode } from "react";
import { Link, useParams } from "react-router-dom";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";

function Panel(props: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel min-w-0">
      <div className="forge-ops-panel__head flex-col items-stretch sm:flex-row sm:items-center">
        <div className="min-w-0">
          <div className="forge-ops-title break-words">{props.title}</div>
          {props.subtitle ? (
            <div className="mt-1 max-w-3xl break-words text-xs leading-5 text-forge-mist/65">
              {props.subtitle}
            </div>
          ) : null}
        </div>
        {props.actions ? (
          <div className="flex flex-wrap items-center gap-2">
            {props.actions}
          </div>
        ) : null}
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}

function EmptyState(props: {
  title: string;
  detail: string;
  tone?: "muted" | "bad";
}) {
  const toneClass =
    props.tone === "bad"
      ? "border-forge-ember/30 bg-forge-ember/10"
      : "border-forge-platinum/10 bg-black/20";
  return (
    <div className={["rounded border border-dashed p-4", toneClass].join(" ")}>
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 break-words text-xs leading-5 text-forge-mist/75">
        {props.detail}
      </div>
    </div>
  );
}

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
      <div className="forge-ops-board space-y-5">
        <Panel title="Chunk detail" subtitle="Could not load this chunk.">
          <EmptyState
            title="Chunk could not be loaded"
            detail={err}
            tone="bad"
          />
          <div className="mt-4">
            <Link
              to="/memory"
              className="forge-btn forge-btn--ghost inline-flex"
            >
              Back to Memory
            </Link>
          </div>
        </Panel>
      </div>
    );
  }

  if (!chunk) {
    return (
      <div className="forge-ops-board space-y-5">
        <Panel title="Chunk detail" subtitle="Loading...">
          <EmptyState
            title="Loading chunk evidence"
            detail="Pulling indexed text, file path, modification time, and retrieval identity from the core."
          />
        </Panel>
      </div>
    );
  }

  return (
    <div className="forge-ops-board space-y-5">
      <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="forge-ops-label">Memory Chunk</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Chunk detail
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Durable indexed text unit with file metadata and retrieval identity.
          </p>
        </div>
        <Link
          className="forge-btn forge-btn--ghost mt-4 inline-flex lg:mt-0"
          to="/memory"
        >
          Back
        </Link>
      </header>

      <Panel
        title="Chunk Metadata"
        subtitle="File path, modification time, and chunk index."
      >
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-4">
          <DetailDatum label="Relative" value={chunk.relPath || "n/a"} mono />
          <DetailDatum
            label="Chunk"
            value={`#${chunk.chunkIndex} / id ${chunk.chunkId}`}
          />
          <DetailDatum
            label="Modified"
            value={formatTime(Math.floor(chunk.mtimeNs / 1_000_000))}
          />
          <DetailDatum label="Bytes" value={chunk.contentLength} />
        </div>
        <div className="mt-3 rounded border border-forge-platinum/10 bg-black/20 p-3">
          <div className="forge-ops-label">Absolute Path</div>
          <div className="mt-2 break-all font-mono text-xs text-forge-ash">
            {chunk.absPath}
          </div>
        </div>
      </Panel>

      <Panel
        title="Text"
        subtitle={`${chunk.contentLength} bytes (UTF-8 length may differ from byte length for non-ASCII).`}
      >
        <pre className="max-h-[560px] overflow-auto whitespace-pre-wrap rounded-md border border-forge-platinum/10 bg-black/30 p-4 font-mono text-xs leading-relaxed text-forge-ash">
          {chunk.content}
        </pre>
      </Panel>
    </div>
  );
}

function DetailDatum(props: {
  label: string;
  value: string | number;
  mono?: boolean;
}) {
  return (
    <div className="forge-ops-card p-3">
      <div className="forge-ops-label">{props.label}</div>
      <div
        className={[
          "mt-2 min-w-0 break-words text-sm text-forge-ash",
          props.mono ? "break-all font-mono text-xs" : "font-semibold",
        ].join(" ")}
      >
        {props.value}
      </div>
    </div>
  );
}
