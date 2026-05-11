import { describe, expect, it } from "vitest";

import { allShellTools, getShellTool } from "./shellConfig";

describe("operator apps shell tool", () => {
  it("is discoverable from the shell tool list and route lookup", () => {
    const tool = allShellTools.find((candidate) => candidate.id === "operator-apps");

    expect(tool).toMatchObject({
      id: "operator-apps",
      label: "Operator Apps",
      route: "/operator-apps",
      primary: false,
    });
    expect(getShellTool("/operator-apps")).toEqual(tool);
  });
});
