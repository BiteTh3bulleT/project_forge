import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { EventsPage } from "./EventsPage";

const apiMocks = vi.hoisted(() => ({
  events: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    events: apiMocks.events,
  },
}));

describe("EventsPage", () => {
  it("renders events returned by the API", async () => {
    apiMocks.events.mockResolvedValueOnce({
      events: [
        {
          id: 1,
          type: "forge.test.event",
          createdAtMs: Date.UTC(2026, 4, 19, 12, 0, 0),
          payload: { status: "ok" },
        },
      ],
    });

    render(<EventsPage />);

    expect(await screen.findByText("forge.test.event")).toBeTruthy();
    expect(apiMocks.events).toHaveBeenCalledWith(200);
  });
});
