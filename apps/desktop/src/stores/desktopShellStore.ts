import { create } from "zustand";

const STORAGE_KEY = "forge.desktop.session.v2";
const MAX_RECENT = 10;

type WindowSession = {
  openRoutes: string[];
  recentRoutes: string[];
};

type PersistedShellState = {
  currentWindowId: string;
  sessions: Record<string, WindowSession>;
};

type DesktopShellState = {
  currentWindowId: string;
  openRoutes: string[];
  recentRoutes: string[];
  hydrate: (windowId: string) => void;
  openRoute: (route: string) => void;
  closeRoute: (route: string) => void;
  touchRoute: (route: string) => void;
};

function defaultSession(): WindowSession {
  return { openRoutes: ["/chat"], recentRoutes: ["/chat"] };
}

function load(): PersistedShellState {
  if (typeof window === "undefined")
    return { currentWindowId: "main", sessions: { main: defaultSession() } };
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw)
      return { currentWindowId: "main", sessions: { main: defaultSession() } };
    const parsed = JSON.parse(raw) as PersistedShellState;
    return {
      currentWindowId: parsed.currentWindowId || "main",
      sessions:
        parsed.sessions && typeof parsed.sessions === "object"
          ? parsed.sessions
          : { main: defaultSession() },
    };
  } catch {
    return { currentWindowId: "main", sessions: { main: defaultSession() } };
  }
}

function persist(state: PersistedShellState) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
}

function sessionFor(state: PersistedShellState, windowId: string) {
  return state.sessions[windowId] ?? defaultSession();
}

export const useDesktopShellStore = create<DesktopShellState>((set, get) => ({
  currentWindowId: "main",
  openRoutes: ["/chat"],
  recentRoutes: ["/chat"],
  hydrate: (windowId) => {
    const loaded = load();
    const currentWindowId = windowId || loaded.currentWindowId || "main";
    const next: PersistedShellState = {
      currentWindowId,
      sessions: {
        ...loaded.sessions,
        [currentWindowId]: sessionFor(loaded, currentWindowId),
      },
    };
    persist(next);
    const session = sessionFor(next, currentWindowId);
    set({
      currentWindowId,
      openRoutes: session.openRoutes,
      recentRoutes: session.recentRoutes,
    });
  },
  openRoute: (route) => {
    const currentWindowId = get().currentWindowId;
    const loaded = load();
    const session = sessionFor(loaded, currentWindowId);
    const nextSession: WindowSession = {
      openRoutes: session.openRoutes.includes(route)
        ? session.openRoutes
        : [...session.openRoutes, route],
      recentRoutes: [
        route,
        ...session.recentRoutes.filter((value) => value !== route),
      ].slice(0, MAX_RECENT),
    };
    const next = {
      currentWindowId,
      sessions: { ...loaded.sessions, [currentWindowId]: nextSession },
    };
    persist(next);
    set({
      openRoutes: nextSession.openRoutes,
      recentRoutes: nextSession.recentRoutes,
    });
  },
  closeRoute: (route) => {
    const currentWindowId = get().currentWindowId;
    const loaded = load();
    const session = sessionFor(loaded, currentWindowId);
    const openRoutes = session.openRoutes.filter((value) => value !== route);
    const nextSession: WindowSession = {
      openRoutes: openRoutes.length > 0 ? openRoutes : ["/chat"],
      recentRoutes: session.recentRoutes.filter((value) => value !== route),
    };
    const next = {
      currentWindowId,
      sessions: { ...loaded.sessions, [currentWindowId]: nextSession },
    };
    persist(next);
    set({
      openRoutes: nextSession.openRoutes,
      recentRoutes: nextSession.recentRoutes,
    });
  },
  touchRoute: (route) => {
    const currentWindowId = get().currentWindowId;
    const loaded = load();
    const session = sessionFor(loaded, currentWindowId);
    const nextSession: WindowSession = {
      openRoutes: session.openRoutes,
      recentRoutes: [
        route,
        ...session.recentRoutes.filter((value) => value !== route),
      ].slice(0, MAX_RECENT),
    };
    const next = {
      currentWindowId,
      sessions: { ...loaded.sessions, [currentWindowId]: nextSession },
    };
    persist(next);
    set({ recentRoutes: nextSession.recentRoutes });
  },
}));
