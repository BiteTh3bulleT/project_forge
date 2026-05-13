import { j } from "./client";
import type { JobDetail, JobRecord, JobTemplate } from "@forge/shared";

export const jobsApi = {
  templates: () => j<{ templates: JobTemplate[] }>("/api/jobs/templates"),
  list: (status = "", limit = 120) => {
    const params = new URLSearchParams();
    if (status) params.set("status", status);
    params.set("limit", String(limit));
    return j<{ jobs: JobRecord[] }>(`/api/jobs?${params.toString()}`);
  },
  create: (body: Record<string, unknown>) =>
    j<{ job: JobRecord }>("/api/jobs", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  detail: (id: string, afterEventId = 0) =>
    j<JobDetail>(
      `/api/jobs/${encodeURIComponent(id)}?afterEventId=${encodeURIComponent(String(afterEventId))}`,
    ),
  cancel: (id: string, actor = "operator") =>
    j<{ ok: boolean; jobId: string }>(
      `/api/jobs/${encodeURIComponent(id)}/cancel`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ actor }),
      },
    ),
  retry: (id: string, body: Record<string, unknown>) =>
    j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/retry`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  replay: (id: string, body: Record<string, unknown>) =>
    j<{ job: JobRecord }>(`/api/jobs/${encodeURIComponent(id)}/replay`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
