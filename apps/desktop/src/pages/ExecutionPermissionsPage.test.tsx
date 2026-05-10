import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { ExecutionPermissionsPage, type PermissionProfile } from "./ExecutionPermissionsPage";

const mocks = vi.hoisted(() => ({
  profiles: vi.fn(),
  saveProfile: vi.fn(),
  activateProfile: vi.fn(),
  deleteProfile: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    executionPermissions: {
      profiles: mocks.profiles,
      saveProfile: mocks.saveProfile,
      activateProfile: mocks.activateProfile,
      deleteProfile: mocks.deleteProfile,
    },
  },
}));

const editableProfile: PermissionProfile = {
  id: "operator",
  createdAtMs: 1,
  updatedAtMs: 2,
  name: "Operator",
  description: "Operator profile",
  allowedReadPaths: ["/workspace"],
  allowedWritePaths: ["/workspace"],
  allowedExecutePaths: ["/workspace"],
  forbiddenPaths: [],
  allowedTools: ["fs.read"],
  approvalRequiredRisks: ["dangerous"],
  maxBytesPerWrite: 512,
  allowNetwork: true,
  editable: true,
  active: true,
};

describe("ExecutionPermissionsPage stale authority state", () => {
  beforeEach(() => {
    for (const mock of Object.values(mocks)) {
      mock.mockReset();
    }
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  it("renders authority unavailable on initial load failure and disables mutations", async () => {
    mocks.profiles.mockRejectedValue(new Error("core offline"));

    render(
      <MemoryRouter>
        <ExecutionPermissionsPage />
      </MemoryRouter>,
    );

    expect(
      (await screen.findAllByText(/permission authority unavailable/i)).length,
    ).toBeGreaterThan(0);
    expect(screen.queryByText(/no active profile/i)).toBeNull();
    expect(
      screen.getByRole<HTMLButtonElement>("button", { name: "Save profile" })
        .disabled,
    ).toBe(true);
  });

  it("keeps stale profiles visible but disables authority mutations after refresh failure", async () => {
    mocks.profiles
      .mockResolvedValueOnce({
        profiles: [editableProfile],
        active: editableProfile,
        summary: { profileCount: 1 },
      })
      .mockRejectedValueOnce(new Error("permission endpoint down"));

    render(
      <MemoryRouter>
        <ExecutionPermissionsPage />
      </MemoryRouter>,
    );

    expect((await screen.findAllByText("operator")).length).toBeGreaterThan(0);
    fireEvent.click(screen.getByRole("button", { name: "Refresh" }));

    expect(await screen.findByText(/permission authority stale/i)).toBeTruthy();
    expect(
      screen.getByRole<HTMLButtonElement>("button", { name: "Save profile" })
        .disabled,
    ).toBe(true);
    expect(
      screen.getByRole<HTMLButtonElement>("button", { name: "Activate" })
        .disabled,
    ).toBe(true);
    expect(
      screen.getByRole<HTMLButtonElement>("button", { name: "Delete" })
        .disabled,
    ).toBe(true);

    fireEvent.click(screen.getByRole("button", { name: "Activate" }));
    fireEvent.click(screen.getByRole("button", { name: "Save profile" }));
    expect(mocks.activateProfile).not.toHaveBeenCalled();
    expect(mocks.saveProfile).not.toHaveBeenCalled();

    await waitFor(() => expect(mocks.profiles).toHaveBeenCalledTimes(2));
  });
});
