import type { DesktopSystemDiagnostics } from "../../lib/desktop";

export type SettingsView =
  | "all"
  | "core"
  | "remote"
  | "retrieval"
  | "chat"
  | "display"
  | "diagnostics";

export type CoreMeta = {
  dataDir: string;
  dbPath: string;
  workspaceDir: string;
};

export type PcDiagnostics = {
  userAgent: string;
  platform: string;
  language: string;
  languages: string;
  cores: string;
  memoryGiB: string;
  screenWidth: number;
  screenHeight: number;
  availWidth: number;
  availHeight: number;
  colorDepth: number;
  pixelRatio: number;
  runtime: string;
  memoryUsedMB: string;
  memoryLimitMB: string;
  desktop: DesktopSystemDiagnostics | null;
};

export type MetricTone = "ok" | "warn" | "bad" | "muted";
