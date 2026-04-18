import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { Link, useSearchParams } from "react-router-dom";

import { api, type CanvasBoard, type CanvasBoardDetail, type CanvasNote } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function normalizeBoards(v: CanvasBoard[] | null | undefined): CanvasBoard[] {
  return Array.isArray(v) ? v : [];
}

function normalizeBoardDetail(v: CanvasBoardDetail): CanvasBoardDetail {
  return {
    ...v,
    notes: Array.isArray(v.notes) ? v.notes : [],
  };
}

export function CanvasPage() {
  const [params, setParams] = useSearchParams();
  const setStatus = useUiStore((s) => s.setStatusLine);
  const boardIdParam = params.get("boardId");
  const [boards, setBoards] = useState<CanvasBoard[]>([]);
  const [active, setActive] = useState<CanvasBoardDetail | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [newNote, setNewNote] = useState({ title: "", body: "", x: 40, y: 40 });

  const refreshBoards = useCallback(async () => {
    const res = await api.canvas.boards.list(80);
    setBoards(normalizeBoards(res.boards));
  }, []);

  const loadBoard = useCallback(
    async (id: number) => {
      const b = await api.canvas.boards.get(id);
      setActive(normalizeBoardDetail(b));
      setParams({ boardId: String(id) });
    },
    [setParams],
  );

  useEffect(() => {
    let cancelled = false;
    async function boot() {
      try {
        const res = await api.canvas.boards.list(80);
        if (cancelled) return;
        const nextBoards = normalizeBoards(res.boards);
        setBoards(nextBoards);
        const fromUrl = boardIdParam ? Number(boardIdParam) : NaN;
        if (Number.isFinite(fromUrl) && fromUrl > 0) {
          const b = await api.canvas.boards.get(fromUrl);
          if (!cancelled) setActive(normalizeBoardDetail(b));
        } else if (nextBoards.length > 0 && !cancelled) {
          const b = await api.canvas.boards.get(nextBoards[0].id);
          if (!cancelled) {
            setActive(normalizeBoardDetail(b));
            setParams({ boardId: String(b.id) });
          }
        }
        setErr(null);
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      }
    }
    void boot();
    return () => {
      cancelled = true;
    };
  }, [boardIdParam, setParams]);

  async function createBoard() {
    setBusy(true);
    try {
      const res = await api.canvas.boards.create({ title: "New board" });
      await refreshBoards();
      await loadBoard(res.board.id);
      setStatus(`Board ${res.board.id} created.`);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function addNote() {
    if (!active) return;
    setBusy(true);
    try {
      await api.canvas.boards.createNote(active.id, {
        title: newNote.title || "Note",
        body: newNote.body,
        x: newNote.x,
        y: newNote.y,
        width: 280,
        height: 200,
      });
      setNewNote({ title: "", body: "", x: newNote.x + 28, y: newNote.y + 28 });
      await loadBoard(active.id);
      await refreshBoards();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveNote(n: CanvasNote, patch: Record<string, unknown>) {
    if (!active) return;
    setBusy(true);
    try {
      await api.canvas.boards.patchNote(active.id, n.id, patch);
      await loadBoard(active.id);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="grid min-h-[560px] gap-4 lg:grid-cols-[220px_minmax(0,1fr)]">
      <Panel title="Boards" subtitle="Persisted layout + notes (SQLite)." actions={<GhostButton onClick={() => void refreshBoards()}>Refresh</GhostButton>}>
        <div className="mb-2">
          <PrimaryButton onClick={() => void createBoard()} disabled={busy}>
            New board
          </PrimaryButton>
        </div>
        {err ? <div className="mb-2 rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs text-forge-ash">{err}</div> : null}
        <div className="max-h-[520px] space-y-1 overflow-auto">
          {boards.length === 0 ? (
            <div className="text-xs text-forge-mist">No boards yet.</div>
          ) : (
            boards.map((b) => (
              <button
                key={b.id}
                type="button"
                onClick={() => void loadBoard(b.id)}
                className={[
                  "w-full rounded border px-2 py-2 text-left text-xs",
                  active?.id === b.id ? "border-forge-ember/40 bg-forge-slate/50 text-forge-ash" : "border-white/10 bg-black/20 text-forge-mist hover:border-forge-ember/25",
                ].join(" ")}
              >
                <div className="truncate font-semibold">{b.title}</div>
                <div className="text-[10px] text-forge-mist/80">#{b.id} · {formatTime(b.updatedAtMs)}</div>
              </button>
            ))
          )}
        </div>
      </Panel>

      <Panel
        title={active ? active.title : "Canvas"}
        subtitle="Freeform notes with explicit coordinates. Drag cards directly, then save to persist updated layout."
        actions={
          active ? (
            <GhostButton
              onClick={async () => {
                if (!active || !window.confirm("Delete board and all notes?")) return;
                await api.canvas.boards.delete(active.id);
                setActive(null);
                setParams({});
                await refreshBoards();
                setStatus("Board deleted.");
              }}
              disabled={busy}
            >
              Delete board
            </GhostButton>
          ) : null
        }
      >
        {!active ? (
          <div className="text-sm text-forge-mist">Select or create a board.</div>
        ) : (
          <div className="space-y-4">
            <div className="rounded border border-white/10 bg-black/25 p-3 text-xs text-forge-mist">
              <div className="font-semibold text-forge-ash">Add note</div>
              <div className="mt-2 grid gap-2 md:grid-cols-4">
                <label className="md:col-span-2">
                  <span className="text-[10px] uppercase tracking-wide">Title</span>
                  <input
                    aria-label="New note title"
                    className="forge-input mt-1 w-full"
                    value={newNote.title}
                    onChange={(e) => setNewNote((n) => ({ ...n, title: e.target.value }))}
                  />
                </label>
                <label>
                  <span className="text-[10px] uppercase tracking-wide">X</span>
                  <input
                    aria-label="New note X position"
                    type="number"
                    className="forge-input mt-1 w-full"
                    value={newNote.x}
                    onChange={(e) => setNewNote((n) => ({ ...n, x: Number(e.target.value) }))}
                  />
                </label>
                <label>
                  <span className="text-[10px] uppercase tracking-wide">Y</span>
                  <input
                    aria-label="New note Y position"
                    type="number"
                    className="forge-input mt-1 w-full"
                    value={newNote.y}
                    onChange={(e) => setNewNote((n) => ({ ...n, y: Number(e.target.value) }))}
                  />
                </label>
                <label className="md:col-span-4">
                  <span className="text-[10px] uppercase tracking-wide">Body</span>
                  <textarea
                    aria-label="New note body"
                    className="forge-input mt-1 min-h-[72px] w-full"
                    value={newNote.body}
                    onChange={(e) => setNewNote((n) => ({ ...n, body: e.target.value }))}
                  />
                </label>
              </div>
              <div className="mt-2">
                <PrimaryButton onClick={() => void addNote()} disabled={busy}>
                  Place note
                </PrimaryButton>
              </div>
            </div>

            <div className="relative min-h-[720px] overflow-auto rounded border border-white/10 bg-[#0a0c10]">
              {active.notes.length === 0 ? (
                <div className="p-4 text-sm text-forge-mist">Board is empty. Add a note block above.</div>
              ) : (
                active.notes.map((n) => <NoteCard key={n.id} boardId={active.id} note={n} busy={busy} onSave={saveNote} onReload={() => void loadBoard(active.id)} />)
              )}
            </div>

            <p className="text-[11px] leading-relaxed text-forge-mist/80">
              Cross-links: put references in note body (for example job ids, dossier ids, or paths). Structured{" "}
              <code className="text-forge-ash">links</code> JSON is supported by the API; the UI does not yet expose a link builder.
            </p>
          </div>
        )}
      </Panel>
    </div>
  );
}

function NoteCard(props: {
  boardId: number;
  note: CanvasNote;
  busy: boolean;
  onSave: (n: CanvasNote, patch: Record<string, unknown>) => void | Promise<void>;
  onReload: () => void | Promise<void>;
}) {
  const { note, busy, onSave, onReload } = props;
  const [title, setTitle] = useState(note.title);
  const [body, setBody] = useState(note.body);
  const [x, setX] = useState(note.x);
  const [y, setY] = useState(note.y);
  const [w, setW] = useState(note.width);
  const [h, setH] = useState(note.height);
  const [pinned, setPinned] = useState(note.pinned);
  const [dragging, setDragging] = useState(false);
  const dragRef = useRef<{ pointerId: number; startX: number; startY: number; originX: number; originY: number } | null>(null);

  useEffect(() => {
    setTitle(note.title);
    setBody(note.body);
    setX(note.x);
    setY(note.y);
    setW(note.width);
    setH(note.height);
    setPinned(note.pinned);
  }, [note]);

  function startDrag(e: React.PointerEvent<HTMLButtonElement>) {
    if (busy) return;
    e.preventDefault();
    const current = { pointerId: e.pointerId, startX: e.clientX, startY: e.clientY, originX: x, originY: y };
    dragRef.current = current;
    setDragging(true);

    const onMove = (ev: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag || ev.pointerId !== drag.pointerId) return;
      const dx = ev.clientX - drag.startX;
      const dy = ev.clientY - drag.startY;
      setX(Math.max(0, Math.round(drag.originX + dx)));
      setY(Math.max(0, Math.round(drag.originY + dy)));
    };

    const stop = (ev: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag || ev.pointerId !== drag.pointerId) return;
      dragRef.current = null;
      setDragging(false);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }

  return (
    <div
      className={[
        "absolute rounded border border-white/15 bg-forge-iron/80 shadow-lg backdrop-blur",
        dragging ? "ring-1 ring-inset ring-forge-ember/40" : "",
      ].join(" ")}
      style={{ left: x, top: y, width: w, height: h, zIndex: note.pinned ? 2 : 1 }}
    >
      <div className="flex items-center justify-between gap-1 border-b border-white/10 px-2 py-1">
        <button
          type="button"
          className="forge-btn forge-btn--ghost mr-1 cursor-move px-1.5 py-0 text-[10px] text-forge-mist/85"
          aria-label="Drag note"
          title="Drag note"
          disabled={busy}
          onPointerDown={startDrag}
        >
          drag
        </button>
        <input
          aria-label="Note title"
          className="min-w-0 flex-1 bg-transparent text-xs font-semibold text-forge-ash outline-none"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <label className="flex items-center gap-1 text-[10px] text-forge-mist">
          <input aria-label="Pin note" type="checkbox" checked={pinned} onChange={(e) => setPinned(e.target.checked)} />
          pin
        </label>
      </div>
      <textarea
        aria-label="Note body"
        className="h-[calc(100%-88px)] w-full resize-none bg-transparent p-2 text-[11px] leading-relaxed text-forge-mist outline-none"
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <div className="absolute bottom-0 left-0 right-0 space-y-1 border-t border-white/10 bg-black/40 p-2 text-[10px] text-forge-mist">
        <div className="grid grid-cols-4 gap-1">
          <input aria-label="Note X" type="number" className="forge-input px-1 py-0.5" value={x} onChange={(e) => setX(Number(e.target.value))} title="x" />
          <input aria-label="Note Y" type="number" className="forge-input px-1 py-0.5" value={y} onChange={(e) => setY(Number(e.target.value))} title="y" />
          <input aria-label="Note width" type="number" className="forge-input px-1 py-0.5" value={w} onChange={(e) => setW(Number(e.target.value))} title="w" />
          <input aria-label="Note height" type="number" className="forge-input px-1 py-0.5" value={h} onChange={(e) => setH(Number(e.target.value))} title="h" />
        </div>
        <div className="flex flex-wrap gap-1">
          <button
            type="button"
            className="forge-btn forge-btn--primary px-2 py-0.5 text-[10px]"
            disabled={busy}
            onClick={() =>
              void onSave(note, {
                title,
                body,
                x,
                y,
                width: w,
                height: h,
                pinned,
              })
            }
          >
            Save
          </button>
          <button type="button" className="forge-btn forge-btn--ghost px-2 py-0.5 text-[10px]" disabled={busy} onClick={() => void onReload()}>
            Reset
          </button>
          <button
            type="button"
            className="forge-btn forge-btn--ghost px-2 py-0.5 text-[10px] text-forge-emberSoft"
            disabled={busy}
            onClick={async () => {
              if (!window.confirm("Delete this note?")) return;
              await api.canvas.boards.deleteNote(props.boardId, note.id);
              await onReload();
            }}
          >
            Delete
          </button>
        </div>
        <div className="flex gap-2 text-[10px]">
          <Link className="text-forge-emberSoft underline" to="/chat">
            Chat
          </Link>
          <Link className="text-forge-emberSoft underline" to="/workbench">
            Workbench
          </Link>
        </div>
      </div>
    </div>
  );
}
