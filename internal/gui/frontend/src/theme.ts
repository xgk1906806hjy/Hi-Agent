export type ThemeChoice = "system" | "light" | "dark";

const KEY = "hi-agent-theme";

export function readTheme(): ThemeChoice {
  const v = localStorage.getItem(KEY);
  if (v === "light" || v === "dark" || v === "system") return v;
  return "system";
}

export function isDark(choice: ThemeChoice): boolean {
  if (choice === "dark") return true;
  if (choice === "light") return false;
  return window.matchMedia("(prefers-color-scheme: dark)").matches;
}

export function applyTheme(choice: ThemeChoice): boolean {
  const dark = isDark(choice);
  document.documentElement.classList.toggle("dark", dark);
  document.documentElement.dataset.theme = choice;
  return dark;
}

export function saveTheme(choice: ThemeChoice): boolean {
  localStorage.setItem(KEY, choice);
  return applyTheme(choice);
}

export function nextTheme(choice: ThemeChoice): ThemeChoice {
  if (choice === "system") return "light";
  if (choice === "light") return "dark";
  return "system";
}

export const themeLabel: Record<ThemeChoice, string> = {
  system: "跟随系统",
  light: "浅色",
  dark: "深色",
};
