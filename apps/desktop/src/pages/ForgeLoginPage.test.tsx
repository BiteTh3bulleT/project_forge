import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { ForgeLoginPage } from "./ForgeLoginPage";

describe("ForgeLoginPage accessibility feedback", () => {
  afterEach(() => {
    cleanup();
  });

  it("announces invalid operator login errors as alerts", () => {
    render(<ForgeLoginPage onUnlock={vi.fn()} />);

    fireEvent.change(screen.getByLabelText("Password"), {
      target: { value: "wrong-password" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Sign in" }));

    const alert = screen.getByRole("alert");
    expect(alert.textContent).toContain("Invalid local operator login.");
    expect(alert.getAttribute("aria-live")).toBe("assertive");
  });

  it("renders a distinct lock screen re-auth mode", () => {
    render(<ForgeLoginPage mode="lock" onUnlock={vi.fn()} />);

    expect(
      screen.getByRole("region", { name: "FORGE lock screen" }),
    ).toBeTruthy();
    expect(screen.getByText("Session Locked")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Unlock" })).toBeTruthy();
  });
});
