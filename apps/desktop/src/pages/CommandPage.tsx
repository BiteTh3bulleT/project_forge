import { GhostButton, Panel, PrimaryButton } from "@forge/ui";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";

export function CommandPage() {
  const navigate = useNavigate();
  const setStatus = useUiStore((s) => s.setStatusLine);

  async function queue(templateId: string, title: string, objective: string, query: string) {
    const res = await api.jobs.create({
      templateId,
      title,
      userRequest: query,
      objective,
      query,
      initiatingSource: "command_page",
    });
    setStatus(`Job queued: ${res.job.id}`);
    navigate(`/jobs/${res.job.id}`);
  }

  return (
    <div className="space-y-6">
      <Panel
        title="Command Desk"
        subtitle="Ignition panel for bounded job templates. No hidden autonomy, no silent scope shifts."
        actions={
          <GhostButton
            onClick={async () => {
              const res = await api.commands.execute("reindex", { via: "command.page" });
              setStatus(res.jobId ? `Re-index job queued: ${res.jobId}` : "Re-index queued.");
              navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
            }}
          >
            Queue Re-index Job
          </GhostButton>
        }
      >
        <div className="grid gap-3 md:grid-cols-2">
          <Quick title="Chat" body="Persisted operator threads; optional Ollama replies; queue jobs from a thread." onClick={() => navigate("/chat")} />
          <Quick title="Workbench" body="Artifact index, textual preview, optional job correlation." onClick={() => navigate("/workbench")} />
          <Quick title="Canvas" body="Scratch boards with positioned notes (SQLite-backed)." onClick={() => navigate("/canvas")} />
          <Quick title="Dashboard" body="View live queues, failures, reviews, and routing advisories." onClick={() => navigate("/dashboard")} />
          <Quick title="Policy" body="Inspect presets, recommendations, and packet guidance." onClick={() => navigate("/policy")} />
          <Quick title="Jobs" body="Inspect lifecycle, logs, packet contract, and artifacts." onClick={() => navigate("/jobs")} />
          <Quick title="Approvals" body="Review pending high-risk requests and decide." onClick={() => navigate("/approvals")} />
          <Quick title="Strategies" body="Tune reusable strategy contracts for common task types." onClick={() => navigate("/strategies")} />
          <Quick title="Automation" body="Run bounded rules with dry-run previews and history." onClick={() => navigate("/automation")} />
          <Quick title="Reviews" body="Resolve imported/output review queue with persisted decisions." onClick={() => navigate("/reviews")} />
          <Quick title="Project Context" body="Import and normalize FORGE context into durable guidance files." onClick={() => navigate("/project-context")} />
          <Quick title="Memory" body="Search indexed retrieval context and open chunk detail." onClick={() => navigate("/memory")} />
        </div>
      </Panel>

      <Panel title="Template Jobs" subtitle="Each launch maps to a controlled template with explicit risk class and artifact expectations.">
        <div className="flex flex-wrap gap-2">
          <PrimaryButton onClick={() => void queue("search_packet", "Search packet", "Assemble a packet from current indexed memory", "project context")}>Create packet</PrimaryButton>
          <PrimaryButton onClick={() => void queue("ollama_summary", "Ollama summary", "Summarize relevant context from retrieval", "summarize current forge phase")}>Ollama summary</PrimaryButton>
          <PrimaryButton onClick={() => void queue("plan_from_index", "Implementation plan", "Draft implementation plan from local index", "implementation plan for next phase")}>Plan from index</PrimaryButton>
          <PrimaryButton onClick={() => void queue("prepare_codex_handoff", "Codex handoff", "Prepare bounded Codex handoff contract", "implement job cancellation checks")}>Prepare Codex handoff</PrimaryButton>
          <PrimaryButton onClick={() => void queue("prepare_claude_handoff", "Claude handoff", "Prepare bounded Claude Code handoff contract", "draft docs for approvals")}>Prepare Claude handoff</PrimaryButton>
          <PrimaryButton onClick={() => void queue("normalize_project_context", "Normalize context", "Import source context and regenerate AGENTS/CLAUDE guidance", "normalize forge context")}>Normalize context</PrimaryButton>
        </div>
      </Panel>
    </div>
  );
}

function Quick(props: { title: string; body: string; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={props.onClick}
      className="rounded-md border border-white/10 bg-forge-slate/20 p-4 text-left hover:border-forge-ember/35 hover:bg-forge-slate/35"
    >
      <div className="text-sm font-semibold text-forge-ash">{props.title}</div>
      <div className="mt-2 text-xs text-forge-mist">{props.body}</div>
    </button>
  );
}
