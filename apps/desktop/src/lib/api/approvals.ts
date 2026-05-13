import { j } from "./client";
import type { ApprovalRequest } from "@forge/shared";

export const approvalsApi = {
  list: (status = "pending", limit = 100) =>
    j<{ approvals: ApprovalRequest[] }>(
      `/api/approvals?status=${encodeURIComponent(status)}&limit=${encodeURIComponent(String(limit))}`,
    ),
  get: (id: number) =>
    j<{ approval: ApprovalRequest }>(
      `/api/approvals/${encodeURIComponent(String(id))}`,
    ),
  approve: (id: number, note = "") =>
    j<{ decision: unknown }>(`/api/approvals/${id}/approve`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ actor: "operator", note }),
    }),
  deny: (id: number, note = "") =>
    j<{ decision: unknown }>(`/api/approvals/${id}/deny`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ actor: "operator", note }),
    }),
  cancel: (id: number, note = "") =>
    j<{ decision: unknown }>(`/api/approvals/${id}/cancel`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ actor: "operator", note }),
    }),
};
