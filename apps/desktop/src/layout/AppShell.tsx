import type { ApprovalRequest, DashboardSummary, JobDetail, ReviewRecord } from "@forge/shared";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { useLocation, useNavigate } from "react-router-dom";

import { CommandBar } from "../components/CommandBar";
import { api, type CanvasBoardDetail, type ChatThreadDetail, type ForgeArtifact } from "../lib/api";
import { formatTime } from "../lib/format";
import { useDesktopShellStore } from "../stores/desktopShellStore";
import { useUiStore } from "../stores/uiStore";
import { useWorkspaceLayoutStore } from "../stores/workspaceLayoutStore";
import { useWorkspaceStore } from "../stores/workspaceStore";

import { getShellTool, primaryShellTools } from "./shellConfig";

function corePill(core: "online" | "offline" | "unknown") {
  if (core === "online") return "forge-chip forge-chip--ok";
  if (core === "offline") return "forge-chip forge-chip--warn";
  return "forge-chip forge-chip--muted";
}

function extractJobId(pathname: string) {
  const match = pathname.match(/^\/jobs\/([^/]+)/);
  return match?.[1] ?? "";
}

function formatClock(now: number) {
  return new Intl.DateTimeFormat(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    month: "short",
    day: "numeric",
  }).format(new Date(now));
}

type AppShellProps = {
  children: ReactNode;
  isMainWindow: boolean;
};

export function AppShell(props: AppShellProps) {
  const navigate = useNavigate();
  const location = useLocation();
  const pathname = location.pathname;
  const currentTool = useMemo(() => getShellTool(pathname), [pathname]);

  const core = useWorkspaceStore((s) => s.core);
  const meta = useWorkspaceStore((s) => s.meta);
  const lastErr = useWorkspaceStore((s) => s.lastCoreError);
  const statusLine = useUiStore((s) => s.statusLine);
  const uiMode = useUiStore((s) => s.uiMode);
  const toggleUiMode = useUiStore((s) => s.toggleUiMode);
  const activeLayoutId = useWorkspaceLayoutStore((s) => s.activeLayoutId);
  const layouts = useWorkspaceLayoutStore((s) => s.layouts);
  const runtimeWindows = useWorkspaceLayoutStore((s) => s.runtimeWindows);
  const monitors = useWorkspaceLayoutStore((s) => s.monitors);
  const fallbackNotice = useWorkspaceLayoutStore((s) => s.fallbackNotice);
  const currentWindowLabel = useWorkspaceLayoutStore((s) => s.currentWindowLabel);
  const activateLayout = useWorkspaceLayoutStore((s) => s.activateLayout);
  const clearFallbackNotice = useWorkspaceLayoutStore((s) => s.clearFallbackNotice);

  const openRoutes = useDesktopShellStore((s) => s.openRoutes);
  const recentRoutes = useDesktopShellStore((s) => s.recentRoutes);
  const openRoute = useDesktopShellStore((s) => s.openRoute);
  const closeRoute = useDesktopShellStore((s) => s.closeRoute);
  const touchRoute = useDesktopShellStore((s) => s.touchRoute);

  const [summary, setSummary] = useState<DashboardSummary | null>(null);
  const [shellErr, setShellErr] = useState<string | null>(null);
  const [clockNow, setClockNow] = useState(() => Date.now());
  const isMainWindow = props.isMainWindow;

  useEffect(() => {
    openRoute(pathname);
    touchRoute(pathname);
  }, [openRoute, pathname, touchRoute]);

  useEffect(() => {
    if (!isMainWindow) return;
    let cancelled = false;
    async function loadShellState() {
      try {
        const nextSummary = await api.dashboard.summary();
        if (cancelled) return;
        setSummary(nextSummary);
        setShellErr(null);
      } catch (e) {
        if (cancelled) return;
        setShellErr(e instanceof Error ? e.message : String(e));
      }
    }
    void loadShellState();
    const id = window.setInterval(() => void loadShellState(), 5000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [isMainWindow]);

  useEffect(() => {
    const id = window.setInterval(() => setClockNow(Date.now()), 30000);
    return () => window.clearInterval(id);
  }, []);

  const chatFocused = pathname === "/chat";
  const lockPageScroll = chatFocused;
  const activeLayout = layouts.find((layout) => layout.id === activeLayoutId) ?? null;
  const currentWindowRegistration = runtimeWindows.find((item) => item.runtimeLabel === currentWindowLabel) ?? null;
  const currentMonitor = monitors.find((monitor) => monitor.id === currentWindowRegistration?.monitorId) ?? null;
  const taskItems = openRoutes.map((route) => ({ route, tool: getShellTool(route) }));
  const recentItems = recentRoutes
    .filter((route) => route !== pathname)
    .slice(0, 4)
    .map((route) => ({ route, tool: getShellTool(route) }));
  const summaryActiveJobs = Array.isArray(summary?.activeJobs) ? summary.activeJobs : [];
  const activeJobCount = summaryActiveJobs.length;
  const approvalsPending = summary?.approvalsPending ?? 0;
  const reviewsPending = summary?.reviewsPending ?? 0;
  const attentionCount = approvalsPending + reviewsPending;

  return (
    <div className="forge-shell-frame flex h-full min-h-0 flex-col text-forge-ash">
      {isMainWindow ? (
        <header className="forge-topbar">
        <div className="flex min-w-0 items-center gap-4">
          <div className="flex min-w-0 items-center gap-3">
            <div className="forge-shell-brand">
              FG
            </div>
            <div className="min-w-0">
              <div className="truncate text-[11px] font-semibold uppercase tracking-[0.22em] text-forge-mist/55">FORGE Operator Desktop</div>
              <div className="truncate text-sm text-forge-ash">{meta?.workspaceDir ?? "Workspace metadata unavailable"}</div>
            </div>
          </div>
          {!chatFocused ? (
            <div className="hidden min-w-0 items-center gap-2 xl:flex">
              <span className="forge-chip forge-chip--muted px-3 py-1.5 text-[11px]">
                <span className="mr-2 uppercase tracking-[0.16em] text-forge-mist/55">Surface</span>
                {currentTool.label}
              </span>
              <span className="forge-chip forge-chip--muted px-3 py-1.5 text-[11px]">
                <span className="mr-2 uppercase tracking-[0.16em] text-forge-mist/55">Layout</span>
                {activeLayout?.name ?? "none"}
              </span>
            </div>
          ) : null}
        </div>

        <div className="hidden min-w-0 flex-1 px-4 lg:block">
          <CommandBar compact />
        </div>

        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => navigate("/dashboard")}
            className="forge-chip forge-chip--muted hidden px-3 py-1.5 text-[10px] font-semibold uppercase tracking-[0.16em] md:inline-flex"
          >
            Dashboard
          </button>
          <select
            className="hidden forge-chip forge-chip--muted px-3 py-1.5 text-[11px] text-forge-mist outline-none lg:block"
            value={activeLayoutId ?? ""}
            onChange={(e) => void activateLayout(e.target.value)}
          >
            {layouts.map((layout) => (
              <option key={layout.id} value={layout.id}>
                {layout.name}
              </option>
            ))}
          </select>
          <button
            type="button"
            onClick={() => navigate("/layouts")}
            className="forge-chip forge-chip--muted hidden px-3 py-1.5 text-[10px] font-semibold uppercase tracking-[0.16em] md:inline-flex"
          >
            Layouts
          </button>
          <span className={["forge-chip", attentionCount > 0 ? "forge-chip--warn" : "forge-chip--muted", "text-[10px] font-semibold uppercase tracking-[0.16em]"].join(" ")}>
            Attention {attentionCount}
          </span>
          <span className={["forge-chip", corePill(core), "text-[10px] font-semibold uppercase tracking-[0.16em]"].join(" ")}>
            Core {core === "online" ? "online" : core === "offline" ? "offline" : "checking"}
          </span>
          <button
            type="button"
            onClick={() => toggleUiMode()}
            className="forge-chip forge-chip--muted hidden px-3 py-1.5 text-[10px] font-semibold uppercase tracking-[0.16em] md:inline-flex"
          >
            {uiMode === "guided" ? "Guided" : "Pro"}
          </button>
          <div className="forge-chip forge-chip--muted hidden px-3 py-1.5 text-[11px] md:block">{formatClock(clockNow)}</div>
        </div>
        </header>
      ) : null}

      <div className="flex min-h-0 flex-1 overflow-hidden">
        {!chatFocused ? (
          <aside className="forge-dock">
            <div className="flex flex-1 flex-col items-center gap-2 py-3">
              {primaryShellTools.map((tool) => {
                const active = pathname === tool.route || pathname.startsWith(`${tool.route}/`);
                return (
                  <button
                    key={tool.id}
                    type="button"
                    onClick={() => navigate(tool.route)}
                    className={["forge-dock__item", active ? "forge-dock__item--active" : ""].join(" ")}
                    title={`${tool.label} · ${tool.description}`}
                    aria-label={tool.label}
                  >
                    <span className="forge-dock__glyph">{tool.shortLabel}</span>
                    <span className="forge-dock__label">{tool.label}</span>
                  </button>
                );
              })}
            </div>
            <div className="border-t border-white/10 p-3 text-[10px] leading-relaxed text-forge-mist/60">
              {shellErr ? shellErr : statusLine || "Workspace idle."}
            </div>
          </aside>
        ) : null}

        <div className="flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
          {fallbackNotice ? (
            <div className="forge-chat-toolbar text-xs text-forge-mist">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <span>{fallbackNotice}</span>
                <button type="button" onClick={() => clearFallbackNotice()} className="forge-inline-link">
                  Dismiss
                </button>
              </div>
            </div>
          ) : null}
          <div className="forge-chat-toolbar backdrop-blur-md lg:hidden">
            <CommandBar compact />
          </div>

          {!chatFocused ? (
            <div className="forge-taskstrip">
              <div className="flex min-w-0 flex-1 items-center gap-2 overflow-x-auto pb-1">
                {taskItems.map(({ route, tool }) => {
                  const active = route === pathname;
                  return (
                    <div key={route} className={["forge-task", active ? "forge-task--active" : ""].join(" ")}>
                      <button type="button" onClick={() => navigate(route)} className="min-w-0 flex-1 truncate text-left">
                        <div className="truncate text-[11px] font-semibold uppercase tracking-[0.16em] text-forge-mist/55">{tool.label}</div>
                        <div className="truncate text-sm text-forge-ash">{describeRoute(route)}</div>
                      </button>
                      {taskItems.length > 1 ? (
                        <button
                          type="button"
                          onClick={() => {
                            closeRoute(route);
                            if (route === pathname) {
                              const fallback = taskItems.find((item) => item.route !== route)?.route ?? "/chat";
                              navigate(fallback);
                            }
                          }}
                          className="rounded-full border border-transparent px-2 py-1 text-[10px] text-forge-mist/60 transition hover:border-white/10 hover:text-forge-ash"
                          aria-label={`Close ${tool.label}`}
                        >
                          x
                        </button>
                      ) : null}
                    </div>
                  );
                })}
              </div>

              <div className="hidden items-center gap-2 xl:flex">
                <span className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] text-forge-mist">
                  {activeJobCount} active jobs
                </span>
                <span className="rounded-full border border-white/10 bg-black/30 px-3 py-1 text-[11px] text-forge-mist">
                  {attentionCount} attention items
                </span>
              </div>
            </div>
          ) : null}

          <div className="flex min-h-0 min-w-0 flex-1 overflow-hidden bg-[radial-gradient(circle_at_top_left,rgba(74,99,255,0.09),transparent_30%),linear-gradient(180deg,rgba(255,255,255,0.02),rgba(0,0,0,0))]">
            <main className="forge-desktop-surface">
              <div className="forge-window-frame">
                {!chatFocused ? (
                  <div className="forge-window-frame__head">
                    <div>
                      <div className="text-[11px] font-semibold uppercase tracking-[0.2em] text-forge-mist/55">{currentTool.label}</div>
                      <div className="mt-1 text-sm text-forge-mist/80">{currentTool.description}</div>
                    </div>
                    <div className="flex flex-wrap items-center gap-2 text-[11px] text-forge-mist/75">
                      <span className="rounded-full border border-white/10 bg-black/25 px-2.5 py-1">
                        {currentMonitor?.name ?? (currentMonitor ? `display ${currentMonitor.ordinal + 1}` : "display unknown")}
                      </span>
                      {lastErr && core === "offline" ? <span className="rounded-full border border-forge-ember/25 bg-forge-ember/10 px-2.5 py-1 text-forge-emberSoft">{lastErr}</span> : null}
                    </div>
                  </div>
                ) : null}
                <div
                  className={
                    lockPageScroll
                      ? "min-h-0 flex flex-1 overflow-hidden p-3 sm:p-4"
                      : "min-h-0 flex-1 overflow-auto px-4 py-4 sm:px-5 lg:px-6"
                  }
                >
                  {props.children}
                </div>
              </div>
            </main>

            {!chatFocused ? (
              <ShellContextPanel
                pathname={pathname}
                search={location.search}
                summary={summary}
                statusLine={statusLine}
                recentItems={recentItems}
                activeLayoutName={activeLayout?.name ?? null}
                runtimeWindows={runtimeWindows}
                monitors={monitors}
              />
            ) : null}
          </div>
        </div>
      </div>
    </div>
  );
}

function describeRoute(route: string) {
  if (route.startsWith("/jobs/")) return `Run ${route.replace("/jobs/", "")}`;
  const tool = getShellTool(route);
  return tool.primary ? tool.label : route;
}

function ShellContextPanel(props: {
  pathname: string;
  search: string;
  summary: DashboardSummary | null;
  statusLine: string;
  recentItems: Array<{ route: string; tool: ReturnType<typeof getShellTool> }>;
  activeLayoutName: string | null;
  runtimeWindows: Array<{ runtimeLabel: string; title: string; currentRoute: string; monitorId: string | null; isFocused: boolean }>;
  monitors: Array<{ id: string; ordinal: number; name: string | null }>;
}) {
  const navigate = useNavigate();
  const params = useMemo(() => new URLSearchParams(props.search), [props.search]);
  const [chatThread, setChatThread] = useState<ChatThreadDetail | null>(null);
  const [board, setBoard] = useState<CanvasBoardDetail | null>(null);
  const [artifact, setArtifact] = useState<ForgeArtifact | null>(null);
  const [jobDetail, setJobDetail] = useState<JobDetail | null>(null);
  const [dossier, setDossier] = useState<{ id: number; name: string; description: string; recentJobs: Array<{ id: string; title: string }> } | null>(null);
  const [approvals, setApprovals] = useState<ApprovalRequest[]>([]);
  const [reviews, setReviews] = useState<ReviewRecord[]>([]);
  const [loading, setLoading] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const threadMessages = Array.isArray(chatThread?.messages) ? chatThread.messages : [];
  const boardNotes = Array.isArray(board?.notes) ? board.notes : [];
  const dossierRecentJobs = Array.isArray(dossier?.recentJobs) ? dossier.recentJobs : [];
  const jobEvents = Array.isArray(jobDetail?.events) ? jobDetail.events : [];
  const summaryRecentFailures = Array.isArray(props.summary?.recentFailures) ? props.summary.recentFailures : [];

  useEffect(() => {
    let cancelled = false;
    async function load() {
      setLoading(true);
      setErr(null);
      setChatThread(null);
      setBoard(null);
      setArtifact(null);
      setJobDetail(null);
      setDossier(null);
      try {
        const tasks: Promise<void>[] = [];
        if (props.pathname === "/chat") {
          const threadId = Number(params.get("threadId"));
          if (Number.isFinite(threadId) && threadId > 0) {
            tasks.push(
              api.chat.threads.get(threadId).then((data) => {
                if (cancelled) return;
                setChatThread({
                  ...data,
                  messages: Array.isArray(data.messages) ? data.messages : [],
                });
              }),
            );
          }
        }
        if (props.pathname === "/canvas") {
          const boardId = Number(params.get("boardId"));
          if (Number.isFinite(boardId) && boardId > 0) {
            tasks.push(
              api.canvas.boards.get(boardId).then((data) => {
                if (cancelled) return;
                setBoard({
                  ...data,
                  notes: Array.isArray(data.notes) ? data.notes : [],
                });
              }),
            );
          }
        }
        if (props.pathname === "/workbench") {
          const artifactId = Number(params.get("artifactId"));
          const jobId = params.get("jobId") ?? "";
          if (Number.isFinite(artifactId) && artifactId > 0) {
            tasks.push(api.artifacts.get(artifactId).then((data) => void (!cancelled && setArtifact(data))));
          }
          if (jobId.trim()) {
            tasks.push(api.jobs.detail(jobId.trim(), 0).then((data) => void (!cancelled && setJobDetail(data))));
          }
        }
        if (props.pathname.startsWith("/jobs/")) {
          const jobId = extractJobId(props.pathname);
          if (jobId) {
            tasks.push(api.jobs.detail(jobId, 0).then((data) => void (!cancelled && setJobDetail(data))));
          }
        }
        if (props.pathname === "/dossiers") {
          const dossierId = Number(params.get("dossierId"));
          if (Number.isFinite(dossierId) && dossierId > 0) {
            tasks.push(
              api.dossiers.detail(dossierId).then((data) => {
                if (cancelled) return;
                setDossier({
                  id: data.detail.dossier.id,
                  name: data.detail.dossier.name,
                  description: data.detail.dossier.description || "",
                  recentJobs: Array.isArray(data.detail.recentJobs) ? data.detail.recentJobs.slice(0, 3).map((job) => ({ id: job.jobId, title: job.title })) : [],
                });
              }),
            );
          }
        }
        if (props.pathname === "/approvals") {
          tasks.push(api.approvals.list("pending", 8).then((data) => void (!cancelled && setApprovals(Array.isArray(data.approvals) ? data.approvals : []))));
        }
        if (props.pathname === "/reviews") {
          tasks.push(api.reviews.list({ status: "pending", limit: 8 }).then((data) => void (!cancelled && setReviews(Array.isArray(data.reviews) ? data.reviews : []))));
        }
        await Promise.all(tasks);
      } catch (e) {
        if (!cancelled) setErr(e instanceof Error ? e.message : String(e));
      } finally {
        if (!cancelled) setLoading(false);
      }
    }
    void load();
    return () => {
      cancelled = true;
    };
  }, [params, props.pathname]);

  return (
    <aside className="forge-context-panel">
      <section className="forge-context-card">
        <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Current Context</div>
        <div className="mt-2 text-sm font-semibold text-forge-ash">{getShellTool(props.pathname).label}</div>
        <div className="mt-1 text-[12px] leading-relaxed text-forge-mist/78">{getInspectorSummary(props.pathname)}</div>
        <div className="mt-3 rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-[11px] text-forge-mist/80">{props.statusLine || "No operator note recorded."}</div>
      </section>

      {loading ? <section className="forge-context-card text-sm text-forge-mist">Loading context…</section> : null}
      {err ? <section className="forge-context-card text-sm text-forge-emberSoft">{err}</section> : null}

      {chatThread ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Active Thread</div>
          <div className="mt-2 text-sm font-semibold text-forge-ash">{chatThread.title}</div>
          <div className="mt-2 text-[11px] text-forge-mist/78">{threadMessages.length} messages · updated {formatTime(chatThread.updatedAtMs)}</div>
          {chatThread.dossierId ? <div className="mt-1 text-[11px] text-forge-mist/78">Dossier {chatThread.dossierId}</div> : null}
        </section>
      ) : null}

      {board ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Active Board</div>
          <div className="mt-2 text-sm font-semibold text-forge-ash">{board.title}</div>
          <div className="mt-2 text-[11px] text-forge-mist/78">{boardNotes.length} notes · updated {formatTime(board.updatedAtMs)}</div>
          <div className="mt-3 space-y-2">
            {boardNotes.slice(0, 3).map((note) => (
              <div key={note.id} className="rounded-xl border border-white/10 bg-black/25 p-2 text-[11px] text-forge-mist/80">
                <div className="font-semibold text-forge-ash">{note.title}</div>
                <div className="mt-1 line-clamp-3">{note.body || "No note body."}</div>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {dossier ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Selected Dossier</div>
          <div className="mt-2 text-sm font-semibold text-forge-ash">{dossier.name}</div>
          <div className="mt-2 text-[11px] leading-relaxed text-forge-mist/78">{dossier.description || "No dossier description."}</div>
          <div className="mt-3 space-y-2">
            {dossierRecentJobs.length === 0 ? (
              <div className="text-[11px] text-forge-mist/78">No recent jobs linked.</div>
            ) : (
              dossierRecentJobs.map((job) => (
                <button
                  key={job.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${encodeURIComponent(job.id)}`)}
                  className="w-full rounded-xl border border-white/10 bg-black/25 p-2 text-left text-[11px] text-forge-mist transition hover:border-white/20 hover:text-forge-ash"
                >
                  <div className="font-semibold text-forge-ash">{job.title}</div>
                  <div className="mt-1">{job.id}</div>
                </button>
              ))
            )}
          </div>
        </section>
      ) : null}

      {artifact ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Selected Artifact</div>
          <div className="mt-2 text-sm font-semibold text-forge-ash">#{artifact.id} · {artifact.title}</div>
          <div className="mt-2 break-all text-[11px] text-forge-mist/78">{artifact.filePath}</div>
          <div className="mt-1 text-[11px] text-forge-mist/78">{artifact.mimeType || "unknown MIME"}</div>
          {artifact.jobId ? (
            <button type="button" className="mt-3 forge-inline-link" onClick={() => navigate(`/jobs/${encodeURIComponent(artifact.jobId as string)}`)}>
              Open source job
            </button>
          ) : null}
        </section>
      ) : null}

      {jobDetail ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Job Projection</div>
          <div className="mt-2 text-sm font-semibold text-forge-ash">{jobDetail.job.title}</div>
          <div className="mt-2 text-[11px] text-forge-mist/78">{jobDetail.job.status} · {jobDetail.job.targetAdapter} · packet {jobDetail.job.taskPacketId ?? "—"}</div>
          {jobDetail.approvalRequest ? <div className="mt-1 text-[11px] text-forge-emberSoft">Approval {jobDetail.approvalRequest.status}</div> : null}
          <div className="mt-3 space-y-2">
            {jobEvents.slice(-3).reverse().map((event) => (
              <div key={event.id} className="rounded-xl border border-white/10 bg-black/25 p-2 text-[11px] text-forge-mist/80">
                <div className="font-semibold text-forge-ash">{event.type}</div>
                <div className="mt-1">{event.message}</div>
              </div>
            ))}
          </div>
        </section>
      ) : null}

      {props.pathname === "/approvals" ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Pending Approvals</div>
          {approvals.length === 0 ? (
            <div className="mt-2 text-sm text-forge-mist">No pending requests.</div>
          ) : (
            <div className="mt-3 space-y-2">
              {approvals.map((approval) => (
                <button
                  key={approval.id}
                  type="button"
                  onClick={() => navigate(`/jobs/${encodeURIComponent(approval.jobId)}`)}
                  className="w-full rounded-xl border border-white/10 bg-black/25 p-2 text-left text-[11px] text-forge-mist transition hover:border-white/20 hover:text-forge-ash"
                >
                  <div className="font-semibold text-forge-ash">#{approval.id} · {approval.requestedAction}</div>
                  <div className="mt-1">{approval.riskClass} · {approval.requestedAdapter}</div>
                </button>
              ))}
            </div>
          )}
        </section>
      ) : null}

      {props.pathname === "/reviews" ? (
        <section className="forge-context-card">
          <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Pending Reviews</div>
          {reviews.length === 0 ? (
            <div className="mt-2 text-sm text-forge-mist">No pending reviews.</div>
          ) : (
            <div className="mt-3 space-y-2">
              {reviews.map((review) => (
                <div key={review.id} className="rounded-xl border border-white/10 bg-black/25 p-2 text-[11px] text-forge-mist/80">
                  <div className="font-semibold text-forge-ash">#{review.id} · {review.targetType}:{review.targetId}</div>
                  <div className="mt-1 line-clamp-3">{review.summary || "No summary."}</div>
                </div>
              ))}
            </div>
          )}
        </section>
      ) : null}

      <section className="forge-context-card">
        <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Workspace State</div>
        <div className="mt-3 grid gap-2">
          <MetricRow label="Active layout" value={props.activeLayoutName ?? "none"} />
          <MetricRow label="Active jobs" value={String(props.summary?.activeJobs?.length ?? 0)} />
          <MetricRow label="Approvals" value={String(props.summary?.approvalsPending ?? 0)} />
          <MetricRow label="Reviews" value={String(props.summary?.reviewsPending ?? 0)} />
          <MetricRow label="Recent failures" value={String(summaryRecentFailures.length)} />
        </div>
      </section>

      <section className="forge-context-card">
        <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Workspace Windows</div>
        <div className="mt-3 space-y-2">
          {props.runtimeWindows.length === 0 ? (
            <div className="text-sm text-forge-mist">No window registrations yet.</div>
          ) : (
            props.runtimeWindows.map((windowRecord) => {
              const monitor = props.monitors.find((item) => item.id === windowRecord.monitorId);
              return (
                <div key={windowRecord.runtimeLabel} className="rounded-xl border border-white/10 bg-black/25 px-3 py-2 text-[11px] text-forge-mist/80">
                  <div className="font-semibold text-forge-ash">{windowRecord.title}</div>
                  <div className="mt-1">{windowRecord.runtimeLabel}</div>
                  <div className="mt-1">{windowRecord.currentRoute} · {monitor?.name ?? (monitor ? `display ${monitor.ordinal + 1}` : "display unknown")}</div>
                  <div className="mt-1">{windowRecord.isFocused ? "focused" : "background"}</div>
                </div>
              );
            })
          )}
        </div>
      </section>

      <section className="forge-context-card">
        <div className="text-[10px] font-semibold uppercase tracking-[0.18em] text-forge-mist/45">Recent Surfaces</div>
        <div className="mt-3 space-y-2">
          {props.recentItems.length === 0 ? (
            <div className="text-sm text-forge-mist">No recent surfaces beyond the active tool.</div>
          ) : (
            props.recentItems.map(({ route, tool }) => (
              <button
                key={route}
                type="button"
                onClick={() => navigate(route)}
                className="w-full rounded-xl border border-white/10 bg-black/25 px-3 py-2 text-left text-[11px] text-forge-mist transition hover:border-white/20 hover:text-forge-ash"
              >
                <div className="font-semibold text-forge-ash">{tool.label}</div>
                <div className="mt-1 truncate">{describeRoute(route)}</div>
              </button>
            ))
          )}
        </div>
      </section>
    </aside>
  );
}

function MetricRow(props: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between rounded-xl border border-white/10 bg-black/20 px-3 py-2 text-[11px] text-forge-mist/80">
      <span>{props.label}</span>
      <span className="font-semibold text-forge-ash">{props.value}</span>
    </div>
  );
}

function getInspectorSummary(pathname: string) {
  if (pathname === "/chat") return "Conversation thread, packet launch, and assistant status tied to persisted thread state.";
  if (pathname === "/workbench") return "Artifact selection and job-linked file inspection using recorded artifact metadata.";
  if (pathname === "/canvas") return "Board note layout and working memory pinned to persisted board coordinates.";
  if (pathname === "/dossiers") return "Project profile, recent job lineage, and routing/approval preferences.";
  if (pathname === "/jobs" || pathname.startsWith("/jobs/")) return "Execution truth from job projections, approval gates, events, and artifacts.";
  if (pathname === "/reviews") return "Explicit operator review records and import reconciliation status.";
  if (pathname === "/approvals") return "Pending risk gates and recorded decisions. No silent escalation.";
  if (pathname === "/autonomy") return "Dream-state telemetry, autonomy intents, policy decisions, budgets, and charter boundaries.";
  if (pathname === "/settings") return "Local model, retrieval, and workspace configuration persisted by the core.";
  return "Workspace surface details and recent operator context.";
}
