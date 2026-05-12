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
    return await fetch(input, {
      ...init,
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
    headers: {
      Accept: "application/json",
      ...(init?.headers ?? {}),
    },
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
