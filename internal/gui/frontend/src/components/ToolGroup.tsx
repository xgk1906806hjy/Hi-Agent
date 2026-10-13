import { useState } from "react";
import { ChevronDown, ChevronRight, Loader2, Wrench } from "lucide-react";

export function ToolGroup({ steps, running }: { steps: string[]; running: boolean }) {
  const [open, setOpen] = useState(running);
  const summary = running ? `过程中 · ${steps.length}` : `过程 · ${steps.length}`;

  return (
    <div className="my-1 rounded-lg border border-line bg-panel/80 text-[13px]">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center gap-1.5 px-2.5 py-1.5 text-left text-muted hover:text-ink"
      >
        {open ? <ChevronDown size={14} /> : <ChevronRight size={14} />}
        {running ? <Loader2 size={13} className="animate-spin text-accent" /> : <Wrench size={13} />}
        <span className="font-medium">{summary}</span>
        {!open && steps.length > 0 ? (
          <span className="ml-1 truncate text-[12px] opacity-70">{steps[steps.length - 1]}</span>
        ) : null}
      </button>
      {open ? (
        <ul className="space-y-1 border-t border-line px-3 py-2 font-mono text-[12px] text-muted">
          {steps.map((s, i) => (
            <li key={i} className="leading-relaxed">
              {s}
            </li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
