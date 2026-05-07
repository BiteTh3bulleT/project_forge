import { describe, expect, it } from "vitest";

import {
  projectDesktopWindowToHost,
  resolveDesktopHostPlacement,
} from "./desktopGeometry";

const shellHost = (label: string) =>
  label === "main" ||
  (label.startsWith("forge-") && !label.startsWith("forge-app-"));

describe("desktop geometry", () => {
  it("moves a window from the main host into the right monitor host", () => {
    const placement = resolveDesktopHostPlacement(
      [
        {
          runtimeLabel: "main",
          bounds: { x: 0, y: 0, width: 1000, height: 800 },
        },
        {
          runtimeLabel: "forge-right",
          bounds: { x: 1000, y: 0, width: 1000, height: 800 },
        },
      ],
      "main",
      { hostLabel: "main", width: 300, height: 200 },
      920,
      120,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "forge-right",
      x: -80,
      y: 120,
    });
  });

  it("moves a window from a right monitor host back to the main host", () => {
    const placement = resolveDesktopHostPlacement(
      [
        {
          runtimeLabel: "main",
          bounds: { x: 0, y: 0, width: 1000, height: 800 },
        },
        {
          runtimeLabel: "forge-right",
          bounds: { x: 1000, y: 0, width: 1000, height: 800 },
        },
      ],
      "forge-right",
      { hostLabel: "forge-right", width: 300, height: 200 },
      -220,
      120,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "main",
      x: 780,
      y: 120,
    });
  });

  it("supports hosts with negative global coordinates", () => {
    const placement = resolveDesktopHostPlacement(
      [
        {
          runtimeLabel: "forge-left",
          bounds: { x: -1000, y: 0, width: 1000, height: 800 },
        },
        {
          runtimeLabel: "main",
          bounds: { x: 0, y: 0, width: 1000, height: 800 },
        },
      ],
      "main",
      { hostLabel: "main", width: 300, height: 200 },
      -220,
      120,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "forge-left",
      x: 780,
      y: 120,
    });
  });

  it("supports monitor hosts above the main desktop", () => {
    const placement = resolveDesktopHostPlacement(
      [
        {
          runtimeLabel: "forge-upper",
          bounds: { x: 0, y: -800, width: 1000, height: 800 },
        },
        {
          runtimeLabel: "main",
          bounds: { x: 0, y: 0, width: 1000, height: 800 },
        },
      ],
      "main",
      { hostLabel: "main", width: 300, height: 200 },
      120,
      -220,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "forge-upper",
      x: 120,
      y: 580,
    });
  });

  it("does not transfer windows into detached app hosts", () => {
    const placement = resolveDesktopHostPlacement(
      [
        {
          runtimeLabel: "main",
          bounds: { x: 0, y: 0, width: 1000, height: 800 },
        },
        {
          runtimeLabel: "forge-app-chat",
          bounds: { x: 1000, y: 0, width: 1000, height: 800 },
        },
      ],
      "main",
      { hostLabel: "main", width: 300, height: 200 },
      920,
      120,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "main",
      x: 920,
      y: 120,
    });
  });

  it("projects a crossing window into both shell hosts", () => {
    const runtimeWindows = [
      {
        runtimeLabel: "main",
        bounds: { x: 0, y: 0, width: 1000, height: 800 },
      },
      {
        runtimeLabel: "forge-right",
        bounds: { x: 1000, y: 0, width: 1000, height: 800 },
      },
    ];

    expect(
      projectDesktopWindowToHost(runtimeWindows, "main", {
        hostLabel: "forge-right",
        x: -80,
        y: 120,
        width: 300,
        height: 200,
      }),
    ).toEqual({
      hostLabel: "main",
      x: 920,
      y: 120,
    });
    expect(
      projectDesktopWindowToHost(runtimeWindows, "forge-right", {
        hostLabel: "forge-right",
        x: -80,
        y: 120,
        width: 300,
        height: 200,
      }),
    ).toEqual({
      hostLabel: "forge-right",
      x: -80,
      y: 120,
    });
  });

  it("does not project windows onto non-intersecting hosts", () => {
    expect(
      projectDesktopWindowToHost(
        [
          {
            runtimeLabel: "main",
            bounds: { x: 0, y: 0, width: 1000, height: 800 },
          },
          {
            runtimeLabel: "forge-right",
            bounds: { x: 1000, y: 0, width: 1000, height: 800 },
          },
        ],
        "main",
        {
          hostLabel: "forge-right",
          x: 120,
          y: 120,
          width: 300,
          height: 200,
        },
      ),
    ).toBeNull();
  });
});
