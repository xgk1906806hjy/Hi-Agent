import { useEffect, useMemo, useRef, useState, type KeyboardEvent } from "react";
import { Square, ArrowUp } from "lucide-react";
import { MAIN_PROMPT, type SlashCmd } from "../bridge";
import { useActions, useStore } from "../store";

export function Composer() {
  const busy = useStore((s) => s.busy);
  const prompt = useStore((s) => s.prompt);
  const slash = useStore((s) => s.slash);
  const { submit, stop } = useActions();
  const [text, setText] = useState("");
  const [index, setIndex] = useState(0);
  const [dismissed, setDismissed] = useState(false);
  const ta = useRef<HTMLTextAreaElement>(null);
  const waiting = prompt !== "" && prompt !== MAIN_PROMPT;

  const matches = useMemo(() => matchSlash(slash, text), [slash, text]);
  const showMenu = !waiting && !dismissed && matches.length > 0;

  useEffect(() => {
    const el = ta.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = Math.min(el.scrollHeight, 180) + "px";
  }, [text]);

  const change = (v: string) => {
    setText(v);
    setIndex(0);
    setDismissed(false);
  };

  const send = () => {
    const t = text.trim();
    if (!t || busy) return;
    change("");
    submit(t);
  };

  const complete = (cmd: SlashCmd) => {
    change(cmd.name + " ");
    setDismissed(true);
    ta.current?.focus();
  };

  const onKey = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    if (showMenu) {
      if (e.key === "ArrowDown" || e.key === "ArrowUp") {
        e.preventDefault();
        const step = e.key === "ArrowDown" ? 1 : -1;
        setIndex((i) => (i + step + matches.length) % matches.length);
        return;
      }
      if (e.key === "Escape") {
        e.preventDefault();
        setDismissed(true);
        return;
      }
      const picked = matches[Math.min(index, matches.length - 1)];
      if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey && picked.name !== text.trim())) {
        e.preventDefault();
        complete(picked);
        return;
      }
    }
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      if (busy) return;
      send();
    }
  };

  return (
    <div className="shrink-0 border-t border-line bg-surface px-4 py-3">
      <div className="relative mx-auto max-w-3xl">
        {showMenu ? (
          <ul className="absolute bottom-full left-0 z-20 mb-2 max-h-64 w-72 overflow-y-auto rounded-xl border border-line bg-panel py-1 shadow-lg">
            {matches.map((c, i) => (
              <li key={c.name}>
                <button
                  type="button"
                  onMouseDown={(e) => {
                    e.preventDefault();
                    complete(c);
                  }}
                  onMouseEnter={() => setIndex(i)}
                  className={`flex w-full items-baseline gap-3 px-3 py-1.5 text-left text-[13px] ${
                    i === index ? "bg-accent/10 text-ink" : "text-ink/80"
                  }`}
                >
                  <span className="font-mono">{c.name}</span>
                  <span className="truncate text-xs text-muted">{c.desc}</span>
                </button>
              </li>
            ))}
          </ul>
        ) : null}
        {waiting ? <div className="mb-1.5 px-1 text-xs text-muted">{prompt}</div> : null}
        <div className="flex items-end gap-2 rounded-2xl border border-line bg-panel px-3 py-2 shadow-sm focus-within:border-accent/50 focus-within:ring-2 focus-within:ring-accent/15">
          <textarea
            ref={ta}
            rows={1}
            value={text}
            onChange={(e) => change(e.target.value)}
            onKeyDown={onKey}
            placeholder={busy ? "生成中… Shift+Enter 换行" : "发消息，或输入 / 查看命令 …"}
            className="max-h-[180px] min-h-[24px] flex-1 resize-none bg-transparent py-1.5 text-[14px] leading-6 outline-none placeholder:text-muted/70"
          />
          {busy ? (
            <button
              type="button"
              title="停止"
              onClick={stop}
              className="mb-0.5 flex h-8 w-8 items-center justify-center rounded-full bg-ink text-surface hover:bg-ink/85"
            >
              <Square size={12} fill="currentColor" />
            </button>
          ) : (
            <button
              type="button"
              title="发送"
              disabled={!text.trim()}
              onClick={send}
              className="mb-0.5 flex h-8 w-8 items-center justify-center rounded-full bg-accent text-white disabled:cursor-not-allowed disabled:opacity-35 hover:bg-accent/90"
            >
              <ArrowUp size={16} strokeWidth={2.25} />
            </button>
          )}
        </div>
        <div className="mt-1.5 flex gap-2 px-1 text-[11px] text-muted">
          {["/model", "/sessions", "/help"].map((c) => (
            <button
              key={c}
              type="button"
              disabled={busy}
              onClick={() => submit(c)}
              className="rounded px-1 py-0.5 hover:bg-line/60 hover:text-ink disabled:opacity-40"
            >
              {c}
            </button>
          ))}
          <span className="ml-auto">Enter 发送 · Shift+Enter 换行 · / 命令补全</span>
        </div>
      </div>
    </div>
  );
}

function matchSlash(list: SlashCmd[], text: string): SlashCmd[] {
  if (!text.startsWith("/") || /\s/.test(text)) return [];
  const q = text.toLowerCase();
  return list.filter((c) => c.name.toLowerCase().startsWith(q));
}
