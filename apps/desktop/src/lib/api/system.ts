import { j } from "./client";
import type { ForgeHealth, ForgeSystemStatus } from "./types";

export const systemApi = {
  status: () => j<ForgeSystemStatus>("/forge/system/status"),
  kernelStatus: () =>
    j<NonNullable<ForgeSystemStatus["kernel_activation"]>>(
      "/forge/kernel/status",
    ),
};

export const healthApi = {
  health: () => j<ForgeHealth>("/health"),
  meta: () =>
    j<{ dataDir: string; dbPath: string; workspaceDir: string }>("/api/meta"),
};
