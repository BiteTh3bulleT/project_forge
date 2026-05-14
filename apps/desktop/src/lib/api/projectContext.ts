import { j } from "./client";
import type { ProjectContextRecord } from "@forge/shared";

export const projectContextApi = {
  get: () =>
    j<{ record: ProjectContextRecord | null }>("/api/project-context"),
  import: (sourcePath = "", notes = "") =>
    j<{ record: ProjectContextRecord }>("/api/project-context/import", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sourcePath, notes }),
    }),
  regenerate: () =>
    j<{ record: ProjectContextRecord }>("/api/project-context/regenerate", {
      method: "POST",
    }),
};
