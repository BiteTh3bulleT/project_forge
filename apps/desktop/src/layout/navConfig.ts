/** Legacy route help retained for command/search surfaces. The shell rail now uses cognitive/metrics modes. */

export type UiMode = "cognitive" | "metrics";

export type NavItem = {
  to: string;
  label: string;
  blurb?: string;
  mode?: "cognitive" | "metrics" | "both";
};

export type NavGroup = {
  id: string;
  label: string;
  mode?: "cognitive" | "metrics" | "both";
  items: readonly NavItem[];
};

export const navGroups: readonly NavGroup[] = [
  {
    id: "start",
    label: "Kernel",
    mode: "both",
    items: [
      {
        to: "/start",
        label: "Start",
        blurb: "Guided boot and recovery actions.",
        mode: "both",
      },
      {
        to: "/dashboard",
        label: "Command Deck",
        blurb: "Kernel status, active work, autonomy, and gateway telemetry.",
        mode: "both",
      },
    ],
  },
  {
    id: "daily",
    label: "Operator",
    mode: "both",
    items: [
      {
        to: "/chat",
        label: "Chat",
        blurb: "Operator threads and job launch.",
        mode: "both",
      },
      {
        to: "/jobs",
        label: "Jobs",
        blurb: "Execution projections and event streams.",
        mode: "both",
      },
      {
        to: "/approvals",
        label: "Approvals",
        blurb: "Risk gates with explicit decisions.",
        mode: "both",
      },
      {
        to: "/reviews",
        label: "Reviews",
        blurb: "Admit or reject imported and generated evidence.",
        mode: "both",
      },
      {
        to: "/workbench",
        label: "Artifacts",
        blurb: "Inspect generated and job-linked files.",
        mode: "cognitive",
      },
    ],
  },
  {
    id: "context",
    label: "Cognitive FS",
    mode: "both",
    items: [
      {
        to: "/memory",
        label: "Episodes",
        blurb: "Search indexed semantic evidence.",
        mode: "both",
      },
      {
        to: "/project-context",
        label: "Context Compile",
        blurb: "Normalize guidance and briefing files.",
        mode: "both",
      },
      {
        to: "/inspectors",
        label: "Inspectors",
        blurb: "Read-only packet, snapshot, and trace evidence.",
        mode: "metrics",
      },
      {
        to: "/dossiers",
        label: "Dossiers",
        blurb: "Dossiers and project preferences.",
        mode: "both",
      },
      {
        to: "/retrieval-runs",
        label: "Retrieval Runs",
        blurb: "Inspect keyword/semantic retrieval.",
        mode: "metrics",
      },
      {
        to: "/lineage",
        label: "Lineage",
        blurb: "Compare retries, replays, and loops.",
        mode: "cognitive",
      },
      {
        to: "/evaluations",
        label: "Evaluations",
        blurb: "Score outcomes and quality.",
        mode: "metrics",
      },
      {
        to: "/insights",
        label: "Insights",
        blurb: "Routing advisories.",
        mode: "cognitive",
      },
    ],
  },
  {
    id: "control",
    label: "Execution",
    mode: "both",
    items: [
      {
        to: "/policy",
        label: "Policy",
        blurb: "Approval profiles and recommendations.",
        mode: "both",
      },
      {
        to: "/strategies",
        label: "Strategies",
        blurb: "Reusable execution playbooks.",
        mode: "both",
      },
      {
        to: "/automation",
        label: "Automation",
        blurb: "Safe rule-based automation.",
        mode: "both",
      },
    ],
  },
  {
    id: "advanced",
    label: "Evidence",
    mode: "metrics",
    items: [
      {
        to: "/canvas",
        label: "Canvas",
        blurb: "Spatial planning board.",
        mode: "cognitive",
      },
      {
        to: "/command",
        label: "Command",
        blurb: "Template launch surface.",
        mode: "metrics",
      },
      {
        to: "/gateway",
        label: "Gateway",
        blurb: "Bounded tool invocation.",
        mode: "metrics",
      },
      {
        to: "/action-lanes",
        label: "Action Lanes",
        blurb: "Operational lanes.",
        mode: "metrics",
      },
      {
        to: "/execution-permissions",
        label: "Permissions",
        blurb: "Permission profile tuning.",
        mode: "metrics",
      },
      {
        to: "/audit",
        label: "Audit",
        blurb: "Audit trail inspection.",
        mode: "metrics",
      },
      {
        to: "/backup",
        label: "Backup / Export",
        blurb: "Create and restore bundles.",
        mode: "metrics",
      },
      {
        to: "/release",
        label: "Release",
        blurb: "Readiness and release records.",
        mode: "metrics",
      },
      {
        to: "/events",
        label: "Events",
        blurb: "Global event stream.",
        mode: "metrics",
      },
    ],
  },
  {
    id: "system",
    label: "Runtime",
    mode: "both",
    items: [
      {
        to: "/sources",
        label: "Sources",
        blurb: "Manage indexed folders.",
        mode: "both",
      },
      {
        to: "/adapters",
        label: "Adapters",
        blurb: "Adapter status and invoke tests.",
        mode: "both",
      },
      {
        to: "/models",
        label: "Models",
        blurb: "Manage bounded runtime assets.",
        mode: "both",
      },
      {
        to: "/settings",
        label: "Settings",
        blurb: "Connection and retrieval defaults.",
        mode: "both",
      },
    ],
  },
] as const;

export const routeHelp: Record<string, { title: string; text: string }> = {
  "/start": {
    title: "Start",
    text: "Use this boot surface to connect sources, search memory, run jobs, and review results.",
  },
  "/dashboard": {
    title: "Command Deck",
    text: "Kernel dashboard for active work, autonomy pulse, capability status, and memory correlation.",
  },
  "/chat": {
    title: "Chat",
    text: "Create threads, discuss work, and queue jobs from conversation context.",
  },
  "/jobs": {
    title: "Jobs",
    text: "Track job lifecycle, logs, packet references, and artifacts.",
  },
  "/approvals": {
    title: "Approvals",
    text: "Approve or deny risky operations before execution continues.",
  },
  "/reviews": {
    title: "Reviews",
    text: "Approve, reject, or defer imported/generated outputs.",
  },
  "/memory": {
    title: "Episodes",
    text: "Find indexed cognitive filesystem evidence and inspect the snippets used to build packets.",
  },
  "/project-context": {
    title: "Context Compile",
    text: "Import and normalize context into durable guidance files and project briefing docs.",
  },
  "/inspectors": {
    title: "Inspectors",
    text: "Read-only operator views for context snapshots, task packets, and execution trace evidence.",
  },
  "/dossiers": {
    title: "Dossiers",
    text: "Manage project dossiers, profile preferences, and linked execution history.",
  },
  "/retrieval-runs": {
    title: "Retrieval Runs",
    text: "Inspect keyword and semantic retrieval runs as evidence.",
  },
  "/lineage": {
    title: "Lineage",
    text: "Compare retries, replays, loops, and linked execution history.",
  },
  "/evaluations": {
    title: "Evaluations",
    text: "Score outcomes and inspect quality records.",
  },
  "/insights": {
    title: "Insights",
    text: "Inspect routing advisories and cognitive filesystem signals.",
  },
  "/policy": {
    title: "Policy",
    text: "Control approval presets and inspect recommendation evidence.",
  },
  "/strategies": {
    title: "Strategies",
    text: "Define repeatable execution strategies and success criteria.",
  },
  "/automation": {
    title: "Automation",
    text: "Run bounded automation rules with dry-run previews and history.",
  },
  "/canvas": {
    title: "Canvas",
    text: "Use the spatial planning board for persisted notes and working memory.",
  },
  "/command": {
    title: "Command",
    text: "Launch system command templates from an operator surface.",
  },
  "/gateway": {
    title: "Gateway",
    text: "Inspect bounded tool invocation and gateway authority evidence.",
  },
  "/action-lanes": {
    title: "Action Lanes",
    text: "Inspect operational lanes for bounded execution.",
  },
  "/execution-permissions": {
    title: "Permissions",
    text: "Tune permission profiles and execution capability boundaries.",
  },
  "/audit": {
    title: "Audit",
    text: "Inspect audit records for committed transitions.",
  },
  "/backup": {
    title: "Backup / Export",
    text: "Create and restore portable bundles.",
  },
  "/release": {
    title: "Release",
    text: "Inspect release readiness and status records.",
  },
  "/events": {
    title: "Events",
    text: "Inspect the global event stream.",
  },
  "/workbench": {
    title: "Artifacts",
    text: "Open generated files and compare text artifacts quickly.",
  },
  "/sources": {
    title: "Sources",
    text: "Manage indexed folders and ingestion scope.",
  },
  "/adapters": {
    title: "Adapters",
    text: "Inspect bounded worker status and adapter diagnostics.",
  },
  "/models": {
    title: "Models",
    text: "Import, verify, load, disable, archive, and inspect bounded model runtime assets.",
  },
  "/settings": {
    title: "Settings",
    text: "Configure adapters, embedding defaults, and retrieval weighting.",
  },
};
