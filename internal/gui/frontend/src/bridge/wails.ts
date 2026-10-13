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
  RenameSession(oldId: string, newId: string): Promise<void>;
  DeleteSession(id: string): Promise<void>;
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
    renameSession: (o, n) => void app.RenameSession(o, n),
    deleteSession: (id) => void app.DeleteSession(id),
    on<E extends EventName>(event: E, fn: (payload: EventMap[E]) => void) {
      const rt = window.runtime;
      if (!rt?.EventsOn) {
        return () => {};
      }
      return rt.EventsOn(event, (data) => fn(data as EventMap[E]));
    },
  };
}
