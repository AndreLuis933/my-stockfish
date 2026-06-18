# Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI in TypeScript
- **Xadrez** (Chess) — playable, move generation runs in Go compiled to WebAssembly

## Stack

| Layer | Tech |
|---|---|
| Frontend | React 19 + TypeScript (strict) + Vite + CSS Modules |
| Routing | react-router-dom v7 |
| Package manager | Bun |
| Checkers AI | TypeScript — Minimax + Alpha-Beta + IDDFS (depth 8) |
| Chess engine | Go 1.25 → WebAssembly (move generation, FEN, en passant, promotion) |

## Getting started

```bash
# Install dependencies
bun install

# Start dev server (also compiles Go WASM and watches .go files)
bun dev

# Type-check + lint
bun run check

# Tests
bun test
```

Requires Go installed on the machine for the WASM build step.

## Project structure

```
src/
├── components/
│   ├── Board/             # Checkers board UI
│   ├── ChessBoard/        # Chess board UI (cburnett SVG pieces)
│   ├── PromotionPicker/   # Pawn promotion modal (Q/N/R/B)
│   └── Nav/               # Top nav bar
├── pages/
│   ├── checkers/          # Route /checkers
│   └── chess/             # Route /chess (Chess.tsx + Chess.hooks.ts + .module.css)
├── hooks/useGame.ts       # Checkers state machine
├── utils/
│   ├── gameEngine.ts      # Checkers: move gen, captures, flying kings, applyMove
│   ├── aiEngine.ts        # Checkers AI: Minimax + Alpha-Beta + IDDFS
│   ├── chessEngine.ts     # Chess: emptyBoard(), pieceByte(), square helpers
│   └── chessAssets.ts     # pieceImageUrl() → cburnett SVG path
├── types/
│   ├── game.ts            # Checkers types
│   └── chess.ts           # Chess types + getPiece/decodePieceByte
└── wasm/
    ├── generated/         # Hand-maintained TS contract — edit directly
    │   └── wasm-contract.ts
    ├── loader.ts          # WasmWorkerEngine (Web Worker bridge to Go WASM)
    └── useWasm.ts         # React hook: { engine, loading, error, restarting }

plugins/
└── go-wasm.ts            # Vite plugin: builds WASM, copies wasm_exec.js, watches .go, sends HMR

public/
├── wasm/
│   ├── engine.wasm        # Compiled Go binary (gitignored)
│   ├── wasm_exec.js       # Go WASM runtime (copied from GOROOT)
│   └── worker.js          # Web Worker: loads runtime + wasm, dispatches calls
└── pieces/                # Chess piece SVGs (cburnett + chessnut sets)
```

## Game rules (Damas Brasileiras)

- 8×8 board, 12 pieces per side
- Men move and capture forward diagonally
- Captures are mandatory; must take the maximum number of pieces (Brazilian rule)
- Men promote to kings on the back rank
- Kings are **flying kings** — slide any number of squares diagonally, capture at range
- Multi-jump chains are required
- Draw after 40 moves without a capture or man move

## Chess (Xadrez)

- Board state and move generation run in Go compiled to WebAssembly
- Go engine handles: FEN loading, all piece types, captures, en passant, pawn promotion
- Pawn promotion shows a picker modal (Q/N/R/B); the chosen piece byte is sent to `engine.makeMove(from, to, promotionByte)`
- Modes: Humano vs Humano (Humano vs IA mode exists but chess AI is not yet implemented)

## Go WASM integration

See `AGENTS.md` at the project root for the full architecture, current state, and what is still missing.

The Vite plugin (`plugins/go-wasm.ts`) handles everything automatically in dev mode:
- Compiles `engine.wasm` on startup and on production build
- Copies `wasm_exec.js` from GOROOT
- Watches `.go` files and rebuilds on change
- Sends a `wasm-rebuild` HMR event so the browser restarts the WASM worker without a full reload

The type generator (`go-wasm/tools/main.go`) is **not** run automatically and is not part of the normal workflow. `wasm-contract.ts` is hand-maintained — edit it directly when Go function signatures change. Run `gen-types.exe` only if you want a regenerated starting point:
```bash
cd go-wasm
go build -o bin/gen-types.exe tools/main.go
./bin/gen-types.exe
```

## Modes

### Damas

| Mode | Description |
|---|---|
| Humano vs IA | Player is white; black is controlled by AI |
| Humano vs Humano | Both sides require a human click |
| IA vs IA | Both sides play automatically |

### Xadrez

| Mode | Description |
|---|---|
| Humano vs Humano | Both sides require a human click |
| Humano vs IA | Defined but chess AI not yet implemented |