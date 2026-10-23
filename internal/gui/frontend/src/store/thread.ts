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

/** 支持的思考标签对，与 TUI 的 think.go 保持一致。 */
const THINK_TAGS = [
  ["<think>", "</think>"],
  ["<thinking>", "</thinking>"],
  ["<reasoning>", "</reasoning>"],
] as const;

/** 把 <think>…</think> 等思考段从正文中拆出；未闭合时视为仍在思考。 */
export function splitThink(text: string): { think: string; body: string; thinking: boolean } {
  let body = "";
  const thinks: string[] = [];
  let thinking = false;
  let rest = text;

  while (rest) {
    let openAt = -1;
    let openTag = "";
    let closeTag = "";
    for (const [open, close] of THINK_TAGS) {
      const i = rest.indexOf(open);
      if (i >= 0 && (openAt < 0 || i < openAt)) {
        openAt = i;
        openTag = open;
        closeTag = close;
      }
    }
    if (openAt < 0) {
      body += rest;
      break;
    }
    body += rest.slice(0, openAt);
    rest = rest.slice(openAt + openTag.length);
    const closeAt = rest.indexOf(closeTag);
    if (closeAt < 0) {
      thinks.push(rest);
      thinking = true;
      break;
    }
    thinks.push(rest.slice(0, closeAt));
    rest = rest.slice(closeAt + closeTag.length);
  }

  return { think: thinks.join("\n\n").trim(), body: body.trimStart(), thinking };
}
