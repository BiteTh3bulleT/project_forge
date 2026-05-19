import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { EventsPage } from "./EventsPage";

const mocks = vi.hoisted(() => ({
  events: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    events: mocks.events,
  },
}));

describe("EventsPage", () => {
  beforeEach(() => {
    mocks.events.mockReset();
  });

  it("renders an empty event stream when the API omits the events array", async () => {
    mocks.events.mockResolvedValue({});

    render(<EventsPage />);

    expect(await screen.findByText(/no events yet/i)).toBeTruthy();
  });
});
