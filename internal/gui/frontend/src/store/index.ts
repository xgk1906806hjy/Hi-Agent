// 单一状态源：bridge 事件写入，组件只读并通过 actions 发起操作。
import { create } from "zustand";
import type { Bridge, ConfirmPayload, Nav, SelectPayload, SlashCmd, Status } from "../bridge";
import { appendDelta, appendStep, endTurn, fromTurns, pushText, pushUser, type Item } from "./thread";

export type Dialog = ({ type: "confirm" } & ConfirmPayload) | ({ type: "select" } & SelectPayload);

interface State {
  connected: boolean;
  items: Item[];
  busy: boolean;
  status: Status | null;
  nav: Nav;
  prompt: string;
  slash: SlashCmd[];
  dialog: Dialog | null;
}

interface Actions {
  submit(text: string): void;
  stop(): void;
  answer(value: number): void;
  addProject(): Promise<void>;
  openProject(path: string): void;
  removeProject(path: string): void;
  openSession(path: string, id: string): void;
  newSession(): void;
  renameSession(oldId: string, newId: string): void;
  deleteSession(id: string): void;
}

const noop = () => {};
let bridge: Bridge | null = null;

export const useStore = create<State & { actions: Actions }>((set, get) => ({
  connected: false,
  items: [],
  busy: false,
  status: null,
  nav: { projects: [] },
  prompt: "",
  slash: [],
  dialog: null,
  actions: {
    submit: (text) => bridge?.submit(text),
    stop: () => bridge?.stop(),
    answer(value) {
      const d = get().dialog;
      if (!d) return;
      set({ dialog: null });
      bridge?.answer(d.id, value);
    },
    async addProject() {
      if (!bridge) {
        set((s) => ({ items: pushText(s.items, "error", "尚未连接引擎，无法添加项目") }));
        return;
      }
      try {
        await bridge.addProject();
      } catch (e) {
        const msg = e instanceof Error ? e.message : String(e);
        set((s) => ({ items: pushText(s.items, "error", "添加项目失败：" + msg) }));
      }
    },
    openProject: (path) => bridge?.openProject(path),
    removeProject(path) {
      void bridge?.removeProject(path).then(() =>
        set((s) => ({ nav: { projects: s.nav.projects.filter((p) => p.path !== path) } })),
      );
    },
    openSession: (path, id) => bridge?.openSession(path, id),
    newSession: () => bridge?.newSession(),
    renameSession: (o, n) => bridge?.renameSession(o, n),
    deleteSession: (id) => bridge?.deleteSession(id),
  },
}));

/** 接上 bridge：先注册全部监听，再 ready() 让 Go 冲刷排队事件。 */
export async function attachBridge(b: Bridge): Promise<() => void> {
  bridge = b;
  const set = useStore.setState;
  const offs = [
    b.on("chat:user", (p) => set((s) => ({ items: pushUser(s.items, p.text) }))),
    b.on("chat:delta", (p) => set((s) => ({ items: appendDelta(s.items, p.text) }))),
    b.on("chat:progress", (p) => set((s) => ({ items: appendStep(s.items, p.text) }))),
    b.on("chat:info", (p) => set((s) => ({ items: pushText(s.items, "info", p.text) }))),
    b.on("chat:notice", (p) => set((s) => ({ items: pushText(s.items, "notice", p.text) }))),
    b.on("chat:error", (p) => set((s) => ({ items: pushText(s.items, "error", p.text) }))),
    b.on("chat:reset", (p) => set({ items: fromTurns(p.turns ?? []) })),
    b.on("turn:begin", () => set({ busy: true })),
    b.on("turn:end", () => set((s) => ({ busy: false, items: endTurn(s.items) }))),
    b.on("status", (p) => set({ status: p })),
    b.on("nav", (p) => set({ nav: p })),
    b.on("input:prompt", (p) => set({ prompt: p.prompt })),
    b.on("dialog:confirm", (p) => set({ dialog: { type: "confirm", ...p } })),
    b.on("dialog:select", (p) => set({ dialog: { type: "select", ...p } })),
    b.on("dialog:dismiss", (p) => set((s) => (s.dialog?.id === p.id ? { dialog: null } : {}))),
  ];
  const snap = await b.ready();
  set({
    connected: true,
    status: snap.status,
    nav: snap.nav ?? { projects: [] },
    prompt: snap.prompt ?? "",
    slash: snap.slash ?? [],
  });
  return () => offs.forEach((off) => (off ?? noop)());
}

export const useActions = () => useStore((s) => s.actions);
