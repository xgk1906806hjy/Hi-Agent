import type { Bridge } from "./types";
import { createMockBridge } from "./mock";
import { createWailsBridge, hasGoApp } from "./wails";

export * from "./types";

export class BridgeUnavailableError extends Error {
  constructor(message = "桌面引擎未就绪：未检测到 Wails 绑定。请用 scripts/build-gui 打包后运行。") {
    super(message);
    this.name = "BridgeUnavailableError";
  }
}

/**
 * 等待 Go 绑定注入。
 * - 开发模式（vite）：超时后回退 mock，便于浏览器调样式。
 * - 生产（Wails embed）：超时仍无绑定则抛错，禁止 silent mock（否则「添加项目」会变成空操作）。
 */
export async function connectBridge(timeoutMs = 5000): Promise<Bridge> {
  const start = Date.now();
  while (!hasGoApp() && Date.now() - start < timeoutMs) {
    await new Promise((r) => setTimeout(r, 30));
  }
  if (hasGoApp()) {
    return createWailsBridge();
  }
  if (import.meta.env.DEV) {
    return createMockBridge();
  }
  throw new BridgeUnavailableError();
}
