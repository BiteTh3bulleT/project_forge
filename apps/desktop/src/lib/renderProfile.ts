export type ForgeRenderProfile = "default" | "vm-safe";
export type EffectsPreference = "off" | "subtle";

const VALID_RENDER_PROFILES = new Set<ForgeRenderProfile>([
  "default",
  "vm-safe",
]);

export function normalizeRenderProfile(
  value: unknown,
): ForgeRenderProfile {
  return typeof value === "string" &&
    VALID_RENDER_PROFILES.has(value as ForgeRenderProfile)
    ? (value as ForgeRenderProfile)
    : "default";
}

export function initialEffectsPreference(
  profile: ForgeRenderProfile,
  storedValue: string | null,
): EffectsPreference {
  if (storedValue === "off" || storedValue === "subtle") return storedValue;
  return profile === "vm-safe" ? "off" : "subtle";
}

export function configuredRenderProfile(): ForgeRenderProfile {
  if (typeof window !== "undefined") {
    const stored = window.localStorage.getItem("forge.render.profile");
    const normalizedStored = normalizeRenderProfile(stored);
    if (stored === normalizedStored) return normalizedStored;

    const runtimeProfile = (window as unknown as {
      __FORGE_RENDER_PROFILE__?: unknown;
    }).__FORGE_RENDER_PROFILE__;
    const normalizedRuntime = normalizeRenderProfile(runtimeProfile);
    if (runtimeProfile === normalizedRuntime) return normalizedRuntime;
  }

  return normalizeRenderProfile(import.meta.env.VITE_FORGE_RENDER_PROFILE);
}
