import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { KeyValueList } from "./KeyValueList";

describe("KeyValueList", () => {
  it("renders rows as a description list", () => {
    render(
      <KeyValueList
        rows={[
          { label: "Adapter", value: "gateway" },
          { label: "Boundary", value: "bounded" },
        ]}
      />,
    );

    expect(screen.getByText("Adapter")).toBeTruthy();
    expect(screen.getByText("gateway")).toBeTruthy();
    expect(screen.getByText("Boundary")).toBeTruthy();
    expect(screen.getByText("bounded")).toBeTruthy();
  });

  it("renders an optional empty state", () => {
    render(<KeyValueList rows={[]} empty="No rows" />);

    expect(screen.getByText("No rows")).toBeTruthy();
  });
});
