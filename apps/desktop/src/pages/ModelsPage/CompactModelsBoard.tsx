import { GhostButton, PrimaryButton } from "@forge/ui";

import type {
  ModelRuntimeHealth,
  ModelRuntimeLoadedStatus,
  ModelRuntimeModel,
  ModelRuntimeQueueStatus,
  ModelRuntimeUsageSummary,
} from "../../lib/api";
import {
  badgeClass,
  cx,
  EmptyState,
  StateBox,
} from "./shared";

type LoadedModelRecord = ModelRuntimeLoadedStatus["models"][number];
type CompactPendingApproval = {
  modelId: string;
  action: string;
  approvalRequestId: number;
};

export function CompactModelsBoard(props: {
  err: string | null;
  health: ModelRuntimeHealth | null;
  runtimeAvailable: boolean;
  loading: boolean;
  loaded: ModelRuntimeLoadedStatus | null;
  queue: ModelRuntimeQueueStatus | null;
  chatSelectableModels: ModelRuntimeModel[];
  selectedModelSummary: ModelRuntimeModel | null;
  selectedLoadedRecord: LoadedModelRecord | null;
  modelCounts: Record<string, number>;
  usage: ModelRuntimeUsageSummary | null;
  chatSelectedModelId: string;
  models: ModelRuntimeModel[];
  selectedModelId: string;
  actionBusy: string | null;
  runtimeControlsDisabled: boolean;
  approvalBusy: boolean;
  pendingApproval: CompactPendingApproval | null;
  onRefresh: () => void;
  onOpenRegistry: () => void;
  onOpenApprovals: () => void;
  onApprovePending: () => void;
  onSelectModel: (modelId: string) => void;
  onSelectChatModel: (modelId: string) => void;
  onModelAction: (modelId: string, action: "load" | "unload") => void;
  setStatus: (status: string) => void;
}) {
  const selectedModelBusy =
    props.selectedModelSummary && props.actionBusy?.endsWith(
      `:${props.selectedModelSummary.id}`,
    );
  const selectedModelArchived =
    props.selectedModelSummary?.status?.toLowerCase().trim() === "archived";

  return (
    <div className="forge-ops-board space-y-5">
      <header className="rounded border border-forge-platinum/10 bg-black/20 p-4 lg:flex lg:items-end lg:justify-between">
        <div className="min-w-0">
          <div className="forge-ops-label">Model Runtime</div>
          <h1 className="mt-2 text-2xl font-semibold tracking-normal text-forge-ash sm:text-3xl">
            Runtime command board
          </h1>
          <p className="mt-1 max-w-3xl text-sm leading-6 text-forge-mist/75">
            Cognitive model view with runtime status, active model,
            availability, chat preference, and selected model lifecycle
            controls.
          </p>
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2 lg:mt-0 lg:justify-end">
          <span
            className={badgeClass(
              props.health?.ok === false || !props.runtimeAvailable
                ? "error"
                : "ok",
            )}
          >
            {props.runtimeAvailable
              ? props.health?.status || "runtime"
              : "unavailable"}
          </span>
          <GhostButton onClick={props.onRefresh} disabled={props.loading}>
            {props.loading ? "Refreshing" : "Refresh"}
          </GhostButton>
          <GhostButton onClick={props.onOpenRegistry}>
            Open Registry
          </GhostButton>
        </div>
      </header>

      <section className="forge-ops-panel">
        {props.err ? (
          <div className="m-3">
            <EmptyState
              title="Runtime request failed"
              detail={props.err}
              tone="bad"
            />
          </div>
        ) : null}
        {!props.runtimeAvailable ? (
          <div className="m-3">
            <EmptyState
              title="Model runtime unavailable"
              detail="Core UI remains healthy; enable modelruntime before registry and load controls become active."
              tone="warn"
            />
          </div>
        ) : null}
        <div className="forge-ops-panel__body grid gap-3 lg:grid-cols-[minmax(0,1fr)_18rem]">
          <div className="rounded border border-forge-platinum/10 bg-black/20 p-4">
            <div className="forge-ops-label">Runtime Status</div>
            <div className="mt-2 text-2xl font-semibold text-forge-ash">
              {props.health?.status || (props.health?.ok ? "ok" : "unknown")}
            </div>
            <div className="mt-2 text-sm text-forge-mist">
              {props.loaded?.count ?? 0} loaded · {props.queue?.depth ?? 0}{" "}
              queued · {props.chatSelectableModels.length} chat-capable
            </div>
            {props.health?.degradedReasons?.length ? (
              <div className="mt-3 rounded-md border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs leading-5 text-forge-ash">
                {props.health.degradedReasons.join("; ")}
              </div>
            ) : null}
            {props.health?.policyWarnings?.length ? (
              <div className="mt-2 rounded-md border border-white/10 bg-black/25 p-2 text-xs leading-5 text-forge-mist">
                {props.health.policyWarnings.join("; ")}
              </div>
            ) : null}
            <div className="mt-4 grid gap-2 md:grid-cols-2">
              <StateBox
                title="Active Model"
                rows={[
                  [
                    "Selected",
                    props.selectedModelSummary?.displayName ||
                      props.selectedModelSummary?.id ||
                      "none",
                  ],
                  [
                    "Backend",
                    props.selectedModelSummary?.backend ||
                      props.health?.backend ||
                      "unknown",
                  ],
                  [
                    "Loaded",
                    props.selectedLoadedRecord?.status || "not loaded",
                  ],
                ]}
              />
              <StateBox
                title="Availability"
                rows={[
                  ["Registered", String(props.models.length)],
                  [
                    "Available",
                    String(
                      props.modelCounts.available ?? props.usage?.available ?? 0,
                    ),
                  ],
                  [
                    "Disabled/Archived",
                    String(
                      (props.usage?.disabled ?? 0) +
                        (props.usage?.archived ?? 0),
                    ),
                  ],
                ]}
              />
            </div>
            {props.selectedModelSummary ? (
              <div className="mt-4 flex flex-wrap gap-2">
                {props.selectedLoadedRecord ? (
                  <PrimaryButton
                    className="min-h-10 px-3"
                    onClick={() =>
                      props.onModelAction(
                        props.selectedModelSummary?.id ?? "",
                        "unload",
                      )
                    }
                    disabled={
                      Boolean(selectedModelBusy) ||
                      props.runtimeControlsDisabled
                    }
                  >
                    {props.actionBusy ===
                    `unload:${props.selectedModelSummary.id}`
                      ? "Unloading..."
                      : "Unload"}
                  </PrimaryButton>
                ) : (
                  <PrimaryButton
                    className="min-h-10 px-3"
                    onClick={() =>
                      props.onModelAction(
                        props.selectedModelSummary?.id ?? "",
                        "load",
                      )
                    }
                    disabled={
                      Boolean(selectedModelBusy) ||
                      props.runtimeControlsDisabled ||
                      selectedModelArchived
                    }
                  >
                    {props.actionBusy === `load:${props.selectedModelSummary.id}`
                      ? "Loading..."
                      : "Load"}
                  </PrimaryButton>
                )}
                <GhostButton onClick={props.onOpenRegistry}>
                  Registry
                </GhostButton>
              </div>
            ) : null}
            {props.pendingApproval ? (
              <div className="mt-3 rounded border border-forge-amber/30 bg-forge-amber/10 p-3 text-xs leading-5 text-forge-ash">
                <div className="font-semibold">
                  Approval required #{props.pendingApproval.approvalRequestId}
                </div>
                <div className="mt-1 text-forge-mist">
                  Approve the request, then run {props.pendingApproval.action}{" "}
                  again for this model.
                </div>
                <GhostButton
                  className="mt-3"
                  onClick={props.onOpenApprovals}
                >
                  Open approvals
                </GhostButton>
                <PrimaryButton
                  className="ml-2 mt-3"
                  onClick={props.onApprovePending}
                  disabled={props.approvalBusy}
                >
                  {props.approvalBusy
                    ? "Approving..."
                    : `Approve and ${props.pendingApproval.action}`}
                </PrimaryButton>
              </div>
            ) : null}
          </div>
          <div className="rounded border border-forge-platinum/10 bg-black/20 p-4">
            <div className="forge-ops-label">Chat Preference</div>
            <select
              className="forge-input mt-3"
              value={props.chatSelectedModelId}
              onChange={(event) => {
                props.onSelectChatModel(event.target.value);
                props.setStatus(
                  event.target.value
                    ? `Chat model preference set to ${event.target.value}`
                    : "Chat model preference cleared (auto)",
                );
              }}
            >
              <option value="">Auto routing</option>
              {props.chatSelectableModels.map((model) => (
                <option key={model.id} value={model.id}>
                  {model.displayName?.trim() || model.id}
                </option>
              ))}
            </select>
            <GhostButton
              className="mt-3 w-full"
              onClick={() => props.onSelectChatModel("")}
              disabled={!props.chatSelectedModelId}
            >
              Clear preference
            </GhostButton>
          </div>
        </div>
      </section>

      <section className="forge-ops-panel">
        <div className="forge-ops-panel__head">
          <div>
            <div className="forge-ops-title">Registry</div>
            <div className="mt-1 text-xs text-forge-mist/65">
              Compact model list. Select a model to focus the active runtime
              controls.
            </div>
          </div>
          <span className="font-mono text-[11px] text-forge-mist/60">
            {props.models.length} registered
          </span>
        </div>
        <div className="forge-ops-panel__body">
          {props.models.length === 0 ? (
            <EmptyState
              title={
                props.loading ? "Refreshing registry" : "No models registered"
              }
              detail={
                props.loading
                  ? "Waiting for the runtime registry, queue, loaded-models, and backend health calls to return."
                  : "Import a local model or open the registry to reconcile a configured model home."
              }
            />
          ) : (
            <div className="space-y-2">
              {props.models.slice(0, 12).map((model) => (
                <button
                  key={model.id}
                  type="button"
                  onClick={() => props.onSelectModel(model.id)}
                  className={cx(
                    "flex w-full items-start justify-between gap-3 rounded border border-forge-platinum/10 bg-black/20 px-4 py-3 text-left transition hover:border-forge-ember/35 hover:bg-black/30 sm:items-center",
                    props.selectedModelId === model.id &&
                      "border-forge-ember/45 bg-forge-ember/10",
                  )}
                >
                  <span className="min-w-0">
                    <span className="block break-all font-mono text-sm text-forge-ash">
                      {model.id}
                    </span>
                    <span className="mt-1 block break-words text-xs leading-5 text-forge-mist">
                      {model.displayName || "Unnamed model"} ·{" "}
                      {model.backend || "backend unset"}
                    </span>
                  </span>
                  <span className={cx("shrink-0", badgeClass(model.status))}>
                    {model.status || "unknown"}
                  </span>
                </button>
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}
