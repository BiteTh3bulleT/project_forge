import { j } from "./client";
import type { AutomationHistory, AutomationRule } from "@forge/shared";

export const automationApi = {
  listRules: (params?: { enabled?: boolean; limit?: number }) => {
    const qs = new URLSearchParams();
    if (params?.enabled != null) qs.set("enabled", String(params.enabled));
    if (params?.limit != null) qs.set("limit", String(params.limit));
    const q = qs.toString();
    return j<{ rules: AutomationRule[] }>(
      `/api/automation/rules${q ? `?${q}` : ""}`,
    );
  },
  saveRule: (body: Record<string, unknown>) =>
    j<{ rule: AutomationRule }>("/api/automation/rules", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
  history: (limit = 120) =>
    j<{ history: AutomationHistory[] }>(
      `/api/automation/history?limit=${encodeURIComponent(String(limit))}`,
    ),
  runRule: (body: Record<string, unknown>) =>
    j<{
      result: {
        rule: AutomationRule;
        matched: boolean;
        dryRun: boolean;
        preview: Record<string, unknown>;
        executed: boolean;
        execution: Record<string, unknown>;
        historyId: number;
      };
    }>("/api/automation/run", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
