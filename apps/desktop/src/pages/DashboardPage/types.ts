import type { api } from "../../lib/api";

export type CapabilityRecord = Awaited<
  ReturnType<typeof api.gateway.capabilities>
>["capabilities"][number];

export type InvocationRecord = Awaited<
  ReturnType<typeof api.gateway.invocations>
>["invocations"][number];

export type RunRow = {
  id: string;
  title: string;
  status: string;
  targetAdapter: string;
  createdAtMs: number;
  kind: "active" | "failed";
};
