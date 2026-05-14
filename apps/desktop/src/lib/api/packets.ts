import { j } from "./client";
import type { TaskPacket } from "@forge/shared";

export const packetsApi = {
  get: (id: number) => j<TaskPacket>(`/api/packets/${id}`),
};
