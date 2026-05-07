import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, expect, it, vi } from "vitest";

import { CanvasPage } from "./CanvasPage";

const apiMocks = vi.hoisted(() => ({
  listBoards: vi.fn(),
  getBoard: vi.fn(),
  createBoard: vi.fn(),
  deleteBoard: vi.fn(),
  createNote: vi.fn(),
  patchNote: vi.fn(),
  deleteNote: vi.fn(),
}));

vi.mock("../lib/api", () => ({
  api: {
    canvas: {
      boards: {
        list: apiMocks.listBoards,
        get: apiMocks.getBoard,
        create: apiMocks.createBoard,
        delete: apiMocks.deleteBoard,
        createNote: apiMocks.createNote,
        patchNote: apiMocks.patchNote,
        deleteNote: apiMocks.deleteNote,
      },
    },
  },
}));

describe("CanvasPage", () => {
  it("opens an empty board when the API returns nullable notes", async () => {
    apiMocks.listBoards.mockResolvedValueOnce({
      boards: [
        {
          id: 7,
          title: "Empty planning board",
          createdAtMs: 1,
          updatedAtMs: 2,
        },
      ],
    });
    apiMocks.getBoard.mockResolvedValueOnce({
      id: 7,
      title: "Empty planning board",
      createdAtMs: 1,
      updatedAtMs: 2,
      notes: null,
    });

    render(
      <MemoryRouter>
        <CanvasPage />
      </MemoryRouter>,
    );

    expect(await screen.findByText("Empty board")).toBeTruthy();
    expect(screen.getByText("0 notes")).toBeTruthy();
  });
});
