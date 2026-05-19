import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { CommandPage } from "./CommandPage";

vi.mock("../lib/api", () => ({
  api: {
    commands: {
      execute: vi.fn(),
    },
    jobs: {
      create: vi.fn(),
    },
  },
}));

describe("CommandPage", () => {
  it("renders command shortcuts and template job launchers", () => {
    render(
      <MemoryRouter>
        <CommandPage />
      </MemoryRouter>,
    );

    expect(screen.getByText("Command Desk")).toBeTruthy();
    expect(screen.getByText("Chat")).toBeTruthy();
    expect(screen.getByText("Workbench")).toBeTruthy();
    expect(screen.getByRole("button", { name: "Create packet" })).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Prepare Codex handoff" }),
    ).toBeTruthy();
  });
});
