import { describe, expect, it } from "vitest";

import rustSource from "../../src-tauri/src/main.rs?raw";
import { FALLBACK_OPERATOR_APPS } from "./operatorApps";

function rustOperatorAppIds() {
  const catalogBlock = rustSource.match(
    /const OPERATOR_APPS:[\s\S]*?;\s*\n\s*fn find_desktop_file/,
  )?.[0];
  if (!catalogBlock) {
    throw new Error("Rust operator app catalog block not found");
  }
  return Array.from(catalogBlock.matchAll(/\bid:\s*"([^"]+)"/g), (match) =>
    String(match[1]),
  );
}

describe("operator app fallback catalog", () => {
  it("keeps fallback ids aligned with the Rust allowlist", () => {
    expect(FALLBACK_OPERATOR_APPS.map((app) => app.id)).toEqual(
      rustOperatorAppIds(),
    );
  });

  it("does not include duplicate fallback ids", () => {
    const ids = FALLBACK_OPERATOR_APPS.map((app) => app.id);
    expect(new Set(ids).size).toBe(ids.length);
  });
});
