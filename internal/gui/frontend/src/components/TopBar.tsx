import { useStore } from "../store";

export function TopBar() {
  const status = useStore((s) => s.status);
  const model = status?.model || "—";
  const provider = status?.provider || "";
  const session = status?.session || "";
  const tokens = status
    ? `${status.ctxTokens.toLocaleString()}${status.ctxEstimated ? "~" : ""} / ${status.ctxWindow.toLocaleString()}`
    : "";

  return (
    <header className="flex h-12 shrink-0 items-center justify-between border-b border-line bg-surface px-4">
      <div className="flex min-w-0 items-baseline gap-2">
        <span className="text-[15px] font-semibold tracking-tight">Hi-agent</span>
        {session ? <span className="truncate text-xs text-muted">· {session}</span> : null}
      </div>
      <div className="flex items-center gap-2">
        {tokens ? <span className="hidden text-xs text-muted sm:inline">{tokens}</span> : null}
        <span className="rounded-full bg-accent-soft px-2.5 py-1 text-xs font-medium text-accent">
          {provider ? `${provider} / ${model}` : model}
        </span>
      </div>
    </header>
  );
}
