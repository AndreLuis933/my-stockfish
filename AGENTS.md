# Project State — Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI runs in TypeScript
- **Xadrez** (Chess) — playable, move generation and board state run in Go compiled to WebAssembly

The Go WASM integration is **live**: the worker loads the engine, the React hook calls it for valid moves and move application, and pawn promotion is wired end-to-end.

---

## Repository Layout

```
my-stockfish/
├── front/                            # React 19 + TypeScript (strict) + Vite (Bun)
│   ├── plugins/go-wasm.ts            # Vite plugin: builds WASM on dev start + prod build, watches .go, sends HMR
│   ├── public/
│   │   ├── wasm/
│   │   │   ├── engine.wasm           # Compiled Go WASM binary (gitignored)
│   │   │   ├── wasm_exec.js          # Go WASM runtime (copied from GOROOT at build time)
│   │   │   └── worker.js             # Web Worker: loads wasm_exec.js + engine.wasm, dispatches { id, fn, args }
│   │   └── pieces/                   # Chess piece SVGs (cburnett + chessnut sets)
│   │       ├── cburnett/{wP,wN,wB,wR,wQ,wK,bP,bN,bB,bR,bQ,bK}.svg
│   │       └── chessnut/...
│   └── src/
│       ├── App.tsx                   # Router: / → /checkers, /checkers, /chess
│       ├── main.tsx                  # React root
│       ├── wasm/
│       │   ├── generated/wasm-contract.ts   # Hand-maintained TS contract for the Go functions (edit directly)
│       │   ├── loader.ts                    # WasmWorkerEngine class (Web Worker bridge, typed async calls)
│       │   └── useWasm.ts                   # React hook: { engine, loading, error, restarting } + HMR restart
│       ├── pages/
│       │   ├── checkers/          # Route /checkers
│       │   │   ├── Checkers.tsx
│       │   │   └── Checkers.module.css
│       │   └── chess/             # Route /chess
│       │       ├── Chess.tsx              # Renders board, turn banner, promotion picker, result overlay
│       │       ├── Chess.hooks.ts         # useChess: state machine bridging React ↔ Go WASM
│       │       └── Chess.module.css
│       ├── components/
│       │   ├── Board/             # Checkers board UI (Board.tsx + Board.module.css)
│       │   ├── ChessBoard/        # Chess board UI (cburnett SVG pieces, selection + move hints)
│       │   ├── PromotionPicker/   # Pawn promotion modal: Q/N/R/B picker using piece SVGs
│       │   └── Nav/               # Top nav bar (links to /checkers and /chess)
│       ├── hooks/useGame.ts       # Checkers state machine (uses TS AI)
│       ├── utils/
│       │   ├── gameEngine.ts      # Checkers: move gen, captures, flying kings, applyMove, turn state
│       │   ├── aiEngine.ts        # Checkers AI: Minimax + Alpha-Beta + IDDFS, depth 8
│       │   ├── chessEngine.ts     # Chess: emptyBoard(), pieceByte(), square helpers (board init is in Go)
│       │   └── chessAssets.ts     # pieceImageUrl(piece) → cburnett SVG path (shared by ChessBoard + PromotionPicker)
│       ├── types/
│       │   ├── game.ts            # Checkers: Color, PieceType, Piece, Cell, Board, Move
│       │   └── chess.ts           # Chess: ChessColor, ChessPieceType, ChessPiece, ChessBoard, getPiece, decodePieceByte
│       └── assets/                # Static images (hero, react/vite svg)
├── go-wasm/                        # Go source compiled to WASM (module: webassemble, go 1.25)
│   ├── cmd/
│   │   ├── wasm/main.go           # WASM entry: registers goWasmEngine.{validMovesChess, initBoard, makeMove}
│   │   └── command/main.go        # CLI debug entry: loads FEN, prints valid moves
│   ├── pkg/
│   │   ├── types/types.go         # Piece uint8 (type bits + color bits), Move struct {From, To, *Promotion}
│   │   └── engine/                # Chess logic (pure Go, no JS deps)
│   │       ├── board.go           # Board [64]Piece, enPassant state, inBounds, abs, PiecePtr
│   │       ├── anotation.go       # LoadFen(): parse FEN string → Board
│   │       ├── print.go           # PrintBoard(): ASCII debug print
│   │       ├── moves.go           # GetValidMoves(): iterate board, dispatch per piece type
│   │       ├── move_pawn.go       # Pawn moves: forward, double, captures, en passant, promotion (4 moves)
│   │       ├── move_knight.go     # Knight L-jumps with row/col diff validation
│   │       ├── move_bishop.go     # Bishop diagonal slides
│   │       ├── move_rook.go       # Rook orthogonal slides
│   │       ├── move_king.go       # King one-step in all 8 directions
│   │       └── move_apply.go      # MakeMovement(from, to, promotion): applies move, sets en passant state
│   ├── tools/main.go              # Type generator: Go AST + type-checker → wasm-contract.ts
│   └── bin/gen-types.exe          # Compiled type generator binary
├── AGENTS.md                       # This file
├── CLAUDE.md                       # Symlink/global coding preferences
└── .gitignore
```

---

## Current State

### What works

**Damas Brasileiras (Checkers)**
- Full game in the browser: human-vs-AI, human-vs-human, AI-vs-AI
- AI: Minimax + Alpha-Beta + IDDFS in TypeScript (`aiEngine.ts`), default depth 8
- Brazilian rules: mandatory max captures, flying kings, multi-jump chains, 40-move draw rule
- Board flip, piece counters, "IA pensando..." indicator, must-move highlighting

**Xadrez (Chess)**
- Game runs in the browser via Go WASM: human-vs-human (human-vs-ai mode exists but AI is not yet implemented for chess)
- Go engine handles: board representation, FEN loading, move generation for all piece types, captures, en passant, pawn promotion
- Pawn promotion: when a pawn reaches the last rank, a `PromotionPicker` modal lets the user choose Q/N/R/B; the chosen piece byte is sent to `engine.makeMove(from, to, promotionByte)`
- Move validation: clicking a piece queries `engine.validMovesChess()` (JSON of `{from, to, promotion?}[]`), highlights legal targets
- Board flip, turn banner, result overlay

**WASM infrastructure**
- `worker.js` exists and works: loads `wasm_exec.js` + `engine.wasm`, instantiates `goWasmEngine`, dispatches `{ id, fn, args }` messages, replies `{ id, result, error }`
- `WasmWorkerEngine` (`loader.ts`): typed async wrappers over the Web Worker, pending-promise map, restart support
- `useWasm` hook: loads engine, exposes `{ engine, loading, error, restarting }`, listens for `wasm-rebuild` HMR events and restarts the worker without a full page reload
- Vite plugin (`plugins/go-wasm.ts`): compiles `engine.wasm` on dev start and prod build, copies `wasm_exec.js` from GOROOT, watches all `.go` files and rebuilds on change, sends `wasm-rebuild` HMR event
- Type generator (`tools/main.go`): reads Go AST + type-checks `pkg/engine` → writes `wasm-contract.ts`

### What is missing / in progress

| Piece | Notes |
|---|---|
| Chess AI | No minimax/alpha-beta in Go yet. `human-vs-ai` mode is defined but black doesn't move automatically in chess. |
| Check/checkmate detection | Go move generation does not filter out moves that leave own king in check; no checkmate/stalemate result logic. |
| Castling | Not implemented in Go move generation. |
| King/Queen move generation edge cases | King has no castling; queen = rook + bishop (correct). King doesn't validate moving into check. |
| Type generator auto-run | The Vite plugin does **not** run the type generator automatically — it only builds the WASM and sends HMR. `wasm-contract.ts` is maintained by hand; run `gen-types.exe` only if you want to regenerate a starting point. |
| Optional arg in contract | The generator emits all params as required. `makeMove`'s 3rd arg (`promotion`) was manually marked optional (`number?`) in `wasm-contract.ts`. Re-apply optionality if you regenerate. |
| Checkers → Go | Checkers logic stays in TypeScript for now; no plan to port it to Go. |

---

## WASM Integration Architecture

### Call flow

```
React component
  → useWasm() hook  →  WasmWorkerEngine.makeMove(from, to, promotion)
    → loader.ts: postMessage({ id: 0, fn: "makeMove", args: [from, to, promotion] })
      → worker.js: self.goWasmEngine.makeMove(from, to, promotion)
        → cmd/wasm/main.go: makeMoveJS → engine.MakeMovement(from, to, promotion)
      → worker.js: postMessage({ id: 0, result: [board bytes...] })
    → loader.ts: Promise resolves with number[]
```

### Registered functions (goWasmEngine)

| JS name | Go bridge | Pure function | Args | Return |
|---|---|---|---|---|
| `validMovesChess` | `getValidMovesJS` | `engine.GetValidMoves()` | — | JSON string of `Move[]` |
| `initBoard` | `initBoardJs` | `engine.LoadFen("rnbqkbnr/...")` | — | `number[]` (64 board bytes) |
| `makeMove` | `makeMoveJS` | `engine.MakeMovement(from, to, promotion)` | `number, number, number?` | `number[]` (64 board bytes) |

### Piece byte encoding (shared between Go and TS)

```
bits 0-5: piece type (one-hot)
  Pawn=1, Knight=2, Bishop=4, Rook=8, Queen=16, King=32
bits 6-7: color
  00=empty, 01=white (0b01000000), 10=black (0b10000000)
```

`Move.Promotion` is a `*Piece` (color bits | type bits) — only set for pawn promotion moves. The Go engine emits 4 separate moves per promotable pawn push (Q, N, B, R), each with a different `Promotion` byte. The frontend collects these into a `PendingPromotion` and shows the picker.

### Type safety pipeline

```
go-wasm/pkg/engine/*.go + pkg/types/types.go   (real Go types)
  ↓  tools/main.go (type generator — optional, for a starting point only)
front/src/wasm/generated/wasm-contract.ts       (hand-maintained TS contract)
  ↓  imported by
front/src/wasm/loader.ts                        (WasmWorkerEngine: typed async wrappers)
```

`wasm-contract.ts` is hand-maintained — edit it directly when Go function signatures change. The type generator (`tools/main.go`) is only a starting point; it does not run automatically and the generated output must be hand-edited (e.g. to mark optional params) before use.

### How to add a new Go function

1. Add the pure function to a file in `pkg/engine/` (or `pkg/types/`)
2. Add a bridge wrapper to `cmd/wasm/main.go` and register it with `e.Set("jsName", js.FuncOf(bridgeFunc))`
3. Save — the Vite plugin rebuilds `engine.wasm` automatically and sends the HMR event
4. **Manually** update `wasm-contract.ts` to reflect the new/changed function signature (edit the file directly; run `gen-types.exe` only if you want a regenerated starting point)
5. Add the typed wrapper to `WasmWorkerEngine` in `loader.ts`: `newFn = this.fn("jsName")`
6. `worker.js` needs no changes — it dispatches generically by function name

---

## Commands

```bash
# Frontend (from front/)
bun dev           # dev server — also builds WASM and watches .go files
bun run check     # tsc -b && eslint:strict  (run after every change)
bun test          # vitest
bun run build     # tsc -b && vite build (also builds WASM with prod flags)

# Go WASM (from go-wasm/)
GOOS=js GOARCH=wasm go build -o ../front/public/wasm/engine.wasm ./cmd/wasm

# Type generator (from go-wasm/) — optional, for a starting point only
go build -o bin/gen-types.exe tools/main.go
./bin/gen-types.exe

# CLI debug (from go-wasm/) — loads standard FEN, prints valid moves
go run ./cmd/command
```

PowerShell WASM build (Windows):
```powershell
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o ../front/public/wasm/engine.wasm ./cmd/wasm
```

---

## Rules

- Run `bun run check` from `front/` after every file change.
- `wasm-contract.ts` is hand-maintained — edit it directly when Go function signatures change. Never run `gen-types.exe` as part of the normal workflow; it is only a starting point and its output must be hand-edited (e.g. to mark optional params) before use.
- The Vite plugin **only** builds the WASM and sends HMR events — it does **not** run the type generator.
- `worker.js` must be plain JavaScript (no bundler) — it runs inside a Web Worker with no import support unless using `importScripts`.
- `wasm_exec.js` is copied from `GOROOT/lib/wasm/wasm_exec.js` by the Vite plugin on build — do not edit it.
- All user-facing text is in Portuguese (pt-BR). All code (variables, functions, files) is in English.