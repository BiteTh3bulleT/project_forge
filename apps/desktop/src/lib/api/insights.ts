import { j } from "./client";
import type { RoutingInsight } from "@forge/shared";

export const insightsApi = {
  list: (limit = 120, dossierId?: number) => {
    const qs = new URLSearchParams();
    qs.set("limit", String(limit));
    if (dossierId != null) qs.set("dossierId", String(dossierId));
    return j<{ insights: RoutingInsight[] }>(
      `/api/insights?${qs.toString()}`,
    );
  },
  generate: (dossierId?: number) =>
    j<{ insights: RoutingInsight[] }>("/api/insights/generate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ dossierId }),
    }),
};
