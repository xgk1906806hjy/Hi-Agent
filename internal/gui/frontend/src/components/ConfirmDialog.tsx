import { useActions, useStore } from "../store";

export function ConfirmDialog() {
  const dialog = useStore((s) => s.dialog);
  const { answer } = useActions();
  if (!dialog || dialog.type !== "confirm") return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/25 p-4 backdrop-blur-[1px]">
      <div role="dialog" className="w-full max-w-md rounded-xl border border-line bg-surface p-5 shadow-xl">
        <h2 className="text-sm font-semibold">确认</h2>
        <p className="mt-3 whitespace-pre-wrap text-[14px] leading-relaxed text-ink/90">{dialog.prompt}</p>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={() => answer(0)}
            className="rounded-lg px-3.5 py-1.5 text-sm text-muted hover:bg-panel"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => answer(1)}
            className="rounded-lg bg-accent px-3.5 py-1.5 text-sm font-medium text-white hover:bg-accent/90"
          >
            允许
          </button>
        </div>
      </div>
    </div>
  );
}
