import { useState } from "react";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";

export function ToolGroup({ steps, running }: { steps: string[]; running: boolean }) {
  const [open, setOpen] = useState(running);
  const summary = running ? `过程中 · ${steps.length}` : `过程 · ${steps.length}`;

  return (
    <div className="my-0.5 overflow-hidden rounded-xl border border-line bg-surface text-[13px]">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-2 border-b border-line px-3 py-2 text-left text-muted hover:text-ink"
      >
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        {running ? (
          <Loader2 size={13} className="animate-spin text-accent" />
        ) : (
          <span className="inline-block h-1.5 w-1.5 rounded-full bg-accent" />
        )}
        <span className="font-medium tracking-tight">{summary}</span>
        {!open && steps.length > 0 ? (
          <span className="ml-1 truncate text-[12px] text-faint">{steps[steps.length - 1]}</span>
        ) : null}
      </button>
      {open ? (
        <ul className="space-y-1 bg-[#faf9f6] px-3 py-2.5 font-mono text-[11.5px] leading-relaxed text-muted dark:bg-bg/50">
          {steps.map((s, i) => (
            <li key={i}>{s}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
