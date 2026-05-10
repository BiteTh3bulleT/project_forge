import type { ComponentType } from "react";

import { ActionLanesPage } from "../pages/ActionLanesPage";
import { AdaptersPage } from "../pages/AdaptersPage";
import { ApprovalsPage } from "../pages/ApprovalsPage";
import { AuditPage } from "../pages/AuditPage";
import { AutomationPage } from "../pages/AutomationPage";
import { AutonomyPage } from "../pages/AutonomyPage";
import { BackupPage } from "../pages/BackupPage";
import { CanvasPage } from "../pages/CanvasPage";
import { ChatPage } from "../pages/ChatPage";
import { CommandPage } from "../pages/CommandPage";
import { DashboardPage } from "../pages/DashboardPage";
import { DossiersPage } from "../pages/DossiersPage";
import { EvaluationsPage } from "../pages/EvaluationsPage";
import { EventsPage } from "../pages/EventsPage";
import { ExecutionPermissionsPage } from "../pages/ExecutionPermissionsPage";
import { InsightsPage } from "../pages/InsightsPage";
import { InspectorsPage } from "../pages/InspectorsPage";
import { JobsPage } from "../pages/JobsPage";
import { LineagePage } from "../pages/LineagePage";
import { MemoryPage } from "../pages/MemoryPage";
import { ModelsPage } from "../pages/ModelsPage";
import { PolicyPage } from "../pages/PolicyPage";
import { ProjectContextPage } from "../pages/ProjectContextPage";
import { ReleasePage } from "../pages/ReleasePage";
import { RetrievalRunsPage } from "../pages/RetrievalRunsPage";
import { ReviewsPage } from "../pages/ReviewsPage";
import { SettingsPage } from "../pages/SettingsPage";
import { SourcesPage } from "../pages/SourcesPage";
import { StartPage } from "../pages/StartPage";
import { StrategiesPage } from "../pages/StrategiesPage";
import { SystemPage } from "../pages/SystemPage";
import { ToolGatewayPage } from "../pages/ToolGatewayPage";
import { WorkbenchPage } from "../pages/WorkbenchPage";
import { WorkspaceLayoutsPage } from "../pages/WorkspaceLayoutsPage";

import type { ShellToolId } from "./shellConfig";

type ToolComponent = ComponentType<Record<string, never>>;

// Map of tool id → page component. Each window in the desktop window manager
// renders the component for its tool, replacing the previous single-route
// foreground pane. Tool ids without a directly mounted component (job-detail,
// memory chunk detail, "other") fall back to the existing route surface via
// the deep-link fallback in the shell.
export const TOOL_COMPONENTS: Partial<Record<ShellToolId, ToolComponent>> = {
  start: StartPage,
  dashboard: DashboardPage,
  system: SystemPage,
  command: CommandPage,
  chat: ChatPage,
  workbench: WorkbenchPage,
  canvas: CanvasPage,
  dossiers: DossiersPage,
  jobs: JobsPage,
  reviews: ReviewsPage,
  approvals: ApprovalsPage,
  autonomy: AutonomyPage,
  models: ModelsPage,
  settings: SettingsPage,
  memory: MemoryPage,
  "project-context": ProjectContextPage,
  insights: InsightsPage,
  lineage: LineagePage,
  "retrieval-runs": RetrievalRunsPage,
  evaluations: EvaluationsPage,
  gateway: ToolGatewayPage,
  "action-lanes": ActionLanesPage,
  "execution-permissions": ExecutionPermissionsPage,
  inspectors: InspectorsPage,
  audit: AuditPage,
  policy: PolicyPage,
  strategies: StrategiesPage,
  automation: AutomationPage,
  sources: SourcesPage,
  adapters: AdaptersPage,
  logs: EventsPage,
  backup: BackupPage,
  release: ReleasePage,
  layouts: WorkspaceLayoutsPage,
};

export function getToolComponent(toolId: ShellToolId): ToolComponent | null {
  return TOOL_COMPONENTS[toolId] ?? null;
}
