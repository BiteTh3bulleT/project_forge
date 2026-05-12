import type { ChatAttachment } from "../../lib/api";

function asRecord(v: unknown): Record<string, unknown> | null {
  if (v && typeof v === "object" && !Array.isArray(v))
    return v as Record<string, unknown>;
  return null;
}

export function readJobId(
  meta: Record<string, unknown> | undefined,
): string | null {
  if (!meta) return null;
  const raw = meta.jobId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

export function readCorrelationId(
  meta: Record<string, unknown> | undefined,
): string | null {
  if (!meta) return null;
  const raw = meta.correlationId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

export function readTraceId(
  meta: Record<string, unknown> | undefined,
): string | null {
  if (!meta) return null;
  const raw = meta.traceId;
  if (typeof raw === "string" && raw.trim()) return raw.trim();
  return null;
}

export function readAttachments(
  meta: Record<string, unknown> | undefined,
): ChatAttachment[] {
  if (!meta) return [];
  const raw = meta.attachments;
  if (!Array.isArray(raw)) return [];
  const out: ChatAttachment[] = [];
  for (const item of raw) {
    const rec = asRecord(item);
    if (!rec) continue;
    const idRaw = rec.artifactId;
    const titleRaw = rec.title;
    const mimeRaw = rec.mimeType;
    const fileNameRaw = rec.fileName;
    const id = Number(idRaw);
    if (!Number.isFinite(id) || id <= 0) continue;
    const title =
      typeof titleRaw === "string" && titleRaw.trim()
        ? titleRaw.trim()
        : `Attachment #${id}`;
    const mimeType =
      typeof mimeRaw === "string" && mimeRaw.trim()
        ? mimeRaw.trim()
        : "application/octet-stream";
    const fileName =
      typeof fileNameRaw === "string" && fileNameRaw.trim()
        ? fileNameRaw.trim()
        : title;
    const textPreview =
      typeof rec.textPreview === "string" ? rec.textPreview : undefined;
    out.push({ artifactId: id, title, mimeType, fileName, textPreview });
  }
  return out;
}
