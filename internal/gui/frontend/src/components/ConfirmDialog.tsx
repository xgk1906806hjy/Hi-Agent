import { useActions, useStore } from "../store";

export function ConfirmDialog() {
  const dialog = useStore((s) => s.dialog);
  const { answer } = useActions();
  if (!dialog || dialog.type !== "confirm") return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/20 p-4 backdrop-blur-[2px]">
      <div role="dialog" className="w-full max-w-xl rounded-2xl border border-line bg-surface p-5 shadow-[0_24px_60px_rgba(40,35,25,0.14)]">
        <h2 className="text-sm font-semibold tracking-tight">确认</h2>
        <ConfirmBody text={dialog.prompt} />
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={() => answer(0)}
            className="rounded-xl px-3.5 py-1.5 text-sm text-muted hover:bg-panel"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => answer(1)}
            className="rounded-xl bg-accent px-3.5 py-1.5 text-sm font-medium text-white shadow-[inset_0_1px_0_rgba(255,255,255,0.25)] hover:bg-accent/90"
          >
            允许
          </button>
        </div>
      </div>
    </div>
  );
}

function ConfirmBody({ text }: { text: string }) {
  const lines = text.split("\n");
  const diff = lines.some((l) => l.startsWith("--- diff ---") || l.startsWith("+") || l.startsWith("-"));
  if (!diff) {
    return <p className="mt-3 whitespace-pre-wrap text-[14px] leading-relaxed text-ink/90">{text}</p>;
  }
  return (
    <pre className="mt-3 max-h-80 overflow-auto rounded-lg bg-panel px-3 py-2 font-mono text-[12.5px] leading-relaxed">
      {lines.map((line, i) => (
        <div key={i} className={diffLineClass(line)}>
          {line || " "}
        </div>
      ))}
    </pre>
  );
}

function diffLineClass(line: string): string {
  if (line.startsWith("+") && !line.startsWith("+++")) return "text-add";
  if (line.startsWith("-") && !line.startsWith("---")) return "text-del";
  if (line.startsWith("---")) return "text-muted";
  return "text-ink/90";
}
