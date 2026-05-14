import { j } from "./client";
import type { Dossier, DossierDetail } from "@forge/shared";

export const dossiersApi = {
  list: (limit = 120) =>
    j<{ dossiers: Dossier[] }>(
      `/api/dossiers?limit=${encodeURIComponent(String(limit))}`,
    ),
  create: (body: Record<string, unknown>) =>
    j<{ dossier: Dossier }>("/api/dossiers", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  detail: (id: number) => j<{ detail: DossierDetail }>(`/api/dossiers/${id}`),
  update: (id: number, body: Record<string, unknown>) =>
    j<{ dossier: Dossier }>(`/api/dossiers/${id}`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  generateBrief: (id: number, notes = "") =>
    j<{ brief: unknown }>(`/api/dossiers/${id}/briefs/generate`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ notes }),
    }),
};
