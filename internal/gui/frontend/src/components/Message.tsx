import { lazy, Suspense } from "react";
import { Loader2 } from "lucide-react";
import { splitThink } from "../store/thread";
import { ToolGroup } from "./ToolGroup";
import type { Item } from "../store/thread";

const Markdown = lazy(() => import("./Markdown"));

export function Message({ item }: { item: Item }) {
  if (item.kind === "user") {
    return (
      <div className="flex justify-end">
        <div className="max-w-[78%] whitespace-pre-wrap rounded-[16px] rounded-br-[5px] border border-[#dbe4ff] bg-user px-3.5 py-2.5 text-[14px] leading-relaxed tracking-tight text-ink dark:border-accent/25">
          {item.text}
        </div>
      </div>
    );
  }

  if (item.kind === "tools") {
    return <ToolGroup steps={item.steps} running={item.running} />;
  }

  if (item.kind === "info" || item.kind === "notice" || item.kind === "error") {
    const tone =
      item.kind === "error" ? "text-danger" : item.kind === "notice" ? "text-warn" : "text-muted";
    return (
      <div className={`whitespace-pre-wrap px-1 text-[13px] leading-relaxed ${tone}`}>{item.text}</div>
    );
  }

  if (item.kind !== "assistant") return null;
  const { think, body, thinking } = splitThink(item.text);
  return (
    <div className="max-w-[92%]">
      {think || thinking ? (
        <details className="mb-2 rounded-xl border border-line bg-surface px-3 py-2 text-[12.5px] text-muted">
          <summary className="flex cursor-pointer select-none items-center gap-1.5 font-medium tracking-tight">
            {thinking ? <Loader2 size={12} className="animate-spin text-accent" /> : null}
            {thinking ? "思考中…" : "思考过程"}
            <span className="text-[11px] font-normal text-faint">（点击展开）</span>
          </summary>
          <pre className="mt-2 whitespace-pre-wrap font-sans leading-relaxed opacity-90">{think.trim()}</pre>
        </details>
      ) : null}
      {body ? (
        <div className="prose prose-sm max-w-none tracking-tight prose-headings:mb-2 prose-headings:mt-3 prose-headings:tracking-tight prose-p:my-1.5 prose-pre:my-2 prose-ul:my-1.5 prose-ol:my-1.5">
          <Suspense fallback={<div className="whitespace-pre-wrap">{body}</div>}>
            <Markdown text={body} />
          </Suspense>
          {item.streaming ? <span className="ml-0.5 inline-block h-3.5 w-1.5 animate-pulse bg-accent align-middle" /> : null}
        </div>
      ) : item.streaming && !thinking ? (
        <span className="inline-block h-3.5 w-1.5 animate-pulse bg-accent" />
      ) : null}
    </div>
  );
}
