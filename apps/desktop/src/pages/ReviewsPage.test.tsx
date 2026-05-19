import { render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ReviewsPage } from "./ReviewsPage";

const apiMocks = vi.hoisted(() => ({
  createReview: vi.fn(),
  listImports: vi.fn(),
  listReconciliations: vi.fn(),
  listReviews: vi.fn(),
  updateReview: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    imports: {
      list: apiMocks.listImports,
    },
    reconciliation: {
      list: apiMocks.listReconciliations,
    },
    reviews: {
      create: apiMocks.createReview,
      list: apiMocks.listReviews,
      update: apiMocks.updateReview,
    },
  },
}));

describe("ReviewsPage", () => {
  it("renders empty review and reconciliation states", async () => {
    apiMocks.listReviews.mockResolvedValueOnce({ reviews: [] });
    apiMocks.listImports.mockResolvedValueOnce({ imports: [] });
    apiMocks.listReconciliations.mockResolvedValueOnce({
      reconciliations: [],
    });

    render(<ReviewsPage />);

    expect(await screen.findByText("No matching reviews")).toBeTruthy();
    expect(screen.getByText("No reconciliation records")).toBeTruthy();
    expect(apiMocks.listReviews).toHaveBeenCalledWith({
      limit: 220,
      status: "pending",
    });
    expect(apiMocks.listImports).toHaveBeenCalledWith(180);
    expect(apiMocks.listReconciliations).toHaveBeenCalledWith({ limit: 180 });
  });
});
