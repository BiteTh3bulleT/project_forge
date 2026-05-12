import { hostLabelForMonitorOrdinal } from "../lib/desktopHostLabels";

export { hostLabelForMonitorOrdinal } from "../lib/desktopHostLabels";

export type DesktopRect = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type DesktopHostBounds = {
  runtimeLabel: string;
  monitorId?: string | null;
  bounds: DesktopRect | null;
};

export type DesktopMonitorBounds = {
  id: string;
  ordinal: number;
  workArea: DesktopRect;
};

export type DesktopHost = {
  hostLabel: string;
  monitorId: string;
  monitorIndex: number;
  bounds: DesktopRect;
  role: "main" | "secondary";
  active: boolean;
};

export type DesktopWindowGeometry = {
  hostLabel?: string;
  width: number;
  height: number;
};

export type DesktopPlacement = {
  hostLabel: string;
  x: number;
  y: number;
};

function containsPoint(bounds: DesktopRect, x: number, y: number) {
  return (
    x >= bounds.x &&
    x < bounds.x + bounds.width &&
    y >= bounds.y &&
    y < bounds.y + bounds.height
  );
}

export function buildDesktopHosts(
  monitors: DesktopMonitorBounds[],
  runtimeWindows: DesktopHostBounds[] = [],
): DesktopHost[] {
  return monitors
    .filter((monitor) => monitor.workArea.width > 0 && monitor.workArea.height > 0)
    .sort((a, b) => a.ordinal - b.ordinal)
    .map((monitor) => {
      const hostLabel = hostLabelForMonitorOrdinal(monitor.ordinal);
      const runtimeWindow = runtimeWindows.find(
        (window_) =>
          window_.runtimeLabel === hostLabel || window_.monitorId === monitor.id,
      );
      return {
        hostLabel: runtimeWindow?.runtimeLabel ?? hostLabel,
        monitorId: monitor.id,
        monitorIndex: monitor.ordinal,
        bounds: runtimeWindow?.bounds ?? monitor.workArea,
        role: monitor.ordinal === 0 ? "main" : "secondary",
        active: Boolean(runtimeWindow),
      };
    });
}

export function getPrimaryDesktopHost(hosts: DesktopHost[]) {
  return (
    hosts.find((host) => host.role === "main") ??
    hosts.slice().sort((a, b) => a.monitorIndex - b.monitorIndex)[0] ??
    null
  );
}

export function getDesktopHostForMonitor(
  hosts: DesktopHost[],
  monitorId: string | null | undefined,
) {
  if (!monitorId) return null;
  return hosts.find((host) => host.monitorId === monitorId) ?? null;
}

export function getDesktopHostAtGlobalPoint(
  hosts: DesktopHost[],
  point: { x: number; y: number },
) {
  return (
    hosts.find((host) => containsPoint(host.bounds, point.x, point.y)) ?? null
  );
}

export function hostToGlobalPoint(
  host: Pick<DesktopHost, "bounds"> | DesktopRect,
  point: { x: number; y: number },
) {
  const bounds = "bounds" in host ? host.bounds : host;
  return { x: bounds.x + point.x, y: bounds.y + point.y };
}

export function globalToHostPoint(
  host: Pick<DesktopHost, "bounds"> | DesktopRect,
  point: { x: number; y: number },
) {
  const bounds = "bounds" in host ? host.bounds : host;
  return { x: point.x - bounds.x, y: point.y - bounds.y };
}

export function clampRectToHost(
  host: Pick<DesktopHost, "bounds"> | DesktopRect,
  rect: DesktopRect,
): DesktopRect {
  const bounds = "bounds" in host ? host.bounds : host;
  const width = Math.min(Math.max(1, rect.width), bounds.width);
  const height = Math.min(Math.max(1, rect.height), bounds.height);
  return {
    x: Math.min(Math.max(rect.x, 0), Math.max(0, bounds.width - width)),
    y: Math.min(Math.max(rect.y, 0), Math.max(0, bounds.height - height)),
    width,
    height,
  };
}

function intersectsRect(left: DesktopRect, right: DesktopRect) {
  return (
    left.x < right.x + right.width &&
    left.x + left.width > right.x &&
    left.y < right.y + right.height &&
    left.y + left.height > right.y
  );
}

export function resolveDesktopHostPlacement(
  runtimeWindows: DesktopHostBounds[],
  currentHostLabel: string,
  windowGeometry: DesktopWindowGeometry,
  nextLocalX: number,
  nextLocalY: number,
  canUseHostLabel: (label: string) => boolean,
): DesktopPlacement {
  const currentHost = runtimeWindows.find(
    (runtimeWindow) => runtimeWindow.runtimeLabel === currentHostLabel,
  );
  if (!currentHost?.bounds) {
    return { hostLabel: currentHostLabel, x: nextLocalX, y: nextLocalY };
  }

  const globalX = currentHost.bounds.x + nextLocalX;
  const globalY = currentHost.bounds.y + nextLocalY;
  const centerX = globalX + windowGeometry.width / 2;
  const centerY = globalY + windowGeometry.height / 2;
  const targetHost = runtimeWindows.find((runtimeWindow) => {
    if (!canUseHostLabel(runtimeWindow.runtimeLabel)) return false;
    return runtimeWindow.bounds
      ? containsPoint(runtimeWindow.bounds, centerX, centerY)
      : false;
  });

  if (!targetHost?.bounds || targetHost.runtimeLabel === currentHostLabel) {
    return { hostLabel: currentHostLabel, x: nextLocalX, y: nextLocalY };
  }

  return {
    hostLabel: targetHost.runtimeLabel,
    x: globalX - targetHost.bounds.x,
    y: globalY - targetHost.bounds.y,
  };
}

export function projectDesktopWindowToHost(
  runtimeWindows: DesktopHostBounds[],
  currentHostLabel: string,
  windowGeometry: DesktopWindowGeometry & { x: number; y: number },
): DesktopPlacement | null {
  const ownerHostLabel = windowGeometry.hostLabel || "main";
  if (ownerHostLabel === currentHostLabel) {
    const currentHost = runtimeWindows.find(
      (runtimeWindow) => runtimeWindow.runtimeLabel === currentHostLabel,
    );
    if (!currentHost?.bounds) {
      return {
        hostLabel: currentHostLabel,
        x: windowGeometry.x,
        y: windowGeometry.y,
      };
    }
  }

  const ownerHost = runtimeWindows.find(
    (runtimeWindow) => runtimeWindow.runtimeLabel === ownerHostLabel,
  );
  const currentHost = runtimeWindows.find(
    (runtimeWindow) => runtimeWindow.runtimeLabel === currentHostLabel,
  );
  if (!ownerHost?.bounds || !currentHost?.bounds) return null;

  const globalWindowBounds = {
    x: ownerHost.bounds.x + windowGeometry.x,
    y: ownerHost.bounds.y + windowGeometry.y,
    width: windowGeometry.width,
    height: windowGeometry.height,
  };
  if (!intersectsRect(globalWindowBounds, currentHost.bounds)) return null;

  return {
    hostLabel: currentHostLabel,
    x: globalWindowBounds.x - currentHost.bounds.x,
    y: globalWindowBounds.y - currentHost.bounds.y,
  };
}
