import { describe, expect, it } from "vitest";

import { formatTime } from "./format";

describe("formatTime", () => {
  it("returns a non-empty string for a valid millisecond timestamp", () => {
    const out = formatTime(1700000000000);
    expect(typeof out).toBe("string");
    expect(out.length).toBeGreaterThan(0);
  });

  it("returns a string for any numeric input (no thrown error)", () => {
    expect(typeof formatTime(0)).toBe("string");
    expect(typeof formatTime(Number.NaN)).toBe("string");
  });
});
