import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useState } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

type Bundle = {
  id: number;
  createdAtMs: number;
  kind: string;
  label: string;
  versionTag: string;
  filePath: string;
  sizeBytes: number;
  sha256: string;
  notes: string;
};

export function BackupPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [bundles, setBundles] = useState<Bundle[]>([]);
  const [dirs, setDirs] = useState<{
    backupDir: string;
    exportDir: string;
    kinds: string[];
  }>({ backupDir: "", exportDir: "", kinds: [] });
  const [kind, setKind] = useState("portable_snapshot");
  const [label, setLabel] = useState("");
  const [versionTag, setVersionTag] = useState("");
  const [restorePath, setRestorePath] = useState("");
  const [restoreApprovalId, setRestoreApprovalId] = useState("");
  const [restoreDry, setRestoreDry] = useState(true);
  const [restoreResult, setRestoreResult] = useState<Record<
    string,
    unknown
  > | null>(null);
  const [err, setErr] = useState<string | null>(null);

  async function refresh() {
    try {
      const r = await api.backup.bundles(80);
      setBundles(r.bundles as Bundle[]);
      setDirs({
        backupDir: r.backupDir,
        exportDir: r.exportDir,
        kinds: r.knownKinds,
      });
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void refresh();
  }, []);

  return (
    <div className="forge-ops-board space-y-5">
      <Panel
        title="Backup & export"
        subtitle="Portable bundles, not raw sqlite dumps. Version tags are operator-owned."
        actions={
          <GhostButton onClick={() => void refresh()}>Refresh</GhostButton>
        }
      >
        {err ? (
          <div className="rounded-md border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
            {err}
          </div>
        ) : null}
        <div className="mt-3 space-y-1 text-xs text-forge-mist">
          <div>
            Backups dir:{" "}
            <span className="font-mono text-forge-ash">
              {dirs.backupDir || "—"}
            </span>
          </div>
          <div>
            Exports dir:{" "}
            <span className="font-mono text-forge-ash">
              {dirs.exportDir || "—"}
            </span>
          </div>
        </div>
      </Panel>

      <Panel
        title="Create bundle"
        subtitle="Kinds are validated server-side against KnownKinds."
      >
        <label className="text-xs text-forge-mist">
          Kind
          <select
            className="forge-input mt-1"
            value={kind}
            onChange={(e) => setKind(e.target.value)}
          >
            {dirs.kinds.map((k) => (
              <option key={k} value={k}>
                {k}
              </option>
            ))}
          </select>
        </label>
        <label className="mt-3 block text-xs text-forge-mist">
          Label
          <input
            className="forge-input mt-1"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder="weekly snapshot"
          />
        </label>
        <label className="mt-3 block text-xs text-forge-mist">
          Version tag
          <input
            className="forge-input mt-1"
            value={versionTag}
            onChange={(e) => setVersionTag(e.target.value)}
            placeholder="v0.5.0-2026-04-15"
          />
        </label>
        <div className="mt-4">
          <PrimaryButton
            onClick={async () => {
              const r = await api.backup.createBundle({
                kind,
                label,
                versionTag,
                notes: "",
                sourceVersion: "",
              });
              setStatus("Bundle created.");
              setRestorePath(
                String((r.bundle as Record<string, unknown>).filePath ?? ""),
              );
              await refresh();
            }}
          >
            Create
          </PrimaryButton>
        </div>
      </Panel>

      <Panel
        title="Restore bundle"
        subtitle="Conservative merge. Always dry-run against an unknown file first."
      >
        <input
          className="forge-input"
          value={restorePath}
          onChange={(e) => setRestorePath(e.target.value)}
          placeholder="absolute path to backup bundle"
        />
        <label className="mt-3 flex items-center gap-2 text-xs text-forge-mist">
          <input
            type="checkbox"
            checked={restoreDry}
            onChange={(e) => setRestoreDry(e.target.checked)}
          />
          Dry run
        </label>
        {!restoreDry ? (
          <label className="mt-3 block text-xs text-forge-mist">
            Approval ID
            <input
              className="forge-input mt-1"
              value={restoreApprovalId}
              onChange={(e) => setRestoreApprovalId(e.target.value)}
              placeholder="required after approval"
            />
          </label>
        ) : null}
        <div className="mt-3">
          <PrimaryButton
            onClick={async () => {
              const body: Record<string, unknown> = {
                filePath: restorePath,
                sections: [],
                dryRun: restoreDry,
              };
              if (!restoreDry && restoreApprovalId.trim()) {
                body.approvalId = restoreApprovalId.trim();
              }
              const r = await api.backup.restore(body);
              if (r.governance) {
                setRestoreResult(r.governance);
                const approvalId = String(
                  r.governance.approvalRequestId ?? "",
                );
                if (approvalId) setRestoreApprovalId(approvalId);
                setStatus(
                  approvalId
                    ? `Restore approval required (#${approvalId}).`
                    : "Restore approval required.",
                );
                return;
              }
              setRestoreResult(r.result ?? null);
              setStatus("Restore attempted (see result).");
            }}
          >
            Restore
          </PrimaryButton>
        </div>
        {restoreResult ? (
          <div className="mt-4 max-h-64 overflow-auto rounded border border-white/10 bg-black/30 p-3 text-[11px] text-forge-mist">
            <HumanDataView value={restoreResult} />
          </div>
        ) : null}
      </Panel>

      <Panel
        title="Bundles on disk"
        subtitle="Rows from backup_bundles; delete removes the catalog row (file delete is best-effort server-side)."
      >
        <div className="space-y-2">
          {bundles.map((b) => (
            <div
              key={b.id}
              className="flex flex-wrap items-start justify-between gap-2 rounded border border-white/10 bg-forge-slate/20 p-3 text-xs text-forge-mist"
            >
              <div>
                <div className="font-mono text-forge-ash">
                  {b.kind} · {b.label || "—"}
                </div>
                <div className="mt-1">{formatTime(b.createdAtMs)}</div>
                <div className="mt-1 max-w-[640px] break-all font-mono text-[10px]">
                  {b.filePath}
                </div>
              </div>
              <GhostButton
                onClick={async () => {
                  await api.backup.deleteBundle(b.id);
                  setStatus(`Deleted bundle record ${b.id}`);
                  await refresh();
                }}
              >
                Delete record
              </GhostButton>
            </div>
          ))}
        </div>
      </Panel>
    </div>
  );
}
