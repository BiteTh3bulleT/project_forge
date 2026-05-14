import { j } from "./client";
import type { ForgeArtifact } from "./types";

export const artifactsApi = {
  list: (params?: { limit?: number; jobId?: string }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.jobId) qs.set("jobId", params.jobId);
    const q = qs.toString();
    return j<{ artifacts: ForgeArtifact[] }>(
      `/api/artifacts${q ? `?${q}` : ""}`,
    );
  },
  get: (id: number) =>
    j<ForgeArtifact>(`/api/artifacts/${encodeURIComponent(String(id))}`),
  content: (id: number) =>
    j<{
      artifact: ForgeArtifact;
      textual: boolean;
      content: string;
      previewLimited: boolean;
    }>(`/api/artifacts/${encodeURIComponent(String(id))}/content`),
};
