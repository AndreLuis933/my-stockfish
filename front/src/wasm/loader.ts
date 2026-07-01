import type { WasmContract, WasmEngine, WasmFunctionName } from "./generated/wasm-contract";

export type { WasmEngine } from "./generated/wasm-contract";

class WasmWorkerEngineCore {
  private worker: Worker;
  private pending = new Map<
    number,
    { resolve: (value: unknown) => void; reject: (reason: Error) => void }
  >();
  private nextId = 0;
  private _ready = false;
  private _initError: string | null = null;
  private _restartPromise: Promise<void> | null = null;

  get initError(): string | null {
    return this._initError;
  }

  constructor() {
    this.worker = new Worker("/wasm/worker.js?v=" + Date.now());
    this.worker.onmessage = (e) => this.handleMessage(e);
  }

  get ready() {
    return this._ready;
  }

  private handleMessage(event: MessageEvent) {
    const data = event.data;

    if (data.type === "ready") {
      this._ready = true;
      return;
    }

    if (data.type === "error") {
      this._ready = false;
      this._initError = data.message ?? "WASM failed to initialize";
      return;
    }

    const { id, result, error } = data;
    const p = this.pending.get(id);
    if (!p) return;

    this.pending.delete(id);
    if (error) {
      p.reject(new Error(error));
    } else {
      p.resolve(result);
    }
  }

  private call<K extends WasmFunctionName>(
    fn: K,
    args: WasmContract[K]["args"],
  ): Promise<WasmContract[K]["return"]> {
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (value: unknown) => void, reject });
      this.worker.postMessage({ id, fn, args });
    });
  }

  restart(): Promise<void> {
    // Multiple consumers share this engine (see loadWasmEngine), so several may
    // request a restart for the same HMR event. De-dupe to a single in-flight
    // restart so we don't terminate a worker mid-swap or leak workers.
    if (this._restartPromise) return this._restartPromise;
    this._restartPromise = this.doRestart();
    const clear = () => {
      this._restartPromise = null;
    };
    this._restartPromise.then(clear, clear);
    return this._restartPromise;
  }

  private doRestart(): Promise<void> {
    return new Promise((resolve, reject) => {
      this._ready = false;
      this._initError = null;
      this.pending.forEach((p) =>
        p.reject(new Error("Worker restarting")),
      );
      this.pending.clear();
      this.worker.terminate();

      const newWorker = new Worker("/wasm/worker.js?v=" + Date.now());

      const timeout = setTimeout(() => {
        newWorker.terminate();
        reject(new Error("Worker restart timed out after 10s"));
      }, 10000);

      newWorker.onmessage = (e) => {
        this.handleMessage(e);
        if (e.data?.type === "ready") {
          clearTimeout(timeout);
          this.worker = newWorker;
          resolve();
        }
        if (e.data?.type === "error") {
          clearTimeout(timeout);
          newWorker.terminate();
          reject(new Error(e.data.message || "Worker failed to start"));
        }
      };

      newWorker.onerror = (err) => {
        clearTimeout(timeout);
        newWorker.terminate();
        reject(new Error(err.message || "Worker script error"));
      };
    });
  }

  protected fn<K extends WasmFunctionName>(name: K) {
    return (
      ...args: WasmContract[K]["args"]
    ): Promise<WasmContract[K]["return"]> => this.call(name, args);
  }
}

export class WasmWorkerEngine extends WasmWorkerEngineCore implements WasmEngine {
  validMovesChess = this.fn("validMovesChess");
  initBoard = this.fn("initBoard");
  makeMove = this.fn("makeMove");
  isCheckJS = this.fn("isCheckJS");
  gameStatus = this.fn("gameStatus");
  aiMove = this.fn("aiMove");
  aiMoveDepth = this.fn("aiMoveDepth");
  aiAnalysis = this.fn("aiAnalysis");
  aiMultiPv = this.fn("aiMultiPv");
  loadBook = this.fn("loadBook");
  bookMoves = this.fn("bookMoves");
  fen = this.fn("fen");
  san = this.fn("san");
  applyPgn = this.fn("applyPgn");
}

// The engine holds the authoritative game position inside its worker. Every
// consumer must share one worker — separate workers would each carry their own
// divergent game state (the bug where bookMoves saw an empty board). So the
// engine is a module-level singleton, created once and reused across all
// useWasm() callers.
let enginePromise: Promise<WasmEngine> | null = null;

export function loadWasmEngine(): Promise<WasmEngine> {
  if (enginePromise) return enginePromise;

  const engine = new WasmWorkerEngine();
  enginePromise = new Promise<WasmEngine>((resolve, reject) => {
    const check = () => {
      if (engine.ready) { resolve(engine); return; }
      if (engine.initError) { reject(new Error(engine.initError)); return; }
      setTimeout(check, 50);
    };
    check();
  });
  // Allow a later mount to retry if initialization failed.
  enginePromise.catch(() => { enginePromise = null; });
  return enginePromise;
}
