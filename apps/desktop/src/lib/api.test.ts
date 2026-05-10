import { afterEach, describe, expect, it, vi } from "vitest";

import { api } from "./api";

describe("api client request bounds", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("attaches an abort signal to JSON requests", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ ok: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );

    await api.health();

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.signal).toBeInstanceOf(AbortSignal);
  });

  it("attaches an abort signal to attachment uploads", async () => {
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            artifact: {},
            bytes: 0,
            previewText: "",
          }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );

    await api.chat.threads.uploadAttachment(
      7,
      new File(["hello"], "hello.txt", { type: "text/plain" }),
    );

    const init = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    expect(init?.signal).toBeInstanceOf(AbortSignal);
  });

  it("aborts stalled JSON requests after the bounded timeout", async () => {
    vi.useFakeTimers();
    let signal: AbortSignal | undefined;
    vi.spyOn(globalThis, "fetch").mockImplementationOnce(((_input, init) => {
      signal = (init as RequestInit | undefined)?.signal ?? undefined;
      return new Promise<Response>((_resolve, reject) => {
        signal?.addEventListener("abort", () => {
          reject(new DOMException("aborted", "AbortError"));
        });
      });
    }) as typeof fetch);

    const request = api.health();
    const assertion = expect(request).rejects.toThrow(
      "FORGE API request timed out after 120000ms",
    );
    await vi.advanceTimersByTimeAsync(120_000);

    await assertion;
    expect(signal?.aborted).toBe(true);
  });
});
