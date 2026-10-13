import { useEffect, useState } from "react";
import { useActions, useStore } from "../store";

export function SelectDialog() {
  const dialog = useStore((s) => s.dialog);
  const { answer } = useActions();
  const [idx, setIdx] = useState(0);
  const select = dialog?.type === "select" ? dialog : null;

  useEffect(() => {
    if (select) setIdx(select.selected ?? 0);
  }, [select?.id]);

  if (!select) return null;
  const selected = idx;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/25 p-4 backdrop-blur-[1px]">
      <div role="dialog" className="w-full max-w-md rounded-xl border border-line bg-surface p-5 shadow-xl">
        <h2 className="text-sm font-semibold">{select.title}</h2>
        <ul className="mt-3 max-h-72 space-y-1 overflow-y-auto">
          {select.items.map((item, i) => (
            <li key={i}>
              <button
                type="button"
                onClick={() => setIdx(i)}
                onDoubleClick={() => answer(i)}
                className={`w-full rounded-lg px-3 py-2 text-left text-[14px] ${
                  i === selected ? "bg-accent-soft font-medium text-accent" : "hover:bg-panel"
                }`}
              >
                {item}
              </button>
            </li>
          ))}
        </ul>
        <div className="mt-5 flex justify-end gap-2">
          <button
            type="button"
            onClick={() => answer(-1)}
            className="rounded-lg px-3.5 py-1.5 text-sm text-muted hover:bg-panel"
          >
            取消
          </button>
          <button
            type="button"
            onClick={() => answer(selected)}
            className="rounded-lg bg-accent px-3.5 py-1.5 text-sm font-medium text-white hover:bg-accent/90"
          >
            确定
          </button>
        </div>
      </div>
    </div>
  );
}
