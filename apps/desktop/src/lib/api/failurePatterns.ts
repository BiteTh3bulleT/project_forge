import { j } from "./client";
import type { FailurePattern } from "@forge/shared";

export const failurePatternsApi = {
  list: (params?: { limit?: number; dossierId?: number }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.dossierId != null)
      qs.set("dossierId", String(params.dossierId));
    const q = qs.toString();
    return j<{ patterns: FailurePattern[] }>(
      `/api/failure-patterns${q ? `?${q}` : ""}`,
    );
  },
  analyze: (body: Record<string, unknown>) =>
    j<{ patterns: FailurePattern[] }>("/api/failure-patterns/analyze", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
