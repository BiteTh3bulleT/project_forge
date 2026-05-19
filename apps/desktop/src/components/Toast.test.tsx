import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Toast } from "./Toast";

describe("Toast", () => {
  it("uses status semantics for non-danger messages", () => {
    render(<Toast tone="warning">Heads up</Toast>);

    expect(screen.getByRole("status").textContent).toBe("Heads up");
  });

  it("uses alert semantics for danger messages", () => {
    render(<Toast tone="danger">Broken</Toast>);

    expect(screen.getByRole("alert").textContent).toBe("Broken");
  });
});
