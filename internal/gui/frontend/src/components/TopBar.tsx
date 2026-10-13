import { useState } from "react";
import { useStore } from "../store";

export function TopBar() {
  const status = useStore((s) => s.status);
  const [policyOpen, setPolicyOpen] = useState(false);
  const model = status?.model || "—";
  const provider = status?.provider || "";
  const session = status?.session || "";
  const tokens = status
    ? `${status.ctxTokens.toLocaleString()}${status.ctxEstimated ? "~" : ""} / ${status.ctxWindow.toLocaleString()}`
    : "";

  return (
    <header className="relative flex h-12 shrink-0 items-center justify-between border-b border-line bg-surface px-4">
      <div className="flex min-w-0 items-baseline gap-2">
        <span className="text-[15px] font-semibold tracking-tight">Hi-agent</span>
        {session ? <span className="truncate text-xs text-muted">· {session}</span> : null}
      </div>
      <div className="flex items-center gap-2">
        {tokens ? <span className="hidden text-xs text-muted sm:inline">{tokens}</span> : null}
        <button
          type="button"
          onClick={() => setPolicyOpen((v) => !v)}
          className="rounded-full px-2 py-1 text-xs text-muted hover:bg-panel hover:text-ink"
        >
          权限
        </button>
        <span className="rounded-full bg-accent-soft px-2.5 py-1 text-xs font-medium text-accent">
          {provider ? `${provider} / ${model}` : model}
        </span>
      </div>
      {policyOpen ? (
        <div className="absolute right-4 top-12 z-30 w-96 max-w-[80vw] rounded-xl border border-line bg-surface p-3 text-xs leading-relaxed text-ink/90 shadow-lg">
          {(status?.policy ?? []).length === 0 ? (
            <p className="text-muted">权限规则尚未同步</p>
          ) : (
            (status?.policy ?? []).map((line, i) => (
              <div key={i} className="whitespace-pre-wrap">
                {line}
              </div>
            ))
          )}
        </div>
      ) : null}
    </header>
  );
}
