import { create } from "zustand";

type UiMode = "cognitive" | "metrics";
type ContrastPreference = "normal" | "high";
type EffectsPreference = "off" | "subtle";
type ThemePreference = "dark" | "light";
type AccentPreference = "cyan" | "amber" | "mint";

function loadMode(): UiMode {
  if (typeof window === "undefined") return "cognitive";
  const raw = window.localStorage.getItem("forge.ui.mode");
  if (raw === "metrics" || raw === "pro") return "metrics";
  return "cognitive";
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

function loadTheme(): ThemePreference {
  if (typeof window === "undefined") return "dark";
  const raw = window.localStorage.getItem("forge.ui.theme");
  if (raw === "dark" || raw === "light") return raw;
  return "dark";
}

function loadAccent(): AccentPreference {
  if (typeof window === "undefined") return "cyan";
  const raw = window.localStorage.getItem("forge.ui.accent");
  if (raw === "cyan" || raw === "amber" || raw === "mint") return raw;
  return "cyan";
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
  themePreference: ThemePreference;
  setThemePreference: (value: ThemePreference) => void;
  toggleThemePreference: () => void;
  accentPreference: AccentPreference;
  setAccentPreference: (value: AccentPreference) => void;
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
  themePreference: loadTheme(),
  setThemePreference: (value) =>
    set(() => {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.theme", value);
      }
      return { themePreference: value };
    }),
  toggleThemePreference: () =>
    set((s) => {
      const next: ThemePreference =
        s.themePreference === "dark" ? "light" : "dark";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.theme", next);
      }
      return { themePreference: next };
    }),
  accentPreference: loadAccent(),
  setAccentPreference: (value) =>
    set(() => {
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.accent", value);
      }
      return { accentPreference: value };
    }),
  toggleUiMode: () =>
    set((s) => {
      const next: UiMode = s.uiMode === "cognitive" ? "metrics" : "cognitive";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.mode", next);
      }
      return { uiMode: next };
    }),
  toggleContrastPreference: () =>
    set((s) => {
      const next: ContrastPreference =
        s.contrastPreference === "high" ? "normal" : "high";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.contrast", next);
      }
      return { contrastPreference: next };
    }),
  toggleEffectsPreference: () =>
    set((s) => {
      const next: EffectsPreference =
        s.effectsPreference === "subtle" ? "off" : "subtle";
      if (typeof window !== "undefined") {
        window.localStorage.setItem("forge.ui.effects", next);
      }
      return { effectsPreference: next };
    }),
}));
