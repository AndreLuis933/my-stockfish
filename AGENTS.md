# Project State — Damas Brasileiras

Brazilian Checkers game. The frontend is fully playable right now. The Go WASM integration is **in progress** — the infrastructure is built but the actual game logic has not been ported to Go yet.

---

## Repository Layout

```
my-stockfish/
├── front/                        # React + TypeScript + Vite (Bun)
│   ├── plugins/go-wasm.ts        # Vite plugin: builds WASM, generates types, HMR
│   ├── public/wasm/
│   │   ├── engine.wasm           # Compiled Go WASM binary
│   │   └── wasm_exec.js          # Go runtime for WASM (copied from GOROOT)
│   └── src/
│       ├── wasm/
│       │   ├── generated/wasm-contract.ts   # AUTO-GENERATED — do not edit
│       │   ├── loader.ts                    # WasmWorkerEngine class (Web Worker bridge)
│       │   └── useWasm.ts                   # React hook for WASM lifecycle
│       ├── pages/
│       │   ├── checkers/          # Route /checkers
│       │   │   ├── Checkers.tsx
│       │   │   └── Checkers.module.css
│       │   └── chess/             # Route /chess
│       │       ├── Chess.tsx
│       │       ├── Chess.hooks.ts
│       │       └── Chess.module.css
│       ├── components/
│       │   ├── Board/             # Checkers board UI
│       │   ├── ChessBoard/        # Chess board UI (Unicode pieces)
│       │   └── Nav/               # Top nav bar (links to /checkers and /chess)
│       ├── hooks/useGame.ts       # Checkers state machine (uses TS AI)
│       ├── utils/gameEngine.ts    # Checkers board logic
│       ├── utils/aiEngine.ts      # Checkers Minimax + Alpha-Beta + IDDFS (TypeScript)
│       ├── utils/chessEngine.ts   # Chess: initChessBoard only (logic goes to Go)
│       ├── types/game.ts          # Checkers types
│       └── types/chess.ts         # Chess types: ChessColor, ChessPieceType, ChessPiece, ChessBoard
└── go-wasm/                      # Go source compiled to WASM
    ├── cmd/wasm/main.go           # Entry point: registers functions on js.Global().goWasmEngine
    ├── pkg/engine/engine.go       # Pure Go logic (currently: demo math functions only)
    ├── tools/main.go              # Type generator: Go AST → wasm-contract.ts
    └── bin/gen-types.exe          # Compiled type generator binary
```

---

## Current State

### What works

- **Full game** runs in the browser: human-vs-AI, human-vs-human, AI-vs-AI
- AI: Minimax + Alpha-Beta + IDDFS, implemented in TypeScript (`aiEngine.ts`), depth 8
- WASM **compilation pipeline**: Vite plugin compiles `engine.wasm` on dev start and production build
- WASM **type generator**: reads Go AST → type-checks `pkg/engine` → writes `wasm-contract.ts`; re-runs on every `.go` file change
- WASM **loader infrastructure** (`loader.ts`): `WasmWorkerEngine` class, typed async calls via Web Worker, HMR restart support
- `useWasm` React hook: loads engine, exposes `{ engine, loading, error, restarting }`, listens for `wasm-rebuild` HMR events

### What is missing / in progress

| Piece | Notes |
|---|---|
| `front/public/wasm/worker.js` | **Critical missing file.** `loader.ts` spawns `new Worker("/wasm/worker.js")` but this file doesn't exist. The worker must load `wasm_exec.js` + `engine.wasm`, instantiate `goWasmEngine`, then handle `{ id, fn, args }` messages and respond with `{ id, result, error }`. |
| Chess logic in Go | `pkg/engine/engine.go` only has demo stubs. Chess needs: board representation, move generation (all piece types, special moves), check/checkmate detection, minimax with Alpha-Beta. |
| Wire chess WASM into `Chess.hooks.ts` | `useChess` has a `// TODO: validate move via Go WASM` placeholder. Once Go exposes chess functions, replace the stub with the WASM call and apply the returned board state. |

---

## WASM Integration Architecture

### How a call flows (once worker.js exists)

```
React component
  → useWasm() hook  →  WasmWorkerEngine.fibonacci(10)
    → loader.ts: postMessage({ id: 0, fn: "fibonacci", args: [10] })
      → worker.js: goWasmEngine.fibonacci(10)   ← Go function on js.Global()
        → pkg/engine: Fibonacci(10) in Go
      → worker.js: postMessage({ id: 0, result: 55 })
    → loader.ts: Promise resolves with 55
```

### Type safety pipeline

```
go-wasm/pkg/engine/engine.go   (real Go types)
  ↓  tools/main.go (type generator, run by Vite plugin)
front/src/wasm/generated/wasm-contract.ts   (AUTO-GENERATED)
  ↓  imported by
front/src/wasm/loader.ts   (WasmWorkerEngine: typed async wrappers)
```

`wasm-contract.ts` is re-generated on every `.go` file change — never edit it manually.

### How to add a new Go function

1. Add the pure function to `pkg/engine/engine.go`
2. Add a bridge wrapper to `cmd/wasm/main.go` and register it with `e.Set("jsName", js.FuncOf(bridgeFunc))`
3. Save — the Vite plugin rebuilds the WASM and regenerates `wasm-contract.ts` automatically
4. Add the typed wrapper to `WasmWorkerEngine` in `loader.ts`: `newFn = this.fn("jsName")`
5. Handle the call in `worker.js` (the worker dispatches by function name)

---

## Commands

```bash
# Frontend (from front/)
bun dev           # dev server — also builds WASM and watches .go files
bun run check     # tsc + eslint:strict  (run after every change)
bun test          # vitest

# Go WASM (from go-wasm/)
GOOS=js GOARCH=wasm go build -o ../front/public/wasm/engine.wasm ./cmd/wasm

# Type generator (from go-wasm/)
go build -o bin/gen-types.exe tools/main.go   # compile once
./bin/gen-types.exe                            # run to regenerate wasm-contract.ts
```

---

## Rules

- Run `bun run check` from `front/` after every file change.
- `wasm-contract.ts` is auto-generated — never edit it by hand.
- The Vite plugin runs the type generator automatically in dev mode.
- `worker.js` must be plain JavaScript (no bundler) — it runs inside a Web Worker with no import support unless using `importScripts`.
