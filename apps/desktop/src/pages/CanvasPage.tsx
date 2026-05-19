import { GhostButton, PrimaryButton } from "@forge/ui";
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from "react";
import { useSearchParams } from "react-router-dom";

import {
  api,
  type CanvasBoard,
  type CanvasBoardDetail,
  type CanvasNote,
} from "../lib/api";
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

function applyNoteGeometry(
  node: HTMLDivElement | null,
  geometry: {
    x: number;
    y: number;
    width: number;
    height: number;
    pinned: boolean;
  },
) {
  if (!node) return;
  node.style.left = `${geometry.x}px`;
  node.style.top = `${geometry.y}px`;
  node.style.width = `${geometry.width}px`;
  node.style.height = `${geometry.height}px`;
  node.style.zIndex = geometry.pinned ? "2" : "1";
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
      const normalized = normalizeBoardDetail(b);
      setActive(normalized);
      setParams({ boardId: String(id) });
      setSelectedNoteId((prev) => {
        if (normalized.notes.length === 0) return null;
        if (prev && normalized.notes.some((note) => note.id === prev)) {
          return prev;
        }
        return normalized.notes[0]!.id;
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
            const normalized = normalizeBoardDetail(b);
            setActive(normalized);
            setSelectedNoteId(normalized.notes[0]?.id ?? null);
          }
        } else if (nextBoards.length > 0 && !cancelled) {
          const b = await api.canvas.boards.get(nextBoards[0].id);
          if (!cancelled) {
            const normalized = normalizeBoardDetail(b);
            setActive(normalized);
            setSelectedNoteId(normalized.notes[0]?.id ?? null);
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
      if (
        target instanceof Element &&
        target.closest("input, textarea, [contenteditable='true']")
      )
        return;
      e.preventDefault();
      void deleteSelectedNote();
    }
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [active, selectedNoteId]);

  const selectedNote =
    active?.notes.find((note) => note.id === selectedNoteId) ?? null;

  return (
    <div className="forge-ops-board space-y-4">
      <header className="rounded-lg border border-forge-platinum/10 bg-forge-carbon/80 p-4 shadow-[0_18px_60px_rgba(0,0,0,0.32)] lg:flex lg:items-end lg:justify-between lg:gap-4">
        <div>
          <div className="forge-ops-label">Canvas</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash">
            {active ? active.title : "Board builder"}
          </h1>
          <p className="mt-1 max-w-2xl text-sm leading-6 text-forge-mist/75">
            Arrange durable notes on a dense workspace surface with board
            controls and note inspection in reach.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="forge-ops-status forge-ops-status--muted">
            {boards.length} boards
          </span>
          {active ? (
            <span className="forge-ops-status forge-ops-status--warn">
              {active.notes.length} notes
            </span>
          ) : null}
          <GhostButton
            className="h-9 px-3"
            onClick={() => void refreshBoards()}
          >
            Refresh
          </GhostButton>
          <PrimaryButton
            className="h-9 px-3"
            onClick={() => void createBoard()}
            disabled={busy}
          >
            New board
          </PrimaryButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <div className="grid min-h-[560px] gap-4 xl:grid-cols-[minmax(14rem,17rem)_minmax(0,1fr)_minmax(17rem,21rem)]">
        <aside className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_18px_50px_rgba(0,0,0,0.28)]">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Boards</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Persisted layouts and notes.
              </div>
            </div>
          </div>
          <div className="forge-ops-panel__body">
            <div className="max-h-[min(62vh,680px)] space-y-1 overflow-auto pr-1">
              {boards.length === 0 ? (
                <div className="rounded border border-dashed border-forge-platinum/15 bg-black/35 p-4 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    No boards yet
                  </div>
                  <div className="mt-1 leading-5 text-forge-mist/75">
                    Create a board to start arranging durable notes.
                  </div>
                </div>
              ) : (
                boards.map((b) => (
                  <button
                    key={b.id}
                    type="button"
                    onClick={() => void loadBoard(b.id)}
                    className={[
                      "w-full rounded border px-2.5 py-2 text-left text-xs transition shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]",
                      active?.id === b.id
                        ? "border-forge-ember/50 bg-forge-ember/10 text-forge-ash shadow-[inset_3px_0_0_rgba(255,122,51,0.8)]"
                        : "border-forge-platinum/10 bg-black/25 text-forge-mist hover:border-forge-ember/30 hover:bg-forge-ember/5",
                    ].join(" ")}
                  >
                    <div className="truncate font-semibold">{b.title}</div>
                    <div className="mt-1 text-[10px] text-forge-mist/80">
                      #{b.id} · {formatTime(b.updatedAtMs)}
                    </div>
                  </button>
                ))
              )}
            </div>
          </div>
        </aside>

        <main className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_22px_70px_rgba(0,0,0,0.34)]">
          <div className="forge-ops-panel__head">
            <div className="min-w-0">
              <div className="forge-ops-title truncate">
                {active ? active.title : "Workspace"}
              </div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Drag note headers, resize from corners, inline edits auto-save.
              </div>
            </div>
            {active ? (
              <GhostButton
                className="h-9 px-3"
                onClick={async () => {
                  if (!active || !window.confirm("Delete board and all notes?"))
                    return;
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
            ) : null}
          </div>
          <div className="forge-ops-panel__body">
            {!active ? (
              <div className="flex min-h-[520px] items-center justify-center rounded border border-dashed border-forge-platinum/15 bg-black/35 p-6 text-center text-sm text-forge-mist">
                <div className="max-w-sm">
                  <div className="text-base font-semibold text-forge-ash">
                    Select or create a board
                  </div>
                  <p className="mt-2 leading-6 text-forge-mist/75">
                    Boards preserve note geometry, content, and selection state
                    for workspace planning.
                  </p>
                  <PrimaryButton
                    className="mt-4 h-9 px-3"
                    onClick={() => void createBoard()}
                    disabled={busy}
                  >
                    New board
                  </PrimaryButton>
                </div>
              </div>
            ) : (
              <div className="relative min-h-[520px] overflow-auto rounded border border-forge-platinum/10 bg-forge-black bg-[linear-gradient(rgba(255,255,255,0.035)_1px,transparent_1px),linear-gradient(90deg,rgba(255,255,255,0.035)_1px,transparent_1px)] bg-[length:32px_32px] shadow-inner sm:min-h-[720px]">
                {active.notes.length === 0 ? (
                  <div className="flex min-h-[520px] items-center justify-center p-6 text-center text-sm text-forge-mist sm:min-h-[720px]">
                    <div className="max-w-xs rounded border border-dashed border-forge-platinum/15 bg-black/45 p-5">
                      <div className="font-semibold text-forge-ash">
                        Empty board
                      </div>
                      <p className="mt-2 leading-6 text-forge-mist/75">
                        Place the first note from the inspector to begin mapping
                        this surface.
                      </p>
                      <PrimaryButton
                        className="mt-4 h-9 px-3"
                        onClick={() => void addNote()}
                        disabled={busy}
                      >
                        Place note
                      </PrimaryButton>
                    </div>
                  </div>
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
            )}
          </div>
        </main>

        <aside className="forge-ops-panel min-w-0 bg-forge-carbon/90 shadow-[0_18px_50px_rgba(0,0,0,0.28)]">
          <div className="forge-ops-panel__head">
            <div>
              <div className="forge-ops-title">Inspector</div>
              <div className="mt-1 text-xs text-forge-mist/65">
                Create notes and review selection state.
              </div>
            </div>
          </div>
          <div className="forge-ops-panel__body space-y-4">
            <div className="rounded border border-forge-platinum/10 bg-black/30 p-3 text-xs text-forge-mist shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
              <div className="flex items-center justify-between gap-2">
                <div className="forge-ops-label">Add note</div>
                <span className="rounded-full border border-forge-ember/20 bg-forge-ember/10 px-2 py-0.5 font-mono text-[10px] text-forge-emberSoft">
                  {nextPlacement.x},{nextPlacement.y}
                </span>
              </div>
              <div className="mt-3 grid gap-2">
                <label>
                  <span className="text-[10px] uppercase tracking-wide">
                    Title
                  </span>
                  <input
                    aria-label="New note title"
                    className="forge-input mt-1 w-full"
                    value={newNote.title}
                    onChange={(e) =>
                      setNewNote((n) => ({ ...n, title: e.target.value }))
                    }
                  />
                </label>
                <label>
                  <span className="text-[10px] uppercase tracking-wide">
                    Placement
                  </span>
                  <div className="mt-1 rounded border border-forge-platinum/10 bg-black/30 px-2 py-2 font-mono text-[11px] text-forge-mist">
                    x {nextPlacement.x} / y {nextPlacement.y}
                  </div>
                </label>
                <label>
                  <span className="text-[10px] uppercase tracking-wide">
                    Body
                  </span>
                  <textarea
                    aria-label="New note body"
                    className="forge-input mt-1 min-h-[92px] w-full"
                    value={newNote.body}
                    onChange={(e) =>
                      setNewNote((n) => ({ ...n, body: e.target.value }))
                    }
                  />
                </label>
              </div>
              <div className="mt-3">
                <PrimaryButton
                  className="h-9 w-full px-3"
                  onClick={() => void addNote()}
                  disabled={busy || !active}
                >
                  Place note
                </PrimaryButton>
              </div>
            </div>

            <div className="rounded border border-forge-platinum/10 bg-black/30 p-3 text-xs text-forge-mist shadow-[inset_0_1px_0_rgba(255,255,255,0.03)]">
              <div className="forge-ops-label">Selected note</div>
              {selectedNote ? (
                <div className="mt-3 space-y-2">
                  <div className="font-semibold text-forge-ash">
                    {selectedNote.title}
                  </div>
                  <div className="grid grid-cols-2 gap-2 text-[11px]">
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      x {selectedNote.x}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      y {selectedNote.y}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      w {selectedNote.width}
                    </div>
                    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
                      h {selectedNote.height}
                    </div>
                  </div>
                  <div className="text-[11px] text-forge-mist/75">
                    {selectedNote.pinned
                      ? "Pinned above board stack."
                      : "Standard board layer."}
                  </div>
                  <GhostButton
                    className="h-9 px-3"
                    onClick={() => void deleteSelectedNote()}
                    disabled={busy}
                  >
                    Delete note
                  </GhostButton>
                </div>
              ) : (
                <div className="mt-3 text-[11px] text-forge-mist/75">
                  Select a note to inspect its geometry and layer state.
                </div>
              )}
            </div>

            <div className="rounded border border-forge-platinum/10 bg-black/30 p-3 text-[11px] leading-relaxed text-forge-mist/80">
              Cross-links can be typed into note bodies as job ids, dossier ids,
              or paths. A dedicated link builder is still pending.
            </div>
          </div>
        </aside>
      </div>
    </div>
  );
}

function NoteCard(props: {
  note: CanvasNote;
  busy: boolean;
  selected: boolean;
  onSelect: () => void;
  onSave: (
    n: CanvasNote,
    patch: Record<string, unknown>,
  ) => void | Promise<void>;
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
  const dragRef = useRef<{
    pointerId: number;
    startX: number;
    startY: number;
    originX: number;
    originY: number;
    currentX: number;
    currentY: number;
  } | null>(null);
  const resizeRef = useRef<{
    pointerId: number;
    startX: number;
    startY: number;
    originW: number;
    originH: number;
    currentW: number;
    currentH: number;
  } | null>(null);
  const noteNodeRef = useRef<HTMLDivElement | null>(null);
  const saveTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useLayoutEffect(() => {
    applyNoteGeometry(noteNodeRef.current, {
      x,
      y,
      width: w,
      height: h,
      pinned,
    });
  }, [h, pinned, w, x, y]);

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
    const current = {
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
      originX: x,
      originY: y,
      currentX: x,
      currentY: y,
    };
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
    const current = {
      pointerId: e.pointerId,
      startX: e.clientX,
      startY: e.clientY,
      originW: w,
      originH: h,
      currentW: w,
      currentH: h,
    };
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
      ref={noteNodeRef}
      className={[
        "absolute overflow-hidden rounded border bg-forge-iron/95 shadow-[0_18px_44px_rgba(0,0,0,0.38)] backdrop-blur",
        selected
          ? "border-forge-ember/55 ring-1 ring-inset ring-forge-ember/45"
          : "border-forge-platinum/15 hover:border-forge-platinum/25",
        dragging || resizing ? "ring-1 ring-inset ring-forge-ember/40" : "",
      ].join(" ")}
      onPointerDown={() => onSelect()}
    >
      <div
        className="flex touch-none cursor-move items-center justify-between gap-1 border-b border-forge-platinum/10 bg-black/45 px-2 py-1"
        onPointerDown={startDrag}
        title="Click and hold to drag"
      >
        <input
          aria-label="Note title"
          className="min-w-0 flex-1 bg-transparent text-xs font-semibold text-forge-ash outline-none"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <label className="flex items-center gap-1 rounded border border-forge-platinum/10 bg-black/25 px-1.5 py-0.5 text-[10px] text-forge-mist">
          <input
            aria-label="Pin note"
            type="checkbox"
            checked={pinned}
            onChange={(e) => setPinned(e.target.checked)}
          />
          pin
        </label>
      </div>
      <textarea
        aria-label="Note body"
        className="h-[calc(100%-66px)] w-full resize-none bg-transparent p-2 pb-12 text-[11px] leading-relaxed text-forge-mist outline-none"
        value={body}
        onChange={(e) => setBody(e.target.value)}
      />
      <div className="absolute bottom-0 left-0 right-0 space-y-1 border-t border-forge-platinum/10 bg-black/65 p-2 text-[10px] text-forge-mist">
        <div className="text-[10px]">
          Auto-saved · select and press Delete/Backspace to remove
        </div>
      </div>
      <button
        type="button"
        aria-label="Resize note"
        title="Click and pull to resize"
        className="absolute bottom-1 right-1 h-4 w-4 touch-none cursor-nwse-resize rounded-sm border border-forge-ember/45 bg-forge-ember/20 shadow-[0_0_16px_rgba(255,122,51,0.18)]"
        onPointerDown={startResize}
      />
    </div>
  );
}
