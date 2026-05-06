import type { DashboardSummary } from "@forge/shared";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { CommandBar } from "../components/CommandBar";
import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";
import { useWorkspaceStore } from "../stores/workspaceStore";

import { getShellTool } from "./shellConfig";

type AppShellProps = {
  children: ReactNode;
  isMainWindow: boolean;
};

type AttentionLevel = "none" | "low" | "medium" | "high";
type FloatingKind =
  | "inspector"
  | "snapshot"
  | "graph"
  | "memory"
  | "model"
  | "diagnostics"
  | "surfaces";

type FloatingWindow = {
  id: number;
  kind: FloatingKind;
  title: string;
  pinned: boolean;
  x: number;
  y: number;
};

type NavItem = {
  label: string;
  route: string;
  short: string;
  mode?: "cognitive" | "metrics";
};

const navGroups: Array<{ label: string; items: NavItem[] }> = [
  {
    label: "Kernel",
    items: [
      {
        label: "Command Deck",
        route: "/dashboard",
        short: "KD",
        mode: "cognitive",
      },
      {
        label: "Context Compile",
        route: "/project-context",
        short: "CC",
        mode: "cognitive",
      },
      { label: "Jobs", route: "/jobs", short: "JB", mode: "cognitive" },
    ],
  },
  {
    label: "Cognitive FS",
    items: [
      { label: "Episodes", route: "/memory", short: "ME", mode: "cognitive" },
      { label: "Dossiers", route: "/dossiers", short: "DS", mode: "cognitive" },
      { label: "Lineage", route: "/lineage", short: "LN", mode: "cognitive" },
    ],
  },
  {
    label: "Runtime",
    items: [
      { label: "Models", route: "/models", short: "MD", mode: "cognitive" },
      {
        label: "Registry",
        route: "/models?view=registry",
        short: "RG",
        mode: "cognitive",
      },
      { label: "Adapters", route: "/adapters", short: "AD", mode: "metrics" },
    ],
  },
  {
    label: "Execution",
    items: [
      { label: "Approvals", route: "/approvals", short: "AP", mode: "metrics" },
      { label: "Gateway", route: "/gateway", short: "GW", mode: "metrics" },
      {
        label: "Action Lanes",
        route: "/action-lanes",
        short: "AL",
        mode: "metrics",
      },
      {
        label: "Permissions",
        route: "/execution-permissions",
        short: "PX",
        mode: "metrics",
      },
    ],
  },
  {
    label: "Operator",
    items: [
      { label: "Chat", route: "/chat", short: "CH", mode: "cognitive" },
      { label: "Canvas", route: "/canvas", short: "CV", mode: "cognitive" },
      {
        label: "Artifacts",
        route: "/workbench",
        short: "AR",
        mode: "cognitive",
      },
      { label: "Layouts", route: "/layouts", short: "LY", mode: "cognitive" },
    ],
  },
  {
    label: "Evidence",
    items: [
      { label: "Metrics", route: "/dashboard", short: "MX", mode: "metrics" },
      {
        label: "Inspectors",
        route: "/inspectors",
        short: "IN",
        mode: "metrics",
      },
      { label: "Audit", route: "/audit", short: "AU", mode: "metrics" },
      { label: "Events", route: "/events", short: "EV", mode: "metrics" },
    ],
  },
];

const surfaceDirectory: NavItem[] = [
  { label: "Start", route: "/start", short: "ST" },
  { label: "Command", route: "/command", short: "CM" },
  { label: "Policy", route: "/policy", short: "PL" },
  { label: "Strategies", route: "/strategies", short: "ST" },
  { label: "Automation", route: "/automation", short: "AM" },
  { label: "Autonomy", route: "/autonomy", short: "AY" },
  { label: "Reviews", route: "/reviews", short: "RV" },
  { label: "Insights", route: "/insights", short: "IS" },
  { label: "Retrieval Runs", route: "/retrieval-runs", short: "RR" },
  { label: "Evaluations", route: "/evaluations", short: "EV" },
  { label: "Sources", route: "/sources", short: "SC" },
  { label: "Backup", route: "/backup", short: "BK" },
  { label: "Release", route: "/release", short: "RL" },
  { label: "Settings", route: "/settings", short: "SE" },
];

const consoleDockItem: NavItem = {
  label: "Console",
  route: "/dashboard",
  short: "FG",
  mode: "cognitive",
};

const dockItems: NavItem[] = [
  { label: "Chat", route: "/chat", short: "CH", mode: "cognitive" },
  { label: "Jobs", route: "/jobs", short: "JB", mode: "cognitive" },
  { label: "Memory", route: "/memory", short: "ME", mode: "cognitive" },
  { label: "Models", route: "/models", short: "MD", mode: "cognitive" },
  { label: "Gateway", route: "/gateway", short: "GW", mode: "metrics" },
  { label: "Settings", route: "/settings", short: "SE", mode: "metrics" },
];

function corePill(core: "online" | "offline" | "unknown") {
  if (core === "online") return "forge-chip--ok";
  if (core === "offline") return "forge-chip--warn";
  return "forge-chip--muted";
}

function getWorkspaceLabel(workspaceDir: string | null | undefined) {
  if (!workspaceDir) return "Workspace unavailable";
  const parts = workspaceDir.split(/[\\/]/).filter(Boolean);
  return parts.at(-1) ?? workspaceDir;
}

function attentionLevel(
  summary: DashboardSummary | null,
  core: "online" | "offline" | "unknown",
): AttentionLevel {
  if (core === "offline") return "high";
  const approvals = summary?.approvalsPending ?? 0;
  const reviews = summary?.reviewsPending ?? 0;
  const failures = Array.isArray(summary?.recentFailures)
    ? summary.recentFailures.length
    : 0;
  const active = Array.isArray(summary?.activeJobs)
    ? summary.activeJobs.length
    : 0;
  const score = approvals * 2 + reviews + failures * 2 + Math.min(active, 3);
  if (score <= 0) return "none";
  if (score <= 2) return "low";
  if (score <= 5) return "medium";
  return "high";
}

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

const SHELL_RAIL_COLLAPSED_KEY = "forge.shellRailCollapsed.v1";

function readStoredShellRailCollapsed() {
  if (typeof window === "undefined") return false;
  return window.localStorage.getItem(SHELL_RAIL_COLLAPSED_KEY) === "true";
}

export function AppShell(props: AppShellProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const currentTool = useMemo(() => getShellTool(pathname), [pathname]);

  const core = useWorkspaceStore((s) => s.core);
  const meta = useWorkspaceStore((s) => s.meta);
  const lastErr = useWorkspaceStore((s) => s.lastCoreError);
  const statusLine = useUiStore((s) => s.statusLine);
  const uiMode = useUiStore((s) => s.uiMode);
  const setUiMode = useUiStore((s) => s.setUiMode);
  const fallbackNotice = useWorkspaceLayoutStore((s) => s.fallbackNotice);
  const clearFallbackNotice = useWorkspaceLayoutStore(
    (s) => s.clearFallbackNotice,
  );

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [shellErr, setShellErr] = useState<string | null>(null);
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(
    () => new Set(["Kernel"]),
  );
  const [windows, setWindows] = useState<FloatingWindow[]>([]);
  const [dragging, setDragging] = useState<{
    id: number;
    dx: number;
    dy: number;
  } | null>(null);
  const [windowSeq, setWindowSeq] = useState(1);
  const [railCollapsed, setRailCollapsed] = useState(() =>
    readStoredShellRailCollapsed(),
  );
  const [mobileRailOpen, setMobileRailOpen] = useState(false);
  const [now, setNow] = useState(() => new Date());
  const isMainWindow = props.isMainWindow;

  useEffect(() => {
    if (!isMainWindow) return;
    let cancelled = false;
    async function loadShellState() {
      try {
        const nextSummary = await api.dashboard.summary();
        if (cancelled) return;
        setSummary(nextSummary);
        setShellErr(null);
      } catch (e) {
        if (cancelled) return;
        setShellErr(e instanceof Error ? e.message : String(e));
      }
    }
    void loadShellState();
    const id = window.setInterval(() => void loadShellState(), 5000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    try {
      window.localStorage.setItem(
        SHELL_RAIL_COLLAPSED_KEY,
        String(railCollapsed),
      );
    } catch {
      // Cosmetic shell preference; ignore storage failures.
    }
  }, [railCollapsed]);

  useEffect(() => {
    if (!isMainWindow) return;
    const id = window.setInterval(() => setNow(new Date()), 30_000);
    return () => window.clearInterval(id);
  }, [isMainWindow]);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      const target = event.target as HTMLElement | null;
      const tag = target?.tagName?.toLowerCase();
      if (tag === "input" || tag === "textarea" || tag === "select") return;
      if (event.ctrlKey && event.key.toLowerCase() === "m") {
        event.preventDefault();
        switchMode(uiMode === "cognitive" ? "metrics" : "cognitive");
      }
      if (event.ctrlKey && event.key.toLowerCase() === "b") {
        event.preventDefault();
        setRailCollapsed((value) => !value);
      }
      if (event.key === "Escape") {
        setMobileRailOpen(false);
        setWindows((items) => items.filter((item) => item.pinned));
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [uiMode]);

  useEffect(() => {
    if (!dragging) return;
    const onMove = (event: PointerEvent) => {
      setWindows((items) =>
        items.map((item) =>
          item.id === dragging.id
            ? {
                ...item,
                x: Math.max(
                  96,
                  Math.min(
                    window.innerWidth - 380,
                    event.clientX - dragging.dx,
                  ),
                ),
                y: Math.max(
                  72,
                  Math.min(
                    window.innerHeight - 180,
                    event.clientY - dragging.dy,
                  ),
                ),
              }
            : item,
        ),
      );
    };
    const onUp = () => setDragging(null);
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp, { once: true });
    return () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
  }, [dragging]);

  function switchMode(nextMode: "cognitive" | "metrics") {
    setUiMode(nextMode);
    setMobileRailOpen(false);
    navigate(nextMode === "metrics" ? "/dashboard" : "/chat");
  }

  function isNavItemActive(item: NavItem) {
    const [itemPath, itemSearch = ""] = item.route.split("?");
    const searchMatches = !itemSearch || location.search === `?${itemSearch}`;
    const pathMatches =
      pathname === itemPath || pathname.startsWith(`${itemPath}/`);
    if (!pathMatches || !searchMatches) return false;
    if (item.route === "/dashboard" && item.mode) return uiMode === item.mode;
    return true;
  }

  function navigateToItem(item: NavItem) {
    if (item.mode) setUiMode(item.mode);
    setMobileRailOpen(false);
    navigate(item.route);
  }

  function toggleGroup(label: string) {
    setExpandedGroups((current) => {
      const next = new Set(current);
      if (next.has(label)) next.delete(label);
      else next.add(label);
      return next;
    });
  }

  function openWindow(kind: FloatingKind, title: string) {
    setWindows((items) => {
      const existing = items.find((item) => item.kind === kind);
      if (existing) {
        return [...items.filter((item) => item.id !== existing.id), existing];
      }
      const next: FloatingWindow = {
        id: windowSeq,
        kind,
        title,
        pinned: false,
        x: 128 + (items.length % 3) * 34,
        y: 112 + (items.length % 3) * 28,
      };
      setWindowSeq((value) => value + 1);
      return [...items, next];
    });
  }

  function closeWindow(id: number) {
    setWindows((items) => items.filter((item) => item.id !== id));
  }

  function togglePinned(id: number) {
    setWindows((items) =>
      items.map((item) =>
        item.id === id ? { ...item, pinned: !item.pinned } : item,
      ),
    );
  }

  const approvalsPending = summary?.approvalsPending ?? 0;
  const reviewsPending = summary?.reviewsPending ?? 0;
  const recentFailures = Array.isArray(summary?.recentFailures)
    ? summary.recentFailures
    : [];
  const attentionCount =
    approvalsPending + reviewsPending + recentFailures.length;
  const level = attentionLevel(summary, core);
  const workspaceLabel = getWorkspaceLabel(meta?.workspaceDir);
  const runtimeState =
    core === "offline" ? "offline" : shellErr ? "degraded" : "online";
  const pinnedWindows = windows.filter((item) => item.pinned);
  const floatingWindows = windows.filter((item) => !item.pinned);
  const isChatRoute = pathname === "/chat";
  const showCollapsedRail = railCollapsed && !mobileRailOpen;

  return (
    <div
      className={cx(
        "forge-shell-frame flex h-full min-h-0 flex-col text-forge-ash",
        mobileRailOpen && "forge-shell-frame--rail-open",
      )}
    >
      {isMainWindow ? (
        <header className="forge-status-strip">
          <div className="flex min-w-0 items-center gap-3">
            <button
              type="button"
              onClick={() => setMobileRailOpen((value) => !value)}
              className="forge-mobile-rail-toggle"
              aria-label={
                mobileRailOpen ? "Close navigation" : "Open navigation"
              }
              aria-expanded={mobileRailOpen}
            >
              <span className="forge-mobile-rail-toggle__mark" aria-hidden>
                <span />
                <span />
                <span />
              </span>
            </button>
            <div className="forge-shell-lockup">
              <div className="forge-shell-brand h-9 w-9 text-[11px]">FG</div>
              <div className="hidden min-w-0 sm:block">
                <div className="forge-shell-product">FORGE</div>
                <div className="truncate text-[10px] uppercase tracking-[0.12em] text-forge-mist/55">
                  {workspaceLabel} / {currentTool.label}
                </div>
              </div>
            </div>
          </div>
          <div className="flex min-w-0 flex-1 items-center justify-end gap-2 overflow-x-auto">
            <span
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                corePill(core),
              )}
            >
              Core:{" "}
              {core === "online"
                ? "online"
                : core === "offline"
                  ? "offline"
                  : "checking"}
            </span>
            <span
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                runtimeState === "degraded"
                  ? "forge-chip--warn"
                  : "forge-chip--muted",
              )}
            >
              Runtime: {runtimeState}
            </span>
            <button
              type="button"
              onClick={() => openWindow("diagnostics", "Attention")}
              className={cx(
                "forge-chip px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]",
                level === "none"
                  ? "forge-chip--muted"
                  : level === "high"
                    ? "forge-chip--warn"
                    : "forge-chip--info",
              )}
            >
              Queue:{" "}
              {level === "none" ? "clear" : `attention ${attentionCount}`}
            </button>
            <button
              type="button"
              onClick={() =>
                switchMode(uiMode === "cognitive" ? "metrics" : "cognitive")
              }
              className="forge-chip forge-chip--accent px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]"
              title="Ctrl+M"
            >
              Mode: {uiMode}
            </button>
          </div>
        </header>
      ) : null}

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <button
          type="button"
          className="forge-mobile-rail-backdrop"
          aria-label="Close navigation"
          onClick={() => setMobileRailOpen(false)}
        />
        <aside
          className={cx(
            "forge-category-rail",
            railCollapsed && "forge-category-rail--collapsed",
            mobileRailOpen && "forge-category-rail--mobile-open",
          )}
        >
          <div className="forge-category-rail__top">
            {!showCollapsedRail ? (
              <div className="forge-rail-brand-card">
                <div>
                  <div className="forge-rail-brand-card__label">Operator</div>
                  <div className="forge-rail-brand-card__title">FORGE OS</div>
                </div>
                <span className="forge-rail-brand-card__light" />
              </div>
            ) : null}
            <button
              type="button"
              onClick={() => setRailCollapsed((value) => !value)}
              className="forge-rail-toggle"
              aria-label={
                railCollapsed ? "Open left toolbar" : "Collapse left toolbar"
              }
              title={
                railCollapsed
                  ? "Open toolbar (Ctrl+B)"
                  : "Collapse toolbar (Ctrl+B)"
              }
            >
              <span aria-hidden>{railCollapsed ? ">" : "<"}</span>
              <span className={railCollapsed ? "sr-only" : ""}>
                {railCollapsed ? "Open" : "Collapse"}
              </span>
            </button>
            {!showCollapsedRail ? <CommandBar compact /> : null}
            {!showCollapsedRail ? (
              <div className="grid grid-cols-2 gap-2">
                <button
                  type="button"
                  onClick={() => openWindow("diagnostics", "Attention")}
                  className="forge-context-row text-left text-[10px]"
                >
                  <span>Queue</span>
                  <span className="font-semibold text-forge-ash">
                    {level === "none" ? "clear" : attentionCount}
                  </span>
                </button>
                <button
                  type="button"
                  onClick={() => openWindow("inspector", "Surface Inspector")}
                  className="forge-context-row text-left text-[10px]"
                >
                  <span>Surface</span>
                  <span className="truncate font-semibold text-forge-ash">
                    {currentTool.shortLabel}
                  </span>
                </button>
              </div>
            ) : null}
          </div>
          <nav className="forge-nav-scroll flex min-h-0 flex-1 flex-col gap-1.5 overflow-y-auto px-2 pb-3">
            {navGroups.map((group) => {
              const expanded = expandedGroups.has(group.label);
              const groupActive = group.items.some((item) =>
                isNavItemActive(item),
              );
              return (
                <section key={group.label} className="forge-nav-group">
                  <button
                    type="button"
                    onClick={() => toggleGroup(group.label)}
                    className={cx(
                      "forge-nav-group__trigger",
                      groupActive && "forge-nav-group__trigger--active",
                    )}
                    aria-expanded={expanded}
                    title={showCollapsedRail ? group.label : undefined}
                  >
                    <span>
                      {showCollapsedRail
                        ? group.label.slice(0, 2).toUpperCase()
                        : group.label}
                    </span>
                    {!showCollapsedRail ? (
                      <span>{expanded ? "-" : "+"}</span>
                    ) : null}
                  </button>
                  {expanded ? (
                    <div className="mt-1 space-y-1">
                      {group.items.map((item) => {
                        const active = isNavItemActive(item);
                        return (
                          <button
                            key={`${group.label}-${item.label}`}
                            type="button"
                            onClick={() => navigateToItem(item)}
                            className={cx(
                              "forge-nav-item",
                              active && "forge-nav-item--active",
                            )}
                            aria-current={active ? "page" : undefined}
                            title={showCollapsedRail ? item.label : undefined}
                          >
                            <span className="forge-nav-item__short">
                              {item.short}
                            </span>
                            <span
                              className={cx(
                                "forge-nav-item__label min-w-0 truncate",
                                showCollapsedRail && "sr-only",
                              )}
                            >
                              {item.label}
                            </span>
                          </button>
                        );
                      })}
                    </div>
                  ) : null}
                </section>
              );
            })}
          </nav>
          <div className="border-t border-forge-platinum/10 p-2">
            <button
              type="button"
              onClick={() => {
                setMobileRailOpen(false);
                openWindow("surfaces", "Surface Directory");
              }}
              className="forge-nav-item w-full"
              title={showCollapsedRail ? "More" : undefined}
            >
              <span className="forge-nav-item__short">··</span>
              <span
                className={cx(
                  "forge-nav-item__label",
                  showCollapsedRail && "sr-only",
                )}
              >
                More
              </span>
            </button>
          </div>
        </aside>

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          {fallbackNotice ? (
            <div className="forge-chat-toolbar text-xs text-forge-mist">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span>{fallbackNotice}</span>
                <button
                  type="button"
                  onClick={() => clearFallbackNotice()}
                  className="forge-inline-link"
                >
                  Dismiss
                </button>
              </div>
            </div>
          ) : null}

          <div className="forge-main-field flex min-h-0 min-w-0 flex-1 overflow-hidden">
            <main
              className={cx(
                "forge-desktop-surface",
                isChatRoute && "forge-desktop-surface--flush",
              )}
            >
              <div
                className={cx(
                  "forge-window-frame forge-window-frame--focus",
                  isChatRoute && "forge-window-frame--chat",
                )}
              >
                {!isChatRoute ? (
                  <div className="forge-focus-head">
                    <div className="min-w-0">
                      <div className="text-[10px] font-semibold uppercase tracking-[0.16em] text-forge-mist/55">
                        {uiMode === "metrics"
                          ? "System Metrics"
                          : "Cognitive State"}
                      </div>
                      <div className="mt-1 truncate text-sm font-semibold text-forge-ash">
                        {currentTool.label}
                      </div>
                    </div>
                    <div className="flex flex-wrap items-center justify-end gap-2">
                      {level !== "none" ? (
                        <button
                          type="button"
                          onClick={() => openWindow("diagnostics", "Attention")}
                          className="forge-chip forge-chip--warn px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]"
                        >
                          Attention {attentionCount}
                        </button>
                      ) : null}
                      <button
                        type="button"
                        onClick={() => openWindow("inspector", "Inspector")}
                        className="forge-chip forge-chip--muted px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]"
                      >
                        Inspector
                      </button>
                      <button
                        type="button"
                        onClick={() =>
                          openWindow("snapshot", "Restore Snapshot")
                        }
                        className="forge-chip forge-chip--muted px-2.5 py-1 text-[10px] font-semibold uppercase tracking-[0.12em]"
                      >
                        Snapshot
                      </button>
                    </div>
                  </div>
                ) : null}
                <div
                  className={
                    isChatRoute
                      ? "min-h-0 flex flex-1 overflow-hidden p-0"
                      : "min-h-0 flex-1 overflow-auto px-4 py-4 sm:px-5 lg:px-6"
                  }
                >
                  {props.children}
                </div>
              </div>
            </main>

            {pinnedWindows.length > 0 ? (
              <aside className="forge-pinned-panel">
                {pinnedWindows.map((item) => (
                  <FloatingContent
                    key={item.id}
                    item={item}
                    summary={summary}
                    pathname={pathname}
                    statusLine={statusLine}
                    lastErr={lastErr}
                    onClose={() => closeWindow(item.id)}
                    onPin={() => togglePinned(item.id)}
                  />
                ))}
              </aside>
            ) : null}
          </div>
        </div>
      </div>

      {isMainWindow ? (
        <footer className="forge-os-dockbar">
          <button
            type="button"
            onClick={() => navigateToItem(consoleDockItem)}
            className="forge-os-dockbar__brand"
            aria-label="Open FORGE console"
          >
            <span className="forge-os-dockbar__sigil">FG</span>
            <span>FORGE</span>
          </button>
          <div className="forge-os-dockbar__items">
            {dockItems.map((item) => {
              const active = isNavItemActive(item);
              return (
                <button
                  key={item.route}
                  type="button"
                  onClick={() => navigateToItem(item)}
                  className={cx(
                    "forge-os-dockbar__item",
                    active && "forge-os-dockbar__item--active",
                  )}
                  aria-current={active ? "page" : undefined}
                  title={item.label}
                >
                  <span>{item.short}</span>
                </button>
              );
            })}
          </div>
          <button
            type="button"
            onClick={() => openWindow("surfaces", "Surface Directory")}
            className="forge-os-dockbar__search"
          >
            <span className="text-forge-mist/45">Search FORGE</span>
            <span className="font-mono text-forge-mist/55">Ctrl K</span>
          </button>
          <div className="forge-os-dockbar__system">
            <span>
              {now.toLocaleDateString([], {
                day: "numeric",
                month: "short",
                year: "numeric",
              })}
            </span>
            <span>
              {now.toLocaleTimeString([], {
                hour: "numeric",
                minute: "2-digit",
              })}
            </span>
          </div>
        </footer>
      ) : null}

      {floatingWindows.map((item) => (
        <div
          key={item.id}
          className="forge-floating-window"
          style={{ left: item.x, top: item.y }}
        >
          <FloatingContent
            item={item}
            summary={summary}
            pathname={pathname}
            statusLine={statusLine}
            lastErr={lastErr}
            onClose={() => closeWindow(item.id)}
            onPin={() => togglePinned(item.id)}
            onDragStart={(clientX, clientY) =>
              setDragging({
                id: item.id,
                dx: clientX - item.x,
                dy: clientY - item.y,
              })
            }
          />
        </div>
      ))}
    </div>
  );
}

function FloatingContent(props: {
  item: FloatingWindow;
  summary: DashboardSummary | null;
  pathname: string;
  statusLine: string;
  lastErr: string | null;
  onClose: () => void;
  onPin: () => void;
  onDragStart?: (clientX: number, clientY: number) => void;
}) {
  const navigate = useNavigate();
  const activeJobs = Array.isArray(props.summary?.activeJobs)
    ? props.summary.activeJobs
    : [];
  const recentFailures = Array.isArray(props.summary?.recentFailures)
    ? props.summary.recentFailures
    : [];
  const recentImports = Array.isArray(props.summary?.recentImports)
    ? props.summary.recentImports
    : [];

  return (
    <section
      className={
        props.item.pinned
          ? "forge-floating-card forge-floating-card--pinned"
          : "forge-floating-card"
      }
    >
      <div
        className="forge-floating-card__head"
        onPointerDown={(event) => {
          if (props.item.pinned || !props.onDragStart) return;
          props.onDragStart(event.clientX, event.clientY);
        }}
      >
        <div>
          <div className="text-[10px] font-semibold uppercase tracking-[0.16em] text-forge-mist/55">
            {props.item.kind}
          </div>
          <div className="mt-0.5 text-sm font-semibold text-forge-ash">
            {props.item.title}
          </div>
        </div>
        <div className="flex gap-1">
          <button
            type="button"
            className="forge-window-btn"
            onClick={props.onPin}
          >
            {props.item.pinned ? "Float" : "Pin"}
          </button>
          <button
            type="button"
            className="forge-window-btn"
            onClick={props.onClose}
          >
            Close
          </button>
        </div>
      </div>
      <div className="space-y-3 p-3 text-sm text-forge-mist">
        {props.item.kind === "surfaces" ? (
          <div className="grid gap-2 sm:grid-cols-2">
            {surfaceDirectory.map((item) => (
              <button
                key={item.route}
                type="button"
                onClick={() => navigate(item.route)}
                className="forge-surface-link"
              >
                <span className="forge-nav-item__short">{item.short}</span>
                <span className="truncate">{item.label}</span>
              </button>
            ))}
          </div>
        ) : null}

        {props.item.kind === "diagnostics" ? (
          <>
            <WindowMetric
              label="Approvals pending"
              value={String(props.summary?.approvalsPending ?? 0)}
              onClick={() => navigate("/approvals")}
            />
            <WindowMetric
              label="Reviews pending"
              value={String(props.summary?.reviewsPending ?? 0)}
              onClick={() => navigate("/reviews")}
            />
            <WindowMetric
              label="Recent failures"
              value={String(recentFailures.length)}
              onClick={() => navigate("/events")}
            />
            {props.lastErr ? (
              <div className="rounded-xl border border-forge-ember/30 bg-forge-ember/10 p-3 text-xs text-forge-emberSoft">
                {props.lastErr}
              </div>
            ) : null}
          </>
        ) : null}

        {props.item.kind === "inspector" ? (
          <>
            <div className="rounded-xl bg-black/20 p-3 text-xs leading-5">
              <div className="font-semibold text-forge-ash">
                {getShellTool(props.pathname).label}
              </div>
              <div className="mt-1">{getInspectorSummary(props.pathname)}</div>
              <div className="mt-2 text-forge-mist/70">
                {props.statusLine || "No operator note recorded."}
              </div>
            </div>
            <WindowMetric
              label="Active jobs"
              value={String(activeJobs.length)}
              onClick={() => navigate("/jobs")}
            />
            <WindowMetric
              label="Recent imports"
              value={String(recentImports.length)}
              onClick={() => navigate("/workbench")}
            />
          </>
        ) : null}

        {props.item.kind === "snapshot" ? (
          <>
            <div className="rounded-xl bg-black/20 p-3 text-xs leading-5">
              Restore scoring is header-first: the default view keeps full graph
              expansion out of the focus path until requested.
            </div>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/inspectors")}
            >
              Open snapshot inspector
            </button>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/project-context")}
            >
              Open context compiler
            </button>
          </>
        ) : null}

        {props.item.kind === "graph" || props.item.kind === "memory" ? (
          <>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/memory")}
            >
              Recent episodes
            </button>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/insights")}
            >
              Important insights
            </button>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/lineage")}
            >
              Active loops
            </button>
          </>
        ) : null}

        {props.item.kind === "model" ? (
          <>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/models")}
            >
              Runtime status
            </button>
            <button
              type="button"
              className="forge-window-action"
              onClick={() => navigate("/models?view=registry")}
            >
              Model registry
            </button>
          </>
        ) : null}
      </div>
    </section>
  );
}

function WindowMetric(props: {
  label: string;
  value: string;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      className="forge-context-row w-full"
    >
      <span>{props.label}</span>
      <span className="font-semibold text-forge-ash">{props.value}</span>
    </button>
  );
}

function getInspectorSummary(pathname: string) {
  if (pathname === "/chat")
    return "Conversation thread, packet launch, and assistant status tied to persisted thread state.";
  if (pathname === "/workbench")
    return "Artifact selection and job-linked file inspection using recorded artifact metadata.";
  if (pathname === "/canvas")
    return "Board note layout and working memory pinned to persisted board coordinates.";
  if (pathname === "/dossiers")
    return "Project profile, recent job lineage, and routing/approval preferences.";
  if (pathname === "/jobs" || pathname.startsWith("/jobs/"))
    return "Execution truth from job projections, approval gates, events, and artifacts.";
  if (pathname === "/reviews")
    return "Explicit operator review records and import reconciliation status.";
  if (pathname === "/approvals")
    return "Pending risk gates and recorded decisions. No silent escalation.";
  if (pathname === "/autonomy")
    return "Dream-state telemetry, autonomy intents, policy decisions, budgets, and charter boundaries.";
  if (pathname === "/settings")
    return "Local model, retrieval, and workspace configuration persisted by the core.";
  return "Focused FORGE surface. Details are available on demand instead of occupying permanent screen space.";
}
