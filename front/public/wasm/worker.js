// Web Worker — loads the Go WASM engine and dispatches calls from the main thread.
//
// Message in:  { id, fn, args }
// Message out: { id, result } | { id, error }
// Lifecycle:   { type: "ready" } | { type: "error", message }

importScripts("/wasm/wasm_exec.js");

const go = new Go();

WebAssembly.instantiateStreaming(fetch("/wasm/engine.wasm"), go.importObject)
  .then(({ instance }) => {
    // go.run() starts Go execution. main() registers goWasmEngine on the global
    // scope and then blocks at select{}, yielding control back to JS.
    go.run(instance);
    postMessage({ type: "ready" });
  })
  .catch((err) => {
    postMessage({ type: "error", message: String(err) });
  });

self.onmessage = ({ data: { id, fn, args } }) => {
  try {
    const result = self.goWasmEngine[fn](...args);
    const payload = result instanceof Uint8Array ? Array.from(result) : result;
    postMessage({ id, result: payload });
  } catch (err) {
    postMessage({ id, error: String(err) });
  }
};
