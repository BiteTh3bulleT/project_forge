import type { ImportReconciliation, ImportedExecution, ReviewRecord } from "@forge/shared";
import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";

import { api } from "../lib/api";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function parseLines(raw: string): string[] {
  return raw
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function lines(v: string[]): string {
  return v.join("\n");
}

export function ReviewsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [statusFilter, setStatusFilter] = useState("pending");
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);
  const [importsList, setImportsList] = useState<ImportedExecution[]>([]);
  const [reconciliations, setReconciliations] = useState<ImportReconciliation[]>([]);
  const [selectedImportId, setSelectedImportId] = useState<string>("");
  const [selectedReconciliation, setSelectedReconciliation] = useState<ImportReconciliation | null>(null);
  const [changedFiles, setChangedFiles] = useState("");
  const [failureReasons, setFailureReasons] = useState("");
  const [unresolvedIssues, setUnresolvedIssues] = useState("");
  const [nextSteps, setNextSteps] = useState("");
  const [agentNotes, setAgentNotes] = useState("");
  const [patchSummary, setPatchSummary] = useState("");
  const [reviewStatus, setReviewStatus] = useState("pending");
  const [manualTargetType, setManualTargetType] = useState("job");
  const [manualTargetId, setManualTargetId] = useState("");
  const [manualSummary, setManualSummary] = useState("");
  const [manualNotes, setManualNotes] = useState("");
  const [err, setErr] = useState<string | null>(null);

  async function load() {
    try {
      const [r, i, rec] = await Promise.all([
        api.reviews.list({ status: statusFilter === "all" ? "" : statusFilter, limit: 220 }),
        api.imports.list(180),
        api.reconciliation.list({ limit: 180 }),
      ]);
      setReviews(r.reviews);
      setImportsList(i.imports);
      setReconciliations(rec.reconciliations);
      setErr(null);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    }
  }

  useEffect(() => {
    void load();
  }, [statusFilter]);

  const importMap = useMemo(() => {
    const map = new Map<number, ImportedExecution>();
    importsList.forEach((i) => map.set(i.id, i));
    return map;
  }, [importsList]);

  async function loadReconciliation(importId: number) {
    try {
      const res = await api.reconciliation.getByImport(importId);
      setSelectedReconciliation(res.reconciliation);
      setChangedFiles(lines(res.reconciliation.changedFiles));
      setFailureReasons(lines(res.reconciliation.failureReasons));
      setUnresolvedIssues(lines(res.reconciliation.unresolvedIssues));
      setNextSteps(lines(res.reconciliation.suggestedNextSteps));
      setAgentNotes(res.reconciliation.agentNotes);
      setPatchSummary(res.reconciliation.patchSummary);
      setReviewStatus(res.reconciliation.reviewStatus || "pending");
    } catch {
      setSelectedReconciliation(null);
      setChangedFiles("");
      setFailureReasons("");
      setUnresolvedIssues("");
      setNextSteps("");
      setAgentNotes("");
      setPatchSummary("");
      setReviewStatus("pending");
    }
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Reviews"
        subtitle="Approval/reject/defer workflows for imported and generated outputs with persisted audit records."
        actions={<GhostButton onClick={() => void load()}>Refresh</GhostButton>}
      >
        {err ? <div className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">{err}</div> : null}
        <div className="grid gap-3 md:grid-cols-3">
          <div>
            <label className="text-xs font-semibold tracking-wide text-forge-mist">Status filter</label>
            <select className="forge-input mt-1" value={statusFilter} onChange={(e) => setStatusFilter(e.target.value)}>
              <option value="pending">pending</option>
              <option value="approved">approved</option>
              <option value="rejected">rejected</option>
              <option value="deferred">deferred</option>
              <option value="all">all</option>
            </select>
          </div>
          <div className="text-xs text-forge-mist md:col-span-2 md:self-end">Reviews are explicit operator decisions; no adapter can silently self-approve.</div>
        </div>
      </Panel>

      <div className="grid gap-6 xl:grid-cols-2">
        <Panel title="Review Queue" subtitle="Persisted review records with explicit status transitions.">
          {reviews.length === 0 ? (
            <div className="text-sm text-forge-mist">No reviews for current filter.</div>
          ) : (
            <div className="space-y-2">
              {reviews.map((r) => (
                <div key={r.id} className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                  <div className="font-semibold text-forge-ash">
                    #{r.id} · {r.targetType}:{r.targetId} · {r.status}
                  </div>
                  <div className="mt-1">{r.summary || "(no summary)"}</div>
                  <div className="mt-1">reviewer {r.reviewer} · {formatTime(r.updatedAtMs)}</div>
                  <div className="mt-2 flex flex-wrap gap-2">
                    <GhostButton
                      onClick={async () => {
                        await api.reviews.update(r.id, { status: "approved" });
                        setStatus(`Review #${r.id} approved.`);
                        await load();
                      }}
                    >
                      Approve
                    </GhostButton>
                    <GhostButton
                      onClick={async () => {
                        await api.reviews.update(r.id, { status: "rejected" });
                        setStatus(`Review #${r.id} rejected.`);
                        await load();
                      }}
                    >
                      Reject
                    </GhostButton>
                    <GhostButton
                      onClick={async () => {
                        await api.reviews.update(r.id, { status: "deferred" });
                        setStatus(`Review #${r.id} deferred.`);
                        await load();
                      }}
                    >
                      Defer
                    </GhostButton>
                  </div>
                </div>
              ))}
            </div>
          )}
        </Panel>

        <Panel title="Create Review" subtitle="Manual review record creation for jobs, imports, artifacts, or packets.">
          <div className="space-y-3">
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <label className="text-xs text-forge-mist">Target type</label>
                <select className="forge-input mt-1" value={manualTargetType} onChange={(e) => setManualTargetType(e.target.value)}>
                  <option value="job">job</option>
                  <option value="import">import</option>
                  <option value="artifact">artifact</option>
                  <option value="packet">packet</option>
                </select>
              </div>
              <div>
                <label className="text-xs text-forge-mist">Target id</label>
                <input className="forge-input mt-1" value={manualTargetId} onChange={(e) => setManualTargetId(e.target.value)} />
              </div>
            </div>
            <div>
              <label className="text-xs text-forge-mist">Summary</label>
              <input className="forge-input mt-1" value={manualSummary} onChange={(e) => setManualSummary(e.target.value)} />
            </div>
            <div>
              <label className="text-xs text-forge-mist">Notes</label>
              <textarea className="forge-input mt-1 min-h-[70px]" value={manualNotes} onChange={(e) => setManualNotes(e.target.value)} />
            </div>
            <PrimaryButton
              onClick={async () => {
                await api.reviews.create({
                  targetType: manualTargetType,
                  targetId: manualTargetId,
                  status: "pending",
                  summary: manualSummary,
                  notes: manualNotes,
                  reviewer: "operator",
                  annotations: [],
                });
                setStatus("Review record created.");
                setManualTargetId("");
                setManualSummary("");
                setManualNotes("");
                await load();
              }}
            >
              Create Review
            </PrimaryButton>
          </div>
        </Panel>
      </div>

      <Panel title="Import Reconciliation" subtitle="Store changed files, failure reasons, unresolved issues, and next steps for imported external execution.">
        <div className="grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs text-forge-mist">Imported run</label>
            <select
              className="forge-input mt-1"
              value={selectedImportId}
              onChange={(e) => {
                setSelectedImportId(e.target.value);
                const n = Number(e.target.value);
                if (Number.isFinite(n) && n > 0) {
                  void loadReconciliation(n);
                }
              }}
            >
              <option value="">Select import…</option>
              {importsList.map((i) => (
                <option key={i.id} value={String(i.id)}>
                  {i.id} - {i.adapterId} - {i.summary.slice(0, 60)}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="text-xs text-forge-mist">Review status</label>
            <select className="forge-input mt-1" value={reviewStatus} onChange={(e) => setReviewStatus(e.target.value)}>
              <option value="pending">pending</option>
              <option value="approved">approved</option>
              <option value="rejected">rejected</option>
              <option value="deferred">deferred</option>
            </select>
          </div>
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <TextAreaLines label="Changed files (one per line)" value={changedFiles} onChange={setChangedFiles} />
          <TextAreaLines label="Failure reasons" value={failureReasons} onChange={setFailureReasons} />
          <TextAreaLines label="Unresolved issues" value={unresolvedIssues} onChange={setUnresolvedIssues} />
          <TextAreaLines label="Suggested next steps" value={nextSteps} onChange={setNextSteps} />
        </div>

        <div className="mt-3 grid gap-3 md:grid-cols-2">
          <div>
            <label className="text-xs text-forge-mist">Agent notes</label>
            <textarea className="forge-input mt-1 min-h-[80px]" value={agentNotes} onChange={(e) => setAgentNotes(e.target.value)} />
          </div>
          <div>
            <label className="text-xs text-forge-mist">Patch summary</label>
            <textarea className="forge-input mt-1 min-h-[80px]" value={patchSummary} onChange={(e) => setPatchSummary(e.target.value)} />
          </div>
        </div>

        <div className="mt-3 flex gap-2">
          <PrimaryButton
            onClick={async () => {
              const importId = Number(selectedImportId);
              if (!Number.isFinite(importId) || importId <= 0) {
                setErr("Choose an import before saving reconciliation.");
                return;
              }
              await api.reconciliation.saveByImport(importId, {
                changedFiles: parseLines(changedFiles),
                failureReasons: parseLines(failureReasons),
                unresolvedIssues: parseLines(unresolvedIssues),
                suggestedNextSteps: parseLines(nextSteps),
                agentNotes,
                patchSummary,
                reviewStatus,
              });
              setStatus(`Reconciliation saved for import ${importId}.`);
              await load();
              await loadReconciliation(importId);
            }}
          >
            Save Reconciliation
          </PrimaryButton>
          <GhostButton onClick={() => void load()}>Reload Reconciliations</GhostButton>
        </div>

        {selectedImportId ? (
          <div className="mt-3 rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
            <div className="font-semibold text-forge-ash">Selected import context</div>
            <pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap rounded border border-forge-platinum/10 bg-black/30 p-2 text-[11px] text-forge-mist">
              {JSON.stringify(importMap.get(Number(selectedImportId)) ?? {}, null, 2)}
            </pre>
            {selectedReconciliation ? (
              <div className="mt-2 text-[11px]">Existing reconciliation #{selectedReconciliation.id} (updated {formatTime(selectedReconciliation.updatedAtMs)})</div>
            ) : (
              <div className="mt-2 text-[11px]">No existing reconciliation; saving will create one.</div>
            )}
          </div>
        ) : null}
      </Panel>

      <Panel title="Reconciliation History" subtitle="Persisted reconciliation records linked to imported executions.">
        {reconciliations.length === 0 ? (
          <div className="text-sm text-forge-mist">No reconciliation records yet.</div>
        ) : (
          <div className="space-y-2">
            {reconciliations.map((r) => (
              <div key={r.id} className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
                <div className="font-semibold text-forge-ash">
                  reconciliation #{r.id} · import {r.importId} · {r.reviewStatus}
                </div>
                <div className="mt-1">files {r.changedFiles.length} · unresolved {r.unresolvedIssues.length}</div>
                <div className="mt-1">{r.patchSummary || "(no patch summary)"}</div>
                <div className="mt-1">updated {formatTime(r.updatedAtMs)}</div>
              </div>
            ))}
          </div>
        )}
      </Panel>
    </div>
  );
}

function TextAreaLines(props: { label: string; value: string; onChange: (v: string) => void }) {
  return (
    <div>
      <label className="text-xs text-forge-mist">{props.label}</label>
      <textarea className="forge-input mt-1 min-h-[90px]" value={props.value} onChange={(e) => props.onChange(e.target.value)} />
    </div>
  );
}
