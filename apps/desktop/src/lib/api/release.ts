import { j } from "./client";

export const releaseApi = {
  readiness: () =>
    j<{ checklist: Record<string, unknown> }>("/api/release/readiness"),
  artifacts: (limit?: number) => {
    const q =
      limit != null ? `?limit=${encodeURIComponent(String(limit))}` : "";
    return j<{ artifacts: unknown[] }>(`/api/release/artifacts${q}`);
  },
  recordArtifact: (body: Record<string, unknown>) =>
    j<{ artifact: unknown }>("/api/release/artifacts", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  firstRun: () =>
    j<{ firstRun: Record<string, unknown> }>("/api/release/first-run"),
};
