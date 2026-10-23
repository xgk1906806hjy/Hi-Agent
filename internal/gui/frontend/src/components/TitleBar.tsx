import { useEffect, useRef, useState } from "react";
import { useActions, useStore } from "../store";
import { nextTheme, readTheme, saveTheme, themeLabel, type ThemeChoice } from "../theme";
import { AppIcon } from "./AppIcon";

type MenuKey = "file" | "edit" | "view" | "help" | null;

export function TitleBar() {
  const status = useStore((s) => s.status);
  const connected = useStore((s) => s.connected);
  const { submit, addProject, newSession, setTheme, windowMinimise, windowToggleMaximise, windowClose } =
    useActions();
  const [theme, setThemeChoice] = useState<ThemeChoice>(readTheme);
  const [menu, setMenu] = useState<MenuKey>(null);
  const [about, setAbout] = useState(false);
  const [maximised, setMaximised] = useState(false);
  const barRef = useRef<HTMLElement>(null);

  useEffect(() => {
    const dark = saveTheme(theme);
    if (connected) setTheme(dark ? "dark" : "light");
    if (theme !== "system") return;
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    const onChange = () => {
      const next = saveTheme("system");
      if (connected) setTheme(next ? "dark" : "light");
    };
    mq.addEventListener("change", onChange);
    return () => mq.removeEventListener("change", onChange);
  }, [theme, connected, setTheme]);

  useEffect(() => {
    if (!menu) return;
    const onDoc = (e: MouseEvent) => {
      if (barRef.current?.contains(e.target as Node)) return;
      setMenu(null);
    };
    document.addEventListener("mousedown", onDoc);
    return () => document.removeEventListener("mousedown", onDoc);
  }, [menu]);

  const model = status?.model || "—";
  const provider = status?.provider || "";
  const session = status?.session || "";
  const tokens = status
    ? `${status.ctxTokens.toLocaleString()}${status.ctxEstimated ? "~" : ""} / ${status.ctxWindow.toLocaleString()}`
    : "";

  const run = (fn: () => void) => {
    setMenu(null);
    fn();
  };

  return (
    <>
      <header
        ref={barRef}
        data-wails-drag
        onDoubleClick={() => {
          windowToggleMaximise();
          setMaximised((v) => !v);
        }}
        className="relative z-40 flex h-10 shrink-0 items-center gap-1 border-b border-line bg-bg pl-3 select-none"
      >
        <div data-wails-no-drag className="flex items-center gap-2">
          <AppIcon size={16} />
          <span className="text-[12.5px] font-semibold tracking-tight">Hi-agent</span>
        </div>

        <nav data-wails-no-drag className="ml-2 flex items-center gap-0.5">
          <MenuButton label="File" open={menu === "file"} onOpen={() => setMenu(menu === "file" ? null : "file")}>
            <Item onClick={() => run(newSession)}>New Session</Item>
            <Item onClick={() => run(() => void addProject())}>Open Project…</Item>
            <Item onClick={() => run(() => submit("/save"))}>Save</Item>
            <Sep />
            <Item onClick={() => run(windowClose)}>Exit</Item>
          </MenuButton>
          <MenuButton label="Edit" open={menu === "edit"} onOpen={() => setMenu(menu === "edit" ? null : "edit")}>
            <Item onClick={() => run(() => submit("/undo"))}>Undo File Write</Item>
          </MenuButton>
          <MenuButton label="View" open={menu === "view"} onOpen={() => setMenu(menu === "view" ? null : "view")}>
            <Item
              onClick={() =>
                run(() => {
                  const next = nextTheme(theme);
                  setThemeChoice(next);
                })
              }
            >
              Theme · {themeLabel[theme]}
            </Item>
            <Item onClick={() => run(() => submit("/permissions"))}>Permissions</Item>
            <Item onClick={() => run(() => submit("/sessions"))}>Sessions</Item>
          </MenuButton>
          <MenuButton label="Help" open={menu === "help"} onOpen={() => setMenu(menu === "help" ? null : "help")}>
            <Item onClick={() => run(() => submit("/help"))}>Help</Item>
            <Item
              onClick={() =>
                run(() => {
                  setAbout(true);
                })
              }
            >
              About
            </Item>
          </MenuButton>
        </nav>

        <div className="min-w-0 flex-1" />

        <div data-wails-no-drag className="mr-1 flex min-w-0 items-center gap-1.5">
          {session ? (
            <span className="hidden max-w-[140px] truncate rounded-full border border-line bg-surface px-2 py-0.5 text-[10.5px] text-muted sm:inline">
              {session}
            </span>
          ) : null}
          {tokens ? <span className="hidden text-[10.5px] tabular-nums text-faint md:inline">{tokens}</span> : null}
          <button
            type="button"
            title="切换主题"
            onClick={() => setThemeChoice((c) => nextTheme(c))}
            className="rounded-full border border-line bg-surface px-2 py-0.5 text-[10.5px] font-medium text-muted hover:text-ink"
          >
            {themeLabel[theme]}
          </button>
          <span className="hidden rounded-full border border-[#d6e2ff] bg-accent-soft px-2 py-0.5 text-[10.5px] font-medium text-accent dark:border-accent/30 sm:inline">
            {provider ? `${provider} / ${model}` : model}
          </span>
        </div>

        <div data-wails-no-drag className="flex h-full">
          <WinBtn title="最小化" onClick={windowMinimise}>
            <span className="mb-1.5 block h-px w-2.5 bg-current" />
          </WinBtn>
          <WinBtn
            title={maximised ? "还原" : "最大化"}
            onClick={() => {
              windowToggleMaximise();
              setMaximised((v) => !v);
            }}
          >
            {maximised ? (
              <span className="relative h-2 w-2.5">
                <span className="absolute right-0 top-0 h-1.5 w-2 border border-current" />
                <span className="absolute bottom-0 left-0 h-1.5 w-2 border border-current bg-bg" />
              </span>
            ) : (
              <span className="block h-2.5 w-2.5 border border-current" />
            )}
          </WinBtn>
          <WinBtn title="关闭" danger onClick={windowClose}>
            <svg width="10" height="10" viewBox="0 0 10 10" aria-hidden>
              <path d="M1 1l8 8M9 1L1 9" stroke="currentColor" strokeWidth="1.2" />
            </svg>
          </WinBtn>
        </div>
      </header>

      {about ? (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-ink/20 p-4 backdrop-blur-[2px]" data-wails-no-drag>
          <div className="w-full max-w-sm rounded-2xl border border-line bg-surface p-5 shadow-[0_24px_60px_rgba(40,35,25,0.14)]">
            <div className="flex items-center gap-3">
              <AppIcon size={36} />
              <div>
                <div className="text-sm font-semibold tracking-tight">Hi-agent</div>
                <div className="text-xs text-muted">本地桌面 AI coding agent</div>
              </div>
            </div>
            <p className="mt-3 text-[13px] leading-relaxed text-muted">
              精密纸面界面 · Wails + React。菜单项对应会话、项目与斜杠命令。
            </p>
            <div className="mt-4 flex justify-end">
              <button
                type="button"
                onClick={() => setAbout(false)}
                className="rounded-xl bg-accent px-3.5 py-1.5 text-sm font-medium text-white hover:bg-accent/90"
              >
                好的
              </button>
            </div>
          </div>
        </div>
      ) : null}
    </>
  );
}

function MenuButton({
  label,
  open,
  onOpen,
  children,
}: {
  label: string;
  open: boolean;
  onOpen: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="relative">
      <button
        type="button"
        onClick={onOpen}
        className={`rounded px-2 py-1 text-[12.5px] ${open ? "bg-line/60 text-ink" : "text-ink hover:bg-line/45"}`}
      >
        {label}
      </button>
      {open ? (
        <div className="absolute left-0 top-full z-50 mt-0.5 min-w-[180px] rounded-xl border border-line bg-surface py-1 shadow-lg">
          {children}
        </div>
      ) : null}
    </div>
  );
}

function Item({ children, onClick }: { children: React.ReactNode; onClick: () => void }) {
  return (
    <button
      type="button"
      onClick={onClick}
      className="flex w-full px-3 py-1.5 text-left text-[12.5px] text-ink hover:bg-accent-soft hover:text-accent"
    >
      {children}
    </button>
  );
}

function Sep() {
  return <div className="my-1 border-t border-line" />;
}

function WinBtn({
  children,
  title,
  onClick,
  danger,
}: {
  children: React.ReactNode;
  title: string;
  onClick: () => void;
  danger?: boolean;
}) {
  return (
    <button
      type="button"
      title={title}
      onClick={onClick}
      className={`flex h-full w-11 items-center justify-center text-ink/80 ${
        danger ? "hover:bg-[#e81123] hover:text-white" : "hover:bg-line/70"
      }`}
    >
      {children}
    </button>
  );
}
