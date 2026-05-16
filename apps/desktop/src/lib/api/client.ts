import { invoke } from "@tauri-apps/api/core";

export const base = () =>
  import.meta.env.VITE_FORGE_API_URL ?? "http://127.0.0.1:18492";

const defaultRequestTimeoutMs = 120_000;

const requestTimeoutMs = () => {
  const raw = import.meta.env.VITE_FORGE_API_TIMEOUT_MS;
  const parsed = Number(raw);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return defaultRequestTimeoutMs;
  }
  return parsed;
};

let apiTokenPromise: Promise<string | undefined> | undefined;

async function forgeApiToken(): Promise<string | undefined> {
  apiTokenPromise ??= invoke<string | null>("read_forge_api_token")
    .then((token) => token?.trim() || undefined)
    .catch(() => undefined);
  return apiTokenPromise;
}

async function authenticatedHeaders(init?: RequestInit): Promise<Headers> {
  const headers = new Headers(init?.headers);
  if (!headers.has("Accept")) {
    headers.set("Accept", "application/json");
  }
  if (!headers.has("Authorization")) {
    const token = await forgeApiToken();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }
  }
  return headers;
}

export async function fetchWithTimeout(
  input: RequestInfo | URL,
  init?: RequestInit,
): Promise<Response> {
  const timeoutMs = requestTimeoutMs();
  const controller = new AbortController();
  const upstreamSignal = init?.signal;
  let timedOut = false;
  let timer: ReturnType<typeof setTimeout> | undefined;

  const abortFromUpstream = () => {
    controller.abort(upstreamSignal?.reason);
  };

  if (upstreamSignal?.aborted) {
    abortFromUpstream();
  } else {
    upstreamSignal?.addEventListener("abort", abortFromUpstream, {
      once: true,
    });
  }

  timer = setTimeout(() => {
    timedOut = true;
    controller.abort();
  }, timeoutMs);

  try {
    const headers = await authenticatedHeaders(init);
    return await fetch(input, {
      ...init,
      headers,
      signal: controller.signal,
    });
  } catch (err) {
    if (timedOut) {
      throw new Error(`FORGE API request timed out after ${timeoutMs}ms`);
    }
    throw err;
  } finally {
    if (timer) {
      clearTimeout(timer);
    }
    upstreamSignal?.removeEventListener("abort", abortFromUpstream);
  }
}

export async function j<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetchWithTimeout(`${base()}${path}`, {
    ...init,
  });
  if (!res.ok) {
    const t = await res.text().catch(() => "");
    throw new Error(t || `${res.status} ${res.statusText}`);
  }
  if (res.status === 204) {
    return undefined as T;
  }
  return (await res.json()) as T;
}
