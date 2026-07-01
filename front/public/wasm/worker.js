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

    // Wait for goWasmEngine to be registered by Go's main(), then load the book.
    // go.run() is async, so goWasmEngine may not be available immediately.
    const waitForEngine = (callback, retries = 0) => {
      if (self.goWasmEngine && self.goWasmEngine.loadBook) {
        callback();
      } else if (retries < 100) {
        setTimeout(() => waitForEngine(callback, retries + 1), 50);
      } else {
        // Engine didn't initialize in time — proceed without book
        postMessage({ type: "ready" });
      }
    };

    waitForEngine(() => {
      // Load the opening book if it exists. Fails silently on 404 — the engine
      // works fine without a book (falls back to search).
      fetch("/books/book.bin")
        .then((res) => (res.ok ? res.arrayBuffer() : null))
        .then((buf) => {
          if (buf && self.goWasmEngine && self.goWasmEngine.loadBook) {
            const ok = self.goWasmEngine.loadBook(new Uint8Array(buf));
            console.log("[worker] loadBook result:", ok, "bytes:", buf.byteLength);
          } else {
            console.log("[worker] loadBook skipped: buf=", !!buf, "engine=", !!self.goWasmEngine);
          }
        })
        .catch((err) => console.log("[worker] book fetch error:", err))
        .finally(() => {
          postMessage({ type: "ready" });
        });
    });
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
