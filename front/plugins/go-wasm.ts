import { execSync, spawnSync } from "child_process";
import { mkdirSync, copyFileSync } from "fs";
import { watch } from "fs";
import { dirname, join } from "path";
import { fileURLToPath } from "url";
import type { Plugin, ViteDevServer } from "vite";

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = join(__dirname, "..", "..");
const goDir = join(root, "go-wasm");
const wasmMainDir = join(goDir, "cmd", "wasm");
const outDir = join(__dirname, "..", "public", "wasm");

let wasmBuilding = false;
let wasmPending = false;

function copyWasmExec(): void {
  try {
    const goRoot = execSync("go env GOROOT", { encoding: "utf-8" }).trim();
    copyFileSync(
      join(goRoot, "lib", "wasm", "wasm_exec.js"),
      join(outDir, "wasm_exec.js"),
    );
  } catch {
    /* already exists or go not found */
  }
}

function buildWasm(prod: boolean): void {
  if (wasmBuilding) {
    wasmPending = true;
    return;
  }
  wasmBuilding = true;
  wasmPending = false;

  const start = Date.now();
  const label = prod ? "prod" : "dev";
  console.log(`[go-wasm] building (${label})...`);

  try {
    mkdirSync(outDir, { recursive: true });

    const args = [
      "build",
      ...(prod ? ["-ldflags=-s -w", "-trimpath"] : []),
      "-o",
      join(outDir, "engine.wasm"),
      ".",
    ];

    const result = spawnSync("go", args, {
      cwd: wasmMainDir,
      env: { ...process.env, GOOS: "js", GOARCH: "wasm" },
      stdio: "inherit",
    });

    if (result.status !== 0) {
      throw new Error(`go build exited with code ${result.status}`);
    }

    copyWasmExec();

    const elapsed = Date.now() - start;
    console.log(`[go-wasm] done in ${elapsed}ms`);
  } catch (err) {
    console.log(`[go-wasm] build failed: ${err}`);
  } finally {
    wasmBuilding = false;
    if (wasmPending) buildWasm(false);
  }
}


export function goWasmPlugin(): Plugin {
  let server: ViteDevServer | null = null;

  return {
    name: "go-wasm",

    // Runs once at build start (dev AND production)
    buildStart() {
      buildWasm(true);
    },

    // Runs only in dev mode when the server starts
    configureServer(srv) {
      server = srv;

      // Watch all .go files under go-wasm
      const watcher = watch(goDir, { recursive: true }, (_event, filename) => {
        if (!filename || !filename.endsWith(".go")) return;

        buildWasm(false);

        // Notify browser via WebSocket to restart worker immediately
        server?.ws.send({ type: "custom", event: "wasm-rebuild" });
      });

      srv.httpServer?.once("close", () => watcher.close());
    },
  };
}
