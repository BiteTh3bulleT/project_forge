import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter, useLocation } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useUiStore } from "../stores/uiStore";
import { CommandBar } from "./CommandBar";

const apiMocks = vi.hoisted(() => ({
  commandExecute: vi.fn(),
  createRetrievalRun: vi.fn(),
  health: vi.fn(),
  meta: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    commands: {
      execute: apiMocks.commandExecute,
    },
    health: apiMocks.health,
    meta: apiMocks.meta,
    retrieval: {
      createRun: apiMocks.createRetrievalRun,
    },
  },
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location">{location.pathname}</div>;
}

describe("CommandBar", () => {
  beforeEach(() => {
    apiMocks.commandExecute.mockReset();
    apiMocks.createRetrievalRun.mockReset();
    apiMocks.health.mockResolvedValue({ ok: true });
    apiMocks.meta.mockResolvedValue({
      dataDir: "",
      dbPath: "",
      workspaceDir: "",
    });
    useUiStore.setState({
      commandDraft: "",
      statusLine: "Workshop idle.",
      uiMode: "cognitive",
    });
  });

  it("stages visible command actions into the command input", () => {
    render(
      <MemoryRouter>
        <CommandBar />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Use command Jobs" }));

    expect(
      (screen.getByLabelText("Command input") as HTMLInputElement).value,
    ).toBe("go /jobs");
    expect(screen.getByText("Actions")).toBeTruthy();
  });

  it("runs staged navigation commands through the command parser", async () => {
    apiMocks.commandExecute.mockResolvedValueOnce({ jobId: "" });

    render(
      <MemoryRouter initialEntries={["/start"]}>
        <LocationProbe />
        <CommandBar />
      </MemoryRouter>,
    );

    fireEvent.click(screen.getByRole("button", { name: "Use command Jobs" }));
    fireEvent.click(screen.getByRole("button", { name: "Run" }));

    await waitFor(() =>
      expect(screen.getByTestId("location").textContent).toBe("/jobs"),
    );
    expect(apiMocks.commandExecute).toHaveBeenCalledWith("navigate", {
      path: "/jobs",
    });
    expect(
      (screen.getByLabelText("Command input") as HTMLInputElement).value,
    ).toBe("");
  });
});
