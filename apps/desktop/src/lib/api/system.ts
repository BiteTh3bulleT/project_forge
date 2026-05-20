import { j } from "./client";
import type { ForgeHealth, ForgeSystemHost, ForgeSystemStatus } from "./types";

export const systemApi = {
  status: () => j<ForgeSystemStatus>("/forge/system/status"),
  host: () => j<ForgeSystemHost>("/forge/system/host"),
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
