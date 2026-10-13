// 对话流的纯数据变换：事件 → Item 列表。无副作用，便于推理。
import type { TurnPayload } from "../bridge";

export type Item =
  | { kind: "user"; id: number; text: string }
  | { kind: "assistant"; id: number; text: string; streaming: boolean }
  | { kind: "tools"; id: number; steps: string[]; running: boolean }
  | { kind: "info" | "notice" | "error"; id: number; text: string };

let seq = 0;
const nextId = () => ++seq;

export function pushUser(items: Item[], text: string): Item[] {
  return [...items, { kind: "user", id: nextId(), text }];
}

export function appendDelta(items: Item[], text: string): Item[] {
  const last = items[items.length - 1];
  if (last?.kind === "assistant" && last.streaming) {
    return [...items.slice(0, -1), { ...last, text: last.text + text }];
  }
  return [...closeTools(items), { kind: "assistant", id: nextId(), text, streaming: true }];
}

/** 连续的过程行合并为一个可折叠块。 */
export function appendStep(items: Item[], text: string): Item[] {
  const step = text.trim().replace(/^\[|\]$/g, "");
  if (!step) return items;
  const last = items[items.length - 1];
  if (last?.kind === "tools") {
    return [...items.slice(0, -1), { ...last, steps: [...last.steps, step] }];
  }
  const sealed = items.map((it) => (it.kind === "assistant" && it.streaming ? { ...it, streaming: false } : it));
  return [...sealed, { kind: "tools", id: nextId(), steps: [step], running: true }];
}

export function pushText(items: Item[], kind: "info" | "notice" | "error", text: string): Item[] {
  if (!text.trim()) return items;
  return [...items, { kind, id: nextId(), text }];
}

export function endTurn(items: Item[]): Item[] {
  return closeTools(items).map((it) => (it.kind === "assistant" && it.streaming ? { ...it, streaming: false } : it));
}

function closeTools(items: Item[]): Item[] {
  return items.map((it) => (it.kind === "tools" && it.running ? { ...it, running: false } : it));
}

export function fromTurns(turns: TurnPayload[]): Item[] {
  let items: Item[] = [];
  for (const t of turns) {
    if (t.role === "user") items = pushUser(items, t.text);
    else if (t.role === "tool") items = appendStep(items, t.text);
    else items = appendDelta(items, t.text);
  }
  return endTurn(items);
}

/** 把 <think>…</think> 从正文中拆出；未闭合时视为仍在思考。 */
export function splitThink(text: string): { think: string; body: string; thinking: boolean } {
  const open = text.indexOf("<think>");
  if (open < 0) return { think: "", body: text, thinking: false };
  const close = text.indexOf("</think>", open);
  if (close < 0) return { think: text.slice(open + 7), body: text.slice(0, open), thinking: true };
  return {
    think: text.slice(open + 7, close).trim(),
    body: (text.slice(0, open) + text.slice(close + 8)).trimStart(),
    thinking: false,
  };
}
