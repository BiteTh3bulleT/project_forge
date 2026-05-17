import { useEffect, useRef, useState } from "react";
import {
  Navigate,
  Route,
  Routes,
  useLocation,
  useNavigate,
} from "react-router-dom";

import { ForgeErrorBoundary } from "./components/ForgeErrorBoundary";
import { AppShell } from "./layout/AppShell";
import { clearForgeApiTokenCache } from "./lib/api/client";
import { isTauriDesktop, isShellHostWindowLabel } from "./lib/desktop";
import {
  subscribeToCurrentWindowLifecycle,
  subscribeToWorkspaceLayoutSync,
  subscribeToWorkspaceNavigation,
} from "./lib/windowManager";
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
import { ForgeLoginPage } from "./pages/ForgeLoginPage";
import { PolicyPage } from "./pages/PolicyPage";
import { ReviewsPage } from "./pages/ReviewsPage";
import { JobDetailPage } from "./pages/JobDetailPage";
import { JobsPage } from "./pages/JobsPage";
import { LineagePage } from "./pages/LineagePage";
import { MemoryDetailPage } from "./pages/MemoryDetailPage";
import { MemoryPage } from "./pages/MemoryPage";
import { ModelsPage } from "./pages/ModelsPage";
import { OperatorAppsPage } from "./pages/OperatorAppsPage";
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
import { SystemPage } from "./pages/SystemPage";
import { WorkspaceLayoutsPage } from "./pages/WorkspaceLayoutsPage";
import { useDesktopWindowStore } from "./stores/desktopWindowStore";
import { useDesktopShellStore } from "./stores/desktopShellStore";
import { useWorkspaceLayoutStore } from "./stores/workspaceLayoutStore";
import { useWorkspaceStore } from "./stores/workspaceStore";
import { useUiStore } from "./stores/uiStore";

const FORGE_OPERATOR_LOGIN_SESSION_KEY = "forge.operator.login.unlocked";
const FORGE_BOOT_LOGIN_REQUIRED =
  import.meta.env.VITE_FORGE_BOOT_LOGIN === "true";
const FORGE_OPERATOR_DESKTOP_ROUTE = "/";
const FORGE_BOOT_SCREEN_MIN_MS = 1600;

function ForgeBootScreen() {
  return (
    <section className="forge-boot-screen" aria-label="FORGE loading">
      <div className="forge-boot-screen__brand">
        <img
          className="forge-boot-screen__mark"
          src="/brand/forge-start-button.png"
          alt=""
          draggable={false}
        />
        <div>
          <div className="forge-boot-screen__product">FORGE-OS</div>
          <div className="forge-boot-screen__subtitle">
            Loading operator shell
          </div>
        </div>
      </div>
      <div className="forge-boot-screen__status">
        <div className="forge-boot-screen__line" />
      </div>
    </section>
  );
}

function RoutedViews({
  onForgeLoginUnlock,
}: {
  onForgeLoginUnlock: () => void;
}) {
  const location = useLocation();
  return (
    <ForgeErrorBoundary resetKey={location.pathname + location.search}>
      <Routes>
        {/* Root renders the FORGE desktop (wallpaper). The shell decides what
            to show; route components only render when a tool is active. */}
        <Route path="/" element={null} />
        <Route
          path="/login"
          element={<ForgeLoginPage onUnlock={onForgeLoginUnlock} />}
        />
        <Route path="/start" element={<StartPage />} />
        <Route path="/dashboard" element={<DashboardPage />} />
        <Route path="/system" element={<SystemPage />} />
        <Route path="/chat" element={<ChatPage />} />
        <Route path="/workbench" element={<WorkbenchPage />} />
        <Route path="/canvas" element={<CanvasPage />} />
        <Route path="/command" element={<CommandPage />} />
        <Route path="/operator-apps" element={<OperatorAppsPage />} />
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
        <Route
          path="/execution-permissions"
          element={<ExecutionPermissionsPage />}
        />
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
  const refreshEnvironment = useWorkspaceLayoutStore(
    (s) => s.refreshEnvironment,
  );
  const syncCurrentRoute = useWorkspaceLayoutStore((s) => s.syncCurrentRoute);
  const hydrateShell = useDesktopShellStore((s) => s.hydrate);
  const layoutReady = useWorkspaceLayoutStore((s) => s.ready);
  const locationRef = useRef(`${location.pathname}${location.search}`);
  const desktopShownAfterUnlockRef = useRef(false);
  const layoutHydratedRef = useRef(false);
  const currentWindowLabel = useWorkspaceLayoutStore(
    (s) => s.currentWindowLabel,
  );
  const isPrimaryShellWindow = layoutReady && currentWindowLabel === "main";
  const isShellHostWindow =
    layoutReady && isShellHostWindowLabel(currentWindowLabel || "main");
  const contrastPreference = useUiStore((s) => s.contrastPreference);
  const effectsPreference = useUiStore((s) => s.effectsPreference);
  const [forgeLoginUnlocked, setForgeLoginUnlocked] = useState(() => {
    if (!FORGE_BOOT_LOGIN_REQUIRED) return true;
    return (
      window.sessionStorage.getItem(FORGE_OPERATOR_LOGIN_SESSION_KEY) === "true"
    );
  });
  const [forgeBootScreenReady, setForgeBootScreenReady] = useState(
    () => !FORGE_BOOT_LOGIN_REQUIRED,
  );
  const requiresForgeLogin =
    FORGE_BOOT_LOGIN_REQUIRED &&
    (isPrimaryShellWindow || !layoutReady || currentWindowLabel === "main");
  const canHydrateLayouts = !FORGE_BOOT_LOGIN_REQUIRED || forgeLoginUnlocked;
  const showingForgeBootSurface =
    FORGE_BOOT_LOGIN_REQUIRED &&
    (!forgeBootScreenReady || !forgeLoginUnlocked || !layoutReady);

  const handleForgeLoginUnlock = () => {
    window.sessionStorage.setItem(FORGE_OPERATOR_LOGIN_SESSION_KEY, "true");
    useDesktopWindowStore.getState().resetDesktopSession();
    desktopShownAfterUnlockRef.current = true;
    setForgeLoginUnlocked(true);
    navigate(FORGE_OPERATOR_DESKTOP_ROUTE, { replace: true });
  };

  const handleForgeLogout = () => {
    clearForgeApiTokenCache();
    window.sessionStorage.removeItem(FORGE_OPERATOR_LOGIN_SESSION_KEY);
    useDesktopWindowStore.getState().resetDesktopSession();
    desktopShownAfterUnlockRef.current = false;
    setForgeLoginUnlocked(false);
    setForgeBootScreenReady(true);
    navigate("/login", { replace: true });
  };

  useEffect(() => {
    if (!FORGE_BOOT_LOGIN_REQUIRED) return;
    const id = window.setTimeout(
      () => setForgeBootScreenReady(true),
      FORGE_BOOT_SCREEN_MIN_MS,
    );
    return () => window.clearTimeout(id);
  }, []);

  useEffect(() => {
    if (!FORGE_BOOT_LOGIN_REQUIRED) return;
    if (!forgeLoginUnlocked) {
      desktopShownAfterUnlockRef.current = false;
      return;
    }
    if (desktopShownAfterUnlockRef.current) return;
    useDesktopWindowStore.getState().resetDesktopSession();
    desktopShownAfterUnlockRef.current = true;
  }, [forgeLoginUnlocked]);

  useEffect(() => {
    locationRef.current = `${location.pathname}${location.search}`;
  }, [location.pathname, location.search]);

  useEffect(() => {
    if (!requiresForgeLogin) return;
    if (!forgeLoginUnlocked && location.pathname !== "/login") {
      navigate("/login", { replace: true });
      return;
    }
    if (forgeLoginUnlocked && location.pathname === "/login") {
      navigate(FORGE_OPERATOR_DESKTOP_ROUTE, { replace: true });
    }
  }, [forgeLoginUnlocked, location.pathname, navigate, requiresForgeLogin]);

  useEffect(() => {
    document.documentElement.dataset.contrast = contrastPreference;
    document.documentElement.dataset.effects = effectsPreference;
  }, [contrastPreference, effectsPreference]);

  useEffect(() => {
    document.documentElement.dataset.forgeBootSurface = showingForgeBootSurface
      ? "true"
      : "false";
    return () => {
      delete document.documentElement.dataset.forgeBootSurface;
    };
  }, [showingForgeBootSurface]);

  useEffect(() => {
    if (!isPrimaryShellWindow) return;
    void ping();
    const id = window.setInterval(() => void ping(), 8000);
    return () => window.clearInterval(id);
  }, [ping, isPrimaryShellWindow]);

  useEffect(() => {
    if (!canHydrateLayouts || layoutHydratedRef.current) return;
    layoutHydratedRef.current = true;
    const hydrationRoute =
      FORGE_BOOT_LOGIN_REQUIRED && location.pathname === "/login"
        ? FORGE_OPERATOR_DESKTOP_ROUTE
        : location.pathname + location.search;
    void hydrateLayouts(hydrationRoute);
  }, [canHydrateLayouts, hydrateLayouts, location.pathname, location.search]);

  useEffect(() => {
    hydrateShell(currentWindowLabel || "main");
  }, [currentWindowLabel, hydrateShell]);

  useEffect(() => {
    if (FORGE_BOOT_LOGIN_REQUIRED && !forgeLoginUnlocked) return;
    void syncCurrentRoute(location.pathname + location.search);
  }, [
    forgeLoginUnlocked,
    location.pathname,
    location.search,
    syncCurrentRoute,
  ]);

  useEffect(() => {
    if (!isPrimaryShellWindow) return;
    const id = window.setInterval(() => void refreshEnvironment(), 5000);
    return () => window.clearInterval(id);
  }, [refreshEnvironment, isPrimaryShellWindow]);

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
      if (!isPrimaryShellWindow) return;
      if (environmentRefreshTimer !== null) return;
      environmentRefreshTimer = setTimeout(() => {
        environmentRefreshTimer = null;
        void refreshEnvironment();
      }, 150);
    };

    (async () => {
      const handleNavigate = await subscribeToWorkspaceNavigation((route) => {
        if (route !== locationRef.current) {
          navigate(route);
        }
      });
      if (handleNavigate) disposers.push(handleNavigate);
      disposers.push(
        ...(await subscribeToCurrentWindowLifecycle(() =>
          scheduleEnvironmentRefresh(),
        )),
      );
      if (isPrimaryShellWindow) {
        const disposeLayoutSync = await subscribeToWorkspaceLayoutSync(
          (payload) => {
            const origin = payload?.origin?.trim() ?? "";
            // Ignore self-originated sync events to prevent refresh/emission loops.
            if (origin && origin === (currentWindowLabel || "main")) {
              return;
            }
            void refreshEnvironment();
          },
        );
        if (disposeLayoutSync) disposers.push(disposeLayoutSync);
      }
    })();

    return () => {
      disposers.forEach((dispose) => void dispose());
      if (environmentRefreshTimer !== null) {
        clearTimeout(environmentRefreshTimer);
      }
    };
  }, [navigate, refreshEnvironment, isPrimaryShellWindow, currentWindowLabel]);

  if (FORGE_BOOT_LOGIN_REQUIRED && !forgeBootScreenReady) {
    return <ForgeBootScreen />;
  }

  if (requiresForgeLogin && !forgeLoginUnlocked) {
    return (
      <div className="forge-tauri-surface forge-tauri-surface--boot">
        <ForgeLoginPage onUnlock={handleForgeLoginUnlock} />
      </div>
    );
  }

  if (FORGE_BOOT_LOGIN_REQUIRED && forgeLoginUnlocked && !layoutReady) {
    return <ForgeBootScreen />;
  }

  // Detached tool windows are compatibility hosts. Main and secondary monitor
  // desktop hosts render the full shell; tool surfaces stay in-shell.
  if (!isShellHostWindow) {
    return (
      <div className="forge-tauri-surface">
        <RoutedViews onForgeLoginUnlock={handleForgeLoginUnlock} />
      </div>
    );
  }
  return (
    <AppShell
      isMainWindow={isPrimaryShellWindow}
      hostLabel={currentWindowLabel || "main"}
      onForgeLogout={handleForgeLogout}
    >
      <RoutedViews onForgeLoginUnlock={handleForgeLoginUnlock} />
    </AppShell>
  );
}
