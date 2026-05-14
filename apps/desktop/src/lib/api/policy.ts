import { j } from "./client";
import type {
  ApprovalPreset,
  DossierProfile,
  PolicyRecommendation,
} from "@forge/shared";

export const policyApi = {
  listPresets: (limit = 60) =>
    j<{ presets: ApprovalPreset[] }>(
      `/api/policy/presets?limit=${encodeURIComponent(String(limit))}`,
    ),
  savePreset: (body: Record<string, unknown>) =>
    j<{ preset: ApprovalPreset }>("/api/policy/presets", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  getGlobalPreset: () => j<{ presetId: string }>("/api/policy/global-preset"),
  setGlobalPreset: (presetId: string) =>
    j<{ ok: boolean; presetId: string }>("/api/policy/global-preset", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ presetId }),
    }),
  getDossierProfile: (id: number) =>
    j<{ profile: DossierProfile | null }>(
      `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
    ),
  saveDossierProfile: (id: number, body: Record<string, unknown>) =>
    j<{ profile: DossierProfile }>(
      `/api/policy/dossiers/${encodeURIComponent(String(id))}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
  recommend: (body: Record<string, unknown>) =>
    j<{ recommendation: PolicyRecommendation }>("/api/policy/recommend", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  listRecommendations: (params?: { limit?: number; dossierId?: number }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null)
      qs.set("dossierId", String(params.dossierId));
    const q = qs.toString();
    return j<{ recommendations: PolicyRecommendation[] }>(
      `/api/policy/recommendations${q ? `?${q}` : ""}`,
    );
  },
};
