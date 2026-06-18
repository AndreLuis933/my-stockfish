import { useState, useEffect, useRef } from "react";
import { loadWasmEngine, WasmWorkerEngine } from "./loader";
import type { WasmEngine } from "./generated/wasm-contract";

export interface WasmState {
  engine: WasmEngine | null;
  loading: boolean;
  error: string | null;
  restarting: boolean;
}

export function useWasm(): WasmState {
  const [engine, setEngine] = useState<WasmEngine | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [restarting, setRestarting] = useState(false);
  const engineRef = useRef<WasmWorkerEngine | null>(null);
  const restartingRef = useRef(false);

  useEffect(() => {
    let cancelled = false;

    loadWasmEngine()
      .then((e) => {
        if (!cancelled) {
          engineRef.current = e as WasmWorkerEngine;
          setEngine(e);
          setLoading(false);
        }
      })
      .catch((err: Error) => {
        if (!cancelled) {
          setError(err.message);
          setLoading(false);
        }
      });

    return () => {
      cancelled = true;
    };
  }, []);

  // Vite HMR: instant restart when plugin sends wasm-rebuild event
  useEffect(() => {
    if (import.meta.env.PROD) return;

    const doRestart = async () => {
      if (!engineRef.current || restartingRef.current) return;
      restartingRef.current = true;
      setRestarting(true);
      try {
        await engineRef.current.restart();
      } catch (err: unknown) {
        setError("Restart failed: " + (err instanceof Error ? err.message : String(err)));
      } finally {
        restartingRef.current = false;
        setRestarting(false);
      }
    };

    const handler = () => {
      console.log("[wasm] rebuild detected via HMR, restarting worker...");
      doRestart();
    };

    if (import.meta.hot) {
      import.meta.hot.on("wasm-rebuild", handler);
    }

    return () => {
      if (import.meta.hot) {
        import.meta.hot.off("wasm-rebuild", handler);
      }
    };
  }, []);

  return { engine, loading, error, restarting };
}
