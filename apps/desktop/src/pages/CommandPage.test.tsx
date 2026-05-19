import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { CommandPage } from "./CommandPage";

const mocks = vi.hoisted(() => ({
  createJob: vi.fn(),
  executeCommand: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    jobs: {
      create: mocks.createJob,
    },
    commands: {
      execute: mocks.executeCommand,
    },
  },
}));

describe("CommandPage", () => {
  beforeEach(() => {
    mocks.createJob.mockReset();
    mocks.executeCommand.mockReset();
  });

  it("shows a page error when a template job cannot be queued", async () => {
    mocks.createJob.mockRejectedValue(new Error("queue unavailable"));

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
