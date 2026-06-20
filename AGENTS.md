# Project State — Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI runs in TypeScript
- **Xadrez** (Chess) — fully playable, including human-vs-AI; move generation and board state run in Go compiled to WebAssembly, AI runs in Go (separate `pkg/ai` package)

The Go WASM integration is **live**: the worker loads the engine, the React hook calls it for valid moves and move application, pawn promotion is wired end-to-end, and the chess AI searches via `ai.Search()` exposed as `aiMove` / `aiMoveDepth`.

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
│   │   └── chess/             # Route /chess
│   │       ├── Chess.tsx              # Renders board, turn banner, promotion picker, "Xeque!" badge, result overlay, AI setup panel, clock config, move history sidebar
│   │       ├── Chess.hooks.ts         # useChess: state machine bridging React ↔ Go WASM, AI turn effect, difficulty/time/depth search modes, move history + navigation, clock integration
│   │       └── Chess.module.css
│       ├── components/
│       │   ├── Board/             # Checkers board UI (Board.tsx + Board.module.css)
│       │   ├── ChessBoard/        # Chess board UI (cburnett SVG pieces, selection + move hints + check glow)
│       │   ├── MoveHistory/       # Move history sidebar (SAN notation, ply navigation, clocks display, result box)
│       │   ├── PromotionPicker/   # Pawn promotion modal: Q/N/R/B picker using piece SVGs
│       │   └── Nav/               # Top nav bar (links to /checkers and /chess)
│       ├── hooks/useGame.ts       # Checkers state machine (uses TS AI)
│       ├── hooks/useChessClock.ts # Chess clock hook: dual countdown, increments, flag-fall detection
│       ├── utils/
│       │   ├── gameEngine.ts      # Checkers: move gen, captures, flying kings, applyMove, turn state
│       │   ├── aiEngine.ts        # Checkers AI: Minimax + Alpha-Beta + IDDFS, depth 8
│       │   ├── chessEngine.ts     # Chess: emptyBoard(), pieceByte(), square helpers (board init is in Go)
│       │   ├── chessAssets.ts     # pieceImageUrl(piece) → cburnett SVG path (shared by ChessBoard + PromotionPicker)
│       │   └── chessNotation.ts   # Chess SAN-like notation generator (toSan, squareName, disambiguation, castling, promotion, check/mate suffixes)
│       ├── types/
│       │   ├── game.ts            # Checkers: Color, PieceType, Piece, Cell, Board, Move
│       │   └── chess.ts           # Chess: ChessColor, ChessPieceType, ChessPiece, ChessBoard, getPiece, decodePieceByte
│       └── assets/                # Static images (hero, react/vite svg)
├── go-wasm/                        # Go source compiled to WASM (module: webassemble, go 1.25)
│   ├── cmd/
│   │   ├── wasm/main.go           # WASM entry: registers goWasmEngine.{validMovesChess, initBoard, makeMove, isCheckJS, gameStatus, aiMove, aiMoveDepth, aiAnalysis}
│   │   └── command/main.go        # CLI debug entry: loads FEN, runs Perft depths 1-5
│   ├── pkg/
│   │   ├── types/types.go         # Piece uint8 (type bits + color bits), CastlingRights uint8 (KQkq), Move struct (with Flag + Captured), MoveFlag enum, Piece methods (IsWhite, IsBlack, IsEnemy, Color, TypePiece)
│   │   ├── engine/                # Chess rules (pure Go, no JS deps)
│   │   │   ├── position.go        # Position struct (Board, WhiteToMove, CastlingRights, EnPassant*, HalfmoveClock, FullmoveNumber, undoStack) + Game global + reset()
│   │   │   ├── helpers.go         # Pure helpers: abs, inBounds, oppositeColor, colorOfSide(); legacy free fns KingCheck()/Perft() delegating to Game
│   │   │   ├── fen.go             # LoadFen(): parses all 6 FEN fields + squareToIndex() + StartingFEN constant
│   │   │   ├── print.go           # PrintBoard(): ASCII debug print
│   │   │   ├── moves.go           # Position.PseudoLegalMoves(ml *MoveList): iterate board, dispatch per piece type, writes into caller-owned MoveList
│   │   │   ├── movelist.go        # MoveList struct: [256]Move inline array + count, methods (Add, Len, Get, Clear, Slice) — stack-allocatable, zero-alloc in hot paths
│   │   │   ├── move_pawn.go       # Pawn moves: forward, double, captures, en passant, promotion; inline capture loop with IsEnemy guard
│   │   │   ├── move_knight.go     # Knight L-jumps with IsEnemy guard
│   │   │   ├── move_bishop.go     # Bishop diagonal slides with IsEnemy guard
│   │   │   ├── move_rook.go       # Rook orthogonal slides with IsEnemy guard
│   │   │   ├── move_king.go       # King one-step + delegates castling to generateCastling()
│   │   │   ├── castling.go        # castleSide struct + castleSides [4]castleSide table + generateCastling(): data-driven, all 6 FIDE conditions as sequential guards with comments
│   │   │   ├── make.go            # Position.Make(move) + Position.Unmake(move): flagged make/unmake with undoStack (O(1) reversible); legacy MakeMove(from,to,promo) bridge fn
│   │   │   ├── attacks.go         # Position.FindKing, Position.IsSquareAttacked (reverse-scan), Position.IsInCheck
│   │   │   ├── legal.go           # Position.LegalMoves(ml *MoveList): pseudo-legal → Make/Unmake filter; LegalMovesSlice() convenience wrapper for WASM bridge
│   │   │   ├── status.go          # GameStatus enum + Position.CurrentStatus(): playing / white-wins / black-wins / draw; statusFor takes moveCount int
│   │   │   ├── perft.go           # Position.Perft(depth): recursive node count using Make/Unmake + stack-allocated MoveList per ply
│   │   │   ├── legal_test.go      # Tests: FEN loading, castling rights, legal move counts, pins, en-passant discovered check, king-in-check
│   │   │   ├── fen_test.go        # Tests: en passant target parsing, halfmove clock, fullmove number, squareToIndex, Make/Unmake clock management
│   │   │   ├── status_test.go     # Tests: CurrentStatus, GameStatus.String/IsGameOver, statusFor
│   │   │   └── perft_test.go      # Tests: Perft on all 6 chessprogramming.org standard positions
│   │   └── ai/                    # Chess AI (pure Go, no JS deps except build-tagged clock)
│   │       ├── ai.go              # Evaluate(p *Position): material values + 6 piece-square tables (64 squares each)
│   │       ├── search.go          # Search(p, timeLimitMs) + SearchFixedDepth(p, depth): negamax + alpha-beta + iterative deepening; uses PseudoLegalMoves + lazy IsInCheck
│   │       ├── clock_wasm.go      # nowMs() via js.performance.now() — build tag: js && wasm
│   │       ├── clock_native.go    # nowMs() via time.Now().UnixMilli() — build tag: !(js && wasm)
│   │       └── ai_test.go         # Tests: evaluation, mate-in-1, mate-in-1-black, win hanging piece, search properties, depth scaling, NPS measurement, benchmarks
│   ├── tools/main.go              # Type generator: Go AST + type-checker → wasm-contract.ts (optional, starting point only)
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
- Game runs in the browser via Go WASM: human-vs-human, **human-vs-AI** (AI plays either color), and **AI-vs-AI** (both sides played by the engine)
- **Chess AI** (`pkg/ai`): negamax + alpha-beta + iterative deepening in Go; material + piece-square table evaluation; captures-first move ordering; time-limited or fixed-depth search
- Go engine handles: board representation, FEN loading (all 6 fields), move generation for all piece types, captures, en passant, pawn promotion, **castling**
- **Position struct**: all game state is encapsulated in `Position` (Board, WhiteToMove, CastlingRights, EnPassantTarget/Capture, HalfmoveClock, FullmoveNumber, undoStack); a global `Game *Position` is used by the WASM bridge; the AI uses the same Position via `Make`/`Unmake`
- **MoveList**: fixed `[256]Move` inline array + count, passed as `*MoveList` to move generators — stack-allocated, zero heap allocation in hot paths (perft, legal moves, AI search)
- **Flagged moves**: `Move` carries a `MoveFlag` (Normal | DoublePush | EnPassant | CastleK | CastleQ | Promotion) and a `Captured` piece — internal-only fields (`json:"-"`) so the frontend contract stays `{from, to, promotion?}`
- **Make/Unmake**: `Position.Make(move)` applies a move incrementally and pushes undo info onto a stack; `Position.Unmake(move)` reverses it in O(1) — no full board copy, the performance foundation for AI search
- **Castling**: data-driven via `castleSides [4]castleSide` table in `castling.go`; all 6 FIDE conditions checked as sequential guards with comments; rook moves with the king on `Make`; castling rights cleared on king/rook moves and rook captures
- **Piece.IsEnemy()**: unified enemy detection (`other&ColorMask != ColorNone && p&ColorMask != other&ColorMask`) — replaces duplicated color-branch logic in all move generators; correctly rejects empty squares
- **Legal move filtering**: pseudo-legal moves are filtered by Make/Unmake — pins, en-passant discovered checks, and king-moves-into-check are all handled automatically
- **Check detection**: `Position.IsSquareAttacked` (reverse-scan from a square) + `Position.IsInCheck` (king square attacked) + `KingCheck()` exposed to the frontend as `isCheckJS`
- **Game status**: `Position.CurrentStatus()` returns `playing | white-wins | black-wins | draw`, exposed as `gameStatus`
- **Perft validation**: `Position.Perft()` runs recursive move enumeration using Make/Unmake + stack-allocated MoveList; validated against all 6 standard positions from chessprogramming.org/Perft_Results
- Pawn promotion: `PromotionPicker` modal lets the user choose Q/N/R/B; the AI returns promotion bytes automatically
- Move validation: clicking a piece queries `engine.validMovesChess()`, highlights legal targets
- **Check highlight**: king's square glows red + "Xeque!" badge in turn banner
- **Result overlay**: "Brancas vencem!" / "Pretas vencem!" / "Empate!"
- **AI setup panel**: user chooses their color (board auto-flips), search mode (difficulty / custom time / custom depth), and difficulty level (Fácil/Médio/Difícil)
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
- Board flip, turn banner, result overlay

**AI architecture**
- **Separate `pkg/ai` package**: imports `pkg/engine` + `pkg/types`, clean one-directional dependency; engine doesn't know AI exists
- **Stateless `Search(p *Position, timeLimitMs int) SearchResult`**: matches the checkers AI pattern; `SearchFixedDepth(p, depth)` for benchmarking
- **Negamax** (not minimax with isMaximizing): negation handles perspective switching — simpler code
- **Pseudo-legal moves + lazy `IsInCheck`**: one Make/Unmake per move (not two like LegalMoves would force); the AI uses `PseudoLegalMoves` directly, skipping `LegalMoves`
- **Captures-first move ordering**: same as checkers; can upgrade to MVV-LVA later
- **Build-tagged clock**: `clock_wasm.go` (JS `performance.now()`) and `clock_native.go` (Go `time.Now()`) — `pkg/ai` compiles and tests natively with `go test ./pkg/ai/`, no WASM needed
- **Escape analysis verified**: `MoveList` stays on stack in perft, legal moves, AI search; only `LegalMovesSlice` (WASM bridge, cold path) allocates

**WASM infrastructure**
- `worker.js` exists and works: loads `wasm_exec.js` + `engine.wasm`, instantiates `goWasmEngine`, dispatches `{ id, fn, args }` messages, replies `{ id, result, error }`
- `WasmWorkerEngine` (`loader.ts`): typed async wrappers over the Web Worker, pending-promise map, restart support
- `useWasm` hook: loads engine, exposes `{ engine, loading, error, restarting }`, listens for `wasm-rebuild` HMR events and restarts the worker without a full page reload
- Vite plugin (`plugins/go-wasm.ts`): compiles `engine.wasm` on dev start and prod build, copies `wasm_exec.js` from GOROOT, watches all `.go` files and rebuilds on change, sends `wasm-rebuild` HMR event

### What is missing / in progress

| Piece | Notes |
|---|---|
| Transposition tables | No Zobrist hashing + TT cache yet — would need a `Searcher` struct with state |
| Quiescence search | No capture-only extension past depth 0 yet — improves tactical accuracy |
| Opening book | No opening repertoire — AI plays from first principles every game |
| Type generator auto-run | The Vite plugin does **not** run the type generator — it only builds the WASM and sends HMR. `wasm-contract.ts` is maintained by hand. |
| Optional arg in contract | The generator emits all params as required. `makeMove`'s 3rd arg (`promotion`) was manually marked optional in `wasm-contract.ts`. |
| Checkers → Go | Checkers logic stays in TypeScript for now; no plan to port it to Go. |

---

## WASM Integration Architecture

### Call flow

```
React component
  → useWasm() hook  →  WasmWorkerEngine.makeMove(from, to, promotion)
    → loader.ts: postMessage({ id: 0, fn: "makeMove", args: [from, to, promotion] })
      → worker.js: self.goWasmEngine.makeMove(from, to, promotion)
        → cmd/wasm/main.go: makeMoveJS → engine.MakeMove(from, to, promotion)
      → worker.js: postMessage({ id: 0, result: [board bytes...] })
    → loader.ts: Promise resolves with number[]
```

### Registered functions (goWasmEngine)

| JS name | Go bridge | Pure function | Args | Return |
|---|---|---|---|---|
| `validMovesChess` | `getValidMovesJS` | `engine.Game.LegalMovesSlice()` | — | JSON string of `Move[]` |
| `initBoard` | `initBoardJs` | `engine.LoadFen(engine.StartingFEN)` | — | `number[]` (64 board bytes) |
| `makeMove` | `makeMoveJS` | `engine.MakeMove(from, to, promotion)` | `number, number, number?` | `number[]` (64 board bytes) |
| `isCheckJS` | `isCheckJS` | `engine.KingCheck()` | — | `number` (checked king's square index, or -1) |
| `gameStatus` | `gameStatusJS` | `engine.CurrentStatus().String()` | — | `string` (`"playing"` \| `"white-wins"` \| `"black-wins"` \| `"draw"`) |
| `aiMove` | `aiMoveJS` | `ai.Search(engine.Game, timeLimitMs)` | `number` (time limit ms) | JSON string `{from, to, promotion?}` |
| `aiMoveDepth` | `aiMoveDepthJS` | `ai.SearchFixedDepth(engine.Game, depth)` | `number` (depth) | JSON string `{from, to, promotion?}` |
| `aiAnalysis` | `aiAnalysisJS` | `ai.Search(engine.Game, timeLimitMs)` | `number` (time limit ms) | JSON string `{from, to, promotion?, score, depth, nodes, timeMs}` |

### Piece byte encoding (shared between Go and TS)

```
bits 0-5: piece type (one-hot)
  Pawn=1, Knight=2, Bishop=4, Rook=8, Queen=16, King=32
bits 6-7: color
  00=empty, 01=white (0b01000000), 10=black (0b10000000)
```

`Move.Promotion` is a `Piece` (color bits | type bits) — `omitempty` in JSON means it's omitted when 0 (no promotion). The Go engine emits 4 separate moves per promotable pawn push (Q, N, B, R), each with a different `Promotion` byte. `Move.Flag` and `Move.Captured` are internal-only (`json:"-"`).

### Castling rights encoding

```
CastlingRights uint8 bitmask:
  bit 0: CastleWhiteK  (white kingside,  e1→g1,  rook h1→f1)
  bit 1: CastleWhiteQ  (white queenside, e1→c1,  rook a1→d1)
  bit 2: CastleBlackK  (black kingside,  e8→g8,  rook h8→f8)
  bit 3: CastleBlackQ  (black queenside, e8→c8,  rook a8→d8)
```

Parsed from FEN field 2 (`KQkq` or `-`) by `LoadFen`. Castling generation is data-driven via the `castleSides` table in `castling.go`; rights are cleared in `MakeMove` on king moves, rook moves from corners, and rook captures on corners.

### AI search architecture

```
cmd/wasm/main.go
    ↓ imports
pkg/ai              ← Search(), SearchFixedDepth(), Evaluate()
    ↓ imports
pkg/engine           ← Position, MoveList, Make/Unmake, PseudoLegalMoves, IsInCheck, CurrentStatus
    ↓ imports
pkg/types            ← Move, Piece, constants
```

The AI uses `PseudoLegalMoves` + lazy `IsInCheck` (one Make/Unmake per move, not two). Move ordering is captures-first. The clock is build-tagged: `clock_wasm.go` uses `js.performance.now()`, `clock_native.go` uses `time.Now()`. The `pkg/ai` package compiles and tests natively with `go test ./pkg/ai/` — no WASM needed for development.

### Type safety pipeline

```
go-wasm/pkg/engine/*.go + pkg/ai/*.go + pkg/types/types.go   (real Go types)
  ↓  tools/main.go (type generator — optional, for a starting point only)
front/src/wasm/generated/wasm-contract.ts       (hand-maintained TS contract)
  ↓  imported by
front/src/wasm/loader.ts                        (WasmWorkerEngine: typed async wrappers)
```

`wasm-contract.ts` is hand-maintained — edit it directly when Go function signatures change.

### How to add a new Go function

1. Add the pure function to a file in `pkg/engine/` or `pkg/ai/` (or `pkg/types/`)
2. Add a bridge wrapper to `cmd/wasm/main.go` and register it with `e.Set("jsName", js.FuncOf(bridgeFunc))`
3. Save — the Vite plugin rebuilds `engine.wasm` automatically and sends the HMR event
4. **Manually** update `wasm-contract.ts` to reflect the new/changed function signature
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

# Go engine (from go-wasm/)
go test ./pkg/engine/ -v          # engine tests (native, no WASM)
go test ./pkg/ai/ -v -short       # AI tests (native, no WASM) — fast mode
go test ./pkg/ai/ -v              # AI tests including depth scaling + NPS measurement
go test ./pkg/ai/ -bench=.        # AI benchmarks (nodes/sec, eval speed)
go run ./cmd/command              # CLI debug: loads FEN, runs Perft depths 1-5

# Go WASM build (from go-wasm/)
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o ../front/public/wasm/engine.wasm ./cmd/wasm

# Type generator (from go-wasm/) — optional, for a starting point only
go build -o bin/gen-types.exe tools/main.go
./bin/gen-types.exe
```

PowerShell WASM build (Windows):
```powershell
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o ../front/public/wasm/engine.wasm ./cmd/wasm
```

---

## Rules

- Run `bun run check` from `front/` after every file change.
- Run `go test ./pkg/engine/ ./pkg/ai/` from `go-wasm/` after Go changes.
- `wasm-contract.ts` is hand-maintained — edit it directly when Go function signatures change. Never run `gen-types.exe` as part of the normal workflow.
- The Vite plugin **only** builds the WASM and sends HMR events — it does **not** run the type generator.
- `worker.js` must be plain JavaScript (no bundler) — it runs inside a Web Worker with no import support unless using `importScripts`.
- `wasm_exec.js` is copied from `GOROOT/lib/wasm/wasm_exec.js` by the Vite plugin on build — do not edit it.
- `pkg/ai` has no `//go:build js && wasm` tag on its core files — only `clock_wasm.go` and `clock_native.go` are build-tagged. The AI package compiles and tests natively.
- All user-facing text is in Portuguese (pt-BR). All code (variables, functions, files) is in English.

## Documentation — update rule

- When asked to "update the docs", update **only** the root `README.md` and `AGENTS.md`.
- When asked to "update all the docs", update **ALL** of these files to keep them in sync:

  1. **`AGENTS.md`** (root) — project state, architecture, repository layout, registered functions, encoding, rules
  2. **`README.md`** (root) — project overview, stack, getting started, current state summary
  3. **`front/README.md`** — frontend structure, WASM integration, UI features, registered JS functions
  4. **`go-wasm/README.md`** — engine structure, piece/castling encoding, perft validation, JS bridge table, AI package

Update each file's relevant section based on what changed (new files, new functions, new features, changed architecture, test results, etc.). Do not skip any of the four files on an "all" update — they are the canonical documentation set.