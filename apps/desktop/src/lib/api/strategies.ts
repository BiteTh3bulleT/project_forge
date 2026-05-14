import { j } from "./client";
import type { ExecutionStrategy } from "@forge/shared";

export const strategiesApi = {
  list: (params?: { enabled?: boolean; limit?: number }) => {
    const qs = new URLSearchParams();
    if (params?.enabled != null) qs.set("enabled", String(params.enabled));
    if (params?.limit != null) qs.set("limit", String(params.limit));
    const q = qs.toString();
    return j<{ strategies: ExecutionStrategy[] }>(
      `/api/strategies${q ? `?${q}` : ""}`,
    );
  },
  save: (body: Record<string, unknown>) =>
    j<{ strategy: ExecutionStrategy }>("/api/strategies", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
