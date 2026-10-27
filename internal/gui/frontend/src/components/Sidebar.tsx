import { useState, type KeyboardEvent } from "react";
import { FolderPlus, MessageSquarePlus, Pencil, Trash2, X } from "lucide-react";
import { MAIN_PROMPT, type NavProject, type NavSession, type TodoItem } from "../bridge";
import { useActions, useStore } from "../store";

export function Sidebar() {
  const nav = useStore((s) => s.nav);
  const status = useStore((s) => s.status);
  const busy = useStore((s) => s.busy);
  const prompt = useStore((s) => s.prompt);
  const connected = useStore((s) => s.connected);
  const { addProject, openProject, removeProject, newSession } = useActions();
  const cwd = status?.workDir ?? "";

  const locked = !connected || busy || prompt !== MAIN_PROMPT;
  const lockReason = !connected ? "未连接引擎" : busy ? "生成中，请稍候" : "请先完成当前输入";

  return (
    <aside className="flex w-[268px] shrink-0 flex-col border-r border-line bg-panel">
      <div className="flex items-center justify-between px-4 pb-2 pt-[18px]">
        <span className="text-[11px] font-semibold uppercase tracking-[0.06em] text-faint">项目</span>
        <button
          type="button"
          title={locked ? lockReason : "添加项目"}
          disabled={locked}
          onClick={() => void addProject()}
          className="flex h-7 w-7 items-center justify-center rounded-lg border border-line-strong bg-surface text-muted shadow-[0_1px_0_rgba(255,255,255,0.8)] hover:text-ink disabled:cursor-not-allowed disabled:opacity-40 dark:shadow-none"
        >
          <FolderPlus size={14} strokeWidth={1.75} />
        </button>
      </div>
      <TodoList items={status?.todos ?? []} />
      <SkillList items={status?.skills ?? []} />
      <MCPList items={status?.mcp ?? []} />
      <div className={`min-h-0 flex-1 overflow-y-auto px-2.5 pb-3 ${locked ? "opacity-70" : ""}`} title={locked ? lockReason : undefined}>
        {nav.projects.length === 0 ? (
          <p className="px-2 py-3 text-xs text-muted">点 + 添加项目文件夹</p>
        ) : (
          nav.projects.map((p) => (
            <div key={p.path} className="mb-2.5">
              <div
                className={`group flex items-center gap-2 rounded-[10px] px-2.5 py-2 text-[13px] ${
                  p.current
                    ? "border border-line bg-surface font-semibold shadow-[0_1px_0_rgba(255,255,255,0.9)] dark:shadow-none"
                    : "hover:bg-line/40"
                }`}
              >
                {p.current ? <span className="h-2 w-2 shrink-0 rounded-full bg-accent" /> : null}
                <button
                  type="button"
                  disabled={locked}
                  className="min-w-0 flex-1 truncate text-left disabled:cursor-not-allowed"
                  title={p.path}
                  onClick={() => openProject(p.path)}
                >
                  {p.name}
                </button>
                {p.current ? (
                  <IconButton title="新建会话" disabled={locked} onClick={newSession}>
                    <MessageSquarePlus size={14} strokeWidth={1.75} />
                  </IconButton>
                ) : (
                  <IconButton title="从列表移除" disabled={locked} danger onClick={() => removeProject(p.path)}>
                    <Trash2 size={13} strokeWidth={1.75} />
                  </IconButton>
                )}
              </div>
              {p.sessions.length > 0 ? (
                <ul className="ml-3 mt-1.5">
                  {sortSessions(p.sessions).map((s) => (
                    <SessionRow key={s.id} project={p} session={s} locked={locked} />
                  ))}
                </ul>
              ) : null}
            </div>
          ))
        )}
      </div>
      {cwd ? (
        <div className="border-t border-line px-4 py-3 font-mono text-[10.5px] leading-snug text-faint" title={cwd}>
          <div className="truncate">{cwd}</div>
        </div>
      ) : null}
    </aside>
  );
}

function SessionRow({ project, session, locked }: { project: NavProject; session: NavSession; locked: boolean }) {
  const { openSession, renameSession, deleteSession } = useActions();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(session.id);
  const active = session.current && project.current;

  const commit = () => {
    const next = draft.trim();
    setEditing(false);
    if (next && next !== session.id) renameSession(project.path, session.id, next);
  };
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") commit();
    if (e.key === "Escape") setEditing(false);
  };

  if (editing) {
    return (
      <li className="mb-0.5 flex items-center gap-1 px-1 py-0.5">
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value.replace(/\s/g, ""))}
          onKeyDown={onKey}
          onBlur={commit}
          className="min-w-0 flex-1 rounded-lg border border-accent/40 bg-surface px-2 py-1 text-[12.5px] outline-none"
        />
        <IconButton title="取消" onMouseDown={(e) => e.preventDefault()} onClick={() => setEditing(false)}>
          <X size={12} />
        </IconButton>
      </li>
    );
  }

  return (
    <li className="group/s relative mb-0.5">
      <button
        type="button"
        disabled={locked}
        onClick={() => openSession(project.path, session.id)}
        title={session.id}
        className={`flex w-full flex-col rounded-[10px] px-2.5 py-2 text-left disabled:cursor-not-allowed ${
          active
            ? "border border-line bg-surface text-ink shadow-[0_1px_2px_rgba(40,35,25,0.04)] dark:shadow-none"
            : "text-muted hover:bg-line/35 hover:text-ink"
        }`}
      >
        <span className={`truncate pr-10 text-[12.5px] tracking-tight ${active ? "font-medium" : ""}`}>
          {session.title || session.id}
        </span>
        <span className="mt-0.5 flex gap-1.5 text-[11px] text-faint">
          <span>{relativeTime(session.updatedAt)}</span>
          {session.messages > 0 ? <span className="tabular-nums">· {session.messages} 条</span> : null}
        </span>
      </button>
      {!locked ? (
        <div className="absolute right-1.5 top-1.5 hidden gap-0.5 group-hover/s:flex">
          <IconButton
            title="重命名"
            onClick={() => {
              setDraft(session.id);
              setEditing(true);
            }}
          >
            <Pencil size={12} />
          </IconButton>
          <IconButton title="删除" danger onClick={() => deleteSession(project.path, session.id)}>
            <Trash2 size={12} />
          </IconButton>
        </div>
      ) : null}
    </li>
  );
}

function IconButton({
  children,
  title,
  disabled,
  danger,
  onClick,
  onMouseDown,
}: {
  children: React.ReactNode;
  title: string;
  disabled?: boolean;
  danger?: boolean;
  onClick: () => void;
  onMouseDown?: (e: React.MouseEvent) => void;
}) {
  return (
    <button
      type="button"
      title={title}
      disabled={disabled}
      onMouseDown={onMouseDown}
      onClick={(e) => {
        e.stopPropagation();
        onClick();
      }}
      className={`rounded-md p-1 text-muted hover:bg-line/60 disabled:cursor-not-allowed disabled:opacity-40 ${
        danger ? "hover:text-danger" : "hover:text-ink"
      }`}
    >
      {children}
    </button>
  );
}

function SkillList({ items }: { items: string[] }) {
  if (items.length === 0) return null;
  return (
    <div className="mx-3 mb-3.5 rounded-xl border border-line bg-surface px-3 py-2.5 shadow-[0_1px_0_rgba(255,255,255,0.9)] dark:shadow-none">
      <div className="mb-2 text-[11px] font-semibold text-muted">技能</div>
      <div className="flex flex-wrap gap-1.5">
        {items.map((name) => (
          <span key={name} className="rounded-full bg-accent/10 px-2 py-0.5 text-[11px] font-medium text-accent" title={name}>
            {name}
          </span>
        ))}
      </div>
    </div>
  );
}

function MCPList({ items }: { items: string[] }) {
  if (items.length === 0) return null;
  return (
    <div className="mx-3 mb-3.5 rounded-xl border border-line bg-surface px-3 py-2.5 shadow-[0_1px_0_rgba(255,255,255,0.9)] dark:shadow-none">
      <div className="mb-2 text-[11px] font-semibold text-muted">MCP</div>
      <ul className="space-y-1">
        {items.map((line) => (
          <li key={line} className="truncate text-[11px] leading-snug text-muted" title={line}>
            {line}
          </li>
        ))}
      </ul>
    </div>
  );
}

function TodoList({ items }: { items: TodoItem[] }) {
  if (items.length === 0) return null;
  const busy = items.some((t) => t.status === "in_progress");
  return (
    <div className="mx-3 mb-3.5 rounded-xl border border-line bg-surface p-3 shadow-[0_1px_0_rgba(255,255,255,0.9)] dark:shadow-none">
      <div className="mb-2.5 flex items-center justify-between text-[11px] font-semibold text-muted">
        <span>任务</span>
        {busy ? <span className="rounded-full bg-[#e8f5ec] px-1.5 py-0.5 text-[10px] font-semibold text-add dark:bg-add/15">进行中</span> : null}
      </div>
      <ul className="space-y-1">
        {items.map((t) => (
          <li key={t.id} className="grid grid-cols-[16px_1fr] gap-2 text-[12.5px] leading-snug" title={t.content}>
            <span className={`mt-0.5 h-3.5 w-3.5 rounded border-[1.5px] ${todoBoxClass(t.status)}`} />
            <span className={`min-w-0 truncate ${t.status === "completed" || t.status === "cancelled" ? "text-muted line-through" : ""}`}>
              {t.content}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function todoBoxClass(status: string): string {
  if (status === "in_progress" || status === "completed") {
    return "border-add bg-add shadow-[inset_0_0_0_2px_#fff] dark:shadow-[inset_0_0_0_2px_var(--color-surface)]";
  }
  return "border-line-strong bg-transparent";
}

function sortSessions(list: NavSession[]): NavSession[] {
  return [...list].sort((a, b) => (b.updatedAt || 0) - (a.updatedAt || 0) || a.id.localeCompare(b.id));
}

export function relativeTime(ms: number): string {
  if (!ms) return "未保存";
  const diff = Math.max(0, Date.now() - ms) / 1000;
  if (diff < 60) return "刚刚";
  if (diff < 3600) return `${Math.floor(diff / 60)} 分钟前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} 小时前`;
  if (diff < 86400 * 30) return `${Math.floor(diff / 86400)} 天前`;
  const d = new Date(ms);
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
}
