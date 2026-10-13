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
    <aside className="flex w-64 shrink-0 flex-col border-r border-line bg-panel">
      <div className="flex items-center justify-between px-3 pb-1 pt-3">
        <span className="text-[11px] font-semibold uppercase tracking-wider text-muted">项目</span>
        <button
          type="button"
          title={locked ? lockReason : "添加项目"}
          disabled={locked}
          onClick={() => void addProject()}
          className="rounded p-1 text-muted hover:bg-line/70 hover:text-ink disabled:cursor-not-allowed disabled:opacity-40"
        >
          <FolderPlus size={15} strokeWidth={1.75} />
        </button>
      </div>
      <TodoList items={status?.todos ?? []} />
      <div className={`min-h-0 flex-1 overflow-y-auto px-1.5 pb-3 ${locked ? "opacity-70" : ""}`} title={locked ? lockReason : undefined}>
        {nav.projects.length === 0 ? (
          <p className="px-2 py-3 text-xs text-muted">点 + 添加项目文件夹</p>
        ) : (
          nav.projects.map((p) => (
            <div key={p.path} className="mb-2">
              <div
                className={`group flex items-center gap-1 rounded-md px-2 py-1.5 text-[13px] ${
                  p.current ? "bg-surface font-medium shadow-sm ring-1 ring-line" : "hover:bg-line/50"
                }`}
              >
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
                <ul className="ml-2 mt-0.5 border-l border-line pl-2">
                  {sortSessions(p.sessions).map((s) => (
                    <SessionRow key={s.id} project={p} session={s} locked={locked} cwd={cwd} />
                  ))}
                </ul>
              ) : null}
            </div>
          ))
        )}
      </div>
      {cwd ? (
        <div className="border-t border-line px-3 py-2 text-[11px] leading-snug text-muted" title={cwd}>
          <div className="truncate">{cwd}</div>
        </div>
      ) : null}
    </aside>
  );
}

function SessionRow({ project, session, locked, cwd }: { project: NavProject; session: NavSession; locked: boolean; cwd: string }) {
  const { openSession, renameSession, deleteSession } = useActions();
  const [editing, setEditing] = useState(false);
  const [draft, setDraft] = useState(session.id);
  const active = session.current && project.current;
  const editable = project.current && project.path === cwd;

  const commit = () => {
    const next = draft.trim();
    setEditing(false);
    if (next && next !== session.id) renameSession(session.id, next);
  };
  const onKey = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") commit();
    if (e.key === "Escape") setEditing(false);
  };

  if (editing) {
    return (
      <li className="flex items-center gap-1 px-1 py-0.5">
        <input
          autoFocus
          value={draft}
          onChange={(e) => setDraft(e.target.value.replace(/\s/g, ""))}
          onKeyDown={onKey}
          onBlur={commit}
          className="min-w-0 flex-1 rounded border border-accent/50 bg-surface px-1.5 py-0.5 text-[12.5px] outline-none"
        />
        <IconButton title="取消" onMouseDown={(e) => e.preventDefault()} onClick={() => setEditing(false)}>
          <X size={12} />
        </IconButton>
      </li>
    );
  }

  return (
    <li className="group/s relative">
      <button
        type="button"
        disabled={locked}
        onClick={() => openSession(project.path, session.id)}
        title={session.id}
        className={`flex w-full flex-col rounded px-2 py-1 text-left disabled:cursor-not-allowed ${
          active ? "bg-accent-soft text-accent" : "text-muted hover:bg-line/40 hover:text-ink"
        }`}
      >
        <span className={`truncate pr-10 text-[12.5px] ${active ? "font-medium" : ""}`}>{session.title || session.id}</span>
        <span className="flex gap-1.5 text-[10.5px] opacity-70">
          <span>{relativeTime(session.updatedAt)}</span>
          {session.messages > 0 ? <span className="tabular-nums">· {session.messages} 条</span> : null}
        </span>
      </button>
      {editable && !locked ? (
        <div className="absolute right-1 top-1 hidden gap-0.5 group-hover/s:flex">
          <IconButton
            title="重命名"
            onClick={() => {
              setDraft(session.id);
              setEditing(true);
            }}
          >
            <Pencil size={12} />
          </IconButton>
          <IconButton title="删除" danger onClick={() => deleteSession(session.id)}>
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
      className={`rounded p-0.5 text-muted hover:bg-line/60 disabled:cursor-not-allowed disabled:opacity-40 ${
        danger ? "hover:text-danger" : "hover:text-ink"
      }`}
    >
      {children}
    </button>
  );
}

function TodoList({ items }: { items: TodoItem[] }) {
  if (items.length === 0) return null;
  return (
    <div className="border-b border-line px-3 py-2">
      <div className="text-[11px] font-semibold uppercase tracking-wider text-muted">任务</div>
      <ul className="mt-1 space-y-0.5">
        {items.map((t) => (
          <li key={t.id} className="flex items-baseline gap-1.5 text-[12.5px] leading-snug" title={t.content}>
            <span className={todoMarkClass(t.status)}>{todoMark(t.status)}</span>
            <span className={`min-w-0 flex-1 truncate ${t.status === "completed" || t.status === "cancelled" ? "text-muted line-through" : ""}`}>
              {t.content}
            </span>
          </li>
        ))}
      </ul>
    </div>
  );
}

function todoMark(status: string): string {
  if (status === "in_progress") return "●";
  if (status === "completed") return "✓";
  if (status === "cancelled") return "–";
  return "○";
}

function todoMarkClass(status: string): string {
  if (status === "in_progress") return "text-accent";
  if (status === "completed") return "text-add";
  return "text-muted";
}

function sortSessions(list: NavSession[]): NavSession[] {
  const stamp = (s: NavSession) => s.updatedAt || Number.MAX_SAFE_INTEGER;
  return [...list].sort((a, b) => stamp(b) - stamp(a) || a.id.localeCompare(b.id));
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
