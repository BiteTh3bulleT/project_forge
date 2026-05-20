import { describe, expect, it } from "vitest";

import { allShellTools, getShellTool, type ShellToolId } from "./shellConfig";
import { getToolComponent } from "./toolRegistry";

describe("shell tool registry", () => {
  it("mounts a component for every launchable shell tool", () => {
    const missing = allShellTools
      .filter((tool) => !getToolComponent(tool.id))
      .map((tool) => tool.id);

    expect(missing).toEqual([]);
  });

  it("keeps route-derived detail surfaces out of the launchable registry", () => {
    const launchableIds = new Set(allShellTools.map((tool) => tool.id));
    const detailIds: ShellToolId[] = [
      getShellTool("/jobs/job-123").id,
      getShellTool("/memory/chunk/chunk-123").id,
      getShellTool("/not-a-registered-surface").id,
    ];

    expect(detailIds).toEqual(["job-detail", "memory-detail", "other"]);
    for (const id of detailIds) {
      expect(launchableIds.has(id)).toBe(false);
      expect(getToolComponent(id)).toBeNull();
    }
  });
});
