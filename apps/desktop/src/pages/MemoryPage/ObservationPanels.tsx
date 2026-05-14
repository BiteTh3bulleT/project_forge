import type {
  MemoryObservation,
  MemoryObservationDetail,
  ObservationVSADetail,
} from "@forge/shared";

import { api } from "../../lib/api";
import { formatTime } from "../../lib/format";
import { EmptyState, Panel } from "./shared";

export function ObservationsPanel(props: {
  observations: MemoryObservation[];
  selectedObsId: number | null;
  setSelectedObsId: (id: number) => void;
}) {
  return (
    <Panel
      title="Observations"
      subtitle="Cold+warm memory records with structural metadata and staleness/usefulness state."
    >
      {props.observations.length === 0 ? (
        <EmptyState
          title="No observations match"
          detail="Adjust the observation type, dossier, or stale-only filter, then refresh the observation list."
        />
      ) : (
        <div className="space-y-2">
          {props.observations.map((obs) => (
            <button
              key={obs.id}
              type="button"
              onClick={() => props.setSelectedObsId(obs.id)}
              className={[
                "w-full rounded border px-3 py-2 text-left",
                props.selectedObsId === obs.id
                  ? "border-forge-ember/40 bg-black/30"
                  : "border-forge-platinum/10 bg-black/20 hover:border-forge-ember/35",
              ].join(" ")}
            >
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0 break-words text-xs font-semibold text-forge-ash">
                  #{obs.id} · {obs.type}
                </div>
                <div className="shrink-0 text-[11px] text-forge-mist">
                  {formatTime(obs.observedAtMs)}
                </div>
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                {obs.summary || obs.sourcePath || "(no summary)"}
              </div>
              <div className="mt-1 break-words text-[11px] leading-5 text-forge-mist">
                dossier {obs.dossierId ?? "none"} · useful{" "}
                {obs.usefulnessCount} · noise {obs.noiseCount} · stale{" "}
                {String(obs.stale)}
              </div>
            </button>
          ))}
        </div>
      )}
    </Panel>
  );
}

export function ObservationDetailPanel(props: {
  obsDetail: MemoryObservationDetail | null;
  obsVSADetail: ObservationVSADetail | null;
  setObsDetail: (detail: MemoryObservationDetail) => void;
  setObsVSADetail: (detail: ObservationVSADetail | null) => void;
  status: string;
  setStatus: (status: string) => void;
}) {
  return (
    <Panel
      title="Observation Detail"
      subtitle="Inspect lineage, links, and usefulness events. Mark stale/useful/noisy to repair memory drift."
    >
      {!props.obsDetail ? (
        <EmptyState
          title="Select an observation"
          detail="Choose a recent memory record to inspect lineage, raw content, VSA bindings, and usefulness controls."
        />
      ) : (
        <ObservationDetailContent
          obsDetail={props.obsDetail}
          obsVSADetail={props.obsVSADetail}
          setObsDetail={props.setObsDetail}
          setObsVSADetail={props.setObsVSADetail}
          status={props.status}
          setStatus={props.setStatus}
        />
      )}
    </Panel>
  );
}

function ObservationDetailContent(props: {
  obsDetail: MemoryObservationDetail;
  obsVSADetail: ObservationVSADetail | null;
  setObsDetail: (detail: MemoryObservationDetail) => void;
  setObsVSADetail: (detail: ObservationVSADetail | null) => void;
  status: string;
  setStatus: (status: string) => void;
}) {
  return (
    <div className="space-y-3">
      <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs leading-5 text-forge-mist">
        <div>
          ID #{props.obsDetail.observation.id} ·{" "}
          {props.obsDetail.observation.type} · verification{" "}
          {props.obsDetail.observation.verificationState}
        </div>
        <div className="mt-1">
          origin {props.obsDetail.observation.originKind || "none"}:
          {props.obsDetail.observation.originId || "none"}
        </div>
        <div className="mt-1">
          score {props.obsDetail.observation.usefulnessScore.toFixed(2)} ·
          useful {props.obsDetail.observation.usefulnessCount} · noisy{" "}
          {props.obsDetail.observation.noiseCount}
        </div>
      </div>
      <div className="max-h-[360px] overflow-auto rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs leading-5 text-forge-mist whitespace-pre-wrap">
        {props.obsDetail.observation.rawContent || "(no raw content)"}
      </div>
      <ObservationVSADetailBlock obsVSADetail={props.obsVSADetail} />
      <div className="flex flex-wrap gap-2">
        {[
          ["useful", "Useful"],
          ["noisy", "Noisy"],
          ["not_useful", "Not useful"],
          ["insufficient", "Insufficient"],
        ].map(([value, label]) => (
          <button
            key={value}
            type="button"
            className="forge-btn forge-btn--ghost"
            onClick={async () => {
              await api.memory.markObservationUsefulness(
                props.obsDetail.observation.id,
                {
                  signal: value,
                  note: `Marked from memory page at ${new Date().toISOString()}`,
                },
              );
              const [detail, vsa] = await Promise.all([
                api.memory.getObservation(props.obsDetail.observation.id),
                api.memory
                  .getObservationVSA(props.obsDetail.observation.id)
                  .catch(() => ({
                    detail: null as ObservationVSADetail | null,
                  })),
              ]);
              props.setObsDetail(detail.observation);
              if (vsa.detail) {
                props.setObsVSADetail(vsa.detail);
              }
              props.setStatus(
                `Observation ${props.obsDetail.observation.id} marked ${value}.`,
              );
            }}
          >
            {label}
          </button>
        ))}
        <button
          type="button"
          className="forge-btn forge-btn--ghost"
          onClick={async () => {
            await api.memory.patchObservation(props.obsDetail.observation.id, {
              stale: !props.obsDetail.observation.stale,
              lastVerifiedAtMs: Date.now(),
            });
            const [detail, vsa] = await Promise.all([
              api.memory.getObservation(props.obsDetail.observation.id),
              api.memory
                .getObservationVSA(props.obsDetail.observation.id)
                .catch(() => ({
                  detail: null as ObservationVSADetail | null,
                })),
            ]);
            props.setObsDetail(detail.observation);
            if (vsa.detail) {
              props.setObsVSADetail(vsa.detail);
            }
            props.setStatus(
              `Observation ${props.obsDetail.observation.id} stale=${String(detail.observation.observation.stale)}.`,
            );
          }}
        >
          Toggle stale
        </button>
      </div>
      {props.status ? (
        <div className="rounded border border-forge-platinum/10 bg-black/20 p-2 text-xs text-forge-mist">
          {props.status}
        </div>
      ) : null}
    </div>
  );
}

function ObservationVSADetailBlock(props: {
  obsVSADetail: ObservationVSADetail | null;
}) {
  return (
    <div className="rounded border border-forge-platinum/10 bg-black/20 p-3 text-xs text-forge-mist">
      <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-ash">
        VSA Detail
      </div>
      {!props.obsVSADetail?.pointer ? (
        <div className="mt-2 text-[11px]">
          No VSA pointer indexed for this observation yet.
        </div>
      ) : (
        <div className="mt-2 space-y-2">
          <div className="text-[11px]">
            pointer #{props.obsVSADetail.pointer.id} · dims{" "}
            {props.obsVSADetail.pointer.dims} · norm{" "}
            {props.obsVSADetail.pointer.norm.toFixed(4)} · stale{" "}
            {String(props.obsVSADetail.pointer.stale)}
          </div>
          <div className="break-all text-[11px]">
            fingerprint {props.obsVSADetail.pointer.sourceFingerprint || "none"}
          </div>
          <div className="text-[11px]">
            vector preview [
            {props.obsVSADetail.pointer.pointer
              .slice(0, 8)
              .map((v) => v.toFixed(3))
              .join(", ")}
            {props.obsVSADetail.pointer.pointer.length > 8 ? ", ..." : ""}]
          </div>
          <VSARoleBindingsBlock obsVSADetail={props.obsVSADetail} />
          <VSAAssociationsBlock obsVSADetail={props.obsVSADetail} />
        </div>
      )}
    </div>
  );
}

function VSARoleBindingsBlock(props: {
  obsVSADetail: ObservationVSADetail;
}) {
  return (
    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
      <div className="text-[11px] font-semibold text-forge-ash">
        Role bindings ({props.obsVSADetail.roleBindings.length})
      </div>
      {props.obsVSADetail.roleBindings.length === 0 ? (
        <div className="mt-1 text-[11px]">No role bindings.</div>
      ) : (
        <div className="mt-1 space-y-1 text-[11px]">
          {props.obsVSADetail.roleBindings.slice(0, 10).map((binding) => (
            <div key={binding.id} className="break-words">
              {binding.role}={binding.filler} · w {binding.weight.toFixed(2)} ·
              support {binding.supportCount} · noise {binding.noiseCount}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function VSAAssociationsBlock(props: {
  obsVSADetail: ObservationVSADetail;
}) {
  return (
    <div className="rounded border border-forge-platinum/10 bg-black/30 p-2">
      <div className="text-[11px] font-semibold text-forge-ash">
        Associations ({props.obsVSADetail.associations.length})
      </div>
      {props.obsVSADetail.associations.length === 0 ? (
        <div className="mt-1 text-[11px]">No associations.</div>
      ) : (
        <div className="mt-1 space-y-1 text-[11px]">
          {props.obsVSADetail.associations.slice(0, 10).map((association) => (
            <div key={association.id} className="break-words">
              {association.fromObservationId} {"->"}{" "}
              {association.toObservationId} · {association.associationType} ·
              strength {association.strength.toFixed(3)} · support{" "}
              {association.supportCount} · noise {association.noiseCount}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
