import { GhostButton, PrimaryButton } from "@forge/ui";

import { EmptyState } from "./shared";

export function ImportRegistrationPanel(props: {
  runtimeAvailable: boolean;
  runtimeControlsDisabled: boolean;
  importPath: string;
  importDisplayName: string;
  importFamily: string;
  importBackend: string;
  importCapabilities: string;
  importPreferred: boolean;
  importBusy: boolean;
  scanBusy: boolean;
  setImportPath: (value: string) => void;
  setImportDisplayName: (value: string) => void;
  setImportFamily: (value: string) => void;
  setImportBackend: (value: string) => void;
  setImportCapabilities: (value: string) => void;
  setImportPreferred: (value: boolean) => void;
  onImport: () => void;
  onScan: () => void;
}) {
  return (
    <section className="forge-ops-panel">
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">Import and Registration</div>
          <div className="mt-1 text-xs text-forge-mist/65">
            Local GGUF and manifest-backed model registration. File deletion
            remains intentionally out of scope.
          </div>
        </div>
        <span className="font-mono text-[11px] text-forge-mist/60">
          model.management
        </span>
      </div>
      <div className="forge-ops-panel__body">
        {!props.runtimeAvailable ? (
          <div className="mb-4">
            <EmptyState
              title="Model runtime unavailable"
              detail="Registration controls are read-only until modelruntime is available through the governed core status path."
              tone="warn"
            />
          </div>
        ) : null}
        <div className="grid gap-4 xl:grid-cols-[minmax(0,1.35fr)_minmax(240px,0.65fr)]">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <label className="text-xs text-forge-mist">
              Path
              <input
                className="forge-input mt-1"
                value={props.importPath}
                onChange={(e) => props.setImportPath(e.target.value)}
                placeholder="/models/coder.gguf"
                disabled={props.runtimeControlsDisabled}
              />
            </label>
            <label className="text-xs text-forge-mist">
              Display name
              <input
                className="forge-input mt-1"
                value={props.importDisplayName}
                onChange={(e) => props.setImportDisplayName(e.target.value)}
                placeholder="Qwen Coder"
                disabled={props.runtimeControlsDisabled}
              />
            </label>
            <label className="text-xs text-forge-mist">
              Family
              <input
                className="forge-input mt-1"
                value={props.importFamily}
                onChange={(e) => props.setImportFamily(e.target.value)}
                placeholder="qwen"
                disabled={props.runtimeControlsDisabled}
              />
            </label>
            <label className="relative z-10 overflow-visible text-xs text-forge-mist">
              Backend
              <select
                className="forge-input relative z-20 mt-1 h-10 w-full"
                value={props.importBackend}
                onChange={(e) => props.setImportBackend(e.target.value)}
                disabled={props.runtimeControlsDisabled}
              >
                <option value="">manifest/default</option>
                <option value="llama_cpp">llama_cpp</option>
                <option value="openai_compat">openai_compat</option>
                <option value="vllm">vllm</option>
              </select>
            </label>
            <label className="text-xs text-forge-mist">
              Capabilities
              <input
                className="forge-input mt-1"
                value={props.importCapabilities}
                onChange={(e) => props.setImportCapabilities(e.target.value)}
                placeholder="chat,completion"
                disabled={props.runtimeControlsDisabled}
              />
            </label>
            <label className="flex items-center gap-2 self-end rounded border border-white/10 bg-black/20 px-3 py-2 text-xs text-forge-mist">
              <input
                type="checkbox"
                checked={props.importPreferred}
                onChange={(e) => props.setImportPreferred(e.target.checked)}
                disabled={props.runtimeControlsDisabled}
              />
              Mark as preferred
            </label>
          </div>
          <div className="rounded border border-white/10 bg-black/20 p-3 text-xs text-forge-mist">
            <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-forge-mist/80">
              Registration Notes
            </div>
            <div className="mt-3 space-y-2">
              <div>
                Import records runtime details only; removal never deletes model
                files.
              </div>
              <div>
                Use <span className="text-forge-ash">preferred</span> when the
                imported model should be favored by runtime compatibility
                checks.
              </div>
              <div>
                Capabilities should stay comma-separated so registry and chat
                filtering remain aligned.
              </div>
            </div>
          </div>
        </div>
        <div className="mt-4 flex flex-wrap gap-2">
          <PrimaryButton
            className="min-h-11 px-4"
            onClick={props.onImport}
            disabled={props.importBusy || props.runtimeControlsDisabled}
          >
            {props.importBusy ? "Importing..." : "Import Model"}
          </PrimaryButton>
          <GhostButton
            className="min-h-11 px-4"
            onClick={props.onScan}
            disabled={props.scanBusy || props.runtimeControlsDisabled}
          >
            {props.scanBusy ? "Scanning..." : "Reconcile Registry"}
          </GhostButton>
        </div>
      </div>
    </section>
  );
}
