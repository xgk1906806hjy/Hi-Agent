import { useEffect, useRef } from "react";
import { useStore } from "../store";
import { Message } from "./Message";

export function Thread() {
  const items = useStore((s) => s.items);
  const busy = useStore((s) => s.busy);
  const bottom = useRef<HTMLDivElement>(null);
  const stick = useRef(true);

  useEffect(() => {
    if (stick.current) bottom.current?.scrollIntoView({ behavior: busy ? "auto" : "smooth" });
  }, [items, busy]);

  return (
    <div
      className="min-h-0 flex-1 overflow-y-auto px-5 py-4"
      onScroll={(e) => {
        const el = e.currentTarget;
        stick.current = el.scrollHeight - el.scrollTop - el.clientHeight < 80;
      }}
    >
      <div className="mx-auto flex max-w-3xl flex-col gap-3">
        {items.length === 0 ? (
          <p className="py-16 text-center text-sm text-muted">输入消息开始对话，或从左侧打开会话</p>
        ) : (
          items.map((it) => <Message key={it.id} item={it} />)
        )}
        <div ref={bottom} />
      </div>
    </div>
  );
}
