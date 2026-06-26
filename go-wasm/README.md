# Go WASM — Chess Engine + AI

Go 1.25 chess engine and AI compiled to WebAssembly. Powers the Xadrez (Chess) game in the React frontend.

---

## What it does

### Engine (`pkg/engine`)

The engine handles all chess logic for the browser app:

- **Board representation**: `[64]Piece` array (mailbox indexing, a1=0, h8=63)
- **FEN loading**: piece placement, side to move, castling rights (`KQkq`), en passant target, halfmove clock, fullmove number; computes initial Zobrist hash + incremental eval score
- **Move generation**: all piece types — pawn (forward, double, captures, en passant, promotion), knight, bishop, rook, queen, king (one-step + castling)
- **Capture-only move generation**: `PseudoLegalCaptures()` — captures + en passant + promotions only, used by quiescence search
- **MoveList**: fixed `[256]Move` inline array + count, passed as `*MoveList` to move generators — stack-allocated, zero heap allocation in hot paths
- **Castling**: data-driven via `castleSides [4]castleSide` table; all 6 FIDE conditions checked as sequential guards with comments; rook moves with the king on `Make`; rights cleared on king/rook moves and rook captures
- **Piece.IsEnemy()**: unified enemy detection (`other&ColorMask != ColorNone && p&ColorMask != other&ColorMask`) — replaces duplicated color-branch logic in all move generators; correctly rejects empty squares
- **Make/Unmake**: `Position.Make(move)` applies a move incrementally (board, hash, eval, king squares, castling, EP, clocks, side) and pushes undo info; `Position.Unmake(move)` reverses it in O(1) — no full board copy, the performance foundation for AI search
- **Incremental evaluation**: `EvalScore` (white material+PST minus black) maintained in `Make`/`Unmake`; PST tables in `evaluate.go`
- **Zobrist hashing**: `[12][64]uint64` piece keys + side + castling + EP keys (fixed seed); `Hash` maintained incrementally in `Make`/`Unmake` via XOR
- **Transposition table**: 32MB default (2M entries × 16 bytes); `TTEntry{key, score int16, depth, flag, gen, move uint16}`; **gen-aware replacement** (gen+depth priority); `Probe`/`Store`/`Clear`/`FillPercent`; `PackMove`/`UnpackMove`
- **King square cache**: `KingSquares[2]` in Position, updated in `Make`/`Unmake`; `FindKing` is O(1) array read
- **Fixed undo stack**: `undoStack [maxPly]undoInfo` (256 entries) + `undoPly int`; zero heap allocation in search; `Ply()` method for TT mate-score adjustment
- **Legal move filtering**: pseudo-legal moves filtered by Make/Unmake — pins, en-passant discovered checks, and king-moves-into-check handled automatically
- **Check detection**: `IsSquareAttacked` (reverse-scan from a square) + `IsInCheck` (O(1) king lookup + attack scan) + `KingCheck()` (returns checked king's square index or -1)
- **Game status**: `CurrentStatus()` returns `playing | white-wins | black-wins | draw` (checkmate = no legal moves + in check, stalemate = no legal moves + not in check, draw = 50-move / threefold repetition / insufficient material)
- **Draw rules**: `IsRepetition()` (undo stack hash scan, bounded by halfmove clock), `IsFiftyMoveRule()`, `IsInsufficientMaterial()` (zero-alloc: K vs K, K+B vs K, K+N vs K, K+B vs K+B same color), `IsDraw()`
- **Perft validation**: `Perft()` recursive move enumeration using Make/Unmake + stack-allocated MoveList; validated against all 6 standard positions from chessprogramming.org/Perft_Results
- **SAN generation + parsing**: `Position.ToSan(m Move)` generates SAN (disambiguation, castling, promotion, en passant, check/mate suffixes); `Position.SanToMove(san)` matches a SAN string to a legal move; `sanSquare`/`sanToIndex` convert board indices ↔ algebraic; exposed to JS via `san` and `applyPgn` bridges
- **PGN replay**: the `applyPgn` bridge replays a full PGN in one call — parses SAN tokens, applies each via `Make`, and returns a JSON array of history entries (board + SAN + status per ply)

### AI (`pkg/ai`)

Separate package with clean one-directional dependency (`pkg/ai` → `pkg/engine` → `pkg/types`). The engine doesn't know the AI exists.

- **Evaluation**: O(1) read of incremental `EvalScore` (material + piece-square tables maintained by `Make`/`Unmake`), negated for side to move
- **Search**: negamax + alpha-beta pruning + iterative deepening (IDDFS)
- **Transposition table**: Zobrist hash → 16-byte entry; **gen-aware replacement** (gen+depth priority); mate score ply adjustment; TT move for ordering
- **Quiescence search**: stand-pat + captures-only (`PseudoLegalCaptures`) at depth 0 — prevents horizon effect
- **Move ordering**: MVV (captures by `MaterialValue`) + previousBest/TT-move-to-front; insertion sort (optimal for ~20-40 moves)
- **Pseudo-legal moves + lazy `IsInCheck`**: one Make/Unmake per move (not two like LegalMoves would force)
- **Draw checks in search**: `IsRepetition()` + `HalfmoveClock >= 100` at every node (cheap, no board scan); `IsInsufficientMaterial()` only in `CurrentStatus` (too expensive per-node)
- **Terminal detection**: no legal moves → checkmate (`-winScore + depth`, prefer faster mates) or stalemate (0)
- **Time-limited search**: `Search(p, timeLimitMs, stopCh)` — iterative deepening until time budget expires
- **Fixed-depth search**: `SearchFixedDepth(p, depth, stopCh)` — iterative deepening to target depth (ID provides fallback on abort)
- **WithTT variants**: `SearchWithTT` / `SearchFixedDepthWithTT` — caller-provided TT for reuse across searches (e.g. UCI engine's persistent TT)
- **Build-tagged clock**: `clock_wasm.go` (JS `performance.now()`) and `clock_native.go` (Go `time.Now()`) — compiles and tests natively, no WASM needed for development
- **Escape analysis verified**: `MoveList` stays on stack in perft, legal moves, AI search, quiescence; only `LegalMovesSlice` (WASM bridge, cold path) allocates

---

## Project structure

```
go-wasm/
├── cmd/
│   ├── wasm/main.go           # WASM entry: registers goWasmEngine JS functions
│   ├── uci/main.go            # UCI engine entry: standalone CLI engine for cutechess-cli testing (persistent TT, panic recovery, fallback move)
│   └── command/main.go        # CLI debug: loads FEN, runs Perft
├── pkg/
│   ├── types/types.go         # Piece uint8, CastlingRights uint8, Move struct, MoveFlag enum, Piece methods (IsWhite, IsBlack, IsEnemy, Color, TypePiece)
│   ├── engine/                # Chess rules (pure Go, no JS deps)
│   │   ├── position.go        # Position struct (Board, WhiteToMove, CastlingRights, EnPassant*, HalfmoveClock, FullmoveNumber, Hash, KingSquares, EvalScore, undoStack[maxPly], undoPly) + Game global + reset() + Ply()
│   │   ├── evaluate.go        # MaterialValue(), piece-square tables (6×64), pstValue(), pieceTotalValue(), signedPieceValue() — shared with AI for incremental eval
│   │   ├── helpers.go         # abs, inBounds, oppositeColor, colorOfSide(); legacy KingCheck()/Perft()
│   │   ├── fen.go             # LoadFen(): parses all 6 FEN fields + computes initial Hash + EvalScore + StartingFEN
│   │   ├── print.go           # PrintBoard(): ASCII debug
│   │   ├── moves.go           # PseudoLegalMoves(ml *MoveList): iterate board, dispatch per piece type
│   │   ├── move_captures.go   # PseudoLegalCaptures(ml *MoveList): captures + en passant + promotions only — for quiescence search
│   │   ├── movelist.go        # MoveList: [256]Move + count (Add, Len, Get, Clear, Slice) — stack-allocatable
│   │   ├── move_pawn.go       # Pawn moves: forward, double, captures, en passant, promotion
│   │   ├── move_knight.go     # Knight L-jumps with IsEnemy guard
│   │   ├── move_bishop.go     # Bishop diagonal slides with IsEnemy guard
│   │   ├── move_rook.go       # Rook orthogonal slides with IsEnemy guard
│   │   ├── move_king.go       # King one-step + delegates castling to generateCastling()
│   │   ├── castling.go        # castleSides [4]castleSide table + generateCastling() + isPathAttacked() + isEmpty()
│   │   ├── make.go            # Make(move) + Unmake(move): flagged make/unmake with fixed undoStack; incremental Hash, EvalScore, KingSquares updates; legacy MakeMove() bridge
│   │   ├── attacks.go         # FindKing (O(1) via KingSquares cache), IsSquareAttacked (reverse-scan), IsInCheck
│   │   ├── legal.go           # LegalMoves(ml *MoveList): pseudo-legal → Make/Unmake filter; LegalMovesSlice() for WASM
│   │   ├── status.go          # GameStatus enum + CurrentStatus(): checks IsDraw() + checkmate/stalemate
│   │   ├── draw.go            # IsRepetition() (undo stack hash scan), IsTwoFoldRepetition(), IsFiftyMoveRule(), IsInsufficientMaterial() (zero-alloc), IsDraw()
│   │   ├── zobrist.go         # Zobrist hashing: [12][64]uint64 piece keys + side + castling + EP keys (fixed seed), ComputeHash() (full), hashDeltaMove/hashDeltaPiece (incremental)
│   │   ├── tt.go              # TranspositionTable: TTEntry (16 bytes, Gen fits in padding), Probe/Store (gen+depth replacement)/Clear/FillPercent/Size, PackMove/UnpackMove, DefaultTranspositionTable (32MB), TTEntrySize + TestTTEntrySize
│   │   ├── perft.go           # Perft(depth): recursive node count using Make/Unmake + stack MoveList per ply
│   │   ├── san.go             # SAN generation + parsing: ToSan (disambiguation, castling, promotion, en passant, check/mate suffixes), SanToMove (match SAN to legal move), sanSquare/sanToIndex, stripCheckSuffix, disambiguation, appendCheckSuffix
│   │   ├── legal_test.go      # Tests: FEN, castling rights, legal moves, pins, en passant, king-in-check
│   │   ├── fen_test.go        # Tests: en passant target, halfmove clock, fullmove number, Make/Unmake clocks
│   │   ├── status_test.go     # Tests: CurrentStatus, GameStatus.String/IsGameOver, statusFor
│   │   ├── perft_test.go      # Tests: Perft on all 6 chessprogramming.org positions
│   │   ├── zobrist_test.go    # Tests: incremental hash matches full recompute, hash uniqueness, side-to-move, promotion
│   │   ├── san_test.go        # Tests: SAN generation (pawn/knight/bishop/queen/king moves, castling, en passant, promotion, disambiguation, check/mate suffixes), SAN↔Move round-trip, castling notation parsing, invalid SAN rejection
│   │   └── draw_test.go       # Tests: threefold repetition, insufficient material (KvK, KBvK, KNvK, KBvsKB same/diff color), 50-move rule, CurrentStatus draw
│   └── ai/                    # Chess AI (pure Go, no JS deps except build-tagged clock)
│       ├── ai.go              # Evaluate(): O(1) read of incremental EvalScore (negated for side to move)
│       ├── types.go           # SearchResult, searchCtx (with tt *TranspositionTable), constants (winScore=30000, negInf=-32000), shouldStop()
│       ├── moveorder.go       # sideColor, moveOrderScore (previousBest + MVV via MaterialValue), orderMoves (insertion sort), noLegalMoveScore
│       ├── search.go          # negamax (recursive core: TT probe + store, alpha-beta, mate score ply adjustment, repetition/50-move draw checks), scoreToTT/scoreFromTT
│       ├── quiescence.go      # quiescence(): stand-pat + captures-only extension past depth 0 — prevents horizon effect
│       ├── search_api.go      # iterativeDeepening, Search() + SearchWithTT(), SearchFixedDepth() + SearchFixedDepthWithTT() — public entry points
│       ├── clock_wasm.go      # nowMs() via js.performance.now() — build tag: js && wasm
│       ├── clock_native.go    # nowMs() via time.Now().UnixMilli() — build tag: !(js && wasm)
│       └── ai_test.go         # Tests: eval, mate-in-1, win material, search properties, depth scaling, NPS, ID vs direct, benchmarks
├── tools/main.go              # Type generator: Go AST → wasm-contract.ts (optional)
├── bin/gen-types.exe          # Compiled type generator
├── save-engine.ps1            # Builds UCI engine, saves timestamped copy to engines/ (for versioned testing)
├── match-engines.ps1          # Runs cutechess-cli match between two saved engine binaries
├── engines/                    # Saved engine binaries + match PGNs (versioned testing)
└── ROADMAP.md                  # AI improvement roadmap
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

## Zobrist hash encoding

```
Hash uint64 = XOR of:
  zobristPiece[pieceIndex][square]  for each piece on board (12 piece types: wP,wN,wB,wR,wQ,wK,bP,bN,bB,bR,bQ,bK)
  zobristSide                       if black to move
  zobristCastle[CastlingRights]     castling rights bitmask (0-15)
  zobristEP[enPassantFile]           en passant target file (0-7), only if EP target set
```

Keys generated with fixed seed (`0xCAFE`) for reproducibility — same hash across runs. Hash is computed in `LoadFen` (`ComputeHash()`) and maintained incrementally in `Make`/`Unmake` via XOR (XOR is its own inverse, so the same delta applies and reverts).

## Transposition table encoding

```
TTEntry (16 bytes):
  Key   uint64  (8) — full Zobrist hash for collision verification
  Score int16   (2) — adjusted for mate distance (score ± ply)
  Depth uint8   (1) — search depth when stored (used for probe validity)
  Flag  TTFlag  (1) — TTExact | TTLower | TTUpper
  Gen   uint8   (1) — search generation when stored (used for replacement priority)
  Move  uint16  (2) — packed best move (from×1024 + to×4 + promoCode)

TranspositionTable:
  entries []TTEntry  — fixed-size array, indexed by hash & mask
  mask    uint64     — size-1 (power of 2)
  used    int        — slot fill counter
```

`DefaultTranspositionTable()` = 32MB (2M entries). **Gen-aware replacement**: a new entry replaces an existing one if `gen + depth >= old.Gen + old.Depth`. The `gen` counter is a package-level `uint8` in `pkg/ai`, incremented before each `Search()`/`SearchFixedDepth()` call; on wraparound (255 → 0), the TT is cleared. This ensures old deep entries from early moves (low gen) are naturally replaced by recent shallow entries (high gen) — entries from move 1's depth-12 search (priority 13) lose to move 20's depth-6 search (priority 26). `FillPercent()` tracks slot usage. Measured fill at 1s on starting position: ~9.5% (no thrashing).

---

## AI architecture

### Package dependency

```
cmd/wasm/main.go / cmd/uci/main.go
    ↓ imports
pkg/ai              ← Search(), SearchWithTT(), SearchFixedDepth(), SearchFixedDepthWithTT(), Evaluate()
    ↓ imports
pkg/engine           ← Position, MoveList, Make/Unmake, PseudoLegalMoves, PseudoLegalCaptures, IsInCheck, CurrentStatus, TranspositionTable, Zobrist
    ↓ imports
pkg/types            ← Move, Piece, constants
```

### Search algorithm

- **Negamax** with alpha-beta pruning (negation handles perspective switching — simpler than minimax with isMaximizing); returns `(score, bestMove)` — no separate root function
- **Iterative deepening**: depth 1, 2, 3... until time budget expires; previous depth's best move searched first (improves cutoffs); aborted partial results discarded; used by `Search` (time-limited) and `SearchFixedDepth` (fallback on abort)
- **Transposition table**: Zobrist hash → 16-byte entry; **gen-aware replacement** (gen+depth priority); mate score ply adjustment (`scoreToTT`/`scoreFromTT`); TT move used for move ordering
- **Quiescence search**: at depth 0, instead of returning `Evaluate()`, search captures only until quiet (stand-pat + `PseudoLegalCaptures`); prevents horizon effect
- **Pseudo-legal moves + lazy `IsInCheck`**: the AI uses `PseudoLegalMoves` directly, skipping `LegalMoves` — one Make/Unmake per move (not two)
- **MVV move ordering**: captures sorted by `MaterialValue(captured)`; previousBest (ID hint) / TT move forced to index 0; insertion sort (optimal for ~20-40 moves)
- **Draw checks**: `IsRepetition()` (undo stack hash scan) + `HalfmoveClock >= 100` at every node; `IsInsufficientMaterial()` only in `CurrentStatus` (too expensive per-node)
- **Terminal detection**: no legal moves → checkmate (`-winScore + depth`, prefer faster mates) or stalemate (0); `winScore = 30000` (fits in int16 for TT)
- **Time check**: every 2048 nodes via `nowMs()` — build-tagged (`clock_wasm.go` uses `js.performance.now()`, `clock_native.go` uses `time.Now()`); external abort via `stopCh` channel

### Native testing

The `pkg/ai` package compiles and tests natively — no WASM needed. Only `clock_wasm.go` and `clock_native.go` are build-tagged; the core files have no build tag.

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
| `aiMove` | `aiMoveJS` | `ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)` | `number` (time limit ms) | JSON string `{from, to, promotion?}` |
| `aiMoveDepth` | `aiMoveDepthJS` | `ai.SearchFixedDepthWithTT(engine.Game, depth, nil, sharedTT)` | `number` (depth) | JSON string `{from, to, promotion?}` |
| `aiAnalysis` | `aiAnalysisJS` | `ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)` | `number` (time limit ms) | JSON string `{from, to, promotion?, score, depth, nodes, timeMs}` |
| `aiMultiPv` | `aiMultiPvJS` | — | `number, number` (time, numLines) | JSON string (multi-PV lines) |
| `fen` | `fenJS` | — | — | `string` (current FEN) |
| `san` | `sanJS` | `engine.Game.ToSan(move)` | `number, number, number?` (from, to, promotion?) | `string` (SAN) |
| `applyPgn` | `applyPgnJS` | — | `string` (PGN) | JSON string of `PgnHistoryEntry[]` |

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

## UCI engine (`cmd/uci`)

Standalone CLI chess engine implementing the UCI protocol. Used for testing against cutechess-cli.

### Features
- Full UCI protocol: `uci`, `isready`, `ucinewgame`, `position [startpos|fen] [moves ...]`, `go [wtime|btime|winc|binc|movetime|depth|infinite]`, `stop`, `quit`
- **Persistent TT**: 32MB transposition table reused across moves (cleared on `ucinewgame`); uses `SearchWithTT` / `SearchFixedDepthWithTT`; gen-aware replacement (old deep entries decay as the game progresses)
- **Panic recovery**: search goroutine recovers from panics, sends a fallback legal move
- **Fallback move**: if search returns empty (aborted/game over), picks the first legal move
- **Stdout flush**: `os.Stdout.Sync()` after `bestmove` to avoid buffering issues with cutechess
- **Time management**: `wtime/40 + winc*4/5`, capped at half the remaining clock; `movetime` overrides; `depth`/`infinite` = no time limit

### Testing tools

- `save-engine.ps1` — builds the UCI engine, saves a timestamped copy to `engines/` (e.g. `my-stockfish-2026-06-20-1458-baseline.exe`)
- `match-engines.ps1` — runs cutechess-cli between two saved engine binaries with sensible defaults (40 games, `tc=1+0.1`, 8 concurrent, EPD openings, draw adjudication, PGN output); all parameters overridable

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