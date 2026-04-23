import { create } from "zustand";

type UiMode = "guided" | "pro";
type ContrastPreference = "normal" | "high";
type EffectsPreference = "off" | "subtle";

function loadMode(): UiMode {
  if (typeof window === "undefined") return "guided";
  const raw = window.localStorage.getItem("forge.ui.mode");
  return raw === "pro" ? "pro" : "guided";
}

function loadContrast(): ContrastPreference {
  if (typeof window === "undefined") return "high";
  const raw = window.localStorage.getItem("forge.ui.contrast");
  if (raw === "normal" || raw === "high") return raw;
  return "high";
}

function loadEffects(): EffectsPreference {
  if (typeof window === "undefined") return "subtle";
  const raw = window.localStorage.getItem("forge.ui.effects");
  if (raw === "off" || raw === "subtle") return raw;
  return "subtle";
}

type UiState = {
  commandDraft: string;
  setCommandDraft: (v: string) => void;
  statusLine: string;
  setStatusLine: (v: string) => void;
  uiMode: UiMode;
  setUiMode: (mode: UiMode) => void;
  toggleUiMode: () => void;
  contrastPreference: ContrastPreference;
  setContrastPreference: (value: ContrastPreference) => void;
  effectsPreference: EffectsPreference;
  setEffectsPreference: (value: EffectsPreference) => void;
  toggleContrastPreference: () => void;
  toggleEffectsPreference: () => void;
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
  contrastPreference: loadContrast(),
  setContrastPreference: (value) =>
    set(() => {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.contrast", value);
      }
      return { contrastPreference: value };
    }),
  effectsPreference: loadEffects(),
  setEffectsPreference: (value) =>
    set(() => {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.effects", value);
      }
      return { effectsPreference: value };
    }),
  toggleUiMode: () =>
    set((s) => {
      const next: UiMode = s.uiMode === "guided" ? "pro" : "guided";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.mode", next);
      }
      return { uiMode: next };
    }),
  toggleContrastPreference: () =>
    set((s) => {
      const next: ContrastPreference = s.contrastPreference === "high" ? "normal" : "high";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.contrast", next);
      }
      return { contrastPreference: next };
    }),
  toggleEffectsPreference: () =>
    set((s) => {
      const next: EffectsPreference = s.effectsPreference === "subtle" ? "off" : "subtle";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.effects", next);
      }
      return { effectsPreference: next };
    }),
}));
