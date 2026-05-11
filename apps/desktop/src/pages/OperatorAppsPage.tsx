import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useEffect, useMemo, useState } from "react";

import {
  isTauriDesktop,
  iconAssetUrl,
  launchOperatorApp,
  listOperatorApps,
  type OperatorApp,
} from "../lib/desktop";

const FALLBACK_OPERATOR_APPS: OperatorApp[] = [
  {
    id: "terminal",
    label: "Terminal",
    description: "Open a Foot terminal in the current FORGE operator session.",
    executable: "foot",
    category: "Workspace",
    iconName: "foot",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
  {
    id: "files",
    label: "Files",
    description:
      "Open the PCManFM file manager in the current FORGE operator session.",
    executable: "pcmanfm",
    category: "Workspace",
    iconName: "system-file-manager",
    iconPath: null,
    desktopFile: null,
    native: false,
  },
];

export function OperatorAppsPage() {
  const tauriAvailable = useMemo(() => isTauriDesktop(), []);
  const [apps, setApps] = useState<OperatorApp[]>(FALLBACK_OPERATOR_APPS);
  const [loading, setLoading] = useState(true);
  const [busyAppId, setBusyAppId] = useState<string | null>(null);
  const [status, setStatus] = useState(
    tauriAvailable
      ? "Operator app launcher ready."
      : "Operator apps require the Tauri desktop runtime.",
  );

  async function refreshApps() {
    setLoading(true);
    try {
      const listed = await listOperatorApps();
      setApps(listed.length > 0 ? listed : FALLBACK_OPERATOR_APPS);
      setStatus(
        tauriAvailable
          ? "Operator app launcher ready."
          : "Operator apps require the Tauri desktop runtime.",
      );
    } catch (error) {
      setApps(FALLBACK_OPERATOR_APPS);
      setStatus(error instanceof Error ? error.message : String(error));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void refreshApps();
  }, []);

  async function launch(app: OperatorApp) {
    if (!tauriAvailable) {
      setStatus("Operator apps require the Tauri desktop runtime.");
      return;
    }
    setBusyAppId(app.id);
    try {
      const result = await launchOperatorApp(app.id);
      setStatus(
        result.pid
          ? `${result.label} launch requested. PID ${result.pid}.`
          : result.message,
      );
    } catch (error) {
      setStatus(error instanceof Error ? error.message : String(error));
    } finally {
      setBusyAppId(null);
    }
  }

  return (
    <div className="forge-ops-board space-y-5">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <div className="forge-ops-label">Operator</div>
          <h1 className="text-2xl font-semibold text-forge-ash">
            Operator Apps
          </h1>
        </div>
        <GhostButton onClick={() => void refreshApps()} disabled={loading}>
          {loading ? "Refreshing" : "Refresh"}
        </GhostButton>
      </div>

      <Panel
        title="Allowlisted Launch Surface"
        subtitle="Fixed terminal and file manager entries for the current desktop session."
      >
        <div
          className={
            tauriAvailable
              ? "rounded-md border border-forge-electric/25 bg-forge-electric/10 p-3 text-sm text-forge-ash"
              : "rounded-md border border-forge-amber/30 bg-forge-amber/10 p-3 text-sm text-forge-ash"
          }
        >
          {status}
        </div>

        <div className="mt-4 space-y-4">
          {groupAppsByCategory(apps).map((group) => (
            <section key={group.category} className="space-y-2">
              <h2 className="text-xs font-semibold uppercase tracking-[0.14em] text-forge-mist">
                {group.category}
              </h2>
              <div className="grid gap-3 md:grid-cols-2">
                {group.items.map((app) => (
                  <div
                    key={app.id}
                    className="rounded border border-forge-platinum/10 bg-black/25 p-4"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="flex min-w-0 items-start gap-3">
                        <OperatorAppIcon app={app} />
                        <div className="min-w-0">
                          <h3 className="text-sm font-semibold text-forge-ash">
                            {app.label}
                          </h3>
                          <p className="mt-1 text-xs leading-5 text-forge-mist/75">
                            {app.description}
                          </p>
                        </div>
                      </div>
                      <div className="flex shrink-0 flex-col items-end gap-1">
                        <span className="rounded border border-forge-platinum/10 bg-black/30 px-2 py-1 font-mono text-[11px] text-forge-mist">
                          {app.executable}
                        </span>
                        {app.native ? (
                          <span className="rounded border border-forge-electric/25 bg-forge-electric/10 px-2 py-1 text-[11px] text-forge-mist">
                            Native
                          </span>
                        ) : null}
                      </div>
                    </div>
                    <div className="mt-4">
                      <PrimaryButton
                        disabled={!tauriAvailable || busyAppId !== null}
                        onClick={() => void launch(app)}
                      >
                        {busyAppId === app.id
                          ? "Launching"
                          : `Launch ${app.label}`}
                      </PrimaryButton>
                    </div>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      </Panel>
    </div>
  );
}

function groupAppsByCategory(apps: OperatorApp[]) {
  const groups = new Map<string, OperatorApp[]>();
  for (const app of apps) {
    const category = app.category?.trim() || "Tools";
    groups.set(category, [...(groups.get(category) ?? []), app]);
  }
  return Array.from(groups, ([category, items]) => ({ category, items }));
}

function OperatorAppIcon(props: { app: OperatorApp }) {
  if (props.app.iconPath) {
    return (
      <img
        src={iconAssetUrl(props.app.iconPath)}
        alt={`${props.app.label} icon`}
        className="h-9 w-9 shrink-0 rounded border border-forge-platinum/10 bg-black/30 object-contain p-1"
        draggable={false}
      />
    );
  }
  const short = props.app.label
    .split(/\s+/)
    .filter(Boolean)
    .slice(0, 2)
    .map((part) => part[0]?.toUpperCase() ?? "")
    .join("");
  return (
    <span
      role="img"
      aria-label={`${props.app.label} icon`}
      className="grid h-9 w-9 shrink-0 place-items-center rounded border border-forge-electric/25 bg-forge-electric/10 font-mono text-xs font-bold text-forge-electric"
    >
      {short || "AP"}
    </span>
  );
}
