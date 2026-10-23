import type { Bridge, EventMap, EventName, Snapshot } from "./types";

interface GoApp {
  Ready(): Promise<Snapshot>;
  Submit(text: string): Promise<void>;
  Stop(): Promise<void>;
  Answer(id: string, value: number): Promise<void>;
  AddProject(): Promise<string>;
  OpenProject(path: string): Promise<void>;
  RemoveProject(path: string): Promise<void>;
  OpenSession(path: string, id: string): Promise<void>;
  NewSession(): Promise<void>;
  RenameSession(path: string, oldId: string, newId: string): Promise<void>;
  DeleteSession(path: string, id: string): Promise<void>;
  SetTheme(mode: string): Promise<void>;
  WindowMinimise(): Promise<void>;
  WindowToggleMaximise(): Promise<void>;
  WindowClose(): Promise<void>;
}

interface WailsRuntime {
  EventsOn(name: string, cb: (...data: unknown[]) => void): () => void;
}

declare global {
  interface Window {
    go?: { gui?: { App?: GoApp } };
    runtime?: WailsRuntime;
  }
}

/** Go 绑定已注入即可建 bridge；runtime 可稍后用于事件。 */
export function hasGoApp(): boolean {
  return !!window.go?.gui?.App;
}

export function hasWails(): boolean {
  return hasGoApp() && !!window.runtime;
}

export function createWailsBridge(): Bridge {
  const app = window.go!.gui!.App!;
  return {
    ready: () => app.Ready(),
    submit: (t) => void app.Submit(t),
    stop: () => void app.Stop(),
    answer: (id, v) => void app.Answer(id, v),
    addProject: () => app.AddProject(),
    openProject: (p) => void app.OpenProject(p),
    removeProject: (p) => app.RemoveProject(p),
    openSession: (p, id) => void app.OpenSession(p, id),
    newSession: () => void app.NewSession(),
    renameSession: (path, o, n) => void app.RenameSession(path, o, n),
    deleteSession: (path, id) => void app.DeleteSession(path, id),
    setTheme: (mode) => void app.SetTheme(mode),
    windowMinimise: () => void app.WindowMinimise(),
    windowToggleMaximise: () => void app.WindowToggleMaximise(),
    windowClose: () => void app.WindowClose(),
    on<E extends EventName>(event: E, fn: (payload: EventMap[E]) => void) {
      const rt = window.runtime;
      if (!rt?.EventsOn) {
        return () => {};
      }
      return rt.EventsOn(event, (data) => fn(data as EventMap[E]));
    },
  };
}
