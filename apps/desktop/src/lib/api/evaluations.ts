import { j } from "./client";
import type { AdapterMetric, EvaluationRecord } from "@forge/shared";

export const evaluationsApi = {
  create: (body: Record<string, unknown>) =>
    j<{ evaluation: EvaluationRecord }>("/api/evaluations", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  list: (limit = 120, dossierId?: number) => {
    const qs = new URLSearchParams();
    qs.set("limit", String(limit));
    if (dossierId != null) qs.set("dossierId", String(dossierId));
    return j<{ evaluations: EvaluationRecord[] }>(
      `/api/evaluations?${qs.toString()}`,
    );
  },
  metrics: (dossierId?: number) => {
    const q =
      dossierId != null
        ? `?dossierId=${encodeURIComponent(String(dossierId))}`
        : "";
    return j<{ metrics: AdapterMetric[] }>(`/api/evaluations/metrics${q}`);
  },
};
