import type { MonitorSnapshot } from "../../lib/desktop";

import { STORAGE_KEY, STORAGE_KEY_LEGACY } from "./constants";
import { emptyDoc, normalizeLayoutDoc } from "./model";
import type { LayoutDoc } from "./types";

function parseStoredDoc(raw: string | null): LayoutDoc | null {
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as LayoutDoc;
    if (!parsed || typeof parsed !== "object") return null;
    return parsed;
  } catch {
    return null;
  }
}

export function loadDoc(monitors: MonitorSnapshot[] = []): LayoutDoc {
  if (typeof window === "undefined") return emptyDoc();
  try {
    const latest = parseStoredDoc(window.localStorage.getItem(STORAGE_KEY));
    const legacy = latest
      ? null
      : parseStoredDoc(window.localStorage.getItem(STORAGE_KEY_LEGACY));
    return normalizeLayoutDoc(latest ?? legacy, monitors);
  } catch {
    return emptyDoc();
  }
}

export function persistDoc(doc: LayoutDoc) {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(STORAGE_KEY, JSON.stringify(doc));
}
