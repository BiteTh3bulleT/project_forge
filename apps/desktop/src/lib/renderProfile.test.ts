import { describe, expect, it } from "vitest";

import {
  initialEffectsPreference,
  normalizeRenderProfile,
} from "./renderProfile";

describe("render profile", () => {
  it("normalizes unknown render profiles to the default profile", () => {
    expect(normalizeRenderProfile("vm-safe")).toBe("vm-safe");
    expect(normalizeRenderProfile("default")).toBe("default");
    expect(normalizeRenderProfile("native")).toBe("default");
    expect(normalizeRenderProfile(undefined)).toBe("default");
  });

  it("uses low-cost effects by default for the VM-safe profile", () => {
    expect(initialEffectsPreference("vm-safe", null)).toBe("off");
    expect(initialEffectsPreference("default", null)).toBe("subtle");
  });

  it("keeps an explicit operator effects preference", () => {
    expect(initialEffectsPreference("vm-safe", "subtle")).toBe("subtle");
    expect(initialEffectsPreference("vm-safe", "off")).toBe("off");
    expect(initialEffectsPreference("vm-safe", "invalid")).toBe("off");
  });
});
