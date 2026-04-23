import { listen } from "@tauri-apps/api/event";
import { getCurrentWindow } from "@tauri-apps/api/window";
import { useEffect, useRef } from "react";
import { Navigate, Route, Routes, useLocation, useNavigate } from "react-router-dom";

import { ForgeErrorBoundary } from "./components/ForgeErrorBoundary";
import { AppShell } from "./layout/AppShell";
import { WORKSPACE_LAYOUT_EVENT, WORKSPACE_NAVIGATE_EVENT, isTauriDesktop } from "./lib/desktop";
import { ActionLanesPage } from "./pages/ActionLanesPage";
import { AdaptersPage } from "./pages/AdaptersPage";
import { AuditPage } from "./pages/AuditPage";
import { AutonomyPage } from "./pages/AutonomyPage";
import { BackupPage } from "./pages/BackupPage";
import { CanvasPage } from "./pages/CanvasPage";
import { ChatPage } from "./pages/ChatPage";
import { StartPage } from "./pages/StartPage";
import { ApprovalsPage } from "./pages/ApprovalsPage";
import { AutomationPage } from "./pages/AutomationPage";
import { CommandPage } from "./pages/CommandPage";
import { DashboardPage } from "./pages/DashboardPage";
import { ExecutionPermissionsPage } from "./pages/ExecutionPermissionsPage";
import { EventsPage } from "./pages/EventsPage";
import { EvaluationsPage } from "./pages/EvaluationsPage";
import { PolicyPage } from "./pages/PolicyPage";
import { ReviewsPage } from "./pages/ReviewsPage";
import { JobDetailPage } from "./pages/JobDetailPage";
import { JobsPage } from "./pages/JobsPage";
import { LineagePage } from "./pages/LineagePage";
import { MemoryDetailPage } from "./pages/MemoryDetailPage";
import { MemoryPage } from "./pages/MemoryPage";
import { ModelsPage } from "./pages/ModelsPage";
import { DossiersPage } from "./pages/DossiersPage";
import { ProjectContextPage } from "./pages/ProjectContextPage";
import { ReleasePage } from "./pages/ReleasePage";
import { RetrievalRunsPage } from "./pages/RetrievalRunsPage";
import { SettingsPage } from "./pages/SettingsPage";
import { ToolGatewayPage } from "./pages/ToolGatewayPage";
import { WorkbenchPage } from "./pages/WorkbenchPage";
import { InsightsPage } from "./pages/InsightsPage";
import { InspectorsPage } from "./pages/InspectorsPage";
import { SourcesPage } from "./pages/SourcesPage";
import { StrategiesPage } from "./pages/StrategiesPage";
import { WorkspaceLayoutsPage } from "./pages/WorkspaceLayoutsPage";
import { useDesktopShellStore } from "./stores/desktopShellStore";
import { useWorkspaceLayoutStore } from "./stores/workspaceLayoutStore";
import { useWorkspaceStore } from "./stores/workspaceStore";
import { useUiStore } from "./stores/uiStore";

function RoutedViews() {
  const location = useLocation();
  return (
    <ForgeErrorBoundary resetKey={location.pathname + location.search}>
      <Routes>
        <Route path="/" element={<Navigate to="/dashboard" replace />} />
        <Route path="/start" element={<StartPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/chat" element={<ChatPage />} />
        <Route path="/workbench" element={<WorkbenchPage />} />
        <Route path="/canvas" element={<CanvasPage />} />
        <Route path="/command" element={<CommandPage />} />
        <Route path="/memory" element={<MemoryPage />} />
        <Route path="/memory/chunk/:id" element={<MemoryDetailPage />} />
        <Route path="/project-context" element={<ProjectContextPage />} />
        <Route path="/inspectors" element={<InspectorsPage />} />
        <Route path="/policy" element={<PolicyPage />} />
        <Route path="/strategies" element={<StrategiesPage />} />
        <Route path="/automation" element={<AutomationPage />} />
        <Route path="/reviews" element={<ReviewsPage />} />
        <Route path="/dossiers" element={<DossiersPage />} />
        <Route path="/retrieval-runs" element={<RetrievalRunsPage />} />
        <Route path="/evaluations" element={<EvaluationsPage />} />
        <Route path="/lineage" element={<LineagePage />} />
        <Route path="/insights" element={<InsightsPage />} />
        <Route path="/jobs" element={<JobsPage />} />
        <Route path="/jobs/:id" element={<JobDetailPage />} />
        <Route path="/approvals" element={<ApprovalsPage />} />
        <Route path="/gateway" element={<ToolGatewayPage />} />
        <Route path="/action-lanes" element={<ActionLanesPage />} />
        <Route path="/execution-permissions" element={<ExecutionPermissionsPage />} />
        <Route path="/audit" element={<AuditPage />} />
        <Route path="/backup" element={<BackupPage />} />
        <Route path="/release" element={<ReleasePage />} />
        <Route path="/sources" element={<SourcesPage />} />
        <Route path="/adapters" element={<AdaptersPage />} />
        <Route path="/models" element={<ModelsPage />} />
        <Route path="/events" element={<EventsPage />} />
        <Route path="/autonomy" element={<AutonomyPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/layouts" element={<WorkspaceLayoutsPage />} />
        <Route path="*" element={<Navigate to="/chat" replace />} />
      </Routes>
    </ForgeErrorBoundary>
  );
}

export default function App() {
  const ping = useWorkspaceStore((s) => s.ping);
  const location = useLocation();
  const navigate = useNavigate();
  const hydrateLayouts = useWorkspaceLayoutStore((s) => s.hydrate);
  const refreshEnvironment = useWorkspaceLayoutStore((s) => s.refreshEnvironment);
  const syncCurrentRoute = useWorkspaceLayoutStore((s) => s.syncCurrentRoute);
  const hydrateShell = useDesktopShellStore((s) => s.hydrate);
  const layoutReady = useWorkspaceLayoutStore((s) => s.ready);
  const locationRef = useRef(`${location.pathname}${location.search}`);
  const currentWindowLabel = useWorkspaceLayoutStore((s) => s.currentWindowLabel);
  const isMainWindow = layoutReady && currentWindowLabel === "main";
  const contrastPreference = useUiStore((s) => s.contrastPreference);
  const effectsPreference = useUiStore((s) => s.effectsPreference);

  useEffect(() => {
    locationRef.current = `${location.pathname}${location.search}`;
  }, [location.pathname, location.search]);

  useEffect(() => {
    document.documentElement.dataset.contrast = contrastPreference;
    document.documentElement.dataset.effects = effectsPreference;
  }, [contrastPreference, effectsPreference]);

  useEffect(() => {
    if (!isMainWindow) return;
    void ping();
    const id = window.setInterval(() => void ping(), 8000);
    return () => window.clearInterval(id);
  }, [ping, isMainWindow]);

  useEffect(() => {
    void hydrateLayouts(location.pathname + location.search);
  }, []);

  useEffect(() => {
    hydrateShell(currentWindowLabel || "main");
  }, [currentWindowLabel, hydrateShell]);

  useEffect(() => {
    void syncCurrentRoute(location.pathname + location.search);
  }, [location.pathname, location.search, syncCurrentRoute]);

  useEffect(() => {
    if (!isMainWindow) return;
    const id = window.setInterval(() => void refreshEnvironment(), 5000);
    return () => window.clearInterval(id);
  }, [refreshEnvironment, isMainWindow]);

  useEffect(() => {
    if (!isTauriDesktop()) {
      const onStorage = (event: StorageEvent) => {
        if (event.key?.startsWith("forge.")) {
          void refreshEnvironment();
        }
      };
      window.addEventListener("storage", onStorage);
      return () => window.removeEventListener("storage", onStorage);
    }
    let disposers: Array<() => void> = [];
    let environmentRefreshTimer: ReturnType<typeof setTimeout> | null = null;

    const scheduleEnvironmentRefresh = () => {
      if (!isMainWindow) return;
      if (environmentRefreshTimer !== null) return;
      environmentRefreshTimer = setTimeout(() => {
        environmentRefreshTimer = null;
        void refreshEnvironment();
      }, 150);
    };

    (async () => {
      const appWindow = getCurrentWindow();
      const handleNavigate = await appWindow.listen<{ route: string }>(WORKSPACE_NAVIGATE_EVENT, (event) => {
        if (event.payload?.route && event.payload.route !== locationRef.current) {
          navigate(event.payload.route);
        }
      });
      disposers.push(handleNavigate);
      disposers.push(await appWindow.onMoved(() => scheduleEnvironmentRefresh()));
      disposers.push(await appWindow.onResized(() => scheduleEnvironmentRefresh()));
      disposers.push(await appWindow.onFocusChanged(() => scheduleEnvironmentRefresh()));
      if (isMainWindow) {
        disposers.push(
          await listen<{ origin?: string }>(WORKSPACE_LAYOUT_EVENT, (event) => {
            const origin = event.payload?.origin?.trim() ?? "";
            // Ignore self-originated sync events to prevent refresh/emission loops.
            if (origin && origin === (currentWindowLabel || "main")) {
              return;
            }
            void refreshEnvironment();
          }),
        );
      }
    })();

    return () => {
      disposers.forEach((dispose) => void dispose());
      if (environmentRefreshTimer !== null) {
        clearTimeout(environmentRefreshTimer);
      }
    };
  }, [navigate, refreshEnvironment, isMainWindow, currentWindowLabel]);

  return (
    <AppShell isMainWindow={isMainWindow}>
      <RoutedViews />
    </AppShell>
  );
}
