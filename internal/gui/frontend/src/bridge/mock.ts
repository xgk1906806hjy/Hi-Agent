// 浏览器里 `npm run dev` 用的假引擎：不连 Go，只模拟事件，方便调样式与交互。
import type { Bridge, EventMap, EventName, Nav, Snapshot, Status } from "./types";
import { MAIN_PROMPT } from "./types";

type Handler = (payload: never) => void;

export function createMockBridge(): Bridge {
  const handlers = new Map<string, Set<Handler>>();
  const emit = <E extends EventName>(event: E, payload?: EventMap[E]) => {
    handlers.get(event)?.forEach((h) => (h as (p: unknown) => void)(payload));
  };

  let status: Status = {
    model: "gpt-4o-mini",
    provider: "openai",
    session: "default",
    workDir: "E:\\workspace\\Hi-agent",
    busy: false,
    ctxTokens: 1830,
    ctxEstimated: false,
    ctxWindow: 128000,
    turnPrompt: 0,
    turnCompletion: 0,
    messages: 0,
    policy: ["配置文件：无", "内置默认：只读工具 allow，run_shell / write / patch ask"],
    todos: [],
    skills: [],
    mcp: [],
  };
  let nav: Nav = {
    projects: [
      {
        name: "Hi-agent",
        path: "E:\\workspace\\Hi-agent",
        current: true,
        sessions: [
          { id: "default", title: "", updatedAt: 0, messages: 0, current: true },
          { id: "3fa9c2e1", title: "帮我看看项目结构", updatedAt: Date.now() - 3600_000, messages: 12, current: false },
        ],
      },
      {
        name: "notes",
        path: "D:\\notes",
        current: false,
        sessions: [{ id: "a1b2c3d4", title: "整理读书笔记", updatedAt: Date.now() - 86400_000 * 2, messages: 4, current: false }],
      },
    ],
  };
  const slash = [
    { name: "/model", desc: "配置供应商与模型" },
    { name: "/help", desc: "显示帮助" },
    { name: "/new", desc: "新建会话" },
    { name: "/open", desc: "打开会话" },
    { name: "/rename", desc: "重命名会话" },
    { name: "/delete", desc: "删除会话" },
    { name: "/confirm", desc: "（预览）弹确认框" },
  ];
  const mapCurrent = (fn: (s: Nav["projects"][number]["sessions"]) => Nav["projects"][number]["sessions"]) => {
    nav = { projects: nav.projects.map((p) => (p.current ? { ...p, sessions: fn(p.sessions) } : p)) };
    emit("nav", nav);
  };
  let dialogSeq = 0;
  const pending = new Map<string, (v: number) => void>();
  let timer: number | undefined;

  const pushStatus = (patch: Partial<Status>) => {
    status = { ...status, ...patch };
    emit("status", status);
  };

  const reply = async (text: string) => {
    emit("chat:user", { text });
    emit("input:prompt", { prompt: "" });
    emit("turn:begin");
    pushStatus({ busy: true });
    const steps = ["[调用工具 list_dir] .", "[调用工具 read_file] README.md（2.1 KB）"];
    for (const s of steps) {
      await sleep(350);
      emit("chat:progress", { text: s });
    }
    const answer =
      `<think>用户问的是「${text}」，先看看项目结构。</think>好的，下面是简要说明：\n\n` +
      "## 项目结构\n\n- `cmd/hi-agent`：入口与 REPL\n- `internal/chat`：对话与工具循环\n\n" +
      "```go\nfunc main() {\n\tfmt.Println(\"hi\")\n}\n```\n\n| 模块 | 说明 |\n|---|---|\n| ui | 界面抽象 |\n| gui | 桌面界面 |\n";
    let i = 0;
    await new Promise<void>((done) => {
      timer = window.setInterval(() => {
        if (i >= answer.length) {
          window.clearInterval(timer);
          done();
          return;
        }
        emit("chat:delta", { text: answer.slice(i, i + 6) });
        i += 6;
      }, 16);
    });
    finish();
  };

  const finish = () => {
    window.clearInterval(timer);
    emit("turn:end");
    pushStatus({ busy: false, turnPrompt: 2410, turnCompletion: 380, messages: status.messages + 2 });
    emit("input:prompt", { prompt: MAIN_PROMPT });
  };

  const ask = (event: "dialog:confirm" | "dialog:select", payload: object) =>
    new Promise<number>((resolve) => {
      const id = `m${++dialogSeq}`;
      pending.set(id, resolve);
      emit(event, { id, ...payload } as never);
    });

  return {
    async ready(): Promise<Snapshot> {
      setTimeout(() => emit("chat:info", { text: "工作目录 E:\\workspace\\Hi-agent · openai / gpt-4o-mini · 会话 default\n（浏览器预览模式：未连接 Go 引擎）" }), 50);
      return { status, nav, prompt: MAIN_PROMPT, slash };
    },
    submit(text) {
      if (text === "/confirm") {
        void ask("dialog:confirm", { prompt: "允许执行 run_shell：go test ./... ？" }).then((v) =>
          emit("chat:info", { text: v === 1 ? "已允许" : "已拒绝" }),
        );
        return;
      }
      if (text === "/model") {
        void ask("dialog:select", { title: "选择模型", items: ["gpt-4o-mini", "gpt-4.1", "deepseek-chat"], selected: 0 }).then((v) => {
          if (v >= 0) pushStatus({ model: ["gpt-4o-mini", "gpt-4.1", "deepseek-chat"][v] });
        });
        return;
      }
      if (text.startsWith("/")) {
        emit("chat:info", { text: `（预览模式）命令 ${text}` });
        return;
      }
      void reply(text);
    },
    stop() {
      if (status.busy) {
        finish();
        emit("chat:notice", { text: "（已中断当前回复，可继续提问）" });
      }
    },
    answer(id, v) {
      pending.get(id)?.(v);
      pending.delete(id);
    },
    async addProject() {
      const path =
        window.prompt("输入项目路径（预览模式）", "D:\\demo-project")?.trim() ?? "";
      if (!path) return "";
      const name = path.split(/[/\\]/).filter(Boolean).pop() || path;
      const existing = nav.projects.find((p) => p.path.toLowerCase() === path.toLowerCase());
      const rest = nav.projects.filter((p) => p.path.toLowerCase() !== path.toLowerCase());
      const added = existing
        ? { ...existing, name, path, current: true }
        : { name, path, current: true, sessions: [{ id: "default", title: "", updatedAt: 0, messages: 0, current: true }] };
      nav = {
        projects: [
          added,
          ...rest.map((p) => ({
            ...p,
            current: false,
            sessions: p.sessions.map((s) => ({ ...s, current: false })),
          })),
        ],
      };
      emit("nav", nav);
      pushStatus({ workDir: path, session: added.sessions.find((s) => s.current)?.id ?? "default" });
      emit("chat:reset", { turns: [] });
      emit("chat:info", { text: `（预览）已添加项目：${path}` });
      return path;
    },
    openProject(path) {
      nav = { projects: nav.projects.map((p) => ({ ...p, current: p.path === path })) };
      emit("nav", nav);
      pushStatus({ workDir: path });
      emit("chat:reset", { turns: [] });
    },
    async removeProject(path) {
      nav = { projects: nav.projects.filter((p) => p.path !== path) };
      emit("nav", nav);
    },
    openSession(path, id) {
      nav = {
        projects: nav.projects.map((p) => ({
          ...p,
          current: p.path === path,
          sessions: p.sessions.map((s) => ({ ...s, current: p.path === path && s.id === id })),
        })),
      };
      emit("nav", nav);
      pushStatus({ session: id, workDir: path });
      emit("chat:reset", {
        turns: [
          { role: "user", text: "之前的问题" },
          { role: "tool", text: "[调用工具 read_file] main.go" },
          { role: "assistant", text: "之前的**回答**。" },
        ],
      });
    },
    newSession() {
      const id = Math.random().toString(16).slice(2, 10);
      nav = {
        projects: nav.projects.map((p) => ({
          ...p,
          sessions: p.current
            ? [...p.sessions.map((s) => ({ ...s, current: false })), { id, title: "", updatedAt: 0, messages: 0, current: true }]
            : p.sessions,
        })),
      };
      emit("nav", nav);
      pushStatus({ session: id, messages: 0 });
      emit("chat:reset", { turns: [] });
    },
    renameSession(_path, oldId, newId) {
      mapCurrent((list) => list.map((s) => (s.id === oldId ? { ...s, id: newId } : s)));
      if (status.session === oldId) pushStatus({ session: newId });
    },
    deleteSession(_path, id) {
      void ask("dialog:confirm", { prompt: `确认删除会话 ${id}？此操作不可撤销。` }).then((v) => {
        if (v !== 1) return;
        mapCurrent((list) => list.filter((s) => s.id !== id));
      });
    },
    setTheme() {},
    windowMinimise() {},
    windowToggleMaximise() {},
    windowClose() {},
    on(event, fn) {
      const set = handlers.get(event) ?? new Set();
      set.add(fn as Handler);
      handlers.set(event, set);
      return () => set.delete(fn as Handler);
    },
  };
}

function sleep(ms: number) {
  return new Promise((r) => setTimeout(r, ms));
}
