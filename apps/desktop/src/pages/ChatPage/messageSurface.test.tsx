import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { MessageRow } from "./messageSurface";

describe("MessageRow traceability pivots", () => {
  it("links correlation and trace metadata directly to the Audit trace explorer", () => {
    render(
      <MemoryRouter>
        <MessageRow
          message={{
            id: 1,
            threadId: 1,
            role: "assistant",
            content: "Done.",
            createdAtMs: 1_800_000_000_000,
            metadata: {
              correlationId: "corr-chat-1",
              traceId: "trace-chat-1",
            },
          }}
          onInspectAttachment={vi.fn()}
        />
      </MemoryRouter>,
    );

    expect(
      screen.getByRole("link", { name: "Audit corr-chat-1" }).getAttribute("href"),
    ).toBe("/audit?correlationId=corr-chat-1");
    expect(
      screen.getByRole("link", { name: "Audit trace-chat-1" }).getAttribute("href"),
    ).toBe("/audit?traceId=trace-chat-1");
  });
});
