import { create } from "zustand";

import { api } from "../lib/api";

type CoreStatus = "unknown" | "online" | "offline";

type WorkspaceState = {
  core: CoreStatus;
  meta: { dataDir: string; dbPath: string; workspaceDir: string } | null;
  lastCoreError: string | null;
  ping: () => Promise<void>;
};

export const useWorkspaceStore = create<WorkspaceState>((set) => ({
  core: "unknown",
  meta: null,
  lastCoreError: null,
  ping: async () => {
    try {
      await api.health();
      const meta = await api.meta();
      set({ core: "online", meta, lastCoreError: null });
    } catch (e) {
      set({
        core: "offline",
        meta: null,
        lastCoreError: e instanceof Error ? e.message : String(e),
      });
    }
  },
}));
