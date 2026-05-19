import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AsyncState } from "./AsyncState";
import { OpsPanel } from "./OpsPanel";

describe("OpsPanel", () => {
  it("renders title, subtitle, actions, and body content", () => {
    render(
      <OpsPanel
        title="Panel title"
        subtitle="Panel subtitle"
        actions={<button type="button">Refresh</button>}
      >
        Panel body
      </OpsPanel>,
    );

    expect(screen.getByText("Panel title")).toBeTruthy();
    expect(screen.getByText("Panel subtitle")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Refresh" })).toBeTruthy();
    expect(screen.getByText("Panel body")).toBeTruthy();
  });
});

describe("AsyncState", () => {
  it("prioritizes errors over loading and empty states", () => {
    render(
      <AsyncState error="Load failed" loading empty>
        Ready
      </AsyncState>,
    );

    expect(screen.getByRole("alert").textContent).toContain("Load failed");
    expect(screen.queryByText("Ready")).toBeNull();
  });

  it("renders empty state details when requested", () => {
    render(
      <AsyncState
        empty
        emptyTitle="Nothing recorded"
        emptyDetail="Create a record first."
      />,
    );

    expect(screen.getByText("Nothing recorded")).toBeTruthy();
    expect(screen.getByText("Create a record first.")).toBeTruthy();
  });
});
