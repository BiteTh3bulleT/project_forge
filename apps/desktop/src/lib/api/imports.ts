import { j } from "./client";
import type { ImportedExecution } from "@forge/shared";

export const importsApi = {
  list: (limit = 100, dossierId?: number) => {
    const qs = new URLSearchParams();
    qs.set("limit", String(limit));
    if (dossierId != null) qs.set("dossierId", String(dossierId));
    return j<{ imports: ImportedExecution[] }>(
      `/api/imports/executions?${qs.toString()}`,
    );
  },
  create: (body: Record<string, unknown>) =>
    j<{ importedExecution: ImportedExecution }>("/api/imports/executions", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
