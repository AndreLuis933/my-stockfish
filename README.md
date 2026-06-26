# Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI runs in TypeScript
- **Xadrez** (Chess) — fully playable, including human-vs-AI; move generation and board state run in Go compiled to WebAssembly, AI runs in Go (separate `pkg/ai` package)

The Go WASM integration is **live**: the worker loads the engine, the React hook calls it for valid moves and move application, pawn promotion is wired end-to-end, and the chess AI searches via `ai.Search()` exposed as `aiMove` / `aiMoveDepth`.

---

## Repository Layout

```
my-stockfish/
├── front/        # React 19 + TypeScript (strict) + Vite (Bun) — see front/README.md
├── go-wasm/      # Go 1.25 chess engine + AI compiled to WASM — see go-wasm/README.md
├── AGENTS.md     # Project state, architecture, and contribution rules
├── CLAUDE.md     # Global coding preferences (symlink)
└── .gitignore
```

Each subproject has its own README with detailed architecture and instructions:

- [`front/README.md`](front/README.md) — React app, Vite plugin, WASM worker bridge, components
- [`go-wasm/README.md`](go-wasm/README.md) — Go chess engine, move generation, AI search, perft validation, WASM build

---

## Stack

| Layer | Tech |
|---|---|
| Frontend | React 19 + TypeScript (strict) + Vite + CSS Modules |
| Routing | react-router-dom v7 |
| Package manager | Bun (frontend) / Go modules (engine) |
| Checkers AI | TypeScript — Minimax + Alpha-Beta + IDDFS (depth 8) |
| Chess engine | Go 1.25 → WebAssembly (board state, move gen, castling, check/checkmate) |
| Chess AI | Go 1.25 → WebAssembly — negamax + alpha-beta + iterative deepening + transposition table + quiescence search (incremental material + piece-square table eval) |

---

## Getting started

### Prerequisites

- [Bun](https://bun.sh) (frontend runtime + package manager)
- [Go 1.25+](https://go.dev/dl/) (WASM build step)
- A modern browser with WebAssembly support

### Install and run

```bash
# Frontend (from front/)
cd front
bun install
bun dev          # dev server — also builds WASM and watches .go files
```

The Vite plugin (`front/plugins/go-wasm.ts`) compiles `engine.wasm` on startup, copies `wasm_exec.js` from GOROOT, watches all `.go` files for changes, rebuilds automatically, and sends an HMR event so the browser restarts the WASM worker without a full page reload.

### Other commands

```bash
# Frontend (from front/)
bun run check     # tsc -b && eslint:strict
bun test          # vitest
bun run build     # tsc -b && vite build (also builds WASM with prod flags)

# Go engine tests (from go-wasm/)
go test ./pkg/engine/ -v          # engine tests (native, no WASM)
go test ./pkg/ai/ -v -short       # AI tests (native, no WASM) — fast mode
go test ./pkg/ai/ -v              # AI tests including depth scaling + NPS measurement
go test ./pkg/ai/ -bench=.        # AI benchmarks

# Go engine perft validation (from go-wasm/)
go run ./cmd/command
```

---

## Current state

### Damas Brasileiras (Checkers) — complete

- Full game in the browser: human-vs-AI, human-vs-human, AI-vs-AI
- AI: Minimax + Alpha-Beta + IDDFS in TypeScript, default depth 8
- Brazilian rules: mandatory max captures, flying kings, multi-jump chains, 40-move draw rule
- Board flip, piece counters, "IA pensando..." indicator, must-move highlighting

### Xadrez (Chess) — fully playable

- Game runs in the browser via Go WASM: **human-vs-human**, **human-vs-AI** (AI plays either color), and **AI-vs-AI** (both sides played by the engine)
- **Standalone UCI engine** (`cmd/uci`): supports cutechess-cli for automated testing; persistent TT (cleared on `ucinewgame`), panic recovery, fallback legal move, `go depth`/`go wtime`/`go movetime`/`go infinite`, `stop`, `quit`, `position startpos|fen moves ...`
- **Chess AI** (`pkg/ai`): negamax + alpha-beta + iterative deepening + **transposition table** + **quiescence search** + **killer moves + history heuristic** + **null-move pruning** + **late move reductions (LMR)** + **aspiration windows** in Go; material + piece-square table evaluation (incremental, O(1)); MVV + killers + history move ordering; time-limited or fixed-depth search; ~7M nodes/sec native
- Go engine handles: board representation, FEN loading (all 6 fields), move generation for all piece types, captures, en passant, pawn promotion, **castling**, **SAN generation + parsing**, **PGN replay** (single round-trip)
- **Position struct**: all game state in `Position` (Board, WhiteToMove, CastlingRights, EnPassant*, HalfmoveClock, FullmoveNumber, Hash, KingSquares, EvalScore, undoStack[maxPly], undoPly); a global `Game *Position` is used by the WASM bridge; the AI uses the same Position via `Make`/`Unmake`
- **Frontend split**: `Chess.tsx` composes sub-components from `components/chess/` (`SanText`, `ModeSelector`, `AiSetupPanel`, `TurnBanner`, `ClockConfigPanel`, `ActionBar`, `AnalysisSummary`, `PgnImportModal`); `Chess.hooks.ts` delegates to `useMultiPv` (continuous multi-PV analysis) and `useChessAnalysis` (single-position analysis + auto-analyze); shared chess sub-component styles live in `ChessShared.module.css`
- **Zobrist hashing**: `[12][64]uint64` piece keys + side + castling + EP keys (fixed seed); `Hash` maintained incrementally in `Make`/`Unmake` via XOR; `ComputeHash()` for initial computation in `LoadFen`
- **Transposition table**: 32MB default (2M entries × 16 bytes); `TTEntry{key, score int16, depth, flag, gen, move uint16}`; **gen-aware replacement** (gen+depth priority: recent shallow entries replace old deep ones); `Probe`/`Store`/`Clear`/`FillPercent`; mate score ply adjustment (`scoreToTT`/`scoreFromTT`); TT move used for move ordering; `Gen` field fits in struct padding (still 16 bytes)
- **Incremental evaluation**: `EvalScore` (white material+PST minus black) maintained in `Make`/`Unmake`; `Evaluate()` is O(1) — single field read + side-to-move negation; PST tables moved to `pkg/engine/evaluate.go` for `Make` to access
- **King square cache**: `KingSquares[2]` in Position, updated in `Make`/`Unmake`; `FindKing` is O(1) array read instead of 64-square scan
- **Fixed undo stack**: `undoStack [maxPly]undoInfo` (256 entries, ~12KB) + `undoPly int`; zero heap allocation in search; `Ply()` method exposes ply for TT mate-score adjustment
- **Quiescence search**: `quiescence()` — stand-pat + captures-only extension past depth 0; prevents horizon effect (AI no longer thinks a hanging queen is safe); uses `PseudoLegalCaptures()` (capture-only move gen); repetition + 50-move draw checks
- **Draw rules**: `CurrentStatus()` checks `IsDraw()` (50-move rule, threefold repetition, insufficient material); `negamax` checks `IsRepetition()` + `HalfmoveClock >= 100` at every node; `IsInsufficientMaterial()` (zero-alloc: K vs K, K+B vs K, K+N vs K, K+B vs K+B same color) checked in `CurrentStatus` only (too expensive per-node in search)
- **MoveList**: fixed `[256]Move` inline array + count, passed as `*MoveList` — stack-allocated, zero heap allocation in hot paths (perft, legal moves, AI search, quiescence)
- **Make/Unmake**: O(1) incremental make/unmake with undo stack — the performance foundation for AI search
- **Castling**: data-driven via `castleSides [4]castleSide` table; all 6 FIDE conditions checked; rook moves with the king; castling rights cleared on king/rook moves and rook captures
- **Piece.IsEnemy()**: unified enemy detection in all move generators — correctly rejects empty squares, replaces duplicated color-branch logic
- **Check detection & game status**: checkmate, stalemate, and draw detection (50-move, threefold repetition, insufficient material); king square glows red when in check; "Xeque!" badge; result overlay ("Brancas vencem!" / "Pretas vencem!" / "Empate!")
- **Perft validation**: all 6 standard positions from chessprogramming.org pass (initial position through depth 5 = 4,865,609 nodes; Kiwipete; Positions 3-6)
- Pawn promotion: picker modal (Q/N/R/B) wired end-to-end
- **AI setup panel**: user chooses their color (board auto-flips), search mode (difficulty / custom time / custom depth), and difficulty level (Fácil/Médio/Difícil)
- **"IA pensando..." indicator**: badge in turn banner while AI searches
- **Move history sidebar**: SAN-like notation (e4, Nf3, exd5, O-O, e8=Q+); click any move to jump to that position; navigation buttons (|<, <, >, >|); per-move eval tags
- **Chess clock**: dual countdown with configurable initial time (1/3/5/10/15 min or none) and increment (0/2/3/5/10s); flag fall → loss
- **No auto-restart**: result overlay stays until the user clicks "Jogar novamente" (no cancel button)
- **Move animations**: pieces slide to destination; captured pieces fade-out; castling animates king + rook simultaneously
- **Coordinate labels**: file letters (a-h) and rank numbers (1-8) on board edges
- **Keyboard navigation**: ArrowLeft/Right navigate history, Home/End jump to start/end (works after game over too)
- **Position analysis**: "Analisar" button shows AI evaluation (pawns), best move with arrow on board, and search depth
- **Auto-analyze**: "Analisar auto" toggle runs analysis after each move and stores eval tags in the move history
- Board flip, turn banner, result overlay

### What is missing

- **Opening book**: no opening repertoire — AI plays from first principles every game
- **Bitboards**: board is `[64]Piece` mailbox — bitboards + magic bitboards would give 10-50× raw speed; full engine rewrite
- **Parallel search**: no goroutine-based parallel search yet — needs thread-safe TT
- **Type generator auto-run**: the Vite plugin does not run the type generator; `wasm-contract.ts` is hand-maintained
- **PV-SAN in TS**: `chessNotation.ts` still has `toSan`, `canPieceReach`, `isPathClear` (used by `pvToSan.ts` for multi-PV line notation); will move to Go

See `AGENTS.md` for the full architecture, call flow, encoding details, and contribution rules.

---

## License

Personal project — no license specified.