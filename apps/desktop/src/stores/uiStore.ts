import { create } from "zustand";

type UiMode = "guided" | "pro";

function loadMode(): UiMode {
  if (typeof window === "undefined") return "guided";
  const raw = window.localStorage.getItem("forge.ui.mode");
  return raw === "pro" ? "pro" : "guided";
}

type UiState = {
  commandDraft: string;
  setCommandDraft: (v: string) => void;
  statusLine: string;
  setStatusLine: (v: string) => void;
  uiMode: UiMode;
  setUiMode: (mode: UiMode) => void;
  toggleUiMode: () => void;
};

export const useUiStore = create<UiState>((set) => ({
  commandDraft: "",
  setCommandDraft: (v) => set({ commandDraft: v }),
  statusLine: "Workshop idle.",
  setStatusLine: (v) => set({ statusLine: v }),
  uiMode: loadMode(),
  setUiMode: (mode) =>
    set(() => {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.mode", mode);
      }
      return { uiMode: mode };
    }),
  toggleUiMode: () =>
    set((s) => {
      const next: UiMode = s.uiMode === "guided" ? "pro" : "guided";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.mode", next);
      }
      return { uiMode: next };
    }),
}));
