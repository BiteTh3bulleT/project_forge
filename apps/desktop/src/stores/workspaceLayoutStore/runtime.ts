import {
  DETACHED_TAURI_TOOL_WINDOWS,
  emitWorkspaceSync,
  isForgeManagedWindowLabel,
  isTauriDesktop,
  listAvailableMonitors,
  monitorSignature,
  spanCurrentWindowAcrossMonitors,
  virtualDesktopBounds,
  type MonitorSnapshot,
} from "../../lib/desktop";
import {
  bringCurrentWindowFront,
  bringWindowToFrontByLabel,
  closeWindowByLabel,
  createShellWindow,
  getCurrentWindowLabel,
  getCurrentWindowSnapshot,
  getWindowByLabel,
  listRuntimeWindows,
  navigateWindowByLabel,
  setCurrentWindowBounds,
  setCurrentWindowTitle,
  setWindowBoundsByLabel,
  setWindowTitleByLabel,
} from "../../lib/windowManager";

import { EXTENDED_DESKTOP_SINGLE_SHELL } from "./constants";
import {
  defaultWindowForLayout,
  ensureDocMonitors,
  ensureLayoutMonitorHosts,
  findLayout,
  nowMs,
  resolveWindowPlacement,
} from "./model";
import { loadDoc, persistDoc } from "./persistence";
import type {
  LayoutDoc,
  LayoutWindowRecord,
  RuntimeWindowRecord,
} from "./types";

function isInvalidWindowHandleError(error: unknown) {
  const message = error instanceof Error ? error.message : String(error ?? "");
  return (
    message.includes("Invalid window handle") ||
    message.includes("is not found") ||
    message.includes("not found")
  );
}

async function reclaimWindowLabel(runtimeLabel: string) {
  await closeWindowByLabel(runtimeLabel).catch(() => undefined);
}

async function syncOrRecreateWindow(
  layoutWindow: LayoutWindowRecord,
  bounds: { x: number; y: number; width: number; height: number },
  options: { route: string; setFocus?: boolean },
) {
  const targetWindow = await getWindowByLabel(layoutWindow.runtimeLabel);
  if (!targetWindow) {
    return createShellWindow({
      label: layoutWindow.runtimeLabel,
      route: options.route,
      title: layoutWindow.title,
      bounds,
    });
  }
  try {
    await setWindowTitleByLabel(layoutWindow.runtimeLabel, layoutWindow.title);
    await setWindowBoundsByLabel(layoutWindow.runtimeLabel, bounds);
    await navigateWindow(layoutWindow.runtimeLabel, options.route);
    await bringWindowToFrontByLabel(
      layoutWindow.runtimeLabel,
      options.setFocus === true,
    );
    return targetWindow;
  } catch (error) {
    if (!isInvalidWindowHandleError(error)) {
      throw error;
    }
    await reclaimWindowLabel(layoutWindow.runtimeLabel).catch(() => undefined);
    if (typeof console !== "undefined") {
      console.warn(
        `[FORGE] window ${layoutWindow.runtimeLabel} has invalid handle, recreating`,
        error,
      );
    }
    return createShellWindow({
      label: layoutWindow.runtimeLabel,
      route: options.route,
      title: layoutWindow.title,
      bounds,
    });
  }
}

function mergeRuntimeWindow(doc: LayoutDoc, next: RuntimeWindowRecord) {
  const runtimeWindows = doc.runtimeWindows.filter(
    (item) => item.runtimeLabel !== next.runtimeLabel,
  );
  runtimeWindows.push(next);
  doc.runtimeWindows = runtimeWindows.sort((a, b) =>
    a.runtimeLabel.localeCompare(b.runtimeLabel),
  );
}

async function syncRuntimeWindowRegistry(doc: LayoutDoc) {
  const runtimeWindows = await listRuntimeWindows();
  const liveLabels = new Set(runtimeWindows.map((item) => item.label));
  doc.runtimeWindows = doc.runtimeWindows.filter(
    (item) => liveLabels.has(item.runtimeLabel) || item.runtimeLabel === "main",
  );
}

async function closeSecondaryShellHosts() {
  const runtimeWindows = await listRuntimeWindows();
  for (const runtimeWindow of runtimeWindows) {
    if (runtimeWindow.label === "main") continue;
    if (isForgeManagedWindowLabel(runtimeWindow.label)) {
      await closeWindowByLabel(runtimeWindow.label).catch(() => undefined);
    }
  }
}

async function navigateWindow(runtimeLabel: string, route: string) {
  await navigateWindowByLabel(runtimeLabel, route);
}

export async function syncCurrentRuntimeWindow(
  pathname: string,
  monitors: MonitorSnapshot[] = [],
) {
  const currentLabel = await getCurrentWindowLabel();
  const snapshot = await getCurrentWindowSnapshot();
  const doc = loadDoc(monitors);
  const activeLayout = findLayout(doc, doc.activeLayoutId);
  const layoutWindow =
    activeLayout?.windows.find((item) => item.runtimeLabel === currentLabel) ??
    null;
  const next: RuntimeWindowRecord = {
    runtimeLabel: currentLabel,
    layoutId: activeLayout?.id ?? null,
    layoutWindowId: layoutWindow?.id ?? null,
    role: layoutWindow?.role ?? "mixed",
    currentRoute: pathname,
    title: snapshot.title,
    monitorId: snapshot.monitorId,
    isFocused: snapshot.isFocused,
    bounds: snapshot.bounds,
    lastSeenAtMs: nowMs(),
  };
  mergeRuntimeWindow(doc, next);
  if (layoutWindow) {
    layoutWindow.activeRoute = pathname;
    layoutWindow.bounds = snapshot.bounds;
    layoutWindow.fallbackReason = null;
    activeLayout!.updatedAtMs = nowMs();
  }
  await syncRuntimeWindowRegistry(doc);
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}

export async function applyLayout(
  layoutId: string,
  markRestore = false,
  monitors: MonitorSnapshot[] = [],
) {
  if (!isTauriDesktop()) return loadDoc(monitors);
  const resolvedMonitors =
    monitors.length > 0 ? monitors : await listAvailableMonitors();
  const doc = loadDoc(resolvedMonitors);
  const layout = findLayout(doc, layoutId);
  if (!layout) return doc;
  const currentLabel = await getCurrentWindowLabel();
  const resolvedMonitorState = ensureDocMonitors(doc, resolvedMonitors);
  const fallbacks: string[] = [];
  doc.activeLayoutId = layoutId;
  doc.selectedLayoutId = layoutId;
  doc.lastKnownMonitors = resolvedMonitors;
  doc.lastMonitorSignature = monitorSignature(resolvedMonitors);
  doc.monitorDesignations = resolvedMonitorState.monitorDesignations;
  doc.lastRestoreAtMs = markRestore ? nowMs() : doc.lastRestoreAtMs;
  ensureLayoutMonitorHosts(layout, resolvedMonitors, doc.monitorDesignations);

  if (EXTENDED_DESKTOP_SINGLE_SHELL && currentLabel === "main") {
    const mainRecord =
      layout.windows.find((item) => item.runtimeLabel === "main") ??
      layout.windows[0] ??
      defaultWindowForLayout({
        runtimeLabel: "main",
        title: "FORGE",
        role: "mixed",
        targetMonitorOrdinal: 0,
        activeRoute: "/chat",
      });
    const resolved = resolveWindowPlacement(
      mainRecord,
      resolvedMonitors,
      doc.monitorDesignations,
    );
    if (resolved.fallbackReason) fallbacks.push(resolved.fallbackReason);

    await setCurrentWindowTitle(mainRecord.title || "FORGE");
    let shellBounds = resolved.bounds;
    if (resolvedMonitors.length > 1) {
      const spanned = await spanCurrentWindowAcrossMonitors(resolvedMonitors);
      shellBounds = virtualDesktopBounds(resolvedMonitors) ?? resolved.bounds;
      if (!spanned) {
        fallbacks.push(
          "Unable to span the desktop shell across all displays; using the main display.",
        );
        await setCurrentWindowBounds(resolved.bounds);
        shellBounds = resolved.bounds;
      }
    } else {
      await setCurrentWindowBounds(resolved.bounds);
    }
    await navigateWindow("main", mainRecord.activeRoute || "/chat");
    await bringCurrentWindowFront(true).catch(() => undefined);
    await closeSecondaryShellHosts();

    const runtimeRecord: RuntimeWindowRecord = {
      runtimeLabel: "main",
      layoutId: layout.id,
      layoutWindowId: mainRecord.id,
      role: mainRecord.role,
      currentRoute: mainRecord.activeRoute || "/chat",
      title: mainRecord.title || "FORGE",
      monitorId: resolved.monitor?.id ?? null,
      isFocused: true,
      bounds: shellBounds,
      lastSeenAtMs: nowMs(),
    };
    mergeRuntimeWindow(doc, runtimeRecord);
    doc.runtimeWindows = doc.runtimeWindows.filter(
      (item) => item.runtimeLabel === "main",
    );
    doc.fallbackNotice = fallbacks.length > 0 ? fallbacks.join(" ") : null;
    layout.lastActivatedAtMs = nowMs();
    layout.updatedAtMs = nowMs();
    persistDoc(doc);
    await emitWorkspaceSync(currentLabel);
    return doc;
  }

  for (const windowRecord of layout.windows) {
    if (!windowRecord.runtimeLabel) continue;
    const resolved = resolveWindowPlacement(
      windowRecord,
      resolvedMonitors,
      doc.monitorDesignations,
    );
    windowRecord.fallbackReason = resolved.fallbackReason;
    if (resolved.fallbackReason) fallbacks.push(resolved.fallbackReason);

    const targetWindow =
      windowRecord.runtimeLabel === currentLabel
        ? null
        : await getWindowByLabel(windowRecord.runtimeLabel);
    if (windowRecord.runtimeLabel === currentLabel) {
      await setCurrentWindowTitle(windowRecord.title);
      await setCurrentWindowBounds(resolved.bounds);
      await navigateWindow(currentLabel, windowRecord.activeRoute);
      await bringCurrentWindowFront(true).catch(() => undefined);
    } else if (targetWindow) {
      try {
        await syncOrRecreateWindow(windowRecord, resolved.bounds, {
          route: windowRecord.activeRoute,
          setFocus: false,
        });
      } catch (error) {
        if (isInvalidWindowHandleError(error)) {
          await reclaimWindowLabel(windowRecord.runtimeLabel).catch(
            () => undefined,
          );
          if (typeof console !== "undefined") {
            console.warn(
              `[FORGE] window ${windowRecord.runtimeLabel} could not be restored`,
              error,
            );
          }
          await createShellWindow({
            label: windowRecord.runtimeLabel,
            route: windowRecord.activeRoute,
            title: windowRecord.title,
            bounds: resolved.bounds,
          });
        }
      }
    } else {
      await createShellWindow({
        label: windowRecord.runtimeLabel,
        route: windowRecord.activeRoute,
        title: windowRecord.title,
        bounds: resolved.bounds,
      });
    }
  }

  const desired = new Set(layout.windows.map((item) => item.runtimeLabel));
  const runtimeWindows = await listRuntimeWindows();
  for (const runtimeWindow of runtimeWindows) {
    if (runtimeWindow.label === "main") continue;
    if (!desired.has(runtimeWindow.label)) {
      if (
        DETACHED_TAURI_TOOL_WINDOWS ||
        isForgeManagedWindowLabel(runtimeWindow.label)
      ) {
        await closeWindowByLabel(runtimeWindow.label).catch(() => undefined);
      }
    }
  }
  await syncRuntimeWindowRegistry(doc);

  doc.fallbackNotice = fallbacks.length > 0 ? fallbacks.join(" ") : null;
  layout.lastActivatedAtMs = nowMs();
  layout.updatedAtMs = nowMs();
  persistDoc(doc);
  await emitWorkspaceSync(currentLabel);
  return doc;
}
