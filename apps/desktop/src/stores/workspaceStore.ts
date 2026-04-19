import { create } from "zustand";

import { api } from "../lib/api";

type CoreStatus = "unknown" | "online" | "offline";

type WorkspaceState = {
  core: CoreStatus;
  meta: { dataDir: string; dbPath: string; workspaceDir: string } | null;
  lastCoreError: string | null;
  ping: () => Promise<void>;
};

function withTimeout<T>(promise: Promise<T>, ms: number): Promise<T> {
  let timeoutId: ReturnType<typeof setTimeout> | undefined;
  const timeoutPromise = new Promise<never>((_, reject) => {
    timeoutId = setTimeout(() => {
      reject(new Error(`request timed out after ${ms}ms`));
    }, ms);
  });
  return Promise.race([
    promise,
    timeoutPromise,
  ]).finally(() => clearTimeout(timeoutId));
}

export const useWorkspaceStore = create<WorkspaceState>((set, get) => ({
  core: "unknown",
  meta: null,
  lastCoreError: null,
  ping: async () => {
    try {
      await withTimeout(api.health(), 2500);
      try {
        const meta = await withTimeout(api.meta(), 1500);
        set({ core: "online", meta, lastCoreError: null });
        return;
      } catch (metaError) {
        set({
          core: "online",
          meta: get().meta,
          lastCoreError: metaError instanceof Error ? `metadata degraded: ${metaError.message}` : `metadata degraded: ${String(metaError)}`,
        });
        return;
      }
    } catch (e) {
      set({
        core: "offline",
        meta: null,
        lastCoreError: e instanceof Error ? e.message : String(e),
      });
    }
  },
}));
