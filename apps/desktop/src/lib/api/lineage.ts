import { j } from "./client";
import type { JobLineage } from "@forge/shared";

export const lineageApi = {
  byJob: (jobId: string) =>
    j<JobLineage>(`/api/lineage/jobs/${encodeURIComponent(jobId)}`),
};
