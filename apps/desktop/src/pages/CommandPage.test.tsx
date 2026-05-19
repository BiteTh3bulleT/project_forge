import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CommandPage } from "./CommandPage";

const apiMocks = vi.hoisted(() => ({
  createJob: vi.fn(),
  executeCommand: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    commands: {
      execute: apiMocks.executeCommand,
    },
    jobs: {
      create: apiMocks.createJob,
    },
  },
}));

describe("CommandPage", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

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

  it("shows a page error when a template job cannot be queued", async () => {
    apiMocks.createJob.mockRejectedValue(new Error("queue unavailable"));

    render(
      <MemoryRouter>
        <CommandPage />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Create packet" }));

    await waitFor(() => {
      expect(screen.getByText("queue unavailable")).toBeTruthy();
    });
  });
});
