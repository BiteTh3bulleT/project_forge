import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";

import { api } from "../lib/api";
import { useUiStore } from "../stores/uiStore";
import { useWorkspaceStore } from "../stores/workspaceStore";

function normalizeCmd(raw: string) {
  return raw.trim().replace(/\s+/g, " ");
}

export function CommandBar(props: { compact?: boolean }) {
  const navigate = useNavigate();
  const draft = useUiStore((s) => s.commandDraft);
  const setDraft = useUiStore((s) => s.setCommandDraft);
  const setStatus = useUiStore((s) => s.setStatusLine);
  const uiMode = useUiStore((s) => s.uiMode);
  const ping = useWorkspaceStore((s) => s.ping);
  const [busy, setBusy] = useState(false);

  const hint = useMemo(
    () =>
      uiMode === "cognitive"
        ? "Cognitive mode keeps command entry focused on search, jobs, approvals, and current context."
        : "Metrics mode accepts route jumps plus diagnostics commands: go /path, :reindex, job <query>, ollama summary <query>.",
    [uiMode],
  );

  async function runQuick(id: string) {
    if (busy) return;
    setBusy(true);
    try {
      if (id === "start") {
        navigate("/start");
        setStatus("Opened start surface.");
        return;
      }
      if (id === "memory") {
        navigate("/memory");
        setStatus("Opened memory search.");
        return;
      }
      if (id === "jobs") {
        navigate("/jobs");
        setStatus("Opened jobs.");
        return;
      }
      if (id === "approvals") {
        navigate("/approvals");
        setStatus("Opened approvals queue.");
        return;
      }
      if (id === "packet") {
        const res = await api.commands.execute("search_packet", {
          query: "Build task packet from current project context.",
        });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
        setStatus(
          res.jobId ? `Packet job queued: ${res.jobId}.` : "Packet job queued.",
        );
        return;
      }
      if (id === "ollama") {
        const res = await api.commands.execute("ollama_summary", {
          query:
            "Summarize relevant project context and active execution state.",
        });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
        setStatus(
          res.jobId
            ? `Ollama summary queued: ${res.jobId}.`
            : "Ollama summary queued.",
        );
        return;
      }
      if (id === "reindex") {
        const res = await api.commands.execute("reindex", {
          via: "command_bar_quick_action",
        });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
        setStatus(
          res.jobId
            ? `Re-index job queued: ${res.jobId}.`
            : "Re-index submitted.",
        );
      }
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setStatus(`Quick action failed: ${msg}`);
      await ping();
    } finally {
      setBusy(false);
    }
  }

  async function run() {
    const line = normalizeCmd(draft);
    if (!line) return;
    setBusy(true);
    try {
      if (line.startsWith("?")) {
        const q = line.slice(1).trim();
        navigate(`/memory?q=${encodeURIComponent(q)}`);
        setStatus(`Searching memory for “${q}”.`);
        setDraft("");
        return;
      }

      if (line.startsWith("/")) {
        navigate(line);
        setStatus(`Navigated to ${line}.`);
        setDraft("");
        return;
      }

      const [verbRaw, ...rest] = line.split(" ");
      const verb = verbRaw.toLowerCase();
      const tail = rest.join(" ").trim();

      if (verb === "help" || verb === "start") {
        navigate("/start");
        setStatus("Opened start surface.");
        setDraft("");
        return;
      }

      if (verb === "go" || verb === "navigate") {
        const path = tail.startsWith("/") ? tail : `/${tail}`;
        navigate(path);
        await api.commands.execute("navigate", { path });
        setStatus(`Navigated to ${path}.`);
        setDraft("");
        return;
      }

      if (
        verb === "jobs" ||
        verb === "dashboard" ||
        verb === "chat" ||
        verb === "reviews" ||
        verb === "approvals" ||
        verb === "memory" ||
        verb === "dossiers" ||
        verb === "policy" ||
        verb === "settings" ||
        verb === "sources" ||
        verb === "adapters"
      ) {
        const path = verb === "dashboard" ? "/dashboard" : `/${verb}`;
        navigate(path);
        setStatus(`Opened ${verb}.`);
        setDraft("");
        return;
      }

      if (verb === "search" || verb === "s") {
        navigate(`/memory?q=${encodeURIComponent(tail)}`);
        setStatus(`Searching memory for “${tail}”.`);
        setDraft("");
        return;
      }

      if (verb === "keyword" || verb === "semantic" || verb === "hybrid") {
        const q = tail || "project context";
        const res = await api.retrieval.createRun({
          query: q,
          mode: verb,
          limit: 30,
          selectForPacket: 8,
          notes: "command_bar_manual",
        });
        navigate("/retrieval-runs");
        setStatus(`Retrieval run ${res.run.id} queued in ${verb} mode.`);
        setDraft("");
        return;
      }

      if (verb === ":reindex" || verb === "reindex") {
        const res = await api.commands.execute("reindex", {
          via: "command.bar",
        });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
        setStatus(
          res.jobId
            ? `Re-index job queued: ${res.jobId}.`
            : "Re-index command submitted.",
        );
        setDraft("");
        return;
      }

      if (verb === "job") {
        const q = tail || "Build context packet";
        const res = await api.commands.execute("search_packet", { query: q });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
        setStatus(
          res.jobId ? `Packet job queued: ${res.jobId}.` : "Packet job queued.",
        );
        setDraft("");
        return;
      }

      if (verb === "ollama") {
        const [subRaw, ...rem] = tail.split(" ");
        const sub = subRaw?.toLowerCase() ?? "";
        if (sub === "summary") {
          const q =
            rem.join(" ").trim() || "Summarize relevant project context.";
          const res = await api.commands.execute("ollama_summary", {
            query: q,
          });
          navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
          setStatus(
            res.jobId
              ? `Ollama summary job queued: ${res.jobId}.`
              : "Ollama summary job queued.",
          );
          setDraft("");
          return;
        }
        if (sub === "plan") {
          const q =
            rem.join(" ").trim() ||
            "Draft implementation plan from indexed context.";
          const res = await api.commands.execute("plan_from_index", {
            query: q,
          });
          navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
          setStatus(
            res.jobId
              ? `Planning job queued: ${res.jobId}.`
              : "Planning job queued.",
          );
          setDraft("");
          return;
        }
      }

      if (verb === "codex") {
        const [subRaw, ...rem] = tail.split(" ");
        const sub = subRaw?.toLowerCase() ?? "";
        if (sub === "packet") {
          const q = rem.join(" ").trim() || "Prepare Codex handoff packet.";
          const res = await api.commands.execute("prepare_codex_handoff", {
            query: q,
          });
          navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
          setStatus(
            res.jobId
              ? `Codex handoff job queued: ${res.jobId}.`
              : "Codex handoff job queued.",
          );
          setDraft("");
          return;
        }
      }

      if (verb === "claude") {
        const [subRaw, ...rem] = tail.split(" ");
        const sub = subRaw?.toLowerCase() ?? "";
        if (sub === "packet") {
          const q =
            rem.join(" ").trim() || "Prepare Claude Code handoff packet.";
          const res = await api.commands.execute("prepare_claude_handoff", {
            query: q,
          });
          navigate(res.jobId ? `/jobs/${res.jobId}` : "/jobs");
          setStatus(
            res.jobId
              ? `Claude handoff job queued: ${res.jobId}.`
              : "Claude handoff job queued.",
          );
          setDraft("");
          return;
        }
      }

      if (
        verb === "context" &&
        (tail.startsWith("import") || tail.startsWith("normalize"))
      ) {
        const maybePath = tail.replace(/^(import|normalize)\s*/, "").trim();
        const res = await api.commands.execute("normalize_project_context", {
          query: "Normalize project context",
          sourcePath: maybePath,
          notes: "Triggered from command bar",
        });
        navigate(res.jobId ? `/jobs/${res.jobId}` : "/project-context");
        setStatus(
          res.jobId
            ? `Project context normalization job queued: ${res.jobId}.`
            : "Project context normalization queued.",
        );
        setDraft("");
        return;
      }

      navigate(`/memory?q=${encodeURIComponent(line)}`);
      setStatus(`Searching memory for “${line}”.`);
      setDraft("");
    } catch (e) {
      const msg = e instanceof Error ? e.message : String(e);
      setStatus(`Command failed: ${msg}`);
      await ping();
    } finally {
      setBusy(false);
    }
  }

  const quickButtons =
    uiMode === "cognitive"
      ? [
          { id: "start", label: "Start" },
          { id: "memory", label: "Search Memory" },
          { id: "packet", label: "Build Packet Job" },
          { id: "ollama", label: "Run Ollama Summary" },
          { id: "approvals", label: "Approvals" },
        ]
      : [
          { id: "jobs", label: "Jobs" },
          { id: "memory", label: "Memory" },
          { id: "packet", label: "Packet Job" },
          { id: "ollama", label: "Ollama" },
          { id: "reindex", label: "Re-index" },
        ];

  if (props.compact) {
    return (
      <div className="flex items-center gap-2">
        <input
          className="forge-input min-h-[2rem] py-1.5 text-[12px]"
          placeholder={
            uiMode === "cognitive"
              ? "search auth | go /jobs"
              : "go /dashboard | :reindex"
          }
          value={draft}
          onChange={(e) => setDraft(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter") void run();
          }}
          disabled={busy}
          aria-label="Command input"
        />
        <button
          type="button"
          className="forge-btn forge-btn--primary h-8 px-3 py-1 text-[11px]"
          onClick={() => void run()}
          disabled={busy}
        >
          {busy ? "…" : "Run"}
        </button>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-2.5">
      <div className="flex flex-wrap gap-1.5">
        {quickButtons.map((btn) => (
          <button
            key={btn.id}
            type="button"
            onClick={() => void runQuick(btn.id)}
            disabled={busy}
            className="forge-chip forge-chip--muted whitespace-nowrap px-2.5 py-1 text-[10px] font-medium uppercase tracking-wide disabled:cursor-not-allowed disabled:opacity-60"
          >
            {btn.label}
          </button>
        ))}
      </div>

      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="flex min-w-0 flex-1 items-center gap-2">
          <span className="shrink-0 text-[10px] font-semibold uppercase tracking-[0.16em] text-forge-mist/55">
            Command
          </span>
          <input
            className="forge-input min-h-[2.25rem] py-1.5 text-[13px]"
            placeholder={
              uiMode === "cognitive"
                ? "Try: search auth flow | go /jobs"
                : "go /dashboard | :reindex | codex packet auth fix"
            }
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Enter") void run();
            }}
            disabled={busy}
            aria-label="Command input"
          />
        </div>
        <div className="flex shrink-0 items-center gap-2 sm:pl-1">
          <button
            type="button"
            className="forge-btn forge-btn--primary min-w-[4.5rem] px-4"
            onClick={() => void run()}
            disabled={busy}
          >
            {busy ? "…" : "Run"}
          </button>
          <span className="forge-kbd hidden sm:inline" title="Submit">
            Enter
          </span>
        </div>
      </div>
      <p className="text-[10px] leading-relaxed text-forge-mist/45">{hint}</p>
    </div>
  );
}
