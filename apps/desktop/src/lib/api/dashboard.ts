import { j } from "./client";
import type { DashboardSummary } from "@forge/shared";

export const dashboardApi = {
  summary: () => j<DashboardSummary>("/api/dashboard"),
};
