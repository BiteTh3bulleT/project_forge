export type ChatPaneLayout = {
  threadWidth: number;
  inspectorWidth: number;
  inspectorListHeight: number;
};

export const CHAT_PANE_LAYOUT_KEY = "forge.chatPaneLayout.v1";

export const DEFAULT_CHAT_PANE_LAYOUT: ChatPaneLayout = {
  threadWidth: 260,
  inspectorWidth: 420,
  inspectorListHeight: 240,
};

export function clampNumber(value: number, min: number, max: number) {
  if (!Number.isFinite(value)) return min;
  return Math.min(max, Math.max(min, Math.round(value)));
}

export function readStoredChatPaneLayout(): ChatPaneLayout {
  if (typeof window === "undefined") return DEFAULT_CHAT_PANE_LAYOUT;
  try {
    const raw = window.localStorage.getItem(CHAT_PANE_LAYOUT_KEY);
    if (!raw) return DEFAULT_CHAT_PANE_LAYOUT;
    const parsed = JSON.parse(raw) as Partial<ChatPaneLayout>;
    return {
      threadWidth: clampNumber(Number(parsed.threadWidth), 200, 420),
      inspectorWidth: clampNumber(Number(parsed.inspectorWidth), 300, 680),
      inspectorListHeight: clampNumber(
        Number(parsed.inspectorListHeight),
        160,
        520,
      ),
    };
  } catch {
    return DEFAULT_CHAT_PANE_LAYOUT;
  }
}
