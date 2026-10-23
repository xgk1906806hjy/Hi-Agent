import { useState } from "react";
import { ChevronDown, ChevronRight, Loader2 } from "lucide-react";

export function ToolGroup({ steps, running }: { steps: string[]; running: boolean }) {
  // 默认折叠：运行中也不自动展开；摘要始终单行，避免被预览挤成逐字换行。
  const [open, setOpen] = useState(false);

  return (
    <div className="my-0.5 overflow-hidden rounded-lg border border-line/80 bg-surface/60 text-[12.5px]">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-expanded={open}
        className="flex w-full items-center gap-1.5 px-2.5 py-1.5 text-left text-muted hover:bg-line/20 hover:text-ink"
      >
        {open ? <ChevronDown size={13} className="shrink-0" /> : <ChevronRight size={13} className="shrink-0" />}
        {running ? (
          <Loader2 size={12} className="shrink-0 animate-spin text-accent" />
        ) : (
          <span className="h-1.5 w-1.5 shrink-0 rounded-full bg-accent/60" />
        )}
        <span className="shrink-0 whitespace-nowrap font-medium tracking-tight text-ink/75">
          {running ? "过程中" : "过程"}
        </span>
        <span className="shrink-0 rounded-full bg-line/50 px-1.5 py-px text-[10.5px] font-medium tabular-nums text-faint">
          {steps.length}
        </span>
        {!open && steps.length > 0 ? (
          <span className="min-w-0 flex-1 truncate font-mono text-[11px] leading-5 text-faint">
            {steps[steps.length - 1]}
          </span>
        ) : (
          <span className="flex-1" />
        )}
      </button>
      {open ? (
        <ul className="space-y-1 border-t border-line/70 bg-[#faf9f6] px-3 py-2 font-mono text-[11px] leading-relaxed text-muted dark:bg-bg/50">
          {steps.map((s, i) => (
            <li key={i}>{s}</li>
          ))}
        </ul>
      ) : null}
    </div>
  );
}
