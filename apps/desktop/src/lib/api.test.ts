import { afterEach, describe, expect, it, vi } from "vitest";

import { api } from "./api";
import { clearForgeApiTokenCache } from "./api/client";

const tauriMocks = vi.hoisted(() => ({
  invoke: vi.fn<() => Promise<string | null>>(() => Promise.resolve(null)),
}));

vi.mock("@tauri-apps/api/core", () => ({
  invoke: tauriMocks.invoke,
}));

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

  it("reloads the Tauri API token after the token cache is cleared", async () => {
    clearForgeApiTokenCache();
    const fetchMock = vi
      .spyOn(globalThis, "fetch")
      .mockImplementation(() =>
        Promise.resolve(
          new Response(JSON.stringify({ ok: true }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        ),
      );
    tauriMocks.invoke
      .mockResolvedValueOnce("stale-token")
      .mockResolvedValueOnce(null);

    await api.health();
    clearForgeApiTokenCache();
    await api.health();

    const firstInit = fetchMock.mock.calls[0]?.[1] as RequestInit | undefined;
    const secondInit = fetchMock.mock.calls[1]?.[1] as RequestInit | undefined;
    expect(new Headers(firstInit?.headers).get("Authorization")).toBe(
      "Bearer stale-token",
    );
    expect(new Headers(secondInit?.headers).has("Authorization")).toBe(false);
    expect(tauriMocks.invoke).toHaveBeenCalledTimes(2);
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

  it("formats structured API errors instead of leaking raw JSON", async () => {
    vi.spyOn(globalThis, "fetch").mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          error: {
            code: "unauthorized",
            message: "missing or invalid bearer token",
          },
        }),
        {
          status: 401,
          statusText: "Unauthorized",
          headers: { "Content-Type": "application/json" },
        },
      ),
    );

    await expect(api.health()).rejects.toThrow(
      "401 unauthorized: missing or invalid bearer token",
    );
  });
});
