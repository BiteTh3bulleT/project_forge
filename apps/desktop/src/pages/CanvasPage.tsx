import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "react-router-dom";

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
  const [newNote, setNewNote] = useState({ title: "", body: "" });
  const [nextPlacement, setNextPlacement] = useState({ x: 40, y: 40 });
  const [selectedNoteId, setSelectedNoteId] = useState<number | null>(null);

  const refreshBoards = useCallback(async () => {
    const res = await api.canvas.boards.list(80);
    setBoards(normalizeBoards(res.boards));
  }, []);

  const loadBoard = useCallback(
    async (id: number) => {
      const b = await api.canvas.boards.get(id);
      setActive(normalizeBoardDetail(b));
      setParams({ boardId: String(id) });
      setSelectedNoteId((prev) => {
        if (b.notes.length === 0) return null;
        if (prev && b.notes.some((note) => note.id === prev)) return prev;
        return b.notes[0].id;
      });
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
          if (!cancelled) {
            setActive(normalizeBoardDetail(b));
            setSelectedNoteId(b.notes[0]?.id ?? null);
          }
        } else if (nextBoards.length > 0 && !cancelled) {
          const b = await api.canvas.boards.get(nextBoards[0].id);
          if (!cancelled) {
            setActive(normalizeBoardDetail(b));
            setSelectedNoteId(b.notes[0]?.id ?? null);
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
        x: nextPlacement.x,
        y: nextPlacement.y,
        width: 280,
        height: 200,
      });
      setNewNote({ title: "", body: "" });
      setNextPlacement((prev) => ({ x: prev.x + 28, y: prev.y + 28 }));
      await loadBoard(active.id);
      await refreshBoards();
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  const saveNote = useCallback(
    async (n: CanvasNote, patch: Record<string, unknown>) => {
      if (!active) return;
      try {
        await api.canvas.boards.patchNote(active.id, n.id, patch);
      } catch (e) {
        setErr(e instanceof Error ? e.message : String(e));
      }
    },
    [active],
  );

  async function deleteSelectedNote() {
    if (!active || selectedNoteId == null) return;
    try {
      await api.canvas.boards.deleteNote(active.id, selectedNoteId);
      await loadBoard(active.id);
      setStatus(`Note ${selectedNoteId} deleted.`);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    function onKeyDown(e: KeyboardEvent) {
      if (!active || selectedNoteId == null) return;
      if (e.key !== "Delete" && e.key !== "Backspace") return;
      const target = e.target;
      if (target instanceof Element && target.closest("input, textarea, [contenteditable='true']")) return;
      e.preventDefault();
      void deleteSelectedNote();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, selectedNoteId]);

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
                  active?.id === b.id ? "border-forge-ember/40 bg-forge-slate/50 text-forge-ash" : "border-forge-platinum/10 bg-black/20 text-forge-mist hover:border-forge-ember/25",
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
        subtitle="Obsidian-style editing: inline edits auto-save. Click-hold header to drag and pull the corner handle to resize. Delete selected note with Delete/Backspace."
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
            <div className="rounded border border-forge-platinum/10 bg-black/25 p-3 text-xs text-forge-mist">
              <div className="font-semibold text-forge-ash">Add note</div>
              <div className="mt-2 grid gap-2 md:grid-cols-2">
                <label>
                  <span className="text-[10px] uppercase tracking-wide">Title</span>
                  <input
                    aria-label="New note title"
                    className="forge-input mt-1 w-full"
                    value={newNote.title}
                    onChange={(e) => setNewNote((n) => ({ ...n, title: e.target.value }))}
                  />
                </label>
                <label>
                  <span className="text-[10px] uppercase tracking-wide">Placement</span>
                  <div className="mt-1 rounded border border-forge-platinum/10 bg-black/30 px-2 py-2 text-[11px] text-forge-mist">
                    {nextPlacement.x}, {nextPlacement.y}
                  </div>
                </label>
                <label className="md:col-span-2">
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

            <div className="relative min-h-[720px] overflow-auto rounded border border-forge-platinum/10 bg-forge-black">
              {active.notes.length === 0 ? (
                <div className="p-4 text-sm text-forge-mist">Board is empty. Add a note block above.</div>
              ) : (
                active.notes.map((n) => (
                  <NoteCard
                    key={n.id}
                    note={n}
                    busy={busy}
                    selected={selectedNoteId === n.id}
                    onSelect={() => setSelectedNoteId(n.id)}
                    onSave={saveNote}
                  />
                ))
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
  note: CanvasNote;
  busy: boolean;
  selected: boolean;
  onSelect: () => void;
  onSave: (n: CanvasNote, patch: Record<string, unknown>) => void | Promise<void>;
}) {
  const { note, busy, onSave, onSelect, selected } = props;
  const minWidth = 220;
  const minHeight = 150;
  const [title, setTitle] = useState(note.title);
  const [body, setBody] = useState(note.body);
  const [x, setX] = useState(note.x);
  const [y, setY] = useState(note.y);
  const [w, setW] = useState(note.width);
  const [h, setH] = useState(note.height);
  const [pinned, setPinned] = useState(note.pinned);
  const [dragging, setDragging] = useState(false);
  const [resizing, setResizing] = useState(false);
  const dragRef = useRef<{ pointerId: number; startX: number; startY: number; originX: number; originY: number; currentX: number; currentY: number } | null>(null);
  const resizeRef = useRef<{ pointerId: number; startX: number; startY: number; originW: number; originH: number; currentW: number; currentH: number } | null>(null);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    setTitle(note.title);
    setBody(note.body);
    setX(note.x);
    setY(note.y);
    setW(note.width);
    setH(note.height);
    setPinned(note.pinned);
  }, [note]);

  useEffect(() => {
    if (title === note.title && body === note.body && pinned === note.pinned) {
      return;
    }
    if (saveTimerRef.current) {
      clearTimeout(saveTimerRef.current);
    }
    saveTimerRef.current = setTimeout(() => {
      void onSave(note, { title, body, pinned });
    }, 450);
    return () => {
      if (saveTimerRef.current) {
        clearTimeout(saveTimerRef.current);
      }
    };
  }, [body, note, onSave, pinned, title]);

  function isInteractiveTarget(target: EventTarget | null) {
    if (!(target instanceof Element)) return false;
    return Boolean(target.closest("input, textarea, button, a, label"));
  }

  function startDrag(e: React.PointerEvent<HTMLDivElement>) {
    if (busy) return;
    if (isInteractiveTarget(e.target)) return;
    onSelect();
    e.preventDefault();
    const current = { pointerId: e.pointerId, startX: e.clientX, startY: e.clientY, originX: x, originY: y, currentX: x, currentY: y };
    dragRef.current = current;
    setDragging(true);

    const onMove = (ev: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag || ev.pointerId !== drag.pointerId) return;
      const dx = ev.clientX - drag.startX;
      const dy = ev.clientY - drag.startY;
      const nextX = Math.max(0, Math.round(drag.originX + dx));
      const nextY = Math.max(0, Math.round(drag.originY + dy));
      drag.currentX = nextX;
      drag.currentY = nextY;
      setX(nextX);
      setY(nextY);
    };

    const stop = (ev: PointerEvent) => {
      const drag = dragRef.current;
      if (!drag || ev.pointerId !== drag.pointerId) return;
      dragRef.current = null;
      setDragging(false);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
      void onSave(note, { x: drag.currentX, y: drag.currentY });
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }

  function startResize(e: React.PointerEvent<HTMLButtonElement>) {
    if (busy) return;
    e.preventDefault();
    e.stopPropagation();
    onSelect();
    const current = { pointerId: e.pointerId, startX: e.clientX, startY: e.clientY, originW: w, originH: h, currentW: w, currentH: h };
    resizeRef.current = current;
    setResizing(true);

    const onMove = (ev: PointerEvent) => {
      const resize = resizeRef.current;
      if (!resize || ev.pointerId !== resize.pointerId) return;
      const dw = ev.clientX - resize.startX;
      const dh = ev.clientY - resize.startY;
      const nextW = Math.max(minWidth, Math.round(resize.originW + dw));
      const nextH = Math.max(minHeight, Math.round(resize.originH + dh));
      resize.currentW = nextW;
      resize.currentH = nextH;
      setW(nextW);
      setH(nextH);
    };

    const stop = (ev: PointerEvent) => {
      const resize = resizeRef.current;
      if (!resize || ev.pointerId !== resize.pointerId) return;
      resizeRef.current = null;
      setResizing(false);
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", stop);
      window.removeEventListener("pointercancel", stop);
      void onSave(note, { width: resize.currentW, height: resize.currentH });
    };

    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", stop);
    window.addEventListener("pointercancel", stop);
  }

  return (
    <div
      className={[
        "absolute rounded border bg-forge-iron/80 shadow-lg backdrop-blur",
        selected ? "border-forge-ember/45 ring-1 ring-inset ring-forge-ember/35" : "border-forge-platinum/15",
        dragging || resizing ? "ring-1 ring-inset ring-forge-ember/40" : "",
      ].join(" ")}
      style={{ left: x, top: y, width: w, height: h, zIndex: note.pinned ? 2 : 1 }}
      onPointerDown={() => onSelect()}
    >
      <div
        className="flex cursor-move items-center justify-between gap-1 border-b border-forge-platinum/10 px-2 py-1"
        onPointerDown={startDrag}
        title="Click and hold to drag"
      >
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
        className="h-[calc(100%-66px)] w-full resize-none bg-transparent p-2 pb-12 text-[11px] leading-relaxed text-forge-mist outline-none"
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <div className="absolute bottom-0 left-0 right-0 space-y-1 border-t border-forge-platinum/10 bg-black/40 p-2 text-[10px] text-forge-mist">
        <div className="text-[10px]">Auto-saved · select and press Delete/Backspace to remove</div>
      </div>
      <button
        type="button"
        aria-label="Resize note"
        title="Click and pull to resize"
        className="absolute bottom-1 right-1 h-4 w-4 cursor-nwse-resize rounded-sm border border-forge-platinum/20 bg-forge-platinum/10"
        onPointerDown={startResize}
      />
    </div>
  );
}
