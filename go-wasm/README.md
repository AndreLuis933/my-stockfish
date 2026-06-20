# Go WASM — Chess Engine + AI

Go 1.25 chess engine and AI compiled to WebAssembly. Powers the Xadrez (Chess) game in the React frontend.

---

## What it does

### Engine (`pkg/engine`)

The engine handles all chess logic for the browser app:

- **Board representation**: `[64]Piece` array (mailbox indexing, a1=0, h8=63)
- **FEN loading**: piece placement, side to move, castling rights (`KQkq`), en passant target, halfmove clock, fullmove number
- **Move generation**: all piece types — pawn (forward, double, captures, en passant, promotion), knight, bishop, rook, queen, king (one-step + castling)
- **MoveList**: fixed `[256]Move` inline array + count, passed as `*MoveList` to move generators — stack-allocated, zero heap allocation in hot paths
- **Castling**: data-driven via `castleSides [4]castleSide` table; all 6 FIDE conditions checked as sequential guards with comments; rook moves with the king on `Make`; rights cleared on king/rook moves and rook captures
- **Piece.IsEnemy()**: unified enemy detection (`other&ColorMask != ColorNone && p&ColorMask != other&ColorMask`) — replaces duplicated color-branch logic in all move generators; correctly rejects empty squares
- **Make/Unmake**: `Position.Make(move)` applies a move incrementally and pushes undo info onto a stack; `Position.Unmake(move)` reverses it in O(1) — no full board copy, the performance foundation for AI search
- **Legal move filtering**: pseudo-legal moves filtered by Make/Unmake — pins, en-passant discovered checks, and king-moves-into-check handled automatically
- **Check detection**: `IsSquareAttacked` (reverse-scan from a square) + `IsInCheck` (king square attacked) + `KingCheck()` (returns checked king's square index or -1)
- **Game status**: `CurrentStatus()` returns `playing | white-wins | black-wins | draw` (checkmate = no legal moves + in check, stalemate = no legal moves + not in check)
- **Perft validation**: `Perft()` recursive move enumeration using Make/Unmake + stack-allocated MoveList; validated against all 6 standard positions from chessprogramming.org/Perft_Results

### AI (`pkg/ai`)

Separate package with clean one-directional dependency (`pkg/ai` → `pkg/engine` → `pkg/types`). The engine doesn't know the AI exists.

- **Evaluation**: material values (pawn=100, knight=320, bishop=330, rook=500, queen=900, king=20000) + 6 piece-square tables (64 squares each)
- **Search**: negamax + alpha-beta pruning + iterative deepening (IDDFS)
- **Move ordering**: captures-first (same as checkers AI); can upgrade to MVV-LVA later
- **Pseudo-legal moves + lazy `IsInCheck`**: one Make/Unmake per move (not two like LegalMoves would force)
- **Terminal detection**: `CurrentStatus()` → checkmate/stalemate/draw returns ±winScore or 0
- **Time-limited search**: `Search(p, timeLimitMs)` — iterative deepening until time budget expires
- **Fixed-depth search**: `SearchFixedDepth(p, depth)` — no time limit, for benchmarking
- **Build-tagged clock**: `clock_wasm.go` (JS `performance.now()`) and `clock_native.go` (Go `time.Now()`) — compiles and tests natively, no WASM needed for development
- **Escape analysis verified**: `MoveList` stays on stack in perft, legal moves, AI search; only `LegalMovesSlice` (WASM bridge, cold path) allocates

---

## Project structure

```
go-wasm/
├── cmd/
│   ├── wasm/main.go           # WASM entry: registers goWasmEngine JS functions
│   └── command/main.go        # CLI debug: loads FEN, runs Perft
├── pkg/
│   ├── types/types.go         # Piece uint8, CastlingRights uint8, Move struct, MoveFlag enum, Piece methods (IsWhite, IsBlack, IsEnemy, Color, TypePiece)
│   ├── engine/                # Chess rules (pure Go, no JS deps)
│   │   ├── position.go        # Position struct + Game global + reset()
│   │   ├── helpers.go         # abs, inBounds, oppositeColor, colorOfSide(); legacy KingCheck()/Perft()
│   │   ├── fen.go             # LoadFen(): parses all 6 FEN fields + StartingFEN
│   │   ├── print.go           # PrintBoard(): ASCII debug
│   │   ├── moves.go           # PseudoLegalMoves(ml *MoveList): iterate board, dispatch per piece type
│   │   ├── movelist.go        # MoveList: [256]Move + count (Add, Len, Get, Clear, Slice) — stack-allocatable
│   │   ├── move_pawn.go       # Pawn moves: forward, double, captures, en passant, promotion
│   │   ├── move_knight.go     # Knight L-jumps with IsEnemy guard
│   │   ├── move_bishop.go     # Bishop diagonal slides with IsEnemy guard
│   │   ├── move_rook.go       # Rook orthogonal slides with IsEnemy guard
│   │   ├── move_king.go       # King one-step + delegates castling to generateCastling()
│   │   ├── castling.go        # castleSides [4]castleSide table + generateCastling() + isPathAttacked() + isEmpty()
│   │   ├── make.go            # Make(move) + Unmake(move): flagged make/unmake with undoStack; legacy MakeMove() bridge
│   │   ├── attacks.go         # FindKing, IsSquareAttacked (reverse-scan), IsInCheck
│   │   ├── legal.go           # LegalMoves(ml *MoveList): pseudo-legal → Make/Unmake filter; LegalMovesSlice() for WASM
│   │   ├── status.go          # GameStatus enum + CurrentStatus(); statusFor takes moveCount int
│   │   ├── perft.go           # Perft(depth): recursive node count using Make/Unmake + stack MoveList per ply
│   │   ├── legal_test.go      # Tests: FEN, castling rights, legal moves, pins, en passant, king-in-check
│   │   ├── fen_test.go        # Tests: en passant target, halfmove clock, fullmove number, Make/Unmake clocks
│   │   ├── status_test.go     # Tests: CurrentStatus, GameStatus.String/IsGameOver, statusFor
│   │   └── perft_test.go      # Tests: Perft on all 6 chessprogramming.org positions
│   └── ai/                    # Chess AI (pure Go, no JS deps except build-tagged clock)
│       ├── ai.go              # Evaluate(): material + 6 piece-square tables
│       ├── search.go          # Search() + SearchFixedDepth(): negamax + alpha-beta + IDDFS
│       ├── clock_wasm.go      # nowMs() via js.performance.now() — build tag: js && wasm
│       ├── clock_native.go    # nowMs() via time.Now().UnixMilli() — build tag: !(js && wasm)
│       └── ai_test.go         # Tests: eval, mate-in-1, win material, search properties, depth scaling, NPS, benchmarks
├── tools/main.go              # Type generator: Go AST → wasm-contract.ts (optional)
└── bin/gen-types.exe          # Compiled type generator
```

---

## Getting started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)

### Run engine tests

```bash
go test ./pkg/engine/ -v
```

### Run AI tests (native, no WASM needed)

```bash
# Fast mode — skips depth scaling + NPS measurements
go test ./pkg/ai/ -v -short

# Full mode — includes depth scaling, NPS measurement, position comparison
go test ./pkg/ai/ -v

# Benchmarks — nodes/sec, eval speed
go test ./pkg/ai/ -bench=.
```

### Run perft validation (CLI debug)

```bash
go run ./cmd/command
# Output:
# depth 1  nodes 20
# depth 2  nodes 400
# depth 3  nodes 8902
# depth 4  nodes 197281
# depth 5  nodes 4865609
```

### Build WASM (normally done by the Vite plugin automatically)

```bash
# Linux/macOS
GOOS=js GOARCH=wasm go build -o ../front/public/wasm/engine.wasm ./cmd/wasm

# PowerShell (Windows)
$env:GOOS="js"; $env:GOARCH="wasm"; go build -o ../front/public/wasm/engine.wasm ./cmd/wasm
```

---

## Piece byte encoding

```
Piece uint8:
  bits 0-5: type (one-hot)
    Pawn=1, Knight=2, Bishop=4, Rook=8, Queen=16, King=32
  bits 6-7: color
    00=empty, 01=white (0b01000000), 10=black (0b10000000)
```

`Move.Promotion` is a `Piece` (color bits | type bits) — `omitempty` in JSON means it's omitted when 0 (no promotion). The engine emits 4 separate moves per promotable pawn push (Q, N, B, R), each with a different `Promotion` byte. `Move.Flag` and `Move.Captured` are internal-only (`json:"-"`).

## Castling rights encoding

```
CastlingRights uint8 bitmask:
  bit 0: CastleWhiteK  (white kingside,  e1→g1,  rook h1→f1)
  bit 1: CastleWhiteQ  (white queenside, e1→c1,  rook a1→d1)
  bit 2: CastleBlackK  (black kingside,  e8→g8,  rook h8→f8)
  bit 3: CastleBlackQ  (black queenside, e8→c8,  rook a8→d8)
```

Parsed from FEN field 2 (`KQkq` or `-`) by `LoadFen`. Castling generation is data-driven via the `castleSides` table in `castling.go`; rights are cleared in `Make` on king moves, rook moves from corners, and rook captures on corners.

---

## AI architecture

### Package dependency

```
cmd/wasm/main.go
    ↓ imports
pkg/ai              ← Search(), SearchFixedDepth(), Evaluate()
    ↓ imports
pkg/engine           ← Position, MoveList, Make/Unmake, PseudoLegalMoves, IsInCheck, CurrentStatus
    ↓ imports
pkg/types            ← Move, Piece, constants
```

### Search algorithm

- **Negamax** with alpha-beta pruning (negation handles perspective switching — simpler than minimax with isMaximizing)
- **Iterative deepening**: depth 1, 2, 3... until time budget expires; partial results from aborted depth are discarded
- **Pseudo-legal moves + lazy `IsInCheck`**: the AI uses `PseudoLegalMoves` directly, skipping `LegalMoves` — one Make/Unmake per move (not two)
- **Captures-first move ordering**: same as checkers AI; can upgrade to MVV-LVA later
- **Terminal detection**: `CurrentStatus()` → checkmate returns `-winScore + depth` (prefer faster mates), stalemate/draw returns 0
- **Time check**: every 2048 nodes via `nowMs()` — build-tagged (`clock_wasm.go` uses `js.performance.now()`, `clock_native.go` uses `time.Now()`)

### Native testing

The `pkg/ai` package compiles and tests natively — no WASM needed. Only `clock_wasm.go` and `clock_native.go` are build-tagged; the core files (`ai.go`, `search.go`, `ai_test.go`) have no build tag.

```bash
go test ./pkg/ai/ -v          # full test suite
go test ./pkg/ai/ -v -short   # fast mode (skips depth scaling + NPS)
go test ./pkg/ai/ -bench=.    # benchmarks
```

### Performance (native, reference)

Measured on the starting position with native `go test`:

| Time limit | Depth | Nodes | Nodes/sec |
|---|---|---|---|
| 100ms | 4 | 28,371 | ~498K |
| 500ms | 4 | 28,371 | ~473K |
| 1000ms | 5 | 246,727 | ~430K |
| 2000ms | 5 | 246,727 | ~432K |
| 5000ms | 6 | 1,940,230 | ~464K |

Browser (WASM) performance is typically 2-3x slower due to WASM overhead.

---

## Registered JS functions (cmd/wasm/main.go)

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

---

## Perft validation

The engine is validated against the 6 standard perft positions from [chessprogramming.org/Perft_Results](https://www.chessprogramming.org/Perft_Results):

| Position | FEN | Depths verified |
|---|---|---|
| Initial | `rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1` | 1-5 (20 → 4,865,609) |
| Kiwipete | `r3k2r/p1ppqpb1/bn2pnp1/3PN3/1p2P3/2N2Q1p/PPPBBPPP/R3K2R w KQkq -` | 1-3 (48 → 97,862) |
| Position 3 | `8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1` | 1-4 (14 → 43,238) |
| Position 4 | `r3k2r/Pppp1ppp/1b3nbN/nP6/BBP1P3/q4N2/Pp1P2PP/R2Q1RK1 w kq - 0 1` | 1-3 (6 → 9,467) |
| Position 5 | `rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8` | 1-3 (44 → 62,379) |
| Position 6 | `r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10` | 1-3 (46 → 89,890) |

Run with `go test ./pkg/engine/ -run TestPerft -v`.

---

## Type generator (optional)

`tools/main.go` reads the Go AST and type-checks `pkg/engine` to generate a starting-point `wasm-contract.ts`. It does **not** run automatically and is not part of the normal workflow — the contract file is hand-maintained in `front/src/wasm/generated/wasm-contract.ts`.

```bash
go build -o bin/gen-types.exe tools/main.go
./bin/gen-types.exe
```

The generated output must be hand-edited (e.g. to mark optional params like `makeMove`'s `promotion`) before use.

---

## See also

- [`../README.md`](../README.md) — project overview
- [`../front/README.md`](../front/README.md) — React frontend
- [`../AGENTS.md`](../AGENTS.md) — full architecture, call flow, contribution rules