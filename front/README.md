# Damas Brasileiras

Brazilian Checkers game — React + TypeScript frontend with a Go engine compiled to WebAssembly.

## Stack

| Layer | Tech |
|---|---|
| Frontend | React 19 + TypeScript (strict) + Vite + CSS Modules |
| Package manager | Bun |
| AI engine (current) | TypeScript — Minimax + Alpha-Beta + IDDFS |
| AI engine (target) | Go → WASM |

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
├── components/Board/     # Board UI (Board.tsx + Board.module.css)
├── hooks/useGame.ts      # Game state machine — board, turns, AI trigger
├── utils/
│   ├── gameEngine.ts     # Board logic: move generation, applyMove, computeTurnState
│   └── aiEngine.ts       # Minimax + Alpha-Beta + IDDFS
├── types/game.ts         # Color, PieceType, Piece, Cell, Board, Move
└── wasm/
    ├── generated/        # AUTO-GENERATED — do not edit
    │   └── wasm-contract.ts
    ├── loader.ts         # WasmWorkerEngine (Web Worker bridge to Go WASM)
    └── useWasm.ts        # React hook: { engine, loading, error, restarting }

plugins/
└── go-wasm.ts            # Vite plugin: builds WASM, generates TS types, sends HMR events

public/wasm/
├── engine.wasm           # Compiled Go binary
└── wasm_exec.js          # Go WASM runtime (copied from GOROOT at build time)
```

## Game rules (Damas Brasileiras)

- 8×8 board, 12 pieces per side
- Men move and capture forward diagonally
- Captures are mandatory; must take the maximum number of pieces (Brazilian rule)
- Men promote to kings on the back rank
- Kings are **flying kings** — slide any number of squares diagonally, capture at range
- Multi-jump chains are required
- Draw after 40 moves without a capture or man move

## Go WASM integration

See `AGENTS.md` at the project root for the full architecture, current state, and what is still missing.

The Vite plugin (`plugins/go-wasm.ts`) handles everything automatically in dev mode:
- Compiles `engine.wasm` on startup
- Watches `.go` files and rebuilds on change
- Re-generates `src/wasm/generated/wasm-contract.ts` from Go types
- Sends a `wasm-rebuild` HMR event so the browser restarts the WASM worker without a full reload

## Modes

| Mode | Description |
|---|---|
| Humano vs IA | Player is white; black is controlled by AI |
| Humano vs Humano | Both sides require a human click |
| IA vs IA | Both sides play automatically |
