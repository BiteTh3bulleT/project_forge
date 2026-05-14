import { j } from "./client";

export const backupApi = {
  bundles: (limit?: number) => {
    const q =
      limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
    return j<{
      bundles: unknown[];
      backupDir: string;
      exportDir: string;
      knownKinds: string[];
    }>(`/api/backup/bundles${q}`);
  },
  createBundle: (body: Record<string, unknown>) =>
    j<{ bundle: unknown }>("/api/backup/bundles", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  deleteBundle: (id: number) =>
    j<void>(`/api/backup/bundles/${encodeURIComponent(String(id))}`, {
      method: "DELETE",
    }),
  restore: (body: Record<string, unknown>) =>
    j<{
      result?: Record<string, unknown>;
      governance?: Record<string, unknown>;
    }>("/api/backup/restore", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
