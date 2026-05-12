import type { ChatThreadSummary } from "../../lib/api";
import { formatTime } from "../../lib/format";

function trimLine(value: string, fallback: string) {
  const trimmed = value.trim();
  return trimmed ? trimmed : fallback;
}

type ChatThreadRailProps = {
  threads: ChatThreadSummary[];
  filteredThreads: ChatThreadSummary[];
  activeThreadId: number | null;
  threadFilter: string;
  busy: boolean;
  error: string | null;
  onNewThread: () => void;
  onLoadThread: (threadId: number) => void;
  onThreadFilterChange: (value: string) => void;
};

export function ChatThreadRail(props: ChatThreadRailProps) {
  return (
    <aside className="forge-chat-thread-rail flex min-h-0 flex-col overflow-hidden border-b border-forge-platinum/10 bg-forge-black/95 shadow-[8px_0_40px_rgba(0,0,0,0.28)] md:border-b-0 md:border-r">
      <div className="border-b border-forge-platinum/10 bg-forge-carbon/75 p-3">
        <div className="mb-2 flex items-end justify-between gap-3">
          <div>
            <div className="text-[10px] uppercase tracking-[0.18em] text-forge-mist/65">
              Threads
            </div>
            <div className="mt-1 text-sm font-semibold text-forge-ash">
              {props.threads.length} active conversations
            </div>
          </div>
          <div className="rounded-full border border-forge-platinum/10 bg-forge-platinum/5 px-2.5 py-1 text-[10px] text-forge-mist/80">
            {props.filteredThreads.length} shown
          </div>
        </div>
        <button
          type="button"
          onClick={props.onNewThread}
          disabled={props.busy}
          className="forge-chat-primary-btn w-full px-3 py-2 text-sm"
        >
          + New chat
        </button>
        <input
          aria-label="Filter chats"
          value={props.threadFilter}
          onChange={(e) => props.onThreadFilterChange(e.target.value)}
          placeholder="Search chats"
          className="forge-input mt-3 text-xs"
        />
        {props.error ? (
          <div className="mt-2 rounded border border-forge-ember/30 bg-forge-ember/10 p-2 text-xs text-forge-ash">
            {props.error}
          </div>
        ) : null}
      </div>

      <div className="forge-chat-scroll min-h-0 flex-1 overflow-y-auto p-1.5">
        {props.filteredThreads.length === 0 ? (
          <div className="rounded-lg border border-dashed border-forge-platinum/15 bg-black/35 px-3 py-4 text-xs text-forge-mist">
            <div className="font-semibold text-forge-ash">No chats shown</div>
            <div className="mt-1 leading-5 text-forge-mist/75">
              Start a chat or clear the search filter.
            </div>
          </div>
        ) : (
          <div className="space-y-1">
            {props.filteredThreads.map((thread) => {
              const isActive = props.activeThreadId === thread.id;
              return (
                <button
                  key={thread.id}
                  type="button"
                  onClick={() => props.onLoadThread(thread.id)}
                  className={[
                    "w-full rounded-lg border px-3 py-2 text-left transition shadow-[inset_0_1px_0_rgba(255,255,255,0.025)]",
                    isActive
                      ? "border-forge-ember/45 bg-forge-ember/10 text-forge-ash shadow-[inset_3px_0_0_rgba(255,122,51,0.85)]"
                      : "border-transparent bg-transparent text-forge-mist hover:border-forge-ember/25 hover:bg-forge-ember/5",
                  ].join(" ")}
                >
                  <div className="truncate text-sm font-semibold">
                    {trimLine(thread.title, `Thread #${thread.id}`)}
                  </div>
                  <div className="mt-1 text-[10px] text-forge-mist/70">
                    {formatTime(thread.updatedAtMs)}
                  </div>
                </button>
              );
            })}
          </div>
        )}
      </div>
    </aside>
  );
}
