import {
  useMemo,
  useRef,
  type CSSProperties,
  type MouseEvent as ReactMouseEvent,
  type PointerEvent as ReactPointerEvent,
} from "react";

import {
  iconAssetUrl,
  type ForgeHostPowerAction,
  type LinuxWindowAction,
  type LinuxWindowSnapshot,
  type OperatorApp,
} from "../lib/desktop";
import type { DesktopWindow } from "../stores/desktopWindowStore";
import type { DesktopPlacement } from "./desktopGeometry";
import {
  allShellTools,
  type ShellToolDefinition,
  type ShellToolId,
} from "./shellConfig";
import { getToolComponent } from "./toolRegistry";

const MIN_WINDOW_W = 360;
const MIN_WINDOW_H = 280;

export const FALLBACK_OPERATOR_APPS: OperatorApp[] = [
  {
    id: "terminal",
    label: "Foot",
    description: "A Wayland native terminal emulator",
    executable: "foot",
    category: "Workspace",
    iconName: "foot",
    iconPath: "/run/current-system/sw/share/icons/hicolor/48x48/apps/foot.png",
    desktopFile: "/run/current-system/sw/share/applications/foot.desktop",
    native: true,
  },
  {
    id: "files",
    label: "PCMan File Manager",
    description: "Browse the file system and manage files",
    executable: "pcmanfm",
    category: "Workspace",
    iconName: "system-file-manager",
    iconPath: null,
    desktopFile: "/run/current-system/sw/share/applications/pcmanfm.desktop",
    native: true,
  },
  {
    id: "editor",
    label: "Editor",
    description: "Open the fixed operator editor wrapper",
    executable: "foot",
    category: "Workspace",
    iconName: "accessories-text-editor",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "browser",
    label: "Firefox",
    description: "Open local docs, web consoles, and model tooling",
    executable: "firefox",
    category: "Internet",
    iconName: "firefox",
    iconPath:
      "/run/current-system/sw/share/icons/hicolor/128x128/apps/firefox.png",
    desktopFile: "/run/current-system/sw/share/applications/firefox.desktop",
    native: true,
  },
  {
    id: "ollama-status",
    label: "Ollama Status",
    description: "Show local Ollama status without model mutation",
    executable: "foot",
    category: "AI Runtime",
    iconName: "utilities-terminal",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "system-monitor",
    label: "System Monitor",
    description: "Open the fixed btop/htop wrapper",
    executable: "foot",
    category: "System",
    iconName: "utilities-system-monitor",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "lazygit",
    label: "Git UI",
    description: "Open lazygit in the FORGE workspace",
    executable: "foot",
    category: "Developer",
    iconName: "git",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "forge-status",
    label: "FORGE Status",
    description: "Show local forge-core health",
    executable: "foot",
    category: "FORGE",
    iconName: "utilities-terminal",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
];

export type DockTile = {
  kind: "tile";
  key: string;
  tool: ShellToolDefinition;
  window: DesktopWindow | null;
  pinned: boolean;
};

export type DockContextMenu = {
  x: number;
  y: number;
  tool: ShellToolDefinition;
  window: DesktopWindow | null;
  pinned: boolean;
  open: boolean;
};

export type NativeWindowContextMenu = {
  x: number;
  y: number;
  window: LinuxWindowSnapshot;
};

function cx(...parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export function DesktopWallpaper() {
  return (
    <div className="forge-os-wallpaper" aria-hidden>
      <picture className="forge-os-wallpaper__art">
        <source
          media="(orientation: portrait)"
          srcSet="/brand/forge-vertical.png"
        />
        <img src="/brand/forge-horizontal.png" alt="" draggable={false} />
      </picture>
      <div className="forge-os-wallpaper__gradient" />
      <div className="forge-os-wallpaper__grid" />
      <div className="forge-os-wallpaper__glow" />
    </div>
  );
}

export function ForgeHero(props: { lastErr: string | null }) {
  if (!props.lastErr) return null;
  return (
    <div className="forge-os-hero">
      <div className="forge-os-hero__panel forge-os-hero__panel--error">
        <div className="forge-os-hero__error">{props.lastErr}</div>
      </div>
    </div>
  );
}

export function FloatingWindow(props: {
  window: DesktopWindow;
  placement: DesktopPlacement;
  interactive: boolean;
  focused: boolean;
  onFocus: () => void;
  onMinimize: () => void;
  onClose: () => void;
  onToggleMaximize: () => void;
  onMove: (x: number, y: number) => void;
  onResize: (width: number, height: number) => void;
}) {
  const tool = useMemo(
    () => allShellTools.find((t) => t.id === props.window.toolId) ?? null,
    [props.window.toolId],
  );
  const Component = tool ? getToolComponent(tool.id) : null;
  const dragRef = useRef<{
    pointerStartX: number;
    pointerStartY: number;
    windowStartX: number;
    windowStartY: number;
  } | null>(null);
  const resizeRef = useRef<{
    startX: number;
    startY: number;
    startW: number;
    startH: number;
  } | null>(null);

  function startDrag(event: ReactPointerEvent<HTMLDivElement>) {
    if (!props.interactive) return;
    if (props.window.maximized) return;
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    event.preventDefault();
    dragRef.current = {
      pointerStartX: event.clientX,
      pointerStartY: event.clientY,
      windowStartX: props.window.x,
      windowStartY: props.window.y,
    };
    props.onFocus();
    const onMove = (e: PointerEvent) => {
      if (!dragRef.current) return;
      props.onMove(
        dragRef.current.windowStartX +
          (e.clientX - dragRef.current.pointerStartX),
        dragRef.current.windowStartY +
          (e.clientY - dragRef.current.pointerStartY),
      );
    };
    const onUp = () => {
      dragRef.current = null;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  function startResize(event: ReactPointerEvent<HTMLDivElement>) {
    if (!props.interactive) return;
    if (props.window.maximized) return;
    if (event.button !== 0) return;
    event.currentTarget.setPointerCapture?.(event.pointerId);
    event.preventDefault();
    event.stopPropagation();
    resizeRef.current = {
      startX: event.clientX,
      startY: event.clientY,
      startW: props.window.width,
      startH: props.window.height,
    };
    props.onFocus();
    const onMove = (e: PointerEvent) => {
      if (!resizeRef.current) return;
      const dx = e.clientX - resizeRef.current.startX;
      const dy = e.clientY - resizeRef.current.startY;
      props.onResize(
        Math.max(MIN_WINDOW_W, resizeRef.current.startW + dx),
        Math.max(MIN_WINDOW_H, resizeRef.current.startH + dy),
      );
    };
    const onUp = () => {
      resizeRef.current = null;
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
  }

  if (props.window.minimized) return null;

  const style: CSSProperties = props.window.maximized
    ? {
        left: 0,
        top: 0,
        width: "100%",
        height: "100%",
        zIndex: props.window.z,
      }
    : {
        left: props.placement.x,
        top: props.placement.y,
        width: props.window.width,
        height: props.window.height,
        zIndex: props.window.z,
        pointerEvents: props.interactive ? undefined : "none",
      };

  return (
    <section
      className={cx(
        "forge-os-window",
        props.focused && "forge-os-window--focused",
        props.window.maximized && "forge-os-window--maximized",
      )}
      style={style}
      aria-hidden={props.interactive ? undefined : true}
      onPointerDown={() => {
        if (props.interactive && !props.focused) props.onFocus();
      }}
    >
      <div
        className="forge-os-window__chrome"
        onPointerDown={startDrag}
        onDoubleClick={props.onToggleMaximize}
      >
        <div className="forge-os-window__title">
          <span className="forge-os-window__sigil">
            {tool?.shortLabel ?? "??"}
          </span>
          <div className="min-w-0">
            <div className="forge-os-window__name">
              {tool?.label ?? "Unknown surface"}
            </div>
            <div className="forge-os-window__sub">
              {tool?.description ?? ""}
            </div>
          </div>
        </div>
        <div className="forge-os-window__buttons">
          <button
            type="button"
            className="forge-os-window__btn"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onMinimize();
            }}
            aria-label="Minimize"
            title="Minimize"
          >
            –
          </button>
          <button
            type="button"
            className="forge-os-window__btn"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onToggleMaximize();
            }}
            aria-label={props.window.maximized ? "Restore" : "Maximize"}
            title={props.window.maximized ? "Restore" : "Maximize"}
          >
            {props.window.maximized ? "❐" : "▢"}
          </button>
          <button
            type="button"
            className="forge-os-window__btn forge-os-window__btn--close"
            onPointerDown={(e) => e.stopPropagation()}
            onClick={(e) => {
              e.stopPropagation();
              props.onClose();
            }}
            aria-label="Close"
            title="Close"
          >
            ×
          </button>
        </div>
      </div>
      <div className="forge-os-window__body">
        <div className="forge-os-window__content">
          {Component ? (
            <Component />
          ) : (
            <UnsupportedToolNotice toolId={tool?.id} />
          )}
        </div>
      </div>
      {!props.window.maximized ? (
        <div
          className="forge-os-window__resize"
          onPointerDown={startResize}
          aria-hidden
        />
      ) : null}
    </section>
  );
}

function UnsupportedToolNotice(props: { toolId: string | undefined }) {
  return (
    <div className="forge-os-window__placeholder">
      <div className="forge-os-window__placeholder-title">
        Surface unavailable
      </div>
      <div className="forge-os-window__placeholder-body">
        No window component is registered for {props.toolId ?? "this surface"}.
      </div>
    </div>
  );
}

export function StartMenu(props: {
  query: string;
  onQueryChange: (value: string) => void;
  onClose: () => void;
  onLaunch: (tool: ShellToolDefinition) => void;
  onContextMenu: (
    event: ReactMouseEvent<HTMLButtonElement>,
    tool: ShellToolDefinition,
  ) => void;
  activeTool: ShellToolDefinition | null;
  workspaceLabel: string;
  uiMode: "cognitive" | "metrics";
  pinnedIds: ShellToolId[];
  onPowerAction: (action: "logout" | ForgeHostPowerAction) => void;
  operatorApps: OperatorApp[];
  operatorAppStatus: string | null;
  onLaunchOperatorApp: (app: OperatorApp) => void;
}) {
  const query = props.query.trim().toLowerCase();
  const filteredTools = useMemo(() => {
    return allShellTools.filter((tool) => {
      const haystack =
        `${tool.label} ${tool.shortLabel} ${tool.description}`.toLowerCase();
      return !query || haystack.includes(query);
    });
  }, [query]);
  const filteredApps = useMemo(() => {
    return props.operatorApps.filter((app) => {
      const haystack =
        `${app.label} ${app.description} ${app.executable} ${app.category} ${app.iconName ?? ""}`.toLowerCase();
      return !query || haystack.includes(query);
    });
  }, [props.operatorApps, query]);

  const pinnedTools = useMemo(() => {
    const map = new Map<ShellToolId, ShellToolDefinition>(
      allShellTools.map((t) => [t.id, t] as const),
    );
    return props.pinnedIds
      .map((id) => map.get(id))
      .filter((t): t is ShellToolDefinition => Boolean(t));
  }, [props.pinnedIds]);
  const toolGroups = useMemo(
    () => groupToolsByCategory(filteredTools),
    [filteredTools],
  );
  const appGroups = useMemo(
    () => groupAppsByCategory(filteredApps),
    [filteredApps],
  );

  return (
    <>
      <button
        type="button"
        className="forge-os-startmenu__backdrop"
        aria-label="Close Start menu"
        onClick={props.onClose}
      />
      <section
        className="forge-os-startmenu"
        role="dialog"
        aria-label="FORGE Start"
      >
        <header className="forge-os-startmenu__head">
          <div className="forge-os-startmenu__operator">
            <div className="forge-os-startmenu__avatar">
              <img
                src="/brand/forge-start-button.png"
                alt=""
                draggable={false}
              />
            </div>
            <div className="min-w-0">
              <div className="forge-os-startmenu__operator-name">
                Forge Operator
              </div>
              <div className="forge-os-startmenu__operator-meta">
                {props.workspaceLabel} · Mode {props.uiMode}
              </div>
            </div>
          </div>
          <div className="forge-os-startmenu__search">
            <input
              type="text"
              value={props.query}
              onChange={(event) => props.onQueryChange(event.target.value)}
              placeholder="Search FORGE surfaces"
              className="forge-os-startmenu__input"
              autoFocus
            />
          </div>
        </header>

        {!query && pinnedTools.length > 0 ? (
          <div className="forge-os-startmenu__section forge-os-startmenu__section--pinned">
            <div className="forge-os-startmenu__section-label">
              Quick Launch
            </div>
            <div className="forge-os-startmenu__grid">
              {pinnedTools.map((tool) => (
                <button
                  key={tool.id}
                  type="button"
                  onClick={() => props.onLaunch(tool)}
                  onContextMenu={(event) => props.onContextMenu(event, tool)}
                  className={cx(
                    "forge-os-startmenu__tile",
                    props.activeTool?.id === tool.id &&
                      "forge-os-startmenu__tile--active",
                  )}
                  title={`${tool.description} (right-click to unpin)`}
                >
                  <span className="forge-os-startmenu__tile-short">
                    {tool.shortLabel}
                  </span>
                  <span className="forge-os-startmenu__tile-label">
                    {tool.label}
                  </span>
                </button>
              ))}
            </div>
          </div>
        ) : null}

        <div className="forge-os-startmenu__body">
          <div className="forge-os-startmenu__panel forge-os-startmenu__panel--native">
            <div className="forge-os-startmenu__panel-head">
              <div className="forge-os-startmenu__section-label">
                Native Apps
              </div>
              <div className="forge-os-startmenu__panel-meta">
                {filteredApps.length}{" "}
                {filteredApps.length === 1 ? "app" : "apps"}
              </div>
            </div>
            <div className="forge-os-startmenu__list forge-os-startmenu__list--native">
              {appGroups.length === 0 ? (
                <div className="forge-os-startmenu__empty">
                  {query
                    ? `No native apps match "${props.query}".`
                    : "No native operator apps are available."}
                </div>
              ) : (
                appGroups.map((group) => (
                  <div
                    key={group.category}
                    className="forge-os-startmenu__group"
                  >
                    <div className="forge-os-startmenu__group-head">
                      <div className="forge-os-startmenu__group-label">
                        {group.category}
                      </div>
                      <div className="forge-os-startmenu__group-count">
                        {group.items.length}{" "}
                        {group.items.length === 1 ? "app" : "apps"}
                      </div>
                    </div>
                    {group.items.map((app) => (
                      <button
                        key={app.id}
                        type="button"
                        onClick={() => props.onLaunchOperatorApp(app)}
                        className="forge-os-startmenu__row"
                      >
                        <OperatorAppIcon
                          app={app}
                          className="forge-os-startmenu__app-icon"
                        />
                        <span className="min-w-0 flex-1">
                          <span className="forge-os-startmenu__row-label">
                            {app.label}
                          </span>
                          <span className="forge-os-startmenu__row-desc">
                            {app.description}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                ))
              )}
            </div>
            {props.operatorAppStatus ? (
              <div className="forge-os-startmenu__status">
                {props.operatorAppStatus}
              </div>
            ) : null}
          </div>

          <div className="forge-os-startmenu__panel forge-os-startmenu__panel--surfaces">
            <div className="forge-os-startmenu__panel-head">
              <div className="forge-os-startmenu__section-label">
                {query ? "FORGE Results" : "FORGE Surfaces"}
              </div>
              <div className="forge-os-startmenu__panel-meta">
                {filteredTools.length}{" "}
                {filteredTools.length === 1 ? "surface" : "surfaces"}
              </div>
            </div>
            <div className="forge-os-startmenu__list">
              {toolGroups.length === 0 ? (
                <div className="forge-os-startmenu__empty">
                  No FORGE surfaces match "{props.query}".
                </div>
              ) : (
                toolGroups.map((group) => (
                  <div
                    key={group.category}
                    className="forge-os-startmenu__group"
                  >
                    <div className="forge-os-startmenu__group-head">
                      <div className="forge-os-startmenu__group-label">
                        {group.category}
                      </div>
                      <div className="forge-os-startmenu__group-count">
                        {group.items.length}{" "}
                        {group.items.length === 1 ? "item" : "items"}
                      </div>
                    </div>
                    {group.items.map((tool) => (
                      <button
                        key={tool.id}
                        type="button"
                        onClick={() => props.onLaunch(tool)}
                        onContextMenu={(event) =>
                          props.onContextMenu(event, tool)
                        }
                        className={cx(
                          "forge-os-startmenu__row",
                          props.activeTool?.id === tool.id &&
                            "forge-os-startmenu__row--active",
                        )}
                      >
                        <span className="forge-os-startmenu__row-short">
                          {tool.shortLabel}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="forge-os-startmenu__row-label">
                            {tool.label}
                          </span>
                          <span className="forge-os-startmenu__row-desc">
                            {tool.description}
                          </span>
                        </span>
                      </button>
                    ))}
                  </div>
                ))
              )}
            </div>
          </div>
        </div>
        <footer className="forge-os-startmenu__power">
          <button
            type="button"
            className="forge-os-startmenu__power-btn"
            onClick={() => props.onPowerAction("logout")}
          >
            <span className="forge-os-startmenu__power-icon">LO</span>
            <span>Logout</span>
          </button>
          <button
            type="button"
            className="forge-os-startmenu__power-btn"
            onClick={() => props.onPowerAction("reboot")}
          >
            <span className="forge-os-startmenu__power-icon">RB</span>
            <span>Reboot</span>
          </button>
          <button
            type="button"
            className="forge-os-startmenu__power-btn forge-os-startmenu__power-btn--danger"
            onClick={() => props.onPowerAction("shutdown")}
          >
            <span className="forge-os-startmenu__power-icon">SD</span>
            <span>Shutdown</span>
          </button>
        </footer>
      </section>
    </>
  );
}

function groupToolsByCategory(tools: ShellToolDefinition[]) {
  const groups = new Map<string, ShellToolDefinition[]>();
  for (const tool of tools) {
    const category = shellToolCategory(tool.id);
    groups.set(category, [...(groups.get(category) ?? []), tool]);
  }
  return Array.from(groups, ([category, items]) => ({ category, items }));
}

function groupAppsByCategory(apps: OperatorApp[]) {
  const groups = new Map<string, OperatorApp[]>();
  for (const app of apps) {
    const category = app.category?.trim() || "Tools";
    groups.set(category, [...(groups.get(category) ?? []), app]);
  }
  return Array.from(groups, ([category, items]) => ({ category, items }));
}

function shellToolCategory(id: ShellToolId) {
  if (
    [
      "chat",
      "start",
      "dashboard",
      "system",
      "settings",
      "operator-apps",
    ].includes(id)
  ) {
    return "Core";
  }
  if (
    ["workbench", "canvas", "dossiers", "jobs", "reviews", "layouts"].includes(
      id,
    )
  ) {
    return "Workspace";
  }
  if (
    [
      "memory",
      "project-context",
      "sources",
      "insights",
      "lineage",
      "retrieval-runs",
      "evaluations",
    ].includes(id)
  ) {
    return "Knowledge";
  }
  if (
    [
      "models",
      "gateway",
      "adapters",
      "command",
      "action-lanes",
      "automation",
    ].includes(id)
  ) {
    return "Runtime";
  }
  if (
    [
      "approvals",
      "autonomy",
      "audit",
      "policy",
      "strategies",
      "execution-permissions",
      "inspectors",
    ].includes(id)
  ) {
    return "Governance";
  }
  if (["logs", "backup", "release"].includes(id)) {
    return "Operations";
  }
  return "Tools";
}

export function OperatorAppIcon(props: {
  app: OperatorApp;
  className?: string;
}) {
  const iconPath = props.app.iconPath?.trim();
  if (iconPath) {
    return (
      <img
        src={iconAssetUrl(iconPath)}
        alt={`${props.app.label} icon`}
        className={props.className}
        draggable={false}
      />
    );
  }
  const short = props.app.label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
  return (
    <span className={cx("forge-os-startmenu__app-icon", props.className)}>
      {short || "AP"}
    </span>
  );
}

export function LinuxWindowIcon(props: {
  window: LinuxWindowSnapshot;
  className?: string;
}) {
  const iconPath = props.window.iconPath?.trim();
  if (iconPath) {
    return (
      <img
        src={iconAssetUrl(iconPath)}
        alt={`${props.window.title} icon`}
        className={props.className}
        draggable={false}
      />
    );
  }
  const label = props.window.title || props.window.appId;
  const short = label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
  return (
    <span className={cx("forge-os-startmenu__app-icon", props.className)}>
      {short || "LN"}
    </span>
  );
}

export function DockContextMenuView(props: {
  menu: DockContextMenu;
  onClose: () => void;
  onAction: (action: "open" | "close" | "pin" | "unpin") => void;
}) {
  const left = Math.min(
    props.menu.x,
    typeof window !== "undefined" ? window.innerWidth - 220 : props.menu.x,
  );
  const top = Math.max(
    8,
    typeof window !== "undefined"
      ? Math.min(props.menu.y - 8, window.innerHeight - 220)
      : props.menu.y - 8,
  );
  return (
    <div
      role="menu"
      className="forge-os-context-menu"
      style={{ left, top }}
      onMouseDown={(event) => event.stopPropagation()}
    >
      <div className="forge-os-context-menu__title">
        {props.menu.tool.label}
      </div>
      <button
        type="button"
        role="menuitem"
        className="forge-os-context-menu__item"
        onClick={() => props.onAction("open")}
      >
        {props.menu.open ? "Focus window" : "Open"}
      </button>
      {props.menu.open ? (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("close")}
        >
          Close window
        </button>
      ) : null}
      {props.menu.pinned ? (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("unpin")}
        >
          Unpin from taskbar
        </button>
      ) : (
        <button
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction("pin")}
        >
          Pin to taskbar
        </button>
      )}
      <button
        type="button"
        role="menuitem"
        className="forge-os-context-menu__item forge-os-context-menu__item--muted"
        onClick={props.onClose}
      >
        Cancel
      </button>
    </div>
  );
}

export function NativeWindowContextMenuView(props: {
  menu: NativeWindowContextMenu;
  onClose: () => void;
  onAction: (action: LinuxWindowAction) => void;
}) {
  const left = Math.min(
    props.menu.x,
    typeof window !== "undefined" ? window.innerWidth - 240 : props.menu.x,
  );
  const top = Math.max(
    8,
    typeof window !== "undefined"
      ? Math.min(props.menu.y - 8, window.innerHeight - 260)
      : props.menu.y - 8,
  );
  return (
    <div
      role="menu"
      className="forge-os-context-menu"
      style={{ left, top }}
      onMouseDown={(event) => event.stopPropagation()}
    >
      <div className="forge-os-context-menu__title">
        {props.menu.window.title}
      </div>
      {(
        [
          ["focus", "Focus window"],
          ["minimize", "Minimize window"],
          ["maximize", "Maximize window"],
          ["fullscreen", "Fullscreen window"],
          ["close", "Close window"],
        ] satisfies Array<[LinuxWindowAction, string]>
      ).map(([action, label]) => (
        <button
          key={action}
          type="button"
          role="menuitem"
          className="forge-os-context-menu__item"
          onClick={() => props.onAction(action)}
        >
          {label}
        </button>
      ))}
      <button
        type="button"
        role="menuitem"
        className="forge-os-context-menu__item forge-os-context-menu__item--muted"
        onClick={props.onClose}
      >
        Cancel
      </button>
    </div>
  );
}
