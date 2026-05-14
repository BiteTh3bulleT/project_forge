import { create } from "zustand";

import {
  isTauriDesktop,
  listAvailableMonitors,
  monitorSignature,
} from "../lib/desktop";
import { getCurrentWindowLabel } from "../lib/windowManager";

import { AUTO_RESTORE_TAURI_LAYOUTS } from "./workspaceLayoutStore/constants";
import {
  clone,
  defaultWindowForLayout,
  deriveMonitorState,
  ensureActiveLayout,
  findLayout,
  monitorStateFromDesignations,
  normalizeMonitorDesignations,
  nowMs,
  sanitizeRoutes,
  uid,
} from "./workspaceLayoutStore/model";
import { loadDoc, persistDoc } from "./workspaceLayoutStore/persistence";
import {
  applyLayout,
  syncCurrentRuntimeWindow,
} from "./workspaceLayoutStore/runtime";
import type {
  LayoutPreset,
  WorkspaceLayoutState,
} from "./workspaceLayoutStore/types";

export const useWorkspaceLayoutStore = create<WorkspaceLayoutState>(
  (set, get) => ({
    ready: false,
    supported: false,
    currentWindowLabel: "main",
    monitorDesignations: { mainMonitorId: null, customLabels: {} },
    monitorRoleMap: {},
    activeLayoutId: null,
    selectedLayoutId: null,
    layouts: [],
    monitors: [],
    runtimeWindows: [],
    fallbackNotice: null,
    hydrate: async (pathname) => {
      const supported = isTauriDesktop();
      const currentWindowLabel = await getCurrentWindowLabel();
      const monitors = supported ? await listAvailableMonitors() : [];
      let doc = loadDoc(monitors);
      doc = ensureActiveLayout(doc);
      if (supported) {
        doc = await syncCurrentRuntimeWindow(pathname, monitors);
        if (
          AUTO_RESTORE_TAURI_LAYOUTS &&
          currentWindowLabel === "main" &&
          doc.activeLayoutId
        ) {
          doc = await applyLayout(doc.activeLayoutId, true, monitors);
        }
      }
      const monitorState = deriveMonitorState(monitors, doc);
      set({
        ready: true,
        supported,
        currentWindowLabel,
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        runtimeWindows: clone(doc.runtimeWindows),
        monitors: clone(monitors),
        monitorDesignations: clone(monitorState.monitorDesignations),
        monitorRoleMap: monitorState.monitorRoleMap,
        fallbackNotice: doc.fallbackNotice,
      });
    },
    refreshEnvironment: async () => {
      const supported = get().supported;
      if (!supported) return;
      const currentWindowLabel = get().currentWindowLabel;
      const monitors = await listAvailableMonitors();
      const signature = monitorSignature(monitors);
      const doc = loadDoc(monitors);
      const changed = doc.lastMonitorSignature !== signature;
      doc.lastKnownMonitors = monitors;
      doc.lastMonitorSignature = signature;
      persistDoc(doc);
      if (
        AUTO_RESTORE_TAURI_LAYOUTS &&
        changed &&
        currentWindowLabel === "main" &&
        doc.activeLayoutId
      ) {
        const refreshed = await applyLayout(
          doc.activeLayoutId,
          false,
          monitors,
        );
        doc.layouts = refreshed.layouts;
        doc.runtimeWindows = refreshed.runtimeWindows;
        doc.fallbackNotice = refreshed.fallbackNotice;
        doc.lastRestoreAtMs = refreshed.lastRestoreAtMs;
        doc.monitorDesignations = refreshed.monitorDesignations;
      } else {
        const route =
          doc.runtimeWindows.find(
            (item) => item.runtimeLabel === currentWindowLabel,
          )?.currentRoute ?? "/chat";
        const refreshed = await syncCurrentRuntimeWindow(route, monitors);
        doc.layouts = refreshed.layouts;
        doc.runtimeWindows = refreshed.runtimeWindows;
        doc.fallbackNotice = refreshed.fallbackNotice;
        doc.lastRestoreAtMs = refreshed.lastRestoreAtMs;
      }
      const monitorState = deriveMonitorState(monitors, doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        monitors: clone(doc.lastKnownMonitors),
        runtimeWindows: clone(doc.runtimeWindows),
        fallbackNotice: doc.fallbackNotice,
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
      });
    },
    syncCurrentRoute: async (pathname) => {
      const doc = await syncCurrentRuntimeWindow(pathname, get().monitors);
      const monitorState = deriveMonitorState(get().monitors, doc);
      set({
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        runtimeWindows: clone(doc.runtimeWindows),
        fallbackNotice: doc.fallbackNotice,
      });
    },
    createLayout: (name) => {
      const doc = loadDoc(get().monitors);
      const label = name.trim() || "New Layout";
      const layoutId = uid("layout");
      const createdAtMs = nowMs();
      doc.layouts.push({
        id: layoutId,
        name: label,
        createdAtMs,
        updatedAtMs: createdAtMs,
        lastActivatedAtMs: null,
        windows: [
          defaultWindowForLayout({
            runtimeLabel: "main",
            title: `FORGE ${label}`,
            role: "mixed",
            targetMonitorOrdinal: 0,
            activeRoute: "/chat",
          }),
        ],
      });
      doc.selectedLayoutId = layoutId;
      persistDoc(doc);
      set({ layouts: clone(doc.layouts), selectedLayoutId: layoutId });
    },
    selectLayout: (layoutId) => {
      const doc = loadDoc(get().monitors);
      doc.selectedLayoutId = layoutId;
      persistDoc(doc);
      set({ selectedLayoutId: layoutId });
    },
    renameLayout: (layoutId, name) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      layout.name = name.trim() || layout.name;
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    duplicateLayout: (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const createdAtMs = nowMs();
      const cloneLayout: LayoutPreset = clone(layout);
      cloneLayout.id = uid("layout");
      cloneLayout.name = `${layout.name} Copy`;
      cloneLayout.createdAtMs = createdAtMs;
      cloneLayout.updatedAtMs = createdAtMs;
      cloneLayout.lastActivatedAtMs = null;
      cloneLayout.windows = cloneLayout.windows.map((windowRecord, index) => ({
        ...windowRecord,
        id: uid("window"),
        runtimeLabel: index === 0 ? "main" : uid(`forge-${cloneLayout.id}`),
      }));
      doc.layouts.push(cloneLayout);
      doc.selectedLayoutId = cloneLayout.id;
      persistDoc(doc);
      set({ layouts: clone(doc.layouts), selectedLayoutId: cloneLayout.id });
    },
    deleteLayout: async (layoutId) => {
      const doc = loadDoc(get().monitors);
      if (doc.layouts.length <= 1) return;
      doc.layouts = doc.layouts.filter((layout) => layout.id !== layoutId);
      if (doc.activeLayoutId === layoutId) {
        doc.activeLayoutId = doc.layouts[0]?.id ?? null;
      }
      if (doc.selectedLayoutId === layoutId) {
        doc.selectedLayoutId = doc.layouts[0]?.id ?? null;
      }
      persistDoc(doc);
      if (doc.activeLayoutId) {
        await applyLayout(doc.activeLayoutId, false, get().monitors);
      }
      set({
        layouts: clone(doc.layouts),
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
      });
    },
    addLayoutWindow: (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const nextIndex = layout.windows.length;
      layout.windows.push(
        defaultWindowForLayout({
          runtimeLabel: uid(`forge-${layout.id}`),
          title: `FORGE Window ${nextIndex + 1}`,
          role: "mixed",
          targetMonitorOrdinal: nextIndex,
          activeRoute: "/chat",
        }),
      );
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    removeLayoutWindow: (layoutId, windowId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      if (layout.windows.length <= 1) return;
      const target = layout.windows.find((item) => item.id === windowId);
      if (!target || target.runtimeLabel === "main") return;
      layout.windows = layout.windows.filter((item) => item.id !== windowId);
      layout.updatedAtMs = nowMs();
      persistDoc(doc);
      set({ layouts: clone(doc.layouts) });
    },
    updateLayoutWindow: (layoutId, windowId, patch) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      const target = layout.windows.find((item) => item.id === windowId);
      if (!target) return;
      if (patch.assignedRoutes) {
        patch.assignedRoutes = sanitizeRoutes(patch.assignedRoutes);
      }
      Object.assign(target, patch);
      target.assignedRoutes = sanitizeRoutes(target.assignedRoutes);
      if (!target.assignedRoutes.includes(target.activeRoute)) {
        target.activeRoute = target.assignedRoutes[0] ?? "/chat";
      }
      if (target.runtimeLabel === "main") {
        target.runtimeLabel = "main";
      }
      layout.updatedAtMs = nowMs();
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
    setMainMonitor: (monitorId) => {
      const currentMonitors = get().monitors;
      const doc = loadDoc(currentMonitors);
      const hasMonitor = currentMonitors.some(
        (monitor) => monitor.id === monitorId,
      );
      if (!hasMonitor) return;
      doc.monitorDesignations.mainMonitorId = monitorId;
      doc.monitorDesignations.customLabels = normalizeMonitorDesignations(
        doc.monitorDesignations,
      ).customLabels;
      const next = monitorStateFromDesignations(
        currentMonitors,
        doc.monitorDesignations,
      );
      doc.monitorDesignations = next.monitorDesignations;
      persistDoc(doc);
      set({
        monitorDesignations: next.monitorDesignations,
        monitorRoleMap: next.monitorRoleMap,
      });
      const activeLayoutId = doc.activeLayoutId;
      if (activeLayoutId) {
        void applyLayout(activeLayoutId, false, currentMonitors).catch(
          () => undefined,
        );
      }
    },
    setMonitorRoleLabel: (monitorId, label) => {
      const currentMonitors = get().monitors;
      const doc = loadDoc(currentMonitors);
      const cleanLabel = label.trim();
      if (!currentMonitors.some((monitor) => monitor.id === monitorId)) return;
      if (cleanLabel.length === 0) {
        delete doc.monitorDesignations.customLabels[monitorId];
      } else {
        doc.monitorDesignations.customLabels[monitorId] = cleanLabel;
      }
      const next = monitorStateFromDesignations(
        currentMonitors,
        doc.monitorDesignations,
      );
      doc.monitorDesignations = next.monitorDesignations;
      persistDoc(doc);
      set({
        monitorDesignations: next.monitorDesignations,
        monitorRoleMap: next.monitorRoleMap,
      });
    },
    activateLayout: async (layoutId) => {
      const doc = await applyLayout(layoutId, false, get().monitors);
      const monitorState = deriveMonitorState(get().monitors, doc);
      set({
        activeLayoutId: doc.activeLayoutId,
        selectedLayoutId: doc.selectedLayoutId,
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
        runtimeWindows: clone(doc.runtimeWindows),
        monitors: clone(doc.lastKnownMonitors),
        fallbackNotice: doc.fallbackNotice,
      });
    },
    captureRuntimeIntoLayout: async (layoutId) => {
      const doc = loadDoc(get().monitors);
      const layout = findLayout(doc, layoutId);
      if (!layout) return;
      for (const runtimeWindow of doc.runtimeWindows) {
        const layoutWindow = layout.windows.find(
          (item) => item.runtimeLabel === runtimeWindow.runtimeLabel,
        );
        if (!layoutWindow) continue;
        layoutWindow.bounds = runtimeWindow.bounds;
        layoutWindow.activeRoute = runtimeWindow.currentRoute;
        layoutWindow.targetMonitorId = runtimeWindow.monitorId;
        const matchedMonitor = doc.lastKnownMonitors.find(
          (monitor) => monitor.id === runtimeWindow.monitorId,
        );
        layoutWindow.targetMonitorOrdinal =
          matchedMonitor?.ordinal ?? layoutWindow.targetMonitorOrdinal;
        layoutWindow.targetMonitorRole = matchedMonitor
          ? (monitorStateFromDesignations(
              doc.lastKnownMonitors,
              doc.monitorDesignations,
            ).monitorRoleMap[matchedMonitor.id] ?? null)
          : layoutWindow.targetMonitorRole;
        layoutWindow.title = runtimeWindow.title;
      }
      layout.updatedAtMs = nowMs();
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        layouts: clone(doc.layouts),
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
    clearFallbackNotice: () => {
      const doc = loadDoc(get().monitors);
      doc.fallbackNotice = null;
      const monitorState = deriveMonitorState(get().monitors, doc);
      persistDoc(doc);
      set({
        fallbackNotice: null,
        monitorDesignations: monitorState.monitorDesignations,
        monitorRoleMap: monitorState.monitorRoleMap,
      });
    },
  }),
);
