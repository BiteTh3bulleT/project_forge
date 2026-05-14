import { assignableShellTools } from "../../layout/shellConfig";
import type { MonitorSnapshot } from "../../lib/desktop";
import { hostLabelForMonitorOrdinal } from "../../lib/desktopHostLabels";

import type {
  LayoutDoc,
  LayoutPreset,
  LayoutWindowRecord,
  MonitorDesignation,
  MonitorRoleMap,
  WindowRole,
} from "./types";

export function nowMs() {
  return Date.now();
}

export function parseMonitorRole(raw: unknown): string | null {
  if (typeof raw !== "string") return null;
  if (raw === "main") return "main";
  const secondary = /^secondary_(\d+)$/.exec(raw);
  if (!secondary) return null;
  const n = Number(secondary[1]);
  return Number.isFinite(n) && n > 0 ? `secondary_${n}` : null;
}

export function uid(prefix: string) {
  return `${prefix}-${Math.random().toString(36).slice(2, 10)}`;
}

export function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export function defaultRoutesForRole(role: WindowRole): string[] {
  if (role === "chat") return ["/chat", "/jobs"];
  if (role === "workbench") return ["/workbench", "/jobs"];
  if (role === "canvas") return ["/canvas", "/dossiers"];
  if (role === "dossier") return ["/dossiers", "/memory"];
  if (role === "ops") return ["/jobs", "/approvals", "/events"];
  if (role === "review") return ["/reviews", "/approvals"];
  if (role === "settings") return ["/settings", "/layouts"];
  return ["/chat"];
}

export function defaultWindowForLayout(input: {
  runtimeLabel: string;
  title: string;
  role: WindowRole;
  targetMonitorOrdinal: number;
  activeRoute?: string;
}): LayoutWindowRecord {
  const routes = defaultRoutesForRole(input.role);
  return {
    id: uid("window"),
    runtimeLabel: input.runtimeLabel,
    title: input.title,
    role: input.role,
    assignedRoutes: routes,
    activeRoute: input.activeRoute ?? routes[0] ?? "/chat",
    targetMonitorId: null,
    targetMonitorOrdinal: input.targetMonitorOrdinal,
    targetMonitorRole: null,
    bounds: null,
    fallbackReason: null,
  };
}

export function ensureLayoutMonitorHosts(
  layout: LayoutPreset,
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  if (monitors.length <= 1) return false;
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  let changed = false;

  for (const monitor of monitors) {
    if (monitor.ordinal === 0) continue;
    const monitorRole = catalog.roleByMonitorId[monitor.id] ?? null;
    const hasHost = layout.windows.some((windowRecord) => {
      if (windowRecord.targetMonitorId === monitor.id) return true;
      if (monitorRole && windowRecord.targetMonitorRole === monitorRole) {
        return true;
      }
      return (
        !windowRecord.targetMonitorId &&
        !windowRecord.targetMonitorRole &&
        windowRecord.targetMonitorOrdinal === monitor.ordinal
      );
    });
    if (hasHost) continue;

    layout.windows.push({
      ...defaultWindowForLayout({
        runtimeLabel: hostLabelForMonitorOrdinal(monitor.ordinal),
        title: `FORGE Monitor ${monitor.ordinal + 1}`,
        role: "mixed",
        targetMonitorOrdinal: monitor.ordinal,
        activeRoute: "/operator-apps",
      }),
      assignedRoutes: ["/operator-apps", "/chat"],
      targetMonitorId: monitor.id,
      targetMonitorRole: monitorRole,
    });
    changed = true;
  }

  if (changed) {
    layout.updatedAtMs = nowMs();
  }
  return changed;
}

export function normalizeMonitorDesignations(raw: unknown): MonitorDesignation {
  if (!raw || typeof raw !== "object") {
    return { mainMonitorId: null, customLabels: {} };
  }
  const input = raw as { mainMonitorId?: unknown; customLabels?: unknown };
  const rawMainMonitorId =
    typeof input.mainMonitorId === "string" ? input.mainMonitorId : null;
  const customLabels: Record<string, string> = {};
  if (typeof input.customLabels === "object" && input.customLabels !== null) {
    for (const [monitorId, value] of Object.entries(input.customLabels)) {
      if (typeof monitorId !== "string") continue;
      if (typeof value === "string" && value.trim()) {
        customLabels[monitorId] = value.trim();
      }
    }
  }
  return { mainMonitorId: rawMainMonitorId, customLabels };
}

export function canonicalMonitorDesignations(
  monitors: MonitorSnapshot[],
  incoming: MonitorDesignation,
) {
  const monitorIds = new Set(monitors.map((monitor) => monitor.id));
  const keptLabels: Record<string, string> = {};
  for (const [monitorId, label] of Object.entries(incoming.customLabels)) {
    if (monitorIds.has(monitorId) && label.trim()) {
      keptLabels[monitorId] = label.trim();
    }
  }
  const preferredMain = incoming.mainMonitorId;
  const mainMonitorId =
    preferredMain && monitorIds.has(preferredMain)
      ? preferredMain
      : (monitors[0]?.id ?? null);
  return { mainMonitorId, customLabels: keptLabels };
}

export function buildMonitorRoleCatalog(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const sortedMonitors = [...monitors].sort((a, b) => a.ordinal - b.ordinal);
  const canonical = canonicalMonitorDesignations(sortedMonitors, designations);
  const roleByMonitorId: MonitorRoleMap = {};
  const monitorByRole: Record<string, MonitorSnapshot> = {};

  let secondary = 1;
  for (const monitor of sortedMonitors) {
    const role =
      monitor.id === canonical.mainMonitorId
        ? "main"
        : `secondary_${secondary++}`;
    roleByMonitorId[monitor.id] = role;
    monitorByRole[role] = monitor;
  }
  return { roleByMonitorId, monitorByRole, canonical };
}

export function roleToMonitor(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
  role: string | null,
) {
  if (!role) return null;
  const normalizedRole = parseMonitorRole(role);
  if (!normalizedRole) return null;
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  return catalog.monitorByRole[normalizedRole] ?? null;
}

export function monitorStateFromDesignations(
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const catalog = buildMonitorRoleCatalog(monitors, designations);
  return {
    monitorDesignations: catalog.canonical,
    monitorRoleMap: catalog.roleByMonitorId,
  };
}

export function ensureDocMonitors(doc: LayoutDoc, monitors: MonitorSnapshot[]) {
  const state = monitorStateFromDesignations(monitors, doc.monitorDesignations);
  doc.monitorDesignations = state.monitorDesignations;
  return state;
}

export function deriveMonitorState(monitors: MonitorSnapshot[], doc: LayoutDoc) {
  const state = ensureDocMonitors(doc, monitors);
  return {
    monitorDesignations: state.monitorDesignations,
    monitorRoleMap: state.monitorRoleMap,
  };
}

export function resolveWindowPlacement(
  windowRecord: LayoutWindowRecord,
  monitors: MonitorSnapshot[],
  designations: MonitorDesignation,
) {
  const roleMatch = roleToMonitor(
    monitors,
    designations,
    windowRecord.targetMonitorRole,
  );
  const preferred = windowRecord.targetMonitorId
    ? monitors.find((monitor) => monitor.id === windowRecord.targetMonitorId)
    : null;
  const ordinal = monitors[windowRecord.targetMonitorOrdinal] ?? null;
  const chosen = roleMatch ?? preferred ?? ordinal ?? monitors[0] ?? null;
  const fallbackReason = !chosen
    ? "No displays available."
    : windowRecord.targetMonitorRole && !roleMatch
      ? `Monitor role ${windowRecord.targetMonitorRole} unavailable for ${windowRecord.title}; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
      : preferred
        ? null
        : windowRecord.targetMonitorId
          ? `Target display unavailable for ${windowRecord.title}; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
          : monitors[windowRecord.targetMonitorOrdinal] == null &&
              monitors.length > 0
            ? `Expected display ${windowRecord.targetMonitorOrdinal + 1} unavailable; placed on ${chosen.name ?? `display ${chosen.ordinal + 1}`}.`
            : null;

  return {
    monitor: chosen,
    bounds:
      windowRecord.bounds ??
      (chosen
        ? logicalBoundsForMonitor(chosen, windowRecord.targetMonitorOrdinal)
        : { x: 60, y: 60, width: 1200, height: 780 }),
    fallbackReason,
  };
}

export function seedLayouts(): LayoutPreset[] {
  const createdAtMs = nowMs();
  return [
    {
      id: "build",
      name: "Build",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Build",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/jobs", "/workbench"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-build-workbench",
            title: "FORGE Workbench",
            role: "workbench",
            targetMonitorOrdinal: 1,
            activeRoute: "/workbench",
          }),
          assignedRoutes: ["/workbench", "/jobs"],
        },
      ],
    },
    {
      id: "research",
      name: "Research",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Research",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/memory", "/dossiers"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-research-canvas",
            title: "FORGE Canvas",
            role: "canvas",
            targetMonitorOrdinal: 1,
            activeRoute: "/canvas",
          }),
          assignedRoutes: ["/canvas", "/dossiers"],
        },
      ],
    },
    {
      id: "ops",
      name: "Ops",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Ops",
            role: "ops",
            targetMonitorOrdinal: 0,
            activeRoute: "/jobs",
          }),
          assignedRoutes: ["/jobs", "/approvals", "/reviews", "/events"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-ops-review",
            title: "FORGE Review",
            role: "review",
            targetMonitorOrdinal: 1,
            activeRoute: "/reviews",
          }),
          assignedRoutes: ["/reviews", "/approvals"],
        },
      ],
    },
    {
      id: "deep-work",
      name: "Deep Work",
      createdAtMs,
      updatedAtMs: createdAtMs,
      lastActivatedAtMs: null,
      windows: [
        {
          ...defaultWindowForLayout({
            runtimeLabel: "main",
            title: "FORGE Deep Work",
            role: "chat",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
          assignedRoutes: ["/chat", "/canvas"],
        },
        {
          ...defaultWindowForLayout({
            runtimeLabel: "forge-deep-workbench",
            title: "FORGE Workbench",
            role: "workbench",
            targetMonitorOrdinal: 1,
            activeRoute: "/workbench",
          }),
          assignedRoutes: ["/workbench", "/dossiers"],
        },
      ],
    },
  ];
}

export function emptyDoc(): LayoutDoc {
  return {
    version: 2,
    monitorDesignations: { mainMonitorId: null, customLabels: {} },
    activeLayoutId: "build",
    selectedLayoutId: "build",
    layouts: seedLayouts(),
    runtimeWindows: [],
    lastKnownMonitors: [],
    lastMonitorSignature: "",
    fallbackNotice: null,
    lastRestoreAtMs: null,
  };
}

export function normalizeLayoutDoc(
  raw: LayoutDoc | null,
  monitors: MonitorSnapshot[] = [],
) {
  const doc = {
    ...emptyDoc(),
    ...(raw ?? {}),
    version: 2,
  } as LayoutDoc;

  if (!Array.isArray(doc.layouts) || doc.layouts.length === 0) {
    doc.layouts = seedLayouts();
  }
  doc.layouts = doc.layouts.map((layout) => {
    const source = layout as LayoutPreset & { windows?: unknown };
    return {
      id:
        typeof source.id === "string" && source.id ? source.id : uid("layout"),
      name:
        typeof source.name === "string" && source.name
          ? source.name
          : "Recovered Layout",
      createdAtMs:
        typeof source.createdAtMs === "number" &&
        Number.isFinite(source.createdAtMs)
          ? source.createdAtMs
          : nowMs(),
      updatedAtMs:
        typeof source.updatedAtMs === "number" &&
        Number.isFinite(source.updatedAtMs)
          ? source.updatedAtMs
          : nowMs(),
      lastActivatedAtMs:
        typeof source.lastActivatedAtMs === "number" &&
        Number.isFinite(source.lastActivatedAtMs)
          ? source.lastActivatedAtMs
          : null,
      windows:
        Array.isArray(source.windows) && source.windows.length > 0
          ? source.windows.map((windowRecord) => {
              const windowSource = windowRecord as LayoutWindowRecord & {
                fallbackReason?: unknown;
                assignedRoutes?: unknown;
                targetMonitorRole?: unknown;
              };
              return {
                id:
                  typeof windowSource.id === "string" && windowSource.id
                    ? windowSource.id
                    : uid("window"),
                runtimeLabel:
                  typeof windowSource.runtimeLabel === "string" &&
                  windowSource.runtimeLabel
                    ? windowSource.runtimeLabel
                    : uid("window"),
                title:
                  typeof windowSource.title === "string" && windowSource.title
                    ? windowSource.title
                    : "FORGE Window",
                role:
                  windowSource.role === "chat" ||
                  windowSource.role === "workbench" ||
                  windowSource.role === "canvas" ||
                  windowSource.role === "dossier" ||
                  windowSource.role === "ops" ||
                  windowSource.role === "review" ||
                  windowSource.role === "settings" ||
                  windowSource.role === "mixed"
                    ? windowSource.role
                    : "mixed",
                assignedRoutes: sanitizeRoutes(
                  Array.isArray(windowSource.assignedRoutes)
                    ? (windowSource.assignedRoutes.filter(
                        (route) => typeof route === "string",
                      ) as string[])
                    : defaultRoutesForRole(
                        windowSource.role === "chat" ||
                          windowSource.role === "workbench" ||
                          windowSource.role === "canvas" ||
                          windowSource.role === "dossier" ||
                          windowSource.role === "ops" ||
                          windowSource.role === "review" ||
                          windowSource.role === "settings" ||
                          windowSource.role === "mixed"
                          ? windowSource.role
                          : "mixed",
                      ),
                ),
                activeRoute:
                  typeof windowSource.activeRoute === "string" &&
                  windowSource.activeRoute
                    ? windowSource.activeRoute
                    : "/chat",
                targetMonitorId:
                  typeof windowSource.targetMonitorId === "string" &&
                  windowSource.targetMonitorId.length > 0
                    ? windowSource.targetMonitorId
                    : null,
                targetMonitorOrdinal:
                  typeof windowSource.targetMonitorOrdinal === "number" &&
                  Number.isFinite(windowSource.targetMonitorOrdinal)
                    ? windowSource.targetMonitorOrdinal
                    : 0,
                targetMonitorRole: parseMonitorRole(
                  windowSource.targetMonitorRole,
                ),
                bounds:
                  windowSource.bounds &&
                  typeof windowSource.bounds === "object" &&
                  Number.isFinite(windowSource.bounds.x) &&
                  Number.isFinite(windowSource.bounds.y) &&
                  Number.isFinite(windowSource.bounds.width) &&
                  Number.isFinite(windowSource.bounds.height)
                    ? windowSource.bounds
                    : null,
                fallbackReason:
                  typeof windowSource.fallbackReason === "string" &&
                  windowSource.fallbackReason.length > 0
                    ? windowSource.fallbackReason
                    : null,
              };
            })
          : [
              defaultWindowForLayout({
                runtimeLabel: "main",
                title: "FORGE Window",
                role: "mixed",
                targetMonitorOrdinal: 0,
                activeRoute: "/chat",
              }),
            ],
    };
  });

  doc.monitorDesignations = normalizeMonitorDesignations(
    doc.monitorDesignations,
  );
  ensureDocMonitors(doc, monitors);
  doc.runtimeWindows = Array.isArray(doc.runtimeWindows)
    ? doc.runtimeWindows
    : [];
  doc.lastKnownMonitors = Array.isArray(doc.lastKnownMonitors)
    ? doc.lastKnownMonitors
    : [];
  doc.lastMonitorSignature =
    typeof doc.lastMonitorSignature === "string"
      ? doc.lastMonitorSignature
      : "";
  doc.fallbackNotice =
    typeof doc.fallbackNotice === "string" ? doc.fallbackNotice : null;
  doc.lastRestoreAtMs =
    typeof doc.lastRestoreAtMs === "number" &&
    Number.isFinite(doc.lastRestoreAtMs)
      ? doc.lastRestoreAtMs
      : null;
  return ensureActiveLayout(doc);
}

export function findLayout(doc: LayoutDoc, layoutId: string | null) {
  if (!layoutId) return null;
  return doc.layouts.find((layout) => layout.id === layoutId) ?? null;
}

export function ensureActiveLayout(doc: LayoutDoc): LayoutDoc {
  const hasActiveLayout = findLayout(doc, doc.activeLayoutId) !== null;
  const fallbackLayoutId = doc.layouts[0]?.id ?? null;
  if (!hasActiveLayout && fallbackLayoutId) {
    doc.activeLayoutId = fallbackLayoutId;
  }
  if (!doc.selectedLayoutId || !findLayout(doc, doc.selectedLayoutId)) {
    doc.selectedLayoutId = doc.activeLayoutId ?? fallbackLayoutId;
  }
  return doc;
}

export function sanitizeRoutes(routes: string[]) {
  const allowed = new Set(assignableShellTools.map((tool) => tool.route));
  const sanitized = routes.filter((route) => allowed.has(route));
  return sanitized.length > 0 ? Array.from(new Set(sanitized)) : ["/chat"];
}

export function logicalBoundsForMonitor(
  monitor: MonitorSnapshot,
  index: number,
) {
  const workX = Math.round(monitor.workArea.x / monitor.scaleFactor);
  const workY = Math.round(monitor.workArea.y / monitor.scaleFactor);
  const workWidth = Math.round(monitor.workArea.width / monitor.scaleFactor);
  const workHeight = Math.round(monitor.workArea.height / monitor.scaleFactor);
  const width = Math.max(920, Math.round(workWidth * 0.78));
  const height = Math.max(640, Math.round(workHeight * 0.84));
  const x =
    workX + Math.max(20, Math.round((workWidth - width) * 0.5)) + index * 18;
  const y =
    workY + Math.max(20, Math.round((workHeight - height) * 0.08)) + index * 18;
  return {
    x,
    y,
    width: Math.min(width, workWidth),
    height: Math.min(height, workHeight),
  };
}
