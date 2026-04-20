/** Left-rail navigation — guided mode defaults to practical day-to-day routes. */

export type UiMode = "guided" | "pro";

export type NavItem = {
  to: string;
  label: string;
  blurb?: string;
  mode?: "guided" | "pro" | "both";
};

export type NavGroup = {
  id: string;
  label: string;
  mode?: "guided" | "pro" | "both";
  items: readonly NavItem[];
};

export const navGroups: readonly NavGroup[] = [
  {
    id: "start",
    label: "Start",
    mode: "both",
    items: [
      { to: "/start", label: "Start Here", blurb: "Guided setup and next actions.", mode: "both" },
      { to: "/dashboard", label: "Dashboard", blurb: "Main command dashboard with autonomy and gateway telemetry.", mode: "both" },
    ],
  },
  {
    id: "daily",
    label: "Daily Work",
    mode: "both",
    items: [
      { to: "/chat", label: "Chat", blurb: "Ask and queue work.", mode: "both" },
      { to: "/jobs", label: "Jobs", blurb: "Track running and finished work.", mode: "both" },
      { to: "/approvals", label: "Approvals", blurb: "Allow or deny risky actions.", mode: "both" },
      { to: "/reviews", label: "Reviews", blurb: "Review imported/generated output.", mode: "both" },
      { to: "/workbench", label: "Artifacts", blurb: "Inspect generated files.", mode: "guided" },
    ],
  },
  {
    id: "context",
    label: "Project Memory",
    mode: "both",
    items: [
      { to: "/memory", label: "Search Memory", blurb: "Find indexed context fast.", mode: "both" },
      { to: "/project-context", label: "Project Context", blurb: "Normalize and regenerate guidance.", mode: "both" },
      { to: "/dossiers", label: "Projects", blurb: "Dossiers and project preferences.", mode: "both" },
      { to: "/retrieval-runs", label: "Retrieval Runs", blurb: "Inspect keyword/semantic retrieval.", mode: "pro" },
      { to: "/lineage", label: "Retries & Lineage", blurb: "Compare retries/replays.", mode: "pro" },
      { to: "/evaluations", label: "Evaluations", blurb: "Score outcomes and quality.", mode: "pro" },
      { to: "/insights", label: "Insights", blurb: "Routing advisories.", mode: "pro" },
    ],
  },
  {
    id: "control",
    label: "Control",
    mode: "both",
    items: [
      { to: "/policy", label: "Policy", blurb: "Approval profiles and recommendations.", mode: "both" },
      { to: "/strategies", label: "Strategies", blurb: "Reusable execution playbooks.", mode: "both" },
      { to: "/automation", label: "Automation", blurb: "Safe rule-based automation.", mode: "both" },
    ],
  },
  {
    id: "advanced",
    label: "Advanced",
    mode: "pro",
    items: [
      { to: "/canvas", label: "Canvas", blurb: "Spatial planning board.", mode: "pro" },
      { to: "/command", label: "Command", blurb: "Template launch surface.", mode: "pro" },
      { to: "/gateway", label: "Tool Gateway", blurb: "Bounded tool invocation.", mode: "pro" },
      { to: "/action-lanes", label: "Action Lanes", blurb: "Operational lanes.", mode: "pro" },
      { to: "/execution-permissions", label: "Execution Permissions", blurb: "Permission profile tuning.", mode: "pro" },
      { to: "/audit", label: "Audit", blurb: "Audit trail inspection.", mode: "pro" },
      { to: "/backup", label: "Backup / Export", blurb: "Create and restore bundles.", mode: "pro" },
      { to: "/release", label: "Release", blurb: "Readiness and release records.", mode: "pro" },
      { to: "/events", label: "Events", blurb: "Global event stream.", mode: "pro" },
    ],
  },
  {
    id: "system",
    label: "System",
    mode: "both",
    items: [
      { to: "/sources", label: "Source Folders", blurb: "Manage indexed folders.", mode: "both" },
      { to: "/adapters", label: "Adapters", blurb: "Adapter status and invoke tests.", mode: "both" },
      { to: "/settings", label: "Settings", blurb: "Connection and retrieval defaults.", mode: "both" },
    ],
  },
] as const;

export const routeHelp: Record<string, { title: string; text: string }> = {
  "/start": { title: "Start", text: "Use this page for the easiest path: connect sources, search memory, run jobs, and review results." },
  "/dashboard": { title: "Dashboard", text: "Main command dashboard: active work, autonomy pulse, capability status, and memory correlation graph." },
  "/chat": { title: "Chat", text: "Create threads, discuss work, and queue jobs from conversation context." },
  "/jobs": { title: "Jobs", text: "Track job lifecycle, logs, packet references, and artifacts." },
  "/approvals": { title: "Approvals", text: "Approve or deny risky operations before execution continues." },
  "/reviews": { title: "Reviews", text: "Approve, reject, or defer imported/generated outputs." },
  "/memory": { title: "Search Memory", text: "Find indexed context and inspect the source snippets used to build packets." },
  "/project-context": { title: "Project Context", text: "Import and normalize context into durable guidance files and project briefing docs." },
  "/dossiers": { title: "Projects", text: "Manage project dossiers, profile preferences, and linked execution history." },
  "/policy": { title: "Policy", text: "Control approval presets and inspect recommendation evidence." },
  "/strategies": { title: "Strategies", text: "Define repeatable execution strategies and success criteria." },
  "/automation": { title: "Automation", text: "Run bounded automation rules with dry-run previews and history." },
  "/workbench": { title: "Artifacts", text: "Open generated files and compare text artifacts quickly." },
  "/settings": { title: "Settings", text: "Configure adapters, embedding defaults, and retrieval weighting." },
};
