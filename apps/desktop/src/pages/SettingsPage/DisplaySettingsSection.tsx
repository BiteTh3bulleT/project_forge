import { PrimaryButton } from "@forge/ui";
import type { Dispatch, SetStateAction } from "react";

import { FoldSection } from "../../components/FoldSection";
import { Panel } from "./components";

type DisplaySettingsSectionProps = {
  theme: "dark" | "light";
  setTheme: Dispatch<SetStateAction<"dark" | "light">>;
  contrastPreference: "high" | "normal";
  effectsPreference: "subtle" | "off";
  setContrastPreference: (value: "high" | "normal") => void;
  setEffectsPreference: (value: "subtle" | "off") => void;
  saveTheme: () => Promise<void>;
};

export function DisplaySettingsSection({
  theme,
  setTheme,
  contrastPreference,
  effectsPreference,
  setContrastPreference,
  setEffectsPreference,
  saveTheme,
}: DisplaySettingsSectionProps) {
  return (
    <FoldSection
      title="Display and Workspace"
      subtitle="Theme, readability, and local paths."
      defaultOpen
    >
      <Panel title="Theme" subtitle="Dark is the intended operator default.">
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className={
              theme === "dark"
                ? "forge-btn forge-btn--primary"
                : "forge-btn forge-btn--ghost"
            }
            onClick={() => setTheme("dark")}
          >
            Dark
          </button>
          <button
            type="button"
            className={
              theme === "light"
                ? "forge-btn forge-btn--primary"
                : "forge-btn forge-btn--ghost"
            }
            onClick={() => setTheme("light")}
          >
            Light
          </button>
          <PrimaryButton onClick={saveTheme}>Save theme</PrimaryButton>
        </div>
      </Panel>

      <Panel
        title="Display Preferences"
        subtitle="Local contrast and visual effects for operator readability."
      >
        <div className="grid gap-3 md:grid-cols-2">
          <label className="text-xs font-semibold tracking-wide text-forge-mist">
            Contrast
            <select
              className="forge-input mt-1"
              value={contrastPreference}
              onChange={(e) =>
                setContrastPreference(e.target.value as "high" | "normal")
              }
            >
              <option value="high">High</option>
              <option value="normal">Normal</option>
            </select>
          </label>
          <label className="text-xs font-semibold tracking-wide text-forge-mist">
            Effects
            <select
              className="forge-input mt-1"
              value={effectsPreference}
              onChange={(e) =>
                setEffectsPreference(e.target.value as "subtle" | "off")
              }
            >
              <option value="subtle">Subtle</option>
              <option value="off">Off</option>
            </select>
          </label>
        </div>
        <div className="mt-2 text-xs text-forge-mist">
          Changes are local and persist on this machine.
        </div>
      </Panel>
    </FoldSection>
  );
}
