import { j } from "./client";
import type {
  EmbeddingConfig,
  ReembedResult,
  SourceEmbeddingStatus,
} from "@forge/shared";

export const embeddingsApi = {
  status: (provider = "", model = "") => {
    const params = new URLSearchParams();
    if (provider) params.set("provider", provider);
    if (model) params.set("model", model);
    const q = params.toString();
    return j<{ config: EmbeddingConfig; status: SourceEmbeddingStatus[] }>(
      `/api/embeddings/status${q ? `?${q}` : ""}`,
    );
  },
  reembed: (body: Record<string, unknown>) =>
    j<{ result: ReembedResult }>("/api/embeddings/reembed", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
