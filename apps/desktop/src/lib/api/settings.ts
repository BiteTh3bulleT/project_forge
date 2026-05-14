import { j } from "./client";
import type { SettingsRecord } from "./types";

export const settingsApi = {
  get: () => j<SettingsRecord>("/api/settings"),
  patch: (body: Record<string, unknown>) =>
    j<SettingsRecord>("/api/settings", {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  ollamaModels: (baseUrl?: string) => {
    const q = new URLSearchParams();
    if (baseUrl) q.set("baseUrl", baseUrl);
    const path = `/api/settings/ollama-models${q.toString() ? `?${q.toString()}` : ""}`;
    return j<{
      models: string[];
      baseUrl: string;
      status: string;
      error?: string;
    }>(path);
  },
};
