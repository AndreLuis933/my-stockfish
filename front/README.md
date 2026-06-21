# Front — Damas Brasileiras + Xadrez

React 19 + TypeScript (strict) + Vite (Bun) frontend for two board games:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI in TypeScript
- **Xadrez** (Chess) — fully playable, including human-vs-AI and AI-vs-AI; move generation and board state run in Go compiled to WebAssembly, AI runs in Go

---

## Stack

| Layer | Tech |
|---|---|
| Framework | React 19 |
| Language | TypeScript (strict mode) |
| Bundler | Vite |
| Styling | CSS Modules |
| Routing | react-router-dom v7 |
| Package manager | Bun |
| Checkers AI | TypeScript — Minimax + Alpha-Beta + IDDFS (depth 8) |
| Chess engine | Go 1.25 → WebAssembly (loaded via Web Worker) |
| Chess AI | Go 1.25 → WebAssembly — negamax + alpha-beta + iterative deepening + transposition table + quiescence search |

---

## Getting started

### Prerequisites

- [Bun](https://bun.sh)
- [Go 1.25+](https://go.dev/dl/) (WASM build step — the Vite plugin calls `go build` directly)

### Install and run

```bash
bun install
bun dev          # dev server — also builds WASM and watches .go files
```

Open the dev server URL. Routes:

- `/` → redirects to `/checkers`
- `/checkers` — Damas Brasileiras
- `/chess` — Xadrez

### Other commands

```bash
bun run check     # tsc -b && eslint:strict (run after every change)
bun test          # vitest
bun run build     # tsc -b && vite build (also builds WASM with prod flags)
```

---

## Project structure

```
src/
├── components/
│   ├── Board/             # Checkers board UI (Board.tsx + Board.module.css)
│   ├── ChessBoard/        # Chess board UI (cburnett SVG pieces, selection + move hints + check glow)
│   ├── MoveHistory/       # Move history sidebar (SAN notation, ply navigation, clocks display, result box)
│   ├── PromotionPicker/   # Pawn promotion modal (Q/N/R/B)
│   └── Nav/               # Top nav bar
├── pages/
│   ├── checkers/          # Route /checkers (Checkers.tsx + Checkers.module.css)
│   └── chess/             # Route /chess (Chess.tsx + Chess.hooks.ts + Chess.module.css)
├── hooks/useGame.ts       # Checkers state machine (uses TS AI)
├── hooks/useChessClock.ts # Chess clock hook: dual countdown, increments, flag-fall detection
├── utils/
│   ├── gameEngine.ts      # Checkers: move gen, captures, flying kings, applyMove
│   ├── aiEngine.ts        # Checkers AI: Minimax + Alpha-Beta + IDDFS
│   ├── chessEngine.ts     # Chess: emptyBoard(), pieceByte(), square helpers
│   ├── chessAssets.ts     # pieceImageUrl() → cburnett SVG path
│   └── chessNotation.ts   # Chess SAN-like notation generator (toSan, squareName, disambiguation, castling, promotion, check/mate suffixes)
├── types/
│   ├── game.ts            # Checkers types (Color, PieceType, Piece, Cell, Board, Move)
│   └── chess.ts           # Chess types (ChessColor, ChessPiece, ChessBoard, getPiece, decodePieceByte)
├── wasm/
│   ├── generated/wasm-contract.ts   # Hand-maintained TS contract for Go functions (edit directly)
│   ├── loader.ts                    # WasmWorkerEngine (Web Worker bridge, typed async calls)
│   └── useWasm.ts                   # React hook: { engine, loading, error, restarting } + HMR restart
└── assets/                # Static images

plugins/
└── go-wasm.ts            # Vite plugin: builds WASM, copies wasm_exec.js, watches .go, sends HMR

public/
├── wasm/
│   ├── engine.wasm        # Compiled Go binary (gitignored)
│   ├── wasm_exec.js       # Go WASM runtime (copied from GOROOT at build time)
│   └── worker.js          # Web Worker: loads runtime + wasm, dispatches { id, fn, args }
└── pieces/                # Chess piece SVGs (cburnett + chessnut sets)
```

---

## Go WASM integration

The chess engine and AI run in Go compiled to WebAssembly, loaded inside a Web Worker to avoid blocking the UI thread.

### Call flow

```
React component
  → useWasm() hook  →  WasmWorkerEngine.makeMove(from, to, promotion)
    → loader.ts: postMessage({ id, fn: "makeMove", args: [from, to, promotion] })
      → worker.js: self.goWasmEngine.makeMove(from, to, promotion)
        → cmd/wasm/main.go: makeMoveJS → engine.MakeMove(from, to, promotion)
      → worker.js: postMessage({ id, result: [64 board bytes] })
    → loader.ts: Promise resolves with number[]
```

### Registered functions (goWasmEngine)

| JS name | Args | Return | Purpose |
|---|---|---|---|
| `validMovesChess` | — | JSON string of `{from, to, promotion?}[]` | Legal moves for the side to move |
| `initBoard` | — | `number[]` (64 bytes) | Reset to starting position |
| `makeMove` | `number, number, number?` | `number[]` (64 bytes) | Apply a move; optional promotion byte |
| `isCheckJS` | — | `number` | Checked king's square index, or -1 |
| `gameStatus` | — | `string` | `"playing" \| "white-wins" \| "black-wins" \| "draw"` |
| `aiMove` | `number` (time limit ms) | JSON string `{from, to, promotion?}` | AI best move via time-limited search |
| `aiMoveDepth` | `number` (depth) | JSON string `{from, to, promotion?}` | AI best move via fixed-depth search |
| `aiAnalysis` | `number` (time limit ms) | JSON string `{from, to, promotion?, score, depth, nodes, timeMs}` | AI analysis: best move + evaluation + search info |

### Vite plugin (`plugins/go-wasm.ts`)

Handles everything automatically in dev mode:
- Compiles `engine.wasm` on startup and on production build
- Copies `wasm_exec.js` from GOROOT
- Watches `.go` files and rebuilds on change
- Sends a `wasm-rebuild` HMR event so the browser restarts the WASM worker without a full reload

### Type contract

`wasm-contract.ts` is **hand-maintained** — edit it directly when Go function signatures change. The type generator (`go-wasm/tools/main.go`) is only a starting point and does not run automatically. See `AGENTS.md` for details.

---

## Chess UI features

- **Board**: 8×8 with cburnett SVG piece set, light/dark square colors
- **Selection**: click a piece to highlight it and show legal move targets (dots for empty, red ring for captures)
- **Check highlight**: king's square glows red (pulsing animation) when in check; "Xeque!" badge in turn banner
- **Result overlay**: "Brancas vencem!" / "Pretas vencem!" / "Empate!" shown on game over
- **Pawn promotion**: picker modal with Q/N/R/B using piece SVGs
- **Board flip**: toggle board orientation
- **Turn banner**: shows whose turn it is with colored dots
- **AI setup panel** (human-vs-ai mode):
  - **Color selector**: "Você joga de: Brancas / Pretas" — board auto-flips when human chooses black
  - **Search mode**: difficulty / custom time (ms) / custom depth
  - **Difficulty**: Fácil (100ms) / Médio (500ms) / Difícil (2000ms) — time-limited iterative deepening
  - **Custom time**: number input (10-60000ms) — exact time budget for the AI
  - **Custom depth**: number input (1-10) — fixed-depth search with no time limit
- **"IA pensando..." indicator**: badge in turn banner while AI searches
- **Move history sidebar**: SAN-like notation (e4, Nf3, exd5, O-O, e8=Q+, O-O-O#) with move-pair rows; click any move to jump to that position; navigation buttons (|<, <, >, >|) for start/prev/next/end; auto-scrolls to current ply; per-move evaluation tags shown when analysis is available
- **Position navigation**: viewing past positions does not trigger the AI or allow board interaction; making a new move is only possible from the latest position; a "revisitando" badge appears in the turn banner when viewing history
- **Chess clock**: dual countdown (white/black) with configurable initial time (1/3/5/10/15 min or no clock) and increment (0/2/3/5/10s); clock starts on the first move; increment added after each move; flag fall → loss; clock config disabled during an active game
- **No auto-restart**: on game over the result overlay shows and stays until the user clicks "Jogar novamente"; the overlay has no close/cancel button — the user is forced to restart
- **Move animations**: pieces slide to their destination on move (300ms cubic-bezier); captured pieces fade-out with a scale pulse; castling animates both king and rook sliding simultaneously; last-move squares are highlighted
- **Coordinate labels**: file letters (a-h) and rank numbers (1-8) shown on the board edges, color-matched to the square (light on dark, dark on light)
- **Keyboard navigation**: ArrowLeft/ArrowRight navigate history (prev/next ply), Home/End jump to start/end; disabled when the promotion picker is open (works even after game over)
- **Position analysis**: "Analisar" button runs the AI search on the current position and shows the evaluation score (in pawns), best move (with an arrow drawn on the board), and search depth; closeable panel
- **Auto-analyze**: "Analisar auto" toggle automatically runs a 500ms analysis after each move and stores the evaluation in the move history; per-move eval tags appear next to each move in the sidebar
- **AI vs AI mode**: both sides played by the engine; search settings (difficulty/time/depth) apply to both; color selector hidden

## Damas (Checkers) features

- **Modes**: Humano vs IA, Humano vs Humano, IA vs IA
- **Brazilian rules**: mandatory max captures, flying kings, multi-jump chains, 40-move draw rule
- **AI**: Minimax + Alpha-Beta + IDDFS in TypeScript, depth 8
- **UI**: board flip, piece counters, "IA pensando..." indicator, must-move highlighting

---

## Game rules (Damas Brasileiras)

- 8×8 board, 12 pieces per side
- Men move and capture forward diagonally
- Captures are mandatory; must take the maximum number of pieces (Brazilian rule)
- Men promote to kings on the back rank
- Kings are **flying kings** — slide any number of squares diagonally, capture at range
- Multi-jump chains are required
- Draw after 40 moves without a capture or man move

---

## See also

- [`../README.md`](../README.md) — project overview
- [`../go-wasm/README.md`](../go-wasm/README.md) — Go chess engine + AI
- [`../AGENTS.md`](../AGENTS.md) — full architecture, current state, contribution rules