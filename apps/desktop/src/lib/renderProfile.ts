export type ForgeRenderProfile = "default" | "vm-safe";
export type EffectsPreference = "off" | "subtle";

const VALID_RENDER_PROFILES = new Set<ForgeRenderProfile>([
  "default",
  "vm-safe",
]);

function explicitRenderProfile(value: unknown): ForgeRenderProfile | null {
  return typeof value === "string" &&
    VALID_RENDER_PROFILES.has(value as ForgeRenderProfile)
    ? (value as ForgeRenderProfile)
    : null;
}

export function normalizeRenderProfile(
  value: unknown,
): ForgeRenderProfile {
  return explicitRenderProfile(value) ?? "default";
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
    const runtimeProfile = (window as unknown as {
      __FORGE_RENDER_PROFILE__?: unknown;
    }).__FORGE_RENDER_PROFILE__;
    const explicitRuntime = explicitRenderProfile(runtimeProfile);
    if (explicitRuntime) return explicitRuntime;

    const stored = window.localStorage.getItem("forge.render.profile");
    const explicitStored = explicitRenderProfile(stored);
    if (explicitStored) return explicitStored;
  }

  return explicitRenderProfile(import.meta.env.VITE_FORGE_RENDER_PROFILE) ?? "default";
}
