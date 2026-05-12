import { useMemo, useState, type ReactNode } from "react";

export type MessagePart =
  | { type: "text"; text: string }
  | { type: "code"; text: string; lang: string };

export function parseMessageParts(content: string): MessagePart[] {
  const out: MessagePart[] = [];
  const re = /```([\w-]+)?\n([\s\S]*?)```/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(content)) !== null) {
    if (m.index > last) {
      out.push({ type: "text", text: content.slice(last, m.index) });
    }
    out.push({ type: "code", text: m[2] ?? "", lang: (m[1] ?? "").trim() });
    last = re.lastIndex;
  }
  if (last < content.length)
    out.push({ type: "text", text: content.slice(last) });
  if (out.length === 0) return [{ type: "text", text: content }];
  return out;
}

function renderInline(text: string): ReactNode[] {
  const chunks = text.split(/(`[^`]+`)/g);
  const nodes: ReactNode[] = [];
  chunks.forEach((chunk, i) => {
    if (!chunk) return;
    if (chunk.startsWith("`") && chunk.endsWith("`") && chunk.length >= 2) {
      nodes.push(
        <code
          key={`inline-${i}`}
          className="rounded border border-forge-platinum/15 bg-black/35 px-1.5 py-0.5 font-mono text-[12px] text-forge-ash"
        >
          {chunk.slice(1, -1)}
        </code>,
      );
      return;
    }
    const links = chunk.split(/(https?:\/\/[^\s]+)/g);
    links.forEach((part, j) => {
      if (!part) return;
      if (/^https?:\/\//.test(part)) {
        nodes.push(
          <a
            key={`link-${i}-${j}`}
            href={part}
            target="_blank"
            rel="noreferrer"
            className="text-forge-emberSoft underline underline-offset-2 hover:text-forge-ash"
          >
            {part}
          </a>,
        );
      } else {
        nodes.push(<span key={`txt-${i}-${j}`}>{part}</span>);
      }
    });
  });
  return nodes;
}

export function RichMessage(props: { content: string }) {
  const parts = useMemo(
    () => parseMessageParts(props.content),
    [props.content],
  );
  return (
    <div className="min-w-0 max-w-full space-y-3 overflow-hidden">
      {parts.map((part, idx) => {
        if (part.type === "code") {
          return (
            <CodeBlock key={`code-${idx}`} code={part.text} lang={part.lang} />
          );
        }
        const paragraphs = part.text
          .split(/\n{2,}/)
          .map((v) => v.trimEnd())
          .filter((v) => v.length > 0);
        if (paragraphs.length === 0) return null;
        return (
          <div
            key={`text-${idx}`}
            className="min-w-0 max-w-full space-y-2 text-[15px] leading-7 text-forge-ash"
          >
            {paragraphs.map((p, pIdx) => (
              <p
                key={`p-${idx}-${pIdx}`}
                className="whitespace-pre-wrap break-words"
              >
                {renderInline(p)}
              </p>
            ))}
          </div>
        );
      })}
    </div>
  );
}

export function CodeBlock(props: { code: string; lang: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(props.code);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch {
      setCopied(false);
    }
  }

  return (
    <div className="overflow-hidden rounded-xl border border-forge-platinum/10 bg-forge-carbon">
      <div className="flex items-center justify-between border-b border-forge-platinum/10 px-3 py-1.5 text-[11px] text-forge-mist">
        <span className="uppercase tracking-[0.12em]">
          {props.lang || "code"}
        </span>
        <button
          type="button"
          onClick={() => void copy()}
          className="rounded border border-forge-platinum/10 bg-forge-platinum/5 px-2 py-1 text-[10px] text-forge-mist transition hover:bg-forge-platinum/10"
        >
          {copied ? "Copied" : "Copy"}
        </button>
      </div>
      <pre className="forge-chat-scroll overflow-x-auto px-3 py-3 text-[12px] leading-6 text-forge-ash">
        <code>{props.code}</code>
      </pre>
    </div>
  );
}
