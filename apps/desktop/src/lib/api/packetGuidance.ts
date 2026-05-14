import { j } from "./client";
import type { PacketGuidance } from "@forge/shared";

export const packetGuidanceApi = {
  list: (params?: { limit?: number; packetId?: number }) => {
    const qs = new URLSearchParams();
    if (params?.limit != null) qs.set("limit", String(params.limit));
    if (params?.packetId != null) qs.set("packetId", String(params.packetId));
    const q = qs.toString();
    return j<{ guidance: PacketGuidance[] }>(
      `/api/packet-guidance${q ? `?${q}` : ""}`,
    );
  },
  analyze: (body: Record<string, unknown>) =>
    j<{ guidance: PacketGuidance }>("/api/packet-guidance/analyze", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    }),
};
