import "@testing-library/jest-dom/vitest";
import React from "react";
import { vi } from "vitest";

vi.mock("react-router-dom", async (importOriginal) => {
  const actual = await importOriginal<typeof import("react-router-dom")>();
  return {
    ...actual,
    MemoryRouter: (
      props: React.ComponentProps<typeof actual.MemoryRouter>,
    ) =>
      React.createElement(actual.MemoryRouter, {
        ...props,
        future: {
          v7_relativeSplatPath: true,
          v7_startTransition: true,
          ...(props.future ?? {}),
        },
      }),
  };
});
