// 与 internal/gui/protocol.go 一一对应；改协议时两边同步。

export interface TextPayload {
  text: string;
}

export interface TurnPayload {
  role: "user" | "assistant" | "tool" | string;
  text: string;
}

export interface ResetPayload {
  turns: TurnPayload[];
}

export interface Status {
  model: string;
  provider: string;
  session: string;
  workDir: string;
  busy: boolean;
  ctxTokens: number;
  ctxEstimated: boolean;
  ctxWindow: number;
  turnPrompt: number;
  turnCompletion: number;
  messages: number;
  policy: string[];
  todos: TodoItem[];
  skills: string[];
}

export interface TodoItem {
  id: string;
  content: string;
  status: "pending" | "in_progress" | "completed" | "cancelled" | string;
}

export interface NavSession {
  id: string;
  title: string;
  /** Unix 毫秒；0 表示未落盘 */
  updatedAt: number;
  messages: number;
  current: boolean;
}

export interface SlashCmd {
  name: string;
  desc: string;
}

export interface NavProject {
  name: string;
  path: string;
  current: boolean;
  sessions: NavSession[];
}

export interface Nav {
  projects: NavProject[];
}

export interface ConfirmPayload {
  id: string;
  prompt: string;
}

export interface SelectPayload {
  id: string;
  title: string;
  items: string[];
  selected: number;
}

export interface DismissPayload {
  id: string;
}

export interface PromptPayload {
  prompt: string;
}

export interface Snapshot {
  status: Status;
  nav: Nav;
  prompt: string;
  slash: SlashCmd[];
}

export interface EventMap {
  "chat:user": TextPayload;
  "chat:delta": TextPayload;
  "chat:progress": TextPayload;
  "chat:notice": TextPayload;
  "chat:info": TextPayload;
  "chat:error": TextPayload;
  "chat:reset": ResetPayload;
  "turn:begin": void;
  "turn:end": void;
  status: Status;
  nav: Nav;
  "dialog:confirm": ConfirmPayload;
  "dialog:select": SelectPayload;
  "dialog:dismiss": DismissPayload;
  "input:prompt": PromptPayload;
}

export type EventName = keyof EventMap;

/** 界面与 Go 引擎之间的唯一通道；组件不直接接触 Wails 运行时。 */
export interface Bridge {
  /** 注册完监听后调用，返回当前快照，Go 侧随后开始推送事件。 */
  ready(): Promise<Snapshot>;
  submit(text: string): void;
  stop(): void;
  answer(id: string, value: number): void;
  /** 打开系统文件夹对话框；取消返回空串。 */
  addProject(): Promise<string>;
  openProject(path: string): void;
  removeProject(path: string): Promise<void>;
  openSession(path: string, id: string): void;
  newSession(): void;
  renameSession(path: string, oldId: string, newId: string): void;
  deleteSession(path: string, id: string): void;
  /** 把原生窗口背景设为 light 或 dark，与页面主题一致。 */
  setTheme(appearance: "light" | "dark"): void;
  windowMinimise(): void;
  windowToggleMaximise(): void;
  windowClose(): void;
  on<E extends EventName>(event: E, fn: (payload: EventMap[E]) => void): () => void;
}

/** REPL 主输入提示；其它提示表示命令正在向用户索取参数。 */
export const MAIN_PROMPT = "You › ";
