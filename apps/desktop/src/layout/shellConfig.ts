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
  | "project-context"
  | "models"
  | "gateway"
  | "inspectors"
  | "audit"
  | "policy"
  | "strategies"
  | "automation"
  | "sources"
  | "adapters"
  | "insights"
  | "lineage"
  | "retrieval-runs"
  | "evaluations"
  | "action-lanes"
  | "execution-permissions"
  | "backup"
  | "release"
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
  {
    id: "chat",
    label: "Chat",
    shortLabel: "CH",
    route: "/chat",
    description:
      "Operator threads and job launch from persisted conversation context.",
    primary: true,
  },
  {
    id: "workbench",
    label: "Artifacts",
    shortLabel: "AR",
    route: "/workbench",
    description:
      "Artifact inspection, file previews, and job-linked output review.",
    primary: true,
  },
  {
    id: "canvas",
    label: "Canvas",
    shortLabel: "CV",
    route: "/canvas",
    description: "Working-memory boards with persisted notes and coordinates.",
    primary: true,
  },
  {
    id: "dossiers",
    label: "Dossiers",
    shortLabel: "DS",
    route: "/dossiers",
    description: "Project dossiers, policy bias, and linked execution history.",
    primary: true,
  },
  {
    id: "jobs",
    label: "Jobs",
    shortLabel: "JB",
    route: "/jobs",
    description:
      "Execution projections, approval state, artifacts, and append-only events.",
    primary: true,
  },
  {
    id: "reviews",
    label: "Reviews",
    shortLabel: "RV",
    route: "/reviews",
    description: "Human review decisions for imports and generated outputs.",
    primary: true,
  },
  {
    id: "approvals",
    label: "Approvals",
    shortLabel: "AP",
    route: "/approvals",
    description:
      "Risk-gated approval queue with explicit request and decision records.",
    primary: true,
  },
  {
    id: "autonomy",
    label: "Autonomy",
    shortLabel: "AU",
    route: "/autonomy",
    description:
      "Dream-state, intents, charters, budgets, and self-initiated decision telemetry.",
    primary: true,
  },
  {
    id: "models",
    label: "Models",
    shortLabel: "MD",
    route: "/models",
    description:
      "Bounded model runtime inventory, lifecycle controls, and inspection.",
    primary: true,
  },
  {
    id: "settings",
    label: "Settings",
    shortLabel: "ST",
    route: "/settings",
    description:
      "Local workspace configuration, models, and retrieval defaults.",
    primary: true,
  },
] as const;

const secondaryShellTools: readonly ShellToolDefinition[] = [
  {
    id: "start",
    label: "Start",
    shortLabel: "ST",
    route: "/start",
    description: "Guided boot, source connection, and recovery surface.",
    primary: false,
  },
  {
    id: "dashboard",
    label: "Command Deck",
    shortLabel: "KD",
    route: "/dashboard",
    description:
      "Kernel status, active work, autonomy, gateway, and correlation telemetry.",
    primary: false,
  },
  {
    id: "command",
    label: "Command",
    shortLabel: "CM",
    route: "/command",
    description: "Template launch surface for system commands.",
    primary: false,
  },
  {
    id: "memory",
    label: "Episodes",
    shortLabel: "ME",
    route: "/memory",
    description: "Indexed cognitive filesystem evidence search.",
    primary: false,
  },
  {
    id: "project-context",
    label: "Context Compile",
    shortLabel: "CC",
    route: "/project-context",
    description:
      "Context normalization into durable guidance and briefing files.",
    primary: false,
  },
  {
    id: "insights",
    label: "Insights",
    shortLabel: "IS",
    route: "/insights",
    description: "Routing advisories and cognitive filesystem signals.",
    primary: false,
  },
  {
    id: "lineage",
    label: "Lineage",
    shortLabel: "LN",
    route: "/lineage",
    description: "Retry, replay, and loop comparison surfaces.",
    primary: false,
  },
  {
    id: "retrieval-runs",
    label: "Retrieval Runs",
    shortLabel: "RR",
    route: "/retrieval-runs",
    description: "Keyword and semantic retrieval run evidence.",
    primary: false,
  },
  {
    id: "evaluations",
    label: "Evaluations",
    shortLabel: "EV",
    route: "/evaluations",
    description: "Operator and scoring records for outcome quality.",
    primary: false,
  },
  {
    id: "gateway",
    label: "Gateway",
    shortLabel: "GW",
    route: "/gateway",
    description: "Bounded tool execution authority and invocation evidence.",
    primary: false,
  },
  {
    id: "action-lanes",
    label: "Action Lanes",
    shortLabel: "AL",
    route: "/action-lanes",
    description: "Operational lane state for bounded execution.",
    primary: false,
  },
  {
    id: "execution-permissions",
    label: "Permissions",
    shortLabel: "PX",
    route: "/execution-permissions",
    description:
      "Permission profile tuning and execution capability boundaries.",
    primary: false,
  },
  {
    id: "inspectors",
    label: "Inspectors",
    shortLabel: "IN",
    route: "/inspectors",
    description: "Read-only packet, snapshot, and trace evidence surfaces.",
    primary: false,
  },
  {
    id: "audit",
    label: "Audit",
    shortLabel: "AU",
    route: "/audit",
    description: "Audit trail inspection for committed transitions.",
    primary: false,
  },
  {
    id: "policy",
    label: "Policy",
    shortLabel: "PL",
    route: "/policy",
    description:
      "Approval profiles, policy advisories, and recommendation evidence.",
    primary: false,
  },
  {
    id: "strategies",
    label: "Strategies",
    shortLabel: "ST",
    route: "/strategies",
    description: "Reusable execution playbooks and success criteria.",
    primary: false,
  },
  {
    id: "automation",
    label: "Automation",
    shortLabel: "AM",
    route: "/automation",
    description: "Rule-governed automation with previews and history.",
    primary: false,
  },
  {
    id: "sources",
    label: "Sources",
    shortLabel: "SC",
    route: "/sources",
    description: "Indexed source folders and ingestion scope.",
    primary: false,
  },
  {
    id: "adapters",
    label: "Adapters",
    shortLabel: "AD",
    route: "/adapters",
    description: "Bounded worker status and adapter diagnostics.",
    primary: false,
  },
  {
    id: "logs",
    label: "Events",
    shortLabel: "EV",
    route: "/events",
    description: "Global event and log stream.",
    primary: false,
  },
  {
    id: "backup",
    label: "Backup",
    shortLabel: "BK",
    route: "/backup",
    description: "Portable export and restore bundles.",
    primary: false,
  },
  {
    id: "release",
    label: "Release",
    shortLabel: "RL",
    route: "/release",
    description: "Release readiness and status records.",
    primary: false,
  },
  {
    id: "layouts",
    label: "Layouts",
    shortLabel: "LY",
    route: "/layouts",
    description: "Monitor-aware workspace layout manager.",
    primary: false,
  },
] as const;

export const allShellTools: readonly ShellToolDefinition[] = [
  ...primaryShellTools,
  ...secondaryShellTools,
] as const;
export const assignableShellTools: readonly ShellToolDefinition[] =
  allShellTools.filter(
    (tool) => tool.id !== "start" && tool.id !== "dashboard",
  );

export function getShellTool(pathname: string): ShellToolDefinition {
  if (pathname.startsWith("/jobs/")) {
    return {
      id: "job-detail",
      label: "Job Detail",
      shortLabel: "JD",
      route: pathname,
      description:
        "Focused execution detail with packet, events, approval state, and artifacts.",
      primary: false,
    };
  }
  const direct = allShellTools.find((tool) => tool.route === pathname);
  if (direct) return direct;
  const prefix = allShellTools.find(
    (tool) => pathname.startsWith(`${tool.route}/`) && tool.route !== "/jobs",
  );
  if (prefix) return { ...prefix, route: pathname };
  return {
    id: "other",
    label:
      pathname === "/"
        ? "Workspace"
        : pathname.replace(/^\//, "") || "Workspace",
    shortLabel: "WS",
    route: pathname,
    description: "Additional FORGE system surface.",
    primary: false,
  };
}
