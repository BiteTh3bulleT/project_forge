import { GhostButton, PrimaryButton } from "@forge/ui";

import { FoldSection } from "../../components/FoldSection";
import { api } from "../../lib/api";
import { Panel } from "./components";

export function ChatPromptSection(props: {
  chatPersonalityPrompt: string;
  chatPromptDefault: string;
  setChatPersonalityPrompt: (value: string) => void;
  setStatus: (value: string) => void;
}) {
  const {
    chatPersonalityPrompt,
    chatPromptDefault,
    setChatPersonalityPrompt,
    setStatus,
  } = props;

  return (
    <FoldSection
      title="Chat Prompt"
      subtitle="System prompt and default restoration controls."
      defaultOpen
    >
      <Panel
        title="Chat Personality Prompt"
        subtitle="Live system prompt for chat replies. Changes apply on the next assistant response."
      >
        <label className="text-xs font-semibold tracking-wide text-forge-mist">
          System prompt
        </label>
        <textarea
          className="forge-input mt-2 min-h-[280px] font-mono text-xs leading-relaxed"
          value={chatPersonalityPrompt}
          onChange={(e) => setChatPersonalityPrompt(e.target.value)}
        />
        <div className="mt-3 flex flex-wrap gap-2">
          <PrimaryButton
            onClick={async () => {
              await api.settings.patch({ chatPersonalityPrompt });
              setStatus("Chat personality prompt saved.");
            }}
          >
            Save chat prompt
          </PrimaryButton>
          <GhostButton
            onClick={() => {
              setChatPersonalityPrompt(chatPromptDefault);
              setStatus("Restored default chat prompt in editor.");
            }}
          >
            Reset editor to default
          </GhostButton>
          <GhostButton
            onClick={async () => {
              await api.settings.patch({
                chatPersonalityPrompt: chatPromptDefault,
              });
              setChatPersonalityPrompt(chatPromptDefault);
              setStatus("Chat personality prompt reset to default.");
            }}
          >
            Save default prompt
          </GhostButton>
        </div>
      </Panel>
    </FoldSection>
  );
}
