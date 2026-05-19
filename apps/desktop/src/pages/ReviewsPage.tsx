import type {
  ImportReconciliation,
  ImportedExecution,
  ReviewRecord,
} from "@forge/shared";
import { GhostButton, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState, type ReactNode } from "react";

import { HumanDataView } from "../components/HumanDataView";
import { api } from "../lib/api";
import { arrayOrEmpty } from "../lib/arrays";
import { formatTime } from "../lib/format";
import { useUiStore } from "../stores/uiStore";

function parseLines(raw: string): string[] {
  return raw
    .split(/\r?\n/)
    .map((s) => s.trim())
    .filter(Boolean);
}

function lines(v: unknown): string {
  return arrayOrEmpty<string>(v).join("\n");
}

function normalizeReconciliation(
  reconciliation: ImportReconciliation,
): ImportReconciliation {
  return {
    ...reconciliation,
    changedFiles: arrayOrEmpty<string>(reconciliation.changedFiles),
    failureReasons: arrayOrEmpty<string>(reconciliation.failureReasons),
    unresolvedIssues: arrayOrEmpty<string>(reconciliation.unresolvedIssues),
    suggestedNextSteps: arrayOrEmpty<string>(reconciliation.suggestedNextSteps),
  };
}

export function ReviewsPage() {
  const setStatus = useUiStore((s) => s.setStatusLine);
  const [statusFilter, setStatusFilter] = useState("pending");
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);
  const [importsList, setImportsList] = useState<ImportedExecution[]>([]);
  const [reconciliations, setReconciliations] = useState<
    ImportReconciliation[]
  >([]);
  const [selectedImportId, setSelectedImportId] = useState<string>("");
  const [selectedReconciliation, setSelectedReconciliation] =
    useState<ImportReconciliation | null>(null);
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
        api.reviews.list({
          status: statusFilter === "all" ? "" : statusFilter,
          limit: 220,
        }),
        api.imports.list(180),
        api.reconciliation.list({ limit: 180 }),
      ]);
      setReviews(arrayOrEmpty<ReviewRecord>(r.reviews));
      setImportsList(arrayOrEmpty<ImportedExecution>(i.imports));
      setReconciliations(
        arrayOrEmpty<ImportReconciliation>(rec.reconciliations).map(
          normalizeReconciliation,
        ),
      );
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
      const reconciliation = normalizeReconciliation(res.reconciliation);
      setSelectedReconciliation(reconciliation);
      setChangedFiles(lines(reconciliation.changedFiles));
      setFailureReasons(lines(reconciliation.failureReasons));
      setUnresolvedIssues(lines(reconciliation.unresolvedIssues));
      setNextSteps(lines(reconciliation.suggestedNextSteps));
      setAgentNotes(reconciliation.agentNotes);
      setPatchSummary(reconciliation.patchSummary);
      setReviewStatus(reconciliation.reviewStatus || "pending");
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
    <div className="forge-ops-board space-y-5">
      <header className="flex flex-col gap-3 lg:flex-row lg:items-end lg:justify-between">
        <div>
          <div className="forge-ops-label">Governance Reviews</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Review command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            {reviews.length} review records loaded. Imported execution
            reconciliation remains operator-admitted and audit-backed.
          </p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className={statusPillClass(statusFilter)}>
            {statusFilter === "all" ? "all states" : statusFilter}
          </span>
          <GhostButton onClick={() => void load()}>Refresh</GhostButton>
        </div>
      </header>

      {err ? (
        <div className="forge-ops-panel border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash">
          {err}
        </div>
      ) : null}

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <MetricTile
          label="Reviews"
          value={String(reviews.length)}
          detail={`${statusFilter} filter`}
          tone="muted"
        />
        <MetricTile
          label="Imports"
          value={String(importsList.length)}
          detail="available runs"
          tone="muted"
        />
        <MetricTile
          label="Reconciliations"
          value={String(reconciliations.length)}
          detail="stored records"
          tone="ok"
        />
        <MetricTile
          label="Pending"
          value={String(reviews.filter((r) => r.status === "pending").length)}
          detail="operator decisions"
          tone={reviews.some((r) => r.status === "pending") ? "warn" : "ok"}
        />
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head flex-col items-stretch sm:flex-row sm:items-center">
          <div>
            <div className="forge-ops-title">Queue Controls</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Approval/reject/defer workflows for imported and generated
              outputs.
            </div>
          </div>
        </div>
        <div className="forge-ops-panel__body">
          <div className="grid gap-3 md:grid-cols-3">
            <div>
              <label className="forge-ops-label">Status filter</label>
              <select
                className="forge-input mt-1"
                value={statusFilter}
                onChange={(e) => setStatusFilter(e.target.value)}
              >
                <option value="pending">pending</option>
                <option value="approved">approved</option>
                <option value="rejected">rejected</option>
                <option value="deferred">deferred</option>
                <option value="all">all</option>
              </select>
            </div>
            <div className="text-xs text-forge-mist md:col-span-2 md:self-end">
              Reviews are explicit operator decisions; no adapter can silently
              self-approve.
            </div>
          </div>
        </div>
      </section>

      <div className="grid gap-4 xl:grid-cols-2">
        <OpsPanel
          title="Review Queue"
          subtitle="Persisted review records with explicit status transitions."
        >
          {reviews.length === 0 ? (
            <EmptyState
              title="No matching reviews"
              detail="Change the status filter or create a manual review record for an operator decision."
            />
          ) : (
            <div className="space-y-2">
              {reviews.map((r) => (
                <div
                  key={r.id}
                  className="forge-ops-card p-3 text-xs text-forge-mist"
                >
                  <div className="flex flex-wrap items-start justify-between gap-2">
                    <div className="font-semibold text-forge-ash">
                      #{r.id} - {r.targetType}:{r.targetId}
                    </div>
                    <span className={statusPillClass(r.status)}>
                      {r.status}
                    </span>
                  </div>
                  <div className="mt-1">{r.summary || "(no summary)"}</div>
                  <div className="mt-1">
                    reviewer {r.reviewer} · {formatTime(r.updatedAtMs)}
                  </div>
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
        </OpsPanel>

        <OpsPanel
          title="Create Review"
          subtitle="Manual review record creation for jobs, imports, artifacts, or packets."
        >
          <div className="forge-ops-card space-y-3 p-3">
            <div className="grid gap-3 md:grid-cols-2">
              <div>
                <label className="forge-ops-label">Target type</label>
                <select
                  className="forge-input mt-1"
                  value={manualTargetType}
                  onChange={(e) => setManualTargetType(e.target.value)}
                >
                  <option value="job">job</option>
                  <option value="import">import</option>
                  <option value="artifact">artifact</option>
                  <option value="packet">packet</option>
                </select>
              </div>
              <div>
                <label className="forge-ops-label">Target id</label>
                <input
                  className="forge-input mt-1"
                  value={manualTargetId}
                  onChange={(e) => setManualTargetId(e.target.value)}
                />
              </div>
            </div>
            <div>
              <label className="forge-ops-label">Summary</label>
              <input
                className="forge-input mt-1"
                value={manualSummary}
                onChange={(e) => setManualSummary(e.target.value)}
              />
            </div>
            <div>
              <label className="forge-ops-label">Notes</label>
              <textarea
                className="forge-input mt-1 min-h-[70px]"
                value={manualNotes}
                onChange={(e) => setManualNotes(e.target.value)}
              />
            </div>
            <PrimaryButton
              className="w-full sm:w-auto"
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
        </OpsPanel>
      </div>

      <OpsPanel
        title="Import Reconciliation"
        subtitle="Store changed files, failure reasons, unresolved issues, and next steps for imported external execution."
      >
        <div className="forge-ops-card p-3">
          <div className="grid gap-3 md:grid-cols-2">
            <div>
              <label className="forge-ops-label">Imported run</label>
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
              <label className="forge-ops-label">Review status</label>
              <select
                className="forge-input mt-1"
                value={reviewStatus}
                onChange={(e) => setReviewStatus(e.target.value)}
              >
                <option value="pending">pending</option>
                <option value="approved">approved</option>
                <option value="rejected">rejected</option>
                <option value="deferred">deferred</option>
              </select>
            </div>
          </div>

          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <TextAreaLines
              label="Changed files (one per line)"
              value={changedFiles}
              onChange={setChangedFiles}
            />
            <TextAreaLines
              label="Failure reasons"
              value={failureReasons}
              onChange={setFailureReasons}
            />
            <TextAreaLines
              label="Unresolved issues"
              value={unresolvedIssues}
              onChange={setUnresolvedIssues}
            />
            <TextAreaLines
              label="Suggested next steps"
              value={nextSteps}
              onChange={setNextSteps}
            />
          </div>

          <div className="mt-3 grid gap-3 md:grid-cols-2">
            <div>
              <label className="forge-ops-label">Agent notes</label>
              <textarea
                className="forge-input mt-1 min-h-[80px]"
                value={agentNotes}
                onChange={(e) => setAgentNotes(e.target.value)}
              />
            </div>
            <div>
              <label className="forge-ops-label">Patch summary</label>
              <textarea
                className="forge-input mt-1 min-h-[80px]"
                value={patchSummary}
                onChange={(e) => setPatchSummary(e.target.value)}
              />
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
            <GhostButton onClick={() => void load()}>
              Reload Reconciliations
            </GhostButton>
          </div>

          {selectedImportId ? (
            <div className="mt-3 rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
              <div className="font-semibold text-forge-ash">
                Selected import context
              </div>
              <div className="mt-2 max-h-48 overflow-auto rounded border border-white/10 bg-black/30 p-2 text-[11px] text-forge-mist">
                <HumanDataView
                  value={importMap.get(Number(selectedImportId)) ?? {}}
                  compact
                />
              </div>
              {selectedReconciliation ? (
                <div className="mt-2 text-[11px]">
                  Existing reconciliation #{selectedReconciliation.id} (updated{" "}
                  {formatTime(selectedReconciliation.updatedAtMs)})
                </div>
              ) : (
                <div className="mt-2 text-[11px]">
                  No existing reconciliation; saving will create one.
                </div>
              )}
            </div>
          ) : null}
        </div>
      </OpsPanel>

      <OpsPanel
        title="Reconciliation History"
        subtitle="Persisted reconciliation records linked to imported executions."
      >
        {reconciliations.length === 0 ? (
          <EmptyState
            title="No reconciliation records"
            detail="Select an imported execution and save reconciliation evidence to build this history."
          />
        ) : (
          <div className="space-y-2">
            {reconciliations.map((r) => (
              <div
                key={r.id}
                className="forge-ops-card p-3 text-xs text-forge-mist"
              >
                <div className="font-semibold text-forge-ash">
                  reconciliation #{r.id} · import {r.importId} ·{" "}
                  {r.reviewStatus}
                </div>
                <div className="mt-1">
                  files {r.changedFiles.length} · unresolved{" "}
                  {r.unresolvedIssues.length}
                </div>
                <div className="mt-1">
                  {r.patchSummary || "(no patch summary)"}
                </div>
                <div className="mt-1">updated {formatTime(r.updatedAtMs)}</div>
              </div>
            ))}
          </div>
        )}
      </OpsPanel>
    </div>
  );
}

function OpsPanel(props: {
  title: string;
  subtitle: string;
  children: ReactNode;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            {props.subtitle}
          </div>
        </div>
      </div>
      <div className="forge-ops-panel__body">{props.children}</div>
    </section>
  );
}

function MetricTile(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
}) {
  return (
    <div className="forge-ops-card p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        <span className={statusPillClass(props.tone)}>{props.tone}</span>
      </div>
      <div className="mt-3 text-xs text-forge-mist/65">{props.detail}</div>
    </div>
  );
}

function EmptyState(props: { title: string; detail: string }) {
  return (
    <div className="forge-ops-card border-dashed p-4 text-sm">
      <div className="font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-1 text-xs leading-5 text-forge-mist/70">
        {props.detail}
      </div>
    </div>
  );
}

function statusPillClass(status: string) {
  const normalized = status.toLowerCase();
  if (normalized === "approved" || normalized === "ok") {
    return "forge-ops-status forge-ops-status--ok";
  }
  if (normalized === "rejected" || normalized === "bad") {
    return "forge-ops-status forge-ops-status--bad";
  }
  if (
    normalized === "pending" ||
    normalized === "deferred" ||
    normalized === "warn"
  ) {
    return "forge-ops-status forge-ops-status--warn";
  }
  return "forge-ops-status forge-ops-status--muted";
}

function TextAreaLines(props: {
  label: string;
  value: string;
  onChange: (v: string) => void;
}) {
  return (
    <div>
      <label className="forge-ops-label">{props.label}</label>
      <textarea
        className="forge-input mt-1 min-h-[90px]"
        value={props.value}
        onChange={(e) => props.onChange(e.target.value)}
      />
    </div>
  );
}
