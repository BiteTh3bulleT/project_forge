import { j } from "./client";
import type { ForgeEvent, SearchHit, SourceRow } from "@forge/shared";

export const sourcesApi = {
  list: () => j<{ sources: SourceRow[] }>("/api/sources"),
  add: (path: string) =>
    j<{ id: number; path: string }>("/api/sources", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ path }),
    }),
  delete: (id: number) =>
    j<void>(`/api/sources/${id}`, {
      method: "DELETE",
    }),
};

export const searchApi = {
  reindex: (sourceId?: number) => {
    const q =
      sourceId != null
        ? `?sourceId=${encodeURIComponent(String(sourceId))}`
        : "";
    return j<{ ok: boolean; scope: string }>(`/api/reindex${q}`, {
      method: "POST",
    });
  },
  search: (q: string, limit = 50) =>
    j<{ hits: SearchHit[] }>(
      `/api/search?q=${encodeURIComponent(q)}&limit=${encodeURIComponent(String(limit))}`,
    ),
  chunk: (id: number) => j<SearchHit>(`/api/chunks/${id}`),
  events: (limit = 120) =>
    j<{ events: ForgeEvent[] }>(
      `/api/events?limit=${encodeURIComponent(String(limit))}`,
    ),
};
