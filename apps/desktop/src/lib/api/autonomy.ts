import { j } from "./client";
import type { ForgeEvent } from "@forge/shared";
import type {
  AutonomyBudgetRecord,
  AutonomyCharterRecord,
  AutonomyDecisionRecord,
  AutonomyIntentRecord,
  AutonomyStatusSnapshot,
} from "./types";

export const autonomyApi = {
  status: () => j<AutonomyStatusSnapshot>("/api/autonomy/status"),
  intents: (params?: { status?: string; limit?: number }) => {
    const qs = new URLSearchParams();
    if (params?.status) qs.set("status", params.status);
    if (params?.limit != null) qs.set("limit", String(params.limit));
    const q = qs.toString();
    return j<{ intents: AutonomyIntentRecord[] }>(
      `/api/autonomy/intents${q ? `?${q}` : ""}`,
    );
  },
  explainIntent: (id: string) =>
    j<Record<string, unknown>>(
      `/api/autonomy/intents/${encodeURIComponent(id)}/explain`,
    ),
  decisions: (limit = 80) =>
    j<{ decisions: AutonomyDecisionRecord[] }>(
      `/api/autonomy/decisions?limit=${encodeURIComponent(String(limit))}`,
    ),
  budgets: () =>
    j<{ budgets: AutonomyBudgetRecord[] }>("/api/autonomy/budgets"),
  charters: (activeOnly = false) =>
    j<{ charters: AutonomyCharterRecord[] }>(
      `/api/autonomy/charters?activeOnly=${encodeURIComponent(String(activeOnly))}`,
    ),
  events: (limit = 120) =>
    j<{ events: ForgeEvent[] }>(
      `/api/autonomy/events?limit=${encodeURIComponent(String(limit))}`,
    ),
};
