import {
  type ComponentType,
  lazy,
  Suspense,
  useEffect,
  useRef,
  useState,
} from "react";
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
import { configuredRenderProfile } from "./lib/renderProfile";
import {
  subscribeToCurrentWindowLifecycle,
  subscribeToWorkspaceLayoutSync,
  subscribeToWorkspaceNavigation,
} from "./lib/windowManager";
import { ForgeLoginPage } from "./pages/ForgeLoginPage";
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

type PageModule<K extends string> = Record<K, ComponentType>;

function lazyPage<K extends string>(
  load: () => Promise<PageModule<K>>,
  exportName: K,
) {
  return lazy(async () => ({ default: (await load())[exportName] }));
}

const ActionLanesPage = lazyPage(
  () => import("./pages/ActionLanesPage"),
  "ActionLanesPage",
);
const AdaptersPage = lazyPage(
  () => import("./pages/AdaptersPage"),
  "AdaptersPage",
);
const AuditPage = lazyPage(() => import("./pages/AuditPage"), "AuditPage");
const AutonomyPage = lazyPage(
  () => import("./pages/AutonomyPage"),
  "AutonomyPage",
);
const BackupPage = lazyPage(() => import("./pages/BackupPage"), "BackupPage");
const CanvasPage = lazyPage(() => import("./pages/CanvasPage"), "CanvasPage");
const ChatPage = lazyPage(() => import("./pages/ChatPage"), "ChatPage");
const StartPage = lazyPage(() => import("./pages/StartPage"), "StartPage");
const ApprovalsPage = lazyPage(
  () => import("./pages/ApprovalsPage"),
  "ApprovalsPage",
);
const AutomationPage = lazyPage(
  () => import("./pages/AutomationPage"),
  "AutomationPage",
);
const CommandPage = lazyPage(() => import("./pages/CommandPage"), "CommandPage");
const DashboardPage = lazyPage(
  () => import("./pages/DashboardPage"),
  "DashboardPage",
);
const ExecutionPermissionsPage = lazyPage(
  () => import("./pages/ExecutionPermissionsPage"),
  "ExecutionPermissionsPage",
);
const EventsPage = lazyPage(() => import("./pages/EventsPage"), "EventsPage");
const EvaluationsPage = lazyPage(
  () => import("./pages/EvaluationsPage"),
  "EvaluationsPage",
);
const PolicyPage = lazyPage(() => import("./pages/PolicyPage"), "PolicyPage");
const ReviewsPage = lazyPage(() => import("./pages/ReviewsPage"), "ReviewsPage");
const JobDetailPage = lazyPage(
  () => import("./pages/JobDetailPage"),
  "JobDetailPage",
);
const JobsPage = lazyPage(() => import("./pages/JobsPage"), "JobsPage");
const LineagePage = lazyPage(() => import("./pages/LineagePage"), "LineagePage");
const MemoryDetailPage = lazyPage(
  () => import("./pages/MemoryDetailPage"),
  "MemoryDetailPage",
);
const MemoryPage = lazyPage(() => import("./pages/MemoryPage"), "MemoryPage");
const ModelsPage = lazyPage(() => import("./pages/ModelsPage"), "ModelsPage");
const OperatorAppsPage = lazyPage(
  () => import("./pages/OperatorAppsPage"),
  "OperatorAppsPage",
);
const DossiersPage = lazyPage(
  () => import("./pages/DossiersPage"),
  "DossiersPage",
);
const ProjectContextPage = lazyPage(
  () => import("./pages/ProjectContextPage"),
  "ProjectContextPage",
);
const ReleasePage = lazyPage(() => import("./pages/ReleasePage"), "ReleasePage");
const RetrievalRunsPage = lazyPage(
  () => import("./pages/RetrievalRunsPage"),
  "RetrievalRunsPage",
);
const SettingsPage = lazyPage(
  () => import("./pages/SettingsPage"),
  "SettingsPage",
);
const ToolGatewayPage = lazyPage(
  () => import("./pages/ToolGatewayPage"),
  "ToolGatewayPage",
);
const WorkbenchPage = lazyPage(
  () => import("./pages/WorkbenchPage"),
  "WorkbenchPage",
);
const InsightsPage = lazyPage(
  () => import("./pages/InsightsPage"),
  "InsightsPage",
);
const InspectorsPage = lazyPage(
  () => import("./pages/InspectorsPage"),
  "InspectorsPage",
);
const SourcesPage = lazyPage(() => import("./pages/SourcesPage"), "SourcesPage");
const StrategiesPage = lazyPage(
  () => import("./pages/StrategiesPage"),
  "StrategiesPage",
);
const SystemPage = lazyPage(() => import("./pages/SystemPage"), "SystemPage");
const WorkspaceLayoutsPage = lazyPage(
  () => import("./pages/WorkspaceLayoutsPage"),
  "WorkspaceLayoutsPage",
);

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

function RouteLoadingFallback() {
  return (
    <div
      className="forge-route-loading"
      role="status"
      aria-label="Loading view"
    >
      <span />
    </div>
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
      <Suspense fallback={<RouteLoadingFallback />}>
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
          <Route path="/logs" element={<Navigate to="/events" replace />} />
          <Route path="/autonomy" element={<AutonomyPage />} />
          <Route path="/settings" element={<SettingsPage />} />
          <Route path="/layouts" element={<WorkspaceLayoutsPage />} />
          <Route path="*" element={<Navigate to="/chat" replace />} />
        </Routes>
      </Suspense>
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
  const renderProfile = configuredRenderProfile();
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
    document.documentElement.dataset.renderProfile = renderProfile;
  }, [contrastPreference, effectsPreference, renderProfile]);

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
