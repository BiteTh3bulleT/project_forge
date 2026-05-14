import { j } from "./client";
import type { ReviewRecord } from "@forge/shared";

export const reviewsApi = {
  list: (params?: { limit?: number; status?: string }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.status) qs.set("status", params.status);
    const q = qs.toString();
    return j<{ reviews: ReviewRecord[] }>(`/api/reviews${q ? `?${q}` : ""}`);
  },
  create: (body: Record<string, unknown>) =>
    j<{ review: ReviewRecord }>("/api/reviews", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  update: (id: number, body: Record<string, unknown>) =>
    j<{ review: ReviewRecord }>(
      `/api/reviews/${encodeURIComponent(String(id))}`,
      {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
};
