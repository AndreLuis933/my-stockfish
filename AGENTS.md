# Project State — Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI runs in TypeScript
- **Xadrez** (Chess) — fully playable, including human-vs-AI; move generation and board state run in Go compiled to WebAssembly, AI runs in Go (separate `pkg/ai` package). A standalone UCI engine (`cmd/uci`) allows testing against cutechess-cli.

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
│       │   ├── generated/wasm-contract.ts   # Hand-maintained TS contract for the Go functions (edit directly) — incl. AiAnalysisResult, PgnHistoryEntry
│       │   ├── loader.ts                    # WasmWorkerEngine class (Web Worker bridge, typed async calls) — incl. san, applyPgn wrappers
│       │   └── useWasm.ts                   # React hook: { engine, loading, error, restarting } + HMR restart
│       ├── pages/
│       │   ├── checkers/          # Route /checkers
│       │   │   ├── Checkers.tsx
│       │   │   └── Checkers.module.css
│   │   └── chess/             # Route /chess
│   │       ├── Chess.tsx              # Composes sub-components from components/chess/ (~300 lines; was 652)
│   │       ├── Chess.hooks.ts         # useChess: state machine bridging React ↔ Go WASM, AI turn effect, difficulty/time/depth modes, move history + navigation, clock; delegates to useMultiPv + useChessAnalysis (~580 lines; was 866)
│   │       └── Chess.module.css       # Layout-only styles (43 lines; was 549 — component styles moved to ChessShared.module.css)
│       ├── components/
│       │   ├── Board/             # Checkers board UI (Board.tsx + Board.module.css)
│       │   ├── ChessBoard/        # Chess board UI (cburnett SVG pieces, selection + move hints + check glow)
│       │   ├── MoveHistory/       # Move history sidebar (SAN via <SanText>, ply navigation, clocks display, result box)
│       │   ├── PromotionPicker/   # Pawn promotion modal: Q/N/R/B picker using piece SVGs
│       │   ├── Nav/               # Top nav bar (links to /checkers and /chess)
│       │   ├── BottomBar/         # Bottom action bar (BottomBar.tsx + .module.css)
│       │   ├── AnalysisPanel/     # Position analysis panel (uses <SanText>, no blunder highlighting)
│       │   └── chess/            # Chess page sub-components (composed by Chess.tsx)
│       │       ├── SanText.tsx            # Shared SAN+figurine renderer (replaces duplicated renderSanWithFigurine/renderMoveText)
│       │       ├── ModeSelector.tsx        # Game mode buttons (Humano vs IA, etc.)
│       │       ├── AiSetupPanel.tsx       # AI color/search-mode/difficulty/time/depth config
│       │       ├── TurnBanner.tsx         # Turn indicator with thinking/check/history badges
│       │       ├── ClockConfigPanel.tsx   # Clock presets + increment
│       │       ├── ActionBar.tsx          # Reiniciar/Girar/Copiar/Colar/Analisar/Auto buttons
│       │       ├── AnalysisSummary.tsx     # Inline eval/best-move/depth panel
│       │       ├── BookMovesPanel.tsx     # Opening book moves panel (SAN + weight bars, click-to-play)
│       │       ├── PgnImportModal.tsx     # PGN paste modal
│       │       └── ChessShared.module.css # Shared styles for all chess sub-components (391 lines)
│       ├── hooks/
│       │   ├── useGame.ts         # Checkers state machine (uses TS AI)
│       │   ├── useChessClock.ts   # Chess clock hook: dual countdown, increments, flag-fall detection
│       │   ├── useMultiPv.ts      # Multi-PV continuous analysis: multiPv, runMultiPv, stopMultiPv, multiPvSan/depth/thinking
│       │   └── useChessAnalysis.ts # Analysis: analyze, autoAnalyze, setAutoAnalyze, analyzeCurrentPosition, analysisForPly
│       ├── utils/
│       │   ├── gameEngine.ts      # Checkers: move gen, captures, flying kings, applyMove, turn state
│       │   ├── aiEngine.ts        # Checkers AI: Minimax + Alpha-Beta + IDDFS, depth 8
│       │   ├── chessAssets.ts     # pieceImageUrl(piece) → cburnett SVG path (shared by ChessBoard + PromotionPicker)
│       │   ├── chessNotation.ts   # Chess SAN helpers (toSan, canPieceReach, isPathClear) — used by pvToSan.ts; parsePgn/promoByte/colorBits moved to Go/types
│       │   ├── chessAnalysis.ts   # formatEval (move-quality classification removed — dead code)
│       │   └── chessFigurine.ts   # Figurine glyph helpers (figurineColorForMove removed — identity, unused)
│       ├── types/
│       │   ├── game.ts            # Checkers: Color, PieceType, Piece, Cell, Board, Move
│       │   └── chess.ts           # Chess: ChessColor, ChessPieceType, ChessPiece, ChessBoard, ChessMove, HistoryEntry, getPiece, decodePieceByte, emptyBoard, colorBits, pieceByte
│       └── assets/                # Static images (hero, react/vite svg)
├── go-wasm/                        # Go source compiled to WASM (module: webassemble, go 1.25)
│   ├── cmd/
│   │   ├── wasm/main.go           # WASM entry: registers goWasmEngine.{validMovesChess, initBoard, makeMove, isCheckJS, gameStatus, aiMove, aiMoveDepth, aiAnalysis, aiMultiPv, loadBook, fen, san, applyPgn}
│   │   ├── uci/main.go            # UCI engine entry: standalone CLI engine for cutechess-cli testing (persistent TT, panic recovery, fallback move, -book flag for opening book)
│   │   └── command/main.go        # CLI debug entry: loads FEN, runs Perft depths 1-5
│   ├── pkg/
│   │   ├── types/types.go         # Piece uint8 (type bits + color bits), CastlingRights uint8 (KQkq), Move struct (with Flag + Captured), MoveFlag enum, Piece methods (IsWhite, IsBlack, IsEnemy, Color, TypePiece)
│   │   ├── book/                  # Polyglot opening book (pure Go, no JS deps)
│   │   │   ├── polyglot_keys.go   # 781 fixed polyglot random constants (generated from python-chess)
│   │   │   ├── polyglot_hash.go   # PolyglotHash(p *Position): polyglot Zobrist hash (independent of engine's own Zobrist) + canCaptureEnPassant check
│   │   │   ├── polyglot_move.go   # DecodePolyglotMove(uint16) Move + MatchLegalMove(p, bookMove) — resolves flags via LegalMoves
│   │   │   ├── book.go            # Book type: Load([]byte)/LoadFile(path), Probe(hash) []Entry, PickMove(hash, rng) (weighted random), PickBestMove(hash)
│   │   │   └── book_test.go       # Tests: hash correctness (verified vs python-chess), probe, weighted pick, move decode, real book load
│   │   ├── engine/                # Chess rules (pure Go, no JS deps)
│   │   │   ├── position.go        # Position struct (Board mailbox + 12 piece bitboards + 4 occupancy bitboards + pieceBBTable lookup, WhiteToMove, CastlingRights, EnPassant*, HalfmoveClock, FullmoveNumber, Hash, KingSquares, EvalScore, undoStack[maxPly], undoPly) + Game global + reset() + Ply() + updateBitboards()
│   │   │   ├── bitboard.go        # Bitboard type (uint64, LERF a1=0..h8=63), file/rank masks, bitscan (math/bits), precomputed knightAttacks/kingAttacks tables, shift helpers
│   │   │   ├── magic.go           # Magic bitboards for sliders: bishopAttacksBB/rookAttacksBB (magic multiply + shift + table lookup), per-square attack table init
│   │   │   ├── magic_data.go      # Precomputed magic numbers + relevant-occupancy masks per square
│   │   │   ├── evaluate.go        # MaterialValue(), piece-square tables (6×64), pstValue(), pieceTotalValue(), signedPieceValue() — shared with AI for incremental eval
│   │   │   ├── helpers.go         # Pure helpers: abs, inBounds, oppositeColor, colorOfSide(); legacy free fns KingCheck()/Perft() delegating to Game
│   │   │   ├── fen.go             # LoadFen(): parses all 6 FEN fields + computes initial Hash + EvalScore + rebuilds bitboards + squareToIndex() + StartingFEN constant
│   │   │   ├── print.go           # PrintBoard(): ASCII debug print
│   │   │   ├── moves.go           # Position.PseudoLegalMoves(ml *MoveList): iterate piece bitboards, dispatch per piece type, writes into caller-owned MoveList
│   │   │   ├── move_captures.go   # PseudoLegalCaptures(ml *MoveList): captures + en passant + promotions only (bitboard & enemyPieces) — for quiescence search
│   │   │   ├── movelist.go        # MoveList struct: [256]Move inline array + count, methods (Add, Len, Get, Clear, Slice) — stack-allocatable, zero-alloc in hot paths
│   │   │   ├── move_pawn.go       # Pawn moves via bitboard shifts: forward, double, captures, en passant, promotion
│   │   │   ├── move_knight.go     # Knight L-jumps via precomputed knightAttacks table
│   │   │   ├── move_bishop.go     # Bishop diagonal slides via magic lookup (bishopAttacksBB)
│   │   │   ├── move_rook.go       # Rook orthogonal slides via magic lookup (rookAttacksBB)
│   │   │   ├── move_king.go       # King one-step via kingAttacks table + delegates castling to generateCastling()
│   │   │   ├── castling.go        # castleSide struct + castleSides [4]castleSide table + generateCastling(): data-driven, all 6 FIDE conditions as sequential guards with comments
│   │   │   ├── make.go            # Position.Make(move) + Position.Unmake(move): flagged make/unmake with fixed undoStack [maxPly]undoInfo (zero heap alloc); incremental mailbox + bitboards (inline XOR deltas) + Hash + EvalScore + KingSquares; legacy MakeMove() bridge (sets Captured for capture-promotions)
│   │   │   ├── attacks.go         # Position.FindKing (O(1) via KingSquares cache), Position.attackersTo (parallel bitboard ops), Position.IsSquareAttacked, Position.IsInCheck
│   │   │   ├── legal.go           # Position.LegalMoves(ml *MoveList): pseudo-legal → Make/Unmake filter; LegalMovesSlice() convenience wrapper for WASM bridge
│   │   │   ├── status.go          # GameStatus enum + Position.CurrentStatus(): checks IsDraw() (50-move, repetition, insufficient material) + checkmate/stalemate
│   │   │   ├── draw.go            # IsRepetition() (undo stack hash scan), IsTwoFoldRepetition(), IsFiftyMoveRule(), IsInsufficientMaterial() (zero-alloc: K vs K, K+B vs K, K+N vs K, K+B vs K+B same color), IsDraw()
│   │   │   ├── zobrist.go         # Zobrist hashing: [12][64]uint64 piece keys + side + castling + EP keys (fixed seed), ComputeHash() (full), hashDeltaMove/hashDeltaPiece (incremental)
│   │   │   ├── tt.go              # TranspositionTable: TTEntry (16 bytes: key, score int16, depth, flag, gen, move uint16), Probe/Store/Clear/FillPercent/Size, PackMove/UnpackMove, DefaultTranspositionTable (32MB), gen-aware replacement (gen+depth priority), TTEntrySize constant + TestTTEntrySize
│   │   │   ├── perft.go           # Position.Perft(depth): recursive node count using Make/Unmake + stack-allocated MoveList per ply
│   │   │   ├── san.go             # SAN generation + parsing: ToSan (disambiguation, castling, promotion, en passant, check/mate suffixes), SanToMove (match SAN to legal move), sanSquare/sanToIndex, stripCheckSuffix, disambiguation, appendCheckSuffix
│   │   │   ├── legal_test.go      # Tests: FEN loading, castling rights, legal move counts, pins, en-passant discovered check, king-in-check
│   │   │   ├── fen_test.go        # Tests: en passant target parsing, halfmove clock, fullmove number, squareToIndex, Make/Unmake clock management
│   │   │   ├── status_test.go     # Tests: CurrentStatus, GameStatus.String/IsGameOver, statusFor
│   │   │   ├── perft_test.go      # Tests: Perft on all 6 chessprogramming.org standard positions
│   │   │   ├── zobrist_test.go    # Tests: incremental hash matches full recompute, hash uniqueness, side-to-move, promotion
│   │   │   ├── tt_test.go         # Tests: TTEntry struct size stays 16 bytes (Gen field fits in padding)
│   │   │   ├── san_test.go        # Tests: SAN generation (pawn/knight/bishop/queen/king moves, castling, en passant, promotion, disambiguation, check/mate suffixes), SAN↔Move round-trip, castling notation parsing, invalid SAN rejection
│   │   │   └── draw_test.go       # Tests: threefold repetition, insufficient material (KvK, KBvK, KNvK, KBvsKB same/diff color), 50-move rule, CurrentStatus draw
│   │   └── ai/                    # Chess AI (pure Go, no JS deps except build-tagged clock)
│   │       ├── ai.go              # Evaluate(p *Position): O(1) read of incremental EvalScore (negated for side to move) — material + PST maintained by Make/Unmake
│   │       ├── types.go           # SearchResult, searchCtx (with tt *TranspositionTable, gen uint8, killers killerTable, history historyTable, orderScratch [256]int reusable score buffer, disableNullMove bool), constants (winScore=30000, negInf=-32000, nodeCheckMask, maxDepth=64, maxPly=512), genCounter (package-level uint8), shouldStop() method
│   │       ├── moveorder.go       # sideColor, killerTable [maxPly][2]Move, historyTable [4096]int, moveOrderScore (previousBest + MVV + killers + history), orderMoves (score-once into scratch buffer + insertion sort), noLegalMoveScore
│   │       ├── search.go          # negamax (recursive core: TT probe + store with gen, alpha-beta, null-move pruning, LMR, killer/history recording on cutoffs, mate score ply adjustment, repetition/50-move draw checks), scoreToTT/scoreFromTT, hasNonPawnMaterial (O(1) bitboard test)
│   │       ├── quiescence.go      # quiescence(): stand-pat + captures-only extension past depth 0 — prevents horizon effect
│   │       ├── search_api.go      # Search() + SearchWithTT(), SearchFixedDepth() + SearchFixedDepthWithTT(), nextGen() (gen counter increment + wraparound TT clear), iterativeDeepening (aspiration windows from depth 3, history aging) — public entry points
│   │       ├── clock_wasm.go      # nowMs() via js.performance.now() — build tag: js && wasm
│   │       ├── clock_native.go    # nowMs() via time.Now().UnixMilli() — build tag: !(js && wasm)
│   │       ├── ai_test.go         # Tests: evaluation, mate-in-1, mate-in-1-black, win hanging piece, search properties, depth scaling, NPS measurement, ID vs direct, benchmarks
│   │       └── tt_fill_test.go   # Tests: TT fill % measurement (positions × time limits, per-depth), deep-entry preservation with gen-aware replacement
│   ├── tools/main.go              # Type generator: Go AST + type-checker → wasm-contract.ts (optional, starting point only)
│   ├── bin/gen-types.exe          # Compiled type generator binary
│   ├── save-engine.ps1            # Builds UCI engine, saves timestamped copy to engines/ (for versioned testing)
│   ├── match-engines.ps1          # Runs cutechess-cli match between two saved engine binaries
│   ├── engines/                    # Saved engine binaries + match PGNs (versioned testing)
│   └── ROADMAP.md                  # AI improvement roadmap (TT, quiescence, killers, LMR, bitboards, parallel)
├── scripts/
│   └── build-book.py               # Polyglot opening book builder: parse PGN → filter by Elo → Stockfish quality filter → write .bin (4 subcommands: parse/filter/analyze/write)
├── stockfish.exe                   # Stockfish 17 binary (used by build-book.py for quality filtering; gitignored)
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
- **Standalone UCI engine** (`cmd/uci`): supports cutechess-cli for automated testing; persistent TT (cleared on `ucinewgame`), panic recovery, fallback legal move, `go depth`/`go wtime`/`go movetime`/`go infinite`, `stop`, `quit`, `position startpos|fen moves ...`
- **Chess AI** (`pkg/ai`): negamax + alpha-beta + iterative deepening + **transposition table** + **quiescence search** + **killer moves + history heuristic** + **null-move pruning** + **late move reductions (LMR)** + **aspiration windows** in Go; material + piece-square table evaluation (incremental); MVV + killers + history move ordering (each move scored once into a reusable buffer); time-limited or fixed-depth search; **~5.5M nodes/sec native, depth 13 at 1s** on the starting position (depth 15 at 5s) — ~2× the pre-bitboard throughput
- Go engine handles: board representation, FEN loading (all 6 fields), move generation for all piece types, captures, en passant, pawn promotion, **castling**, **SAN generation + parsing**, **PGN replay**
- **Hybrid bitboards**: 12 `uint64` piece bitboards + 4 occupancy bitboards (`WhitePieces`, `BlackPieces`, `Occupied`, `Empty`) maintained in `Make`/`Unmake` alongside the `[64]Piece` mailbox; **magic bitboards** for sliders (`bishopAttacksBB`/`rookAttacksBB` = magic multiply + shift + table lookup in `magic.go`/`magic_data.go`), precomputed `knightAttacks`/`kingAttacks` tables, shift-based pawn gen; move generation and `attackersTo` are bitboard ops; the mailbox stays for O(1) piece-on-square lookups (SAN, captures, eval)
- **Position struct**: all game state in `Position` (Board mailbox + piece/occupancy bitboards + pieceBBTable, WhiteToMove, CastlingRights, EnPassant*, HalfmoveClock, FullmoveNumber, Hash, KingSquares, EvalScore, undoStack[maxPly], undoPly); a global `Game *Position` is used by the WASM bridge; the AI uses the same Position via `Make`/`Unmake`
- **Zobrist hashing**: `[12][64]uint64` piece keys + side + castling + EP keys (fixed seed); `Hash` maintained incrementally in `Make`/`Unmake` via XOR; `ComputeHash()` for initial computation in `LoadFen`
- **Transposition table**: 32MB default (2M entries × 16 bytes); `TTEntry{key, score int16, depth, flag, gen, move uint16}`; **gen-aware replacement** (gen+depth priority: recent shallow entries replace old deep ones, so entries from early moves decay as the game progresses); `Probe`/`Store`/`Clear`/`FillPercent`; mate score ply adjustment (`scoreToTT`/`scoreFromTT`); TT move used for move ordering; `Gen` field fits in struct padding (still 16 bytes)
- **Incremental evaluation**: `EvalScore` (white material+PST minus black) maintained in `Make`/`Unmake`; `Evaluate()` is O(1) — single field read + side-to-move negation; PST tables moved to `pkg/engine/evaluate.go` for `Make` to access
- **King square cache**: `KingSquares[2]` in Position, updated in `Make`/`Unmake`; `FindKing` is O(1) array read instead of 64-square scan
- **Fixed undo stack**: `undoStack [maxPly]undoInfo` (256 entries, ~12KB) + `undoPly int`; zero heap allocation in search; `Ply()` method exposes ply for TT mate-score adjustment
- **Quiescence search**: `quiescence()` — stand-pat + captures-only extension past depth 0; prevents horizon effect (AI no longer thinks a hanging queen is safe); uses `PseudoLegalCaptures()` (capture-only move gen); repetition + 50-move draw checks
- **Check extension**: when the side to move is in check at depth 0, the search extends one ply with full move generation instead of dropping into quiescence — this is how the engine detects checkmate at the depth horizon (quiescence can't detect mate since it only searches captures); without this, mate-in-1 was scored as a normal eval and the engine played a slower mate instead
- **Draw rules**: `CurrentStatus()` checks checkmate/stalemate first (no legal moves), then `IsDraw()` (50-move rule, threefold repetition, insufficient material) — a checkmated position is never a draw even if the 50-move clock is high; `negamax` checks `IsRepetition()` + `HalfmoveClock >= 100` **after** the move loop (not before) so mate/stalemate is detected first; draws are claimable (`max(0, best)`) — if mate is available the side plays on and wins; `IsInsufficientMaterial()` (zero-alloc: K vs K, K+B vs K, K+N vs K, K+B vs K+B same color) checked in `CurrentStatus` only (too expensive per-node in search)
- **Mate scoring**: `noLegalMoveScore(inCheck, ply)` uses **ply** (distance from root), not depth — a mate at ply N is scored `-winScore + N`, so closer mates score higher and are preferred; using depth was wrong because the check extension modifies depth, making mate-in-1 and mate-in-3 both score 29999 (the original pgn.txt bug)
- **MoveList**: fixed `[256]Move` inline array + count, passed as `*MoveList` to move generators — stack-allocated, zero heap allocation in hot paths (perft, legal moves, AI search, quiescence); with `Move.From`/`Move.To` as `uint8` the struct is 5 bytes (was 24) and the list 1.3KB (was 6KB)
- **Flagged moves**: `Move` carries a `MoveFlag` (Normal | DoublePush | EnPassant | CastleK | CastleQ | Promotion) and a `Captured` piece — internal-only fields (`json:"-"`) so the frontend contract stays `{from, to, promotion?}`
- **Make/Unmake**: `Position.Make(move)` applies a move incrementally (mailbox + piece/occupancy bitboards via inline XOR deltas, hash, eval, king squares, castling, EP, clocks, side) and pushes undo info; `Position.Unmake(move)` reverses it in O(1) — no full board copy, the performance foundation for AI search
- **Castling**: data-driven via `castleSides [4]castleSide` table in `castling.go`; all 6 FIDE conditions checked as sequential guards with comments; rook moves with the king on `Make`; castling rights cleared on king/rook moves and rook captures
- **Piece.IsEnemy()**: unified enemy detection (`other&ColorMask != ColorNone && p&ColorMask != other&ColorMask`) — replaces duplicated color-branch logic in all move generators; correctly rejects empty squares
- **Capture-only move generation**: `PseudoLegalCaptures()` generates only captures, en passant, and promotions — used by quiescence search to extend past the depth horizon with only "noisy" moves
- **Legal move filtering**: pseudo-legal moves are filtered by Make/Unmake — pins, en-passant discovered checks, and king-moves-into-check are all handled automatically
- **Check detection**: `Position.IsSquareAttacked` (reverse-scan from a square) + `Position.IsInCheck` (O(1) king lookup + attack scan) + `KingCheck()` exposed to the frontend as `isCheckJS`
- **Game status**: `Position.CurrentStatus()` returns `playing | white-wins | black-wins | draw`, exposed as `gameStatus`
- **Perft validation**: `Position.Perft()` runs recursive move enumeration using Make/Unmake + stack-allocated MoveList; validated against all 6 standard positions from chessprogramming.org/Perft_Results
- **SAN + PGN in Go**: `Position.ToSan(m)` generates SAN (disambiguation, castling, promotion, en passant, check/mate suffixes); `Position.SanToMove(san)` matches a SAN string to a legal move; the WASM bridge exposes `san` (generate SAN for a move) and `applyPgn` (replay a full PGN in one round-trip, returning a history array). The frontend no longer does SAN generation or PGN parsing in TypeScript — `applyMove` calls `engine.san()`, and `loadPgn` calls `engine.applyPgn()`.
- **Frontend split**: `Chess.tsx` (was 652 lines) composes sub-components from `components/chess/` (`SanText`, `ModeSelector`, `AiSetupPanel`, `TurnBanner`, `ClockConfigPanel`, `ActionBar`, `AnalysisSummary`, `PgnImportModal`); `Chess.hooks.ts` (was 866 lines) delegates to `useMultiPv` and `useChessAnalysis`; `Chess.module.css` holds only layout styles (component styles in `ChessShared.module.css`).
- Pawn promotion: `PromotionPicker` modal lets the user choose Q/N/R/B; the AI returns promotion bytes automatically
- Move validation: clicking a piece queries `engine.validMovesChess()`, highlights legal targets
- **Check highlight**: king's square glows red + "Xeque!" badge in turn banner
- **Result overlay**: "Brancas vencem!" / "Pretas vencem!" / "Empate!"
- **AI setup panel**: user chooses their color (board auto-flips), search mode (difficulty / custom time / custom depth), and difficulty level (Fácil/Médio/Difícil)
- **"IA pensando..." indicator**: badge in turn banner while AI searches
- **Persistent WASM TT**: module-level `sharedTT` (32MB) reused across AI moves and analysis; cleared on `initBoard` (new game); entries accumulate across moves, improving hit rates for recurring positions (mirrors the UCI engine's session-level TT)
- **Move history sidebar**: SAN-like notation (e4, Nf3, exd5, O-O, e8=Q+, O-O-O#) with move-pair rows; click any move to jump to that position; navigation buttons (|<, <, >, >|) for start/prev/next/end; auto-scrolls to current ply; per-move evaluation tags shown when analysis is available
- **Position navigation**: viewing past positions does not trigger the AI or allow board interaction; making a new move is only possible from the latest position; a "revisitando" badge appears in the turn banner when viewing history
- **Chess clock**: dual countdown (white/black) with configurable initial time (1/3/5/10/15 min or no clock) and increment (0/2/3/5/10s); clock starts on the first move; increment added after each move; flag fall → loss; clock config disabled during an active game
- **No auto-restart**: on game over the result overlay shows and stays until the user clicks "Jogar novamente"; the overlay has no close/cancel button — the user is forced to restart
- **Move animations**: pieces slide to their destination on move (300ms cubic-bezier); captured pieces fade-out with a scale pulse; castling animates both king and rook sliding simultaneously; last-move squares are highlighted
- **Coordinate labels**: file letters (a-h) and rank numbers (1-8) shown on the board edges, color-matched to the square (light on dark, dark on light)
- **Keyboard navigation**: ArrowLeft/ArrowRight navigate history (prev/next ply), Home/End jump to start/end; disabled when the promotion picker is open (works even after game over)
- **Position analysis**: "Analisar" button runs the AI search on the current position and shows the evaluation score (in pawns), best move (with an arrow drawn on the board), and search depth; closeable panel
- **Auto-analyze**: "Analisar auto" toggle automatically runs a 500ms analysis after each move and stores the evaluation in the move history; per-move eval tags appear next to each move in the sidebar
- **Opening book**: Polyglot `.bin` opening book integrated — both UCI engine (`-book` flag) and WASM browser (`loadBook` + `worker.js` fetch) probe the book before search; weighted random selection adds variety across games; book built from lichess standard games (Elo ≥ 2000, first 10 moves, Stockfish quality-filtered at ±100cp, blended 50/50 popularity+quality weights normalized to 100 per position); **"Livro" toggle button** in the action bar shows all book moves for the current position with SAN + weight bars (click a move to play it)
- Board flip, turn banner, result overlay

**AI architecture**
- **Separate `pkg/ai` package**: imports `pkg/engine` + `pkg/types`, clean one-directional dependency; engine doesn't know AI exists
- **Unified `negamax`**: single recursive function returns `(score, bestMove)` — no separate `searchRoot`; root passes `previousBest` for ordering (it is actually used — forced to index 0, preferred over the TT move), internal nodes pass `nil`
- **Negamax** (not minimax with isMaximizing): negation handles perspective switching — simpler code
- **Pseudo-legal moves + lazy `IsInCheck`**: one Make/Unmake per move (not two like LegalMoves would force); the AI uses `PseudoLegalMoves` directly, skipping `LegalMoves`
- **MVV + killers + history move ordering**: previousBest (ID hint) / TT move forced to index 0; captures sorted by `MaterialValue(captured)`; killers (two `[maxPly][2]Move` slots per ply, quiet cutoff moves) at score 900; history (`[4096]int` indexed by `from*64+to`, depth² bonus + aging) for remaining quiet moves. Each move is scored **once** into a reusable `searchCtx.orderScratch [256]int` buffer, then insertion-sorted (optimal for ~20-40 moves). Scoring once — not re-scoring inside the O(n²) comparison loop — and reusing the buffer (vs a per-node 2KB stack array zeroed every call) together roughly doubled search NPS
- **Quiescence search**: stand-pat + captures-only (`PseudoLegalCaptures`) at depth 0 — prevents horizon effect
- **Transposition table**: Zobrist hash → 16-byte entry; **gen-aware replacement** (gen+depth priority); mate score ply adjustment; TT move for ordering
- **Null-move pruning**: "pass" once, search opponent at `depth-1-R` (R=2); if still ≥ beta, prune the subtree; guarded by `!inCheck && ply > 0 && depth > 3 && hasNonPawnMaterial && !ctx.disableNullMove`; inline hash/side flip (no Make call — restores manually); `ZobristSideKey` / `ZobristEPKeys` exported from engine for the flip
- **Late move reductions (LMR)**: after 3 full-depth moves, remaining quiet non-killer non-promotion moves searched at `depth-2`; if reduced search beats alpha, re-search at full depth; skips captures, promotions, killers (those need full accuracy)
- **Aspiration windows**: from depth 3, search with `[score-50, score+50]` instead of `[-inf, +inf]`; on fail-low (score ≤ alpha) widen downward, on fail-high (score ≥ beta) widen upward, re-search (TT makes it cheap); history aged between iterations
- **Iterative deepening**: depth 1, 2, 3... until time budget expires; previous depth's best move searched first (improves cutoffs); aborted partial results discarded; used by `Search` (time-limited) and `SearchFixedDepth` (fallback on abort)
- **Build-tagged clock**: `clock_wasm.go` (JS `performance.now()`) and `clock_native.go` (Go `time.Now()`) — `pkg/ai` compiles and tests natively with `go test ./pkg/ai/`, no WASM needed
- **Escape analysis verified**: `MoveList` stays on stack in perft, legal moves, AI search, quiescence; only `LegalMovesSlice` (WASM bridge, cold path) allocates

**WASM infrastructure**
- `worker.js` exists and works: loads `wasm_exec.js` + `engine.wasm`, instantiates `goWasmEngine`, dispatches `{ id, fn, args }` messages, replies `{ id, result, error }`
- `WasmWorkerEngine` (`loader.ts`): typed async wrappers over the Web Worker, pending-promise map, restart support
- `useWasm` hook: loads engine, exposes `{ engine, loading, error, restarting }`, listens for `wasm-rebuild` HMR events and restarts the worker without a full page reload
- Vite plugin (`plugins/go-wasm.ts`): compiles `engine.wasm` on dev start and prod build, copies `wasm_exec.js` from GOROOT, watches all `.go` files and rebuilds on change, sends `wasm-rebuild` HMR event

### What is missing / in progress

| Piece | Notes |
|---|---|
| Opening book | Polyglot `.bin` opening book integrated — `pkg/book/` reads position→move entries with weighted random selection; both UCI (`-book` flag) and WASM (`loadBook` + `worker.js` fetch) probe the book before search. Book built from lichess standard games (Elo ≥ 2000, first 10 moves, Stockfish quality-filtered, blended popularity+quality weights). |
| Lazy move selection | `orderMoves` fully sorts the move list up front (~30% of search time, the #1 hot spot). Picking the best *remaining* move on demand inside the negamax loop would skip ordering moves never searched after a beta cutoff — the next planned search optimization. |
| Parallel search | No goroutine-based parallel search yet — needs thread-safe TT (Phase 4) |
| Type generator auto-run | The Vite plugin does **not** run the type generator — it only builds the WASM and sends HMR. `wasm-contract.ts` is maintained by hand. |
| Optional arg in contract | The generator emits all params as required. `makeMove`'s 3rd arg (`promotion`) was manually marked optional in `wasm-contract.ts`. |
| Checkers → Go | Checkers logic stays in TypeScript for now; no plan to port it to Go. |
| PV-SAN in TS | `chessNotation.ts` still has `toSan`, `canPieceReach`, `isPathClear` (used by `pvToSan.ts` for multi-PV line notation). These will be removed when PV-SAN moves to Go. |

---

## WASM Integration Architecture

### Call flow

```
React component
  → useWasm() hook  →  WasmWorkerEngine.makeMove(from, to, promotion)
    → loader.ts: postMessage({ id: 0, fn: "makeMove", args: [from, to, promotion] })
      → worker.js: self.goWasmEngine.makeMove(from, to, promotion)
        → cmd/wasm/main.go: makeMoveJS → engine.MakeMove(from, to, promotion)
      → worker.js: postMessage({ id: 0, result: [64 board bytes...] })
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
| `aiMove` | `aiMoveJS` | `ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)` — probes `sharedBook` first via `book.PolyglotHash` + `PickMove` | `number` (time limit ms) | JSON string `{from, to, promotion?}` |
| `aiMoveDepth` | `aiMoveDepthJS` | `ai.SearchFixedDepthWithTT(engine.Game, depth, nil, sharedTT)` — probes `sharedBook` first | `number` (depth) | JSON string `{from, to, promotion?}` |
| `aiAnalysis` | `aiAnalysisJS` | `ai.SearchWithTT(engine.Game, timeLimitMs, nil, sharedTT)` — probes `sharedBook` first (returns score=0, depth=0 on book hit) | `number` (time limit ms) | JSON string `{from, to, promotion?, score, depth, nodes, timeMs}` |
| `aiMultiPv` | `aiMultiPvJS` | — | `number, number` (time, numLines) | JSON string (multi-PV lines) |
| `loadBook` | `loadBookJS` | `book.Load([]byte)` — loads a polyglot `.bin` from a Uint8Array | `Uint8Array` | `boolean` |
| `bookMoves` | `bookMovesJS` | `sharedBook.Probe(PolyglotHash)` → decode + SAN for each entry | — | JSON string of `BookMoveEntry[]` (`{from, to, promotion?, weight, san}`) |
| `fen` | `fenJS` | — | — | `string` (current FEN) |
| `san` | `sanJS` | `engine.Game.ToSan(move)` | `number, number, number?` (from, to, promotion?) | `string` (SAN) |
| `applyPgn` | `applyPgnJS` | — | `string` (PGN) | JSON string of `PgnHistoryEntry[]` |

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

### Zobrist hash encoding

```
Hash uint64 = XOR of:
  zobristPiece[pieceIndex][square]  for each piece on board (12 piece types: wP,wN,wB,wR,wQ,wK,bP,bN,bB,bR,bQ,bK)
  zobristSide                       if black to move
  zobristCastle[CastlingRights]     castling rights bitmask (0-15)
  zobristEP[enPassantFile]           en passant target file (0-7), only if EP target set
```

Keys generated with fixed seed (`0xCAFE`) for reproducibility — same hash across runs. Hash is computed in `LoadFen` (`ComputeHash()`) and maintained incrementally in `Make`/`Unmake` via XOR (XOR is its own inverse, so the same delta applies and reverts).

### Transposition table encoding

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

`DefaultTranspositionTable()` = 32MB (2M entries). **Gen-aware replacement**: a new entry replaces an existing one if `gen + depth >= old.Gen + old.Depth`. The `gen` counter is a package-level `uint8` in `pkg/ai`, incremented before each `Search()`/`SearchFixedDepth()` call; on wraparound (255 → 0), the TT is cleared to keep comparisons monotonic. This ensures old deep entries from early moves (low gen) are naturally replaced by recent shallow entries (high gen) — entries from move 1's depth-12 search (priority 13) lose to move 20's depth-6 search (priority 26). `FillPercent()` tracks slot usage. Measured fill at 1s on starting position: ~9.5% (no thrashing).

### AI search architecture

```
cmd/wasm/main.go / cmd/uci/main.go
    ↓ imports
pkg/ai              ← Search(), SearchWithTT(), SearchFixedDepth(), SearchFixedDepthWithTT(), Evaluate()
│   ├── search.go       negamax (recursive core: TT probe/store, alpha-beta, null-move pruning, LMR, killer/history recording, draw checks)
│   ├── quiescence.go   quiescence (stand-pat + captures-only)
│   ├── search_api.go   iterativeDeepening (aspiration windows from depth 3, history aging), public Search/SearchFixedDepth + WithTT variants
│   ├── moveorder.go    previousBest/TT-move-to-front + MVV + killers + history ordering (score-once into scratch buffer), killerTable + historyTable types
│   ├── types.go        SearchResult, searchCtx (with tt, killers, history, orderScratch buffer, disableNullMove flag), shouldStop()
│   └── ai.go           Evaluate() — O(1) incremental read
    ↓ imports
pkg/engine           ← Position, MoveList (uint8 From/To), Make/Unmake (bitboards), PseudoLegalMoves, PseudoLegalCaptures, magic attacks, attackersTo, IsInCheck, CurrentStatus, TranspositionTable, Zobrist
    ↓ imports
pkg/types            ← Move, Piece, constants
```

The AI uses `PseudoLegalMoves` + lazy `IsInCheck` (one Make/Unmake per move, not two). At depth 0, it calls `quiescence()` which uses `PseudoLegalCaptures` (captures only). Move ordering forces previousBest/TT-move to index 0, then MVV (captures by `MaterialValue`) + killers + history, scoring each move once into a reusable buffer. The clock is build-tagged: `clock_wasm.go` uses `js.performance.now()`, `clock_native.go` uses `time.Now()`. The `pkg/ai` package compiles and tests natively with `go test ./pkg/ai/` — no WASM needed for development.

### Opening book architecture

```
cmd/wasm/main.go / cmd/uci/main.go
    ↓ imports
pkg/book             ← Load(), LoadFile(), PolyglotHash(), PickMove(), DecodePolyglotMove(), MatchLegalMove()
│   ├── polyglot_keys.go    781 fixed random constants (from python-chess POLYGLOT_RANDOM_ARRAY)
│   ├── polyglot_hash.go    PolyglotHash(p *Position) — independent Zobrist hash (piece keys: (type-1)*2+color, color 0=black 1=white; castling: 768-771; ep: 772-779 only if capture available; turn: 780 if white to move)
│   ├── polyglot_move.go    DecodePolyglotMove(uint16) Move + MatchLegalMove(p, bookMove) — resolves flags via LegalMoves
│   └── book.go             Book type: Load([]byte)/LoadFile(path), Probe(hash) []Entry (binary search), PickMove(hash, rng) (weighted random), PickBestMove(hash)
    ↓ imports
pkg/engine           ← Position, LegalMoves (for MatchLegalMove)
    ↓ imports
pkg/types            ← Move, Piece, CastlingRights
```

Both the UCI engine and the WASM bridge probe the opening book **before** invoking the AI search. On a book hit, the engine returns the book move instantly (UCI: `info string book move` + `bestmove`; WASM: same JSON shape as a search result). On a miss, the normal search runs. The polyglot hash is independent of the engine's own Zobrist hash (different random constants) — computed on demand from the Position's board/castling/ep/turn state. Book moves are matched against `LegalMoves` to resolve `Flag`/`Captured`/`Promotion` correctly (the polyglot encoding doesn't carry flag information).

**Book building** (`scripts/build-book.py`): a 4-phase Python pipeline that parses a PGN database, filters by Elo, applies a popularity threshold + top-5-per-position limit, runs Stockfish quality filtering (absolute ±100cp gate + quality_score relative to best), and writes a sorted `.bin` with blended 50/50 popularity+quality weights normalized to sum=100 per position. Supports `--max-time` (stop and write partial book), `--resume` (checkpoint recovery), and progress bars (tqdm).

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

## UCI engine (`cmd/uci`)

Standalone CLI chess engine implementing the UCI protocol. Used for testing against cutechess-cli.

### Features
- Full UCI protocol: `uci`, `isready`, `ucinewgame`, `position [startpos|fen] [moves ...]`, `go [wtime|btime|winc|binc|movetime|depth|infinite]`, `stop`, `quit`
- **Persistent TT**: 32MB transposition table reused across moves (cleared on `ucinewgame`); uses `SearchWithTT` / `SearchFixedDepthWithTT`; gen-aware replacement (old deep entries decay as the game progresses)
- **Panic recovery**: search goroutine recovers from panics, sends a fallback legal move
- **Fallback move**: if search returns empty (aborted/game over), picks the first legal move
- **Stdout flush**: `os.Stdout.Sync()` after `bestmove` to avoid buffering issues with cutechess
- **Time management**: `wtime/40 + winc*4/5`, capped at half the remaining clock; `movetime` overrides; `depth`/`infinite` = no time limit
- **Opening book**: `-book <path>` CLI flag loads a polyglot `.bin` book; probed before search with weighted random selection (`info string book move` + instant `bestmove` on hit)

### Testing tools

- `save-engine.ps1` — builds the UCI engine, saves a timestamped copy to `engines/` (e.g. `my-stockfish-2026-06-20-1458-baseline.exe`)
- `match-engines.ps1` — runs cutechess-cli between two saved engine binaries with sensible defaults (40 games, `tc=1+0.1`, 8 concurrent, EPD openings, draw adjudication, PGN output); all parameters overridable

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

# UCI engine build (from go-wasm/)
go build -o bin/my-stockfish.exe ./cmd/uci

# UCI engine with opening book (from go-wasm/)
go build -o bin/my-stockfish.exe ./cmd/uci
./bin/my-stockfish.exe -book books/book.bin

# cutechess-cli testing (from go-wasm/)
.\save-engine.ps1                          # save current engine to engines/
.\save-engine.ps1 -Tag experiment           # save with a tag
.\match-engines.ps1 old.exe new.exe         # match two saved engines
.\match-engines.ps1 old.exe new.exe -rounds 100 -concurrency 16

# Opening book builder (from project root)
# Phase 1: parse PGN → raw entries (byte-based progress bar, ~60 min for 2.7GB)
python scripts/build-book.py parse games.pgn -o raw.json --min-elo 2000
# Phase 2: popularity filter + top-5 per position (instant)
python scripts/build-book.py filter raw.json -o popular.json --min-move-games 5 --max-moves-per-position 5
# Phase 3: Stockfish quality filter + blended weights (progress bar, --max-time safety, --resume)
python scripts/build-book.py analyze popular.json -o filtered.json --depth 12 --eval-range 100 --max-time 3600
# Phase 4: write sorted .bin (instant)
python scripts/build-book.py write filtered.json -o go-wasm/books/book.bin

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