export type DesktopRect = {
  x: number;
  y: number;
  width: number;
  height: number;
};

export type DesktopHostBounds = {
  runtimeLabel: string;
  bounds: DesktopRect | null;
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
