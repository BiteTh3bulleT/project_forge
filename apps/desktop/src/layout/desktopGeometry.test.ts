import { describe, expect, it } from "vitest";

import {
  buildDesktopHosts,
  clampRectToHost,
  getDesktopHostAtGlobalPoint,
  getDesktopHostForMonitor,
  getPrimaryDesktopHost,
  globalToHostPoint,
  hostLabelForMonitorOrdinal,
  hostToGlobalPoint,
  projectDesktopWindowToHost,
  resolveDesktopHostPlacement,
} from "./desktopGeometry";

const shellHost = (label: string) =>
  label === "main" ||
  (label.startsWith("forge-") && !label.startsWith("forge-app-"));

describe("desktop geometry", () => {
  it("builds stable desktop hosts from monitor ordinals", () => {
    expect(hostLabelForMonitorOrdinal(0)).toBe("main");
    expect(hostLabelForMonitorOrdinal(1)).toBe("forge-monitor-2");
    expect(hostLabelForMonitorOrdinal(2)).toBe("forge-monitor-3");
    expect(hostLabelForMonitorOrdinal(7)).toBe("forge-monitor-8");

    const hosts = buildDesktopHosts([
      {
        id: "right",
        ordinal: 1,
        workArea: { x: 1000, y: 0, width: 1200, height: 800 },
      },
      {
        id: "main-monitor",
        ordinal: 0,
        workArea: { x: 0, y: 0, width: 1000, height: 800 },
      },
    ]);

    expect(hosts).toEqual([
      {
        hostLabel: "main",
        monitorId: "main-monitor",
        monitorIndex: 0,
        bounds: { x: 0, y: 0, width: 1000, height: 800 },
        role: "main",
        active: false,
      },
      {
        hostLabel: "forge-monitor-2",
        monitorId: "right",
        monitorIndex: 1,
        bounds: { x: 1000, y: 0, width: 1200, height: 800 },
        role: "secondary",
        active: false,
      },
    ]);
  });

  it("overlays active runtime shell hosts onto monitor hosts", () => {
    const hosts = buildDesktopHosts(
      [
        {
          id: "main-monitor",
          ordinal: 0,
          workArea: { x: 0, y: 0, width: 1000, height: 800 },
        },
        {
          id: "right",
          ordinal: 1,
          workArea: { x: 1000, y: 0, width: 1200, height: 800 },
        },
      ],
      [
        {
          runtimeLabel: "forge-right",
          monitorId: "right",
          bounds: { x: 1000, y: 0, width: 1180, height: 760 },
        },
      ],
    );

    expect(hosts[1]).toEqual({
      hostLabel: "forge-right",
      monitorId: "right",
      monitorIndex: 1,
      bounds: { x: 1000, y: 0, width: 1180, height: 760 },
      role: "secondary",
      active: true,
    });
  });

  it("finds primary, monitor, and global point hosts", () => {
    const hosts = buildDesktopHosts([
      {
        id: "left",
        ordinal: 1,
        workArea: { x: -900, y: 0, width: 900, height: 700 },
      },
      {
        id: "main-monitor",
        ordinal: 0,
        workArea: { x: 0, y: 0, width: 1000, height: 800 },
      },
    ]);

    expect(getPrimaryDesktopHost(hosts)?.hostLabel).toBe("main");
    expect(getDesktopHostForMonitor(hosts, "left")?.hostLabel).toBe(
      "forge-monitor-2",
    );
    expect(getDesktopHostAtGlobalPoint(hosts, { x: -10, y: 20 })?.monitorId).toBe(
      "left",
    );
    expect(getDesktopHostAtGlobalPoint(hosts, { x: 2000, y: 20 })).toBeNull();
  });

  it("converts host-local and global points", () => {
    const host = {
      hostLabel: "forge-monitor-2",
      monitorId: "right",
      monitorIndex: 1,
      bounds: { x: 1000, y: -100, width: 1200, height: 800 },
      role: "secondary" as const,
      active: true,
    };

    expect(hostToGlobalPoint(host, { x: 24, y: 48 })).toEqual({
      x: 1024,
      y: -52,
    });
    expect(globalToHostPoint(host, { x: 1024, y: -52 })).toEqual({
      x: 24,
      y: 48,
    });
  });

  it("clamps migrated windows into host bounds", () => {
    const host = { x: 0, y: 0, width: 1000, height: 700 };

    expect(
      clampRectToHost(host, { x: -80, y: 640, width: 420, height: 180 }),
    ).toEqual({
      x: 0,
      y: 520,
      width: 420,
      height: 180,
    });
    expect(
      clampRectToHost(host, { x: 10, y: 20, width: 1400, height: 900 }),
    ).toEqual({
      x: 0,
      y: 0,
      width: 1000,
      height: 700,
    });
  });

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

  it("moves a window into an inactive monitor host derived from monitor layout", () => {
    const hosts = buildDesktopHosts([
      {
        id: "main-monitor",
        ordinal: 0,
        workArea: { x: 0, y: 0, width: 1000, height: 800 },
      },
      {
        id: "right-monitor",
        ordinal: 1,
        workArea: { x: 1000, y: 0, width: 1000, height: 800 },
      },
    ]);
    const hostBounds = hosts.map((host) => ({
      runtimeLabel: host.hostLabel,
      monitorId: host.monitorId,
      bounds: host.bounds,
    }));

    const placement = resolveDesktopHostPlacement(
      hostBounds,
      "main",
      { hostLabel: "main", width: 300, height: 200 },
      920,
      120,
      shellHost,
    );

    expect(placement).toEqual({
      hostLabel: "forge-monitor-2",
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
