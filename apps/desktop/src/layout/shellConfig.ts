export type ShellToolId =
  | "chat"
  | "workbench"
  | "canvas"
  | "dossiers"
  | "jobs"
  | "reviews"
  | "approvals"
  | "settings"
  | "autonomy"
  | "layouts"
  | "logs"
  | "start"
  | "dashboard"
  | "command"
  | "memory"
  | "models"
  | "job-detail"
  | "other";

export type ShellToolDefinition = {
  id: ShellToolId;
  label: string;
  shortLabel: string;
  route: string;
  description: string;
  primary: boolean;
};

export const primaryShellTools: readonly ShellToolDefinition[] = [
  { id: "chat", label: "Chat", shortLabel: "CH", route: "/chat", description: "Persistent operator threads and job launch from conversation context.", primary: true },
  { id: "workbench", label: "Workbench", shortLabel: "WB", route: "/workbench", description: "Artifact inspection, file previews, and job-linked output review.", primary: true },
  { id: "canvas", label: "Canvas", shortLabel: "CV", route: "/canvas", description: "Working-memory boards with persisted notes and coordinates.", primary: true },
  { id: "dossiers", label: "Dossiers", shortLabel: "DS", route: "/dossiers", description: "Project memory profiles, policy bias, and linked execution history.", primary: true },
  { id: "jobs", label: "Jobs", shortLabel: "JB", route: "/jobs", description: "Execution queue, lifecycle projections, and append-only event streams.", primary: true },
  { id: "reviews", label: "Reviews", shortLabel: "RV", route: "/reviews", description: "Human review decisions for imports and generated outputs.", primary: true },
  { id: "approvals", label: "Approvals", shortLabel: "AP", route: "/approvals", description: "Risk-gated approval queue with explicit request and decision records.", primary: true },
  { id: "autonomy", label: "Autonomy", shortLabel: "AU", route: "/autonomy", description: "Dream-state, intents, charters, budgets, and self-initiated decision telemetry.", primary: true },
  { id: "models", label: "Models", shortLabel: "MD", route: "/models", description: "FORGE-native model runtime inventory, lifecycle controls, and runtime inspection.", primary: true },
  { id: "settings", label: "Settings", shortLabel: "ST", route: "/settings", description: "Local workspace configuration, models, and retrieval defaults.", primary: true },
] as const;

const secondaryShellTools: readonly ShellToolDefinition[] = [
  { id: "start", label: "Start", shortLabel: "ST", route: "/start", description: "Legacy guided launch surface.", primary: false },
  { id: "dashboard", label: "Dashboard", shortLabel: "DB", route: "/dashboard", description: "Main command dashboard for autonomy, gateway, and correlation telemetry.", primary: false },
  { id: "command", label: "Command", shortLabel: "CM", route: "/command", description: "Template launch surface for system commands.", primary: false },
  { id: "memory", label: "Memory", shortLabel: "MM", route: "/memory", description: "Indexed project memory search.", primary: false },
  { id: "logs", label: "Logs", shortLabel: "LG", route: "/events", description: "Global event and log stream.", primary: false },
  { id: "layouts", label: "Layouts", shortLabel: "LY", route: "/layouts", description: "Monitor-aware workspace layout manager.", primary: false },
] as const;

export const allShellTools: readonly ShellToolDefinition[] = [...primaryShellTools, ...secondaryShellTools] as const;
export const assignableShellTools: readonly ShellToolDefinition[] = allShellTools.filter((tool) => tool.id !== "start" && tool.id !== "dashboard");

export function getShellTool(pathname: string): ShellToolDefinition {
  if (pathname.startsWith("/jobs/")) {
    return {
      id: "job-detail",
      label: "Job Detail",
      shortLabel: "JD",
      route: pathname,
      description: "Focused execution detail with packet, events, approval state, and artifacts.",
      primary: false,
    };
  }
  const direct = allShellTools.find((tool) => tool.route === pathname);
  if (direct) return direct;
  const prefix = allShellTools.find((tool) => pathname.startsWith(`${tool.route}/`) && tool.route !== "/jobs");
  if (prefix) return { ...prefix, route: pathname };
  return {
    id: "other",
    label: pathname === "/" ? "Workspace" : pathname.replace(/^\//, "") || "Workspace",
    shortLabel: "WS",
    route: pathname,
    description: "Additional FORGE system surface.",
    primary: false,
  };
}
