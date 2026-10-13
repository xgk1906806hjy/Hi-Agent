import { useEffect, useState } from "react";
import { BridgeUnavailableError, connectBridge } from "./bridge";
import { attachBridge, useStore } from "./store";
import { pushText } from "./store/thread";
import { Sidebar } from "./components/Sidebar";
import { TopBar } from "./components/TopBar";
import { Thread } from "./components/Thread";
import { Composer } from "./components/Composer";
import { ConfirmDialog } from "./components/ConfirmDialog";
import { SelectDialog } from "./components/SelectDialog";

export default function App() {
  const [bootError, setBootError] = useState<string | null>(null);

  useEffect(() => {
    let off: (() => void) | undefined;
    let cancelled = false;
    void connectBridge()
      .then(async (b) => {
        if (cancelled) return;
        off = await attachBridge(b);
      })
      .catch((e) => {
        if (cancelled) return;
        const msg = e instanceof BridgeUnavailableError ? e.message : e instanceof Error ? e.message : String(e);
        setBootError(msg);
        useStore.setState((s) => ({ items: pushText(s.items, "error", msg) }));
      });
    return () => {
      cancelled = true;
      off?.();
    };
  }, []);

  return (
    <div className="flex h-full bg-panel text-ink">
      <Sidebar />
      <main className="flex min-w-0 flex-1 flex-col">
        <TopBar />
        {bootError ? (
          <div className="border-b border-line bg-red-50 px-4 py-2 text-sm text-danger">{bootError}</div>
        ) : null}
        <Thread />
        <Composer />
      </main>
      <ConfirmDialog />
      <SelectDialog />
    </div>
  );
}
