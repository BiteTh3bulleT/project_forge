import { j } from "./client";
import type { CanvasBoard, CanvasBoardDetail, CanvasNote } from "./types";

export const canvasApi = {
  boards: {
    list: (limit = 60) =>
      j<{ boards: CanvasBoard[] }>(
        `/api/canvas/boards?limit=${encodeURIComponent(String(limit))}`,
      ),
    create: (body: { title?: string; dossierId?: number }) =>
      j<{ board: CanvasBoard }>("/api/canvas/boards", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      }),
    get: (id: number) =>
      j<CanvasBoardDetail>(
        `/api/canvas/boards/${encodeURIComponent(String(id))}`,
      ),
    delete: (id: number) =>
      j<void>(`/api/canvas/boards/${encodeURIComponent(String(id))}`, {
        method: "DELETE",
      }),
    createNote: (
      boardId: number,
      body: {
        title?: string;
        body?: string;
        x?: number;
        y?: number;
        width?: number;
        height?: number;
      },
    ) =>
      j<{ note: CanvasNote }>(
        `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    patchNote: (
      boardId: number,
      noteId: number,
      body: Record<string, unknown>,
    ) =>
      j<{ note: CanvasNote }>(
        `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
        {
          method: "PATCH",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify(body),
        },
      ),
    deleteNote: (boardId: number, noteId: number) =>
      j<void>(
        `/api/canvas/boards/${encodeURIComponent(String(boardId))}/notes/${encodeURIComponent(String(noteId))}`,
        { method: "DELETE" },
      ),
  },
};
