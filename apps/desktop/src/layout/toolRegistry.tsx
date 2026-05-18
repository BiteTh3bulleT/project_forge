import { lazy, type ComponentType, type LazyExoticComponent } from "react";

import type { ShellToolId } from "./shellConfig";

type ToolComponent =
  | ComponentType<Record<string, never>>
  | LazyExoticComponent<ComponentType<Record<string, never>>>;
type ToolModule<K extends string> = Record<K, ComponentType<Record<string, never>>>;

function lazyTool<K extends string>(
  load: () => Promise<ToolModule<K>>,
  exportName: K,
): ToolComponent {
  return lazy(async () => ({ default: (await load())[exportName] }));
}

// Map of tool id → page component. Each window in the desktop window manager
// renders the component for its tool, replacing the previous single-route
// foreground pane. Tool ids without a directly mounted component (job-detail,
// memory chunk detail, "other") fall back to the existing route surface via
// the deep-link fallback in the shell.
export const TOOL_COMPONENTS: Partial<Record<ShellToolId, ToolComponent>> = {
  start: lazyTool(() => import("../pages/StartPage"), "StartPage"),
  dashboard: lazyTool(() => import("../pages/DashboardPage"), "DashboardPage"),
  system: lazyTool(() => import("../pages/SystemPage"), "SystemPage"),
  command: lazyTool(() => import("../pages/CommandPage"), "CommandPage"),
  "operator-apps": lazyTool(
    () => import("../pages/OperatorAppsPage"),
    "OperatorAppsPage",
  ),
  chat: lazyTool(() => import("../pages/ChatPage"), "ChatPage"),
  workbench: lazyTool(() => import("../pages/WorkbenchPage"), "WorkbenchPage"),
  canvas: lazyTool(() => import("../pages/CanvasPage"), "CanvasPage"),
  dossiers: lazyTool(() => import("../pages/DossiersPage"), "DossiersPage"),
  jobs: lazyTool(() => import("../pages/JobsPage"), "JobsPage"),
  reviews: lazyTool(() => import("../pages/ReviewsPage"), "ReviewsPage"),
  approvals: lazyTool(() => import("../pages/ApprovalsPage"), "ApprovalsPage"),
  autonomy: lazyTool(() => import("../pages/AutonomyPage"), "AutonomyPage"),
  models: lazyTool(() => import("../pages/ModelsPage"), "ModelsPage"),
  settings: lazyTool(() => import("../pages/SettingsPage"), "SettingsPage"),
  memory: lazyTool(() => import("../pages/MemoryPage"), "MemoryPage"),
  "project-context": lazyTool(
    () => import("../pages/ProjectContextPage"),
    "ProjectContextPage",
  ),
  insights: lazyTool(() => import("../pages/InsightsPage"), "InsightsPage"),
  lineage: lazyTool(() => import("../pages/LineagePage"), "LineagePage"),
  "retrieval-runs": lazyTool(
    () => import("../pages/RetrievalRunsPage"),
    "RetrievalRunsPage",
  ),
  evaluations: lazyTool(
    () => import("../pages/EvaluationsPage"),
    "EvaluationsPage",
  ),
  gateway: lazyTool(() => import("../pages/ToolGatewayPage"), "ToolGatewayPage"),
  "action-lanes": lazyTool(
    () => import("../pages/ActionLanesPage"),
    "ActionLanesPage",
  ),
  "execution-permissions": lazyTool(
    () => import("../pages/ExecutionPermissionsPage"),
    "ExecutionPermissionsPage",
  ),
  inspectors: lazyTool(() => import("../pages/InspectorsPage"), "InspectorsPage"),
  audit: lazyTool(() => import("../pages/AuditPage"), "AuditPage"),
  policy: lazyTool(() => import("../pages/PolicyPage"), "PolicyPage"),
  strategies: lazyTool(() => import("../pages/StrategiesPage"), "StrategiesPage"),
  automation: lazyTool(() => import("../pages/AutomationPage"), "AutomationPage"),
  sources: lazyTool(() => import("../pages/SourcesPage"), "SourcesPage"),
  adapters: lazyTool(() => import("../pages/AdaptersPage"), "AdaptersPage"),
  logs: lazyTool(() => import("../pages/EventsPage"), "EventsPage"),
  backup: lazyTool(() => import("../pages/BackupPage"), "BackupPage"),
  release: lazyTool(() => import("../pages/ReleasePage"), "ReleasePage"),
  layouts: lazyTool(
    () => import("../pages/WorkspaceLayoutsPage"),
    "WorkspaceLayoutsPage",
  ),
};

export function getToolComponent(toolId: ShellToolId): ToolComponent | null {
  return TOOL_COMPONENTS[toolId] ?? null;
}
