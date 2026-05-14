import { j } from "./client";
import type { ImportReconciliation } from "@forge/shared";

export const reconciliationApi = {
  getByImport: (importId: number) =>
    j<{ reconciliation: ImportReconciliation }>(
      `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
    ),
  saveByImport: (importId: number, body: Record<string, unknown>) =>
    j<{ reconciliation: ImportReconciliation }>(
      `/api/reconciliation/imports/${encodeURIComponent(String(importId))}`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
    ),
  list: (params?: { limit?: number; reviewStatus?: string }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.reviewStatus) qs.set("reviewStatus", params.reviewStatus);
    const q = qs.toString();
    return j<{ reconciliations: ImportReconciliation[] }>(
      `/api/reconciliation${q ? `?${q}` : ""}`,
    );
  },
};
