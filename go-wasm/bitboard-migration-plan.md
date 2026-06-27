# Bitboard Migration Plan

Migrate the chess engine from mailbox `[64]Piece` to **hybrid bitboards** (12 piece bitboards + derived occupancy, alongside the existing mailbox for O(1) piece-identity queries). Target: 10-50x speedup in move generation and attack detection via **magic bitboards** for sliders.

## Decisions

- **Hybrid**: keep `Board [64]Piece` mailbox alongside 12 bitboards. Mailbox = "what piece is here?" (SAN, captures, eval, WASM output). Bitboards = "where are all the X?" and "which squares does Y attack?" (move gen, check detection). Standard approach (Stockfish keeps `pieceList[64]`).
- **Magic bitboards**: one multiply + shift + table lookup for sliding-piece attacks. ~1MB attack table. 10x speedup vs classical ray-trim (which is only ~2x over mailbox loops).
- **Runtime init**: magic numbers and attack tables generated in `init()` at startup (~50-100ms one-time). No offline codegen, no committed generated files. Binary stays small.

## Rules

- After every step: `go test ./pkg/engine/ ./pkg/ai/ -short` must pass. Perft is the ground truth — if perft node counts drift, the bitboards are out of sync with the mailbox.
- Commit after every green perft. Each commit is a bisect point.
- The mailbox is never removed. SAN, FEN, PV, WASM output, and capture-piece identity stay mailbox-backed.
- `cmd/playground/` stays as a scratch reference (already proven correct for knight/king/pawn/classical-ray). Not imported by the engine.

## Phases

### Phase 0 — Infrastructure (no behavior change)

#### Step 0.1: bitboard.go + bitboard_test.go
- [ ] New `pkg/engine/bitboard.go`:
  - `type Bitboard uint64`
  - File/rank masks: `fileA`, `fileH`, `notA`, `notH`, `rankMask(r int) Bitboard`
  - Helpers: `setBit`, `clearBit`, `isSet`, `popcount`, `bitscan`
  - `knightAttacks [64]Bitboard`, `kingAttacks [64]Bitboard` — built in `init()` (port from `cmd/playground/main.go`, already proven)
  - `genStepper(sq int, offsets [][2]int) Bitboard` (used by init)
- [ ] New `pkg/engine/bitboard_test.go`:
  - Knight on g1 (sq 6) → attacks {e2, f3, h3} (sqs 12, 21, 23)
  - Knight on a1 (sq 0) → attacks {b3, c2} (sqs 17, 10) — no wrap-around
  - Knight on h8 (sq 63) → attacks {f7, g6} (sqs 53, 46) — no wrap-around
  - King on e1 (sq 4) → attacks {d1, d2, e2, f1, f2} (sqs 3, 11, 12, 5, 13)
  - King on a1 (sq 0) → attacks {a2, b1, b2} (sqs 8, 1, 9) — corner, no wrap
  - `popcount(0xFFFFFFFFFFFFFFFF) == 64`
  - `bitscan(0b...00101000) == 3`
- [ ] `go test ./pkg/engine/ -run Bitboard -v` passes
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` still green (no behavior change)
- [ ] **Commit**: "Add bitboard type and precomputed knight/king attack tables"

#### Step 0.2: magic.go + magic_test.go
- [ ] New `pkg/engine/magic.go`:
  - `rookMask [64]Bitboard` — relevant-occupancy squares for rook on each sq (ray squares excluding board edges)
  - `bishopMask [64]Bitboard` — same for bishops
  - `rookMagic [64]uint64`, `bishopMagic [64]uint64` — magic multipliers found by search
  - `rookShift [64]uint8`, `bishopShift [64]uint8` — shift amounts (`64 - popcount(mask)`)
  - `rookAttacksTable []Bitboard`, `bishopAttacksTable []Bitboard` — attack sets, indexed by `((occ & mask) * magic) >> shift`
  - `init()`:
    1. For each square, compute `mask` (ray squares excluding edges — the edges can never be blockers since they're the terminal squares of each ray)
    2. Enumerate all `2^bits` relevant-occupancy combinations via carries-rippling counter over the mask bits
    3. For each combination, compute the true attack set using classical ray-trim (port `rayPositive`/`rayNegative` from `cmd/playground/main.go`)
    4. Search for a magic multiplier: try random `uint64` values, test that `((occ & mask) * magic) >> shift` produces no index collisions for any occupancy in the enumeration
    5. Store the attack set at each index; record the magic and shift
  - Public functions:
    - `func rookAttacks(sq int, occ Bitboard) Bitboard`
    - `func bishopAttacks(sq int, occ Bitboard) Bitboard`
  - Helper: `func genRayMask(sq int, dr, df int) Bitboard` (ray squares excluding edges — same as `genRay` in playground but stops before the terminal edge square)
- [ ] New `pkg/engine/magic_test.go`:
  - For every square (0..63), for ~100 random occupancies each:
    - `rookAttacks(sq, occ)` matches a reference classical ray-trim implementation
    - `bishopAttacks(sq, occ)` matches the reference
  - Edge squares (a1, h1, a8, h8, d4, e5) tested with extra occupancies including blockers on ray edges
  - Empty board: rook on d4 sees full cross, bishop on d4 sees full diagonals
  - Fully blocked: rook on a1 with all 4 adjacent squares occupied sees only those 4 squares
- [ ] `go test ./pkg/engine/ -run Magic -v` passes
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` still green
- [ ] **Commit**: "Add magic bitboards for rook and bishop attacks"

### Phase 1 — Add bitboards to Position (redundant state, no behavior change)

#### Step 1.1: Extend Position struct + updateBitboards
- [ ] `pkg/engine/position.go`:
  - Add 12 `Bitboard` fields to `Position`: `WhitePawns, WhiteKnights, WhiteBishops, WhiteRooks, WhiteQueens, WhiteKing, BlackPawns, BlackKnights, BlackBishops, BlackRooks, BlackQueens, BlackKing`
  - Add derived: `WhitePieces, BlackPieces, Occupied, Empty Bitboard`
  - Keep `Board types.Board` (the mailbox — hybrid)
  - New method `func (p *Position) updateBitboards()` — scans `Board`, sets the 12 piece bitboards, then derives `WhitePieces`/`BlackPieces`/`Occupied`/`Empty` via OR
  - New method `func (p *Position) pieceBitboard(pt types.Piece, color types.Piece) *Bitboard` — returns pointer to the right bitboard field (used by Make/Unmake to avoid a 12-way switch)
  - `reset()` clears all 16 bitboards to 0
- [ ] `pkg/engine/fen.go`:
  - `LoadFen` calls `p.updateBitboards()` at the end (after mailbox is populated)
- [ ] `pkg/engine/position_test.go` (new or extend):
  - After `LoadFen(StartingFEN)`: `popcount(p.WhitePawns) == 8`, `popcount(p.BlackPawns) == 8`, `popcount(p.Occupied) == 32`, `popcount(p.Empty) == 32`
  - After `LoadFen` of a tactical position (e.g., "r3k2r/p1ppqpb1/bn2Qnp1/2qPN3/1p2P3/2N5/PPPBBPPP/R3K2R b KQkq - 0 1"): verify per-piece counts
  - Consistency check: for every sq, `isSet(p.pieceBitboardFor(p.Board[sq]), sq)` matches `p.Board[sq] != 0`
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green (bitboards not used yet)
- [ ] **Commit**: "Add bitboard fields to Position, sync from mailbox in LoadFen"

#### Step 1.2: Maintain bitboards in Make/Unmake
- [ ] `pkg/engine/make.go`:
  - In `Position.Make`, after each mailbox write, XOR the corresponding bitboard:
    - Quiet move: `pieceBB ^= (1<<from | 1<<to)` for the moving piece's bitboard
    - Capture: same XOR for moving piece, plus `capturedBB &^= (1<<captureSquare)` for the captured piece's bitboard
    - Castling: XOR king bitboard (from→to) AND XOR rook bitboard (rookFrom→rookTo)
    - En passant: XOR moving pawn, clear captured pawn's bitboard at `EnPassantCapture`
    - Promotion: clear pawn bit at `from`, set promoted-piece bit at `to`
  - After per-move bitboard updates: re-derive `WhitePieces`/`BlackPieces`/`Occupied`/`Empty` (6 ORs + 1 NOT — cheap, avoids per-move delta tracking bugs)
  - In `Position.Unmake`: same XORs in reverse (XOR is self-inverse — `pawns ^= (1<<from | 1<<to)` works both ways). For captures, re-set the captured piece's bitboard: `capturedBB |= (1<<captureSquare)`. For promotions, clear promoted bit, set pawn bit. Re-derive occupancies.
  - Use `pieceBitboard()` helper to avoid 12-way switch — or inline the switch if the helper is too slow (benchmark decides)
- [ ] `pkg/engine/make_test.go` (extend or new):
  - For 1000 random positions + random moves: after Make, `p.Board[sq]` and `p.bitboardFor(sq)` agree on all 64 squares. After Unmake, they agree again and match the pre-Make state.
  - Specifically test: castling (both sides, both colors), en passant (both colors), all 4 promotion types, captures of each piece type
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green — **critical gate**: perft must still pass, proving bitboards stay in sync
- [ ] **Commit**: "Maintain bitboards in Make/Unmake, perft still green"

### Phase 2 — Use bitboards for hot paths (one at a time)

#### Step 2.1: Replace IsSquareAttacked with bitboard attackersTo
- [ ] `pkg/engine/attacks.go`:
  - New method `func (p *Position) attackersTo(sq int, byColor types.Piece) Bitboard`:
    - Pawn attackers: if byColor is white, `(whitePawns >> 7 & notH) | (whitePawns >> 9 & notA)` shifted to land on `sq` — equivalently `((Bitboard(1)<<sq) << 7 & notH | (Bitboard(1)<<sq) << 9 & notA) & whitePawns` (attackers of sq by white pawns = white pawns on the squares that would attack sq)
    - Knight attackers: `knightAttacks[sq] & enemyKnights`
    - King attackers: `kingAttacks[sq] & enemyKing`
    - Bishop/queen attackers: `bishopAttacks(sq, p.Occupied) & (enemyBishops | enemyQueens)`
    - Rook/queen attackers: `rookAttacks(sq, p.Occupied) & (enemyRooks | enemyQueens)`
    - OR all five together
  - `IsInCheck` becomes `p.attackersTo(kingSq, enemyColor) != 0`
  - `IsSquareAttacked` delegates to `p.attackersTo(sq, byColor) != 0`
  - Keep `KingCheck()` free function as a wrapper (used by WASM bridge)
- [ ] `pkg/engine/attacks_test.go` (extend):
  - Various positions: `attackersTo(sq, white) != 0` matches old `IsSquareAttacked(sq, white)` for all squares
  - Check detection: positions with check by pawn, knight, bishop, rook, queen, king, discovered checks
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green
- [ ] **Commit**: "Replace IsSquareAttacked with bitboard attack map" — **first speedup in check detection**

#### Step 2.2: Migrate knight move generation
- [ ] `pkg/engine/move_knight.go`:
  - Replace the 8-offset loop with: `targets := knightAttacks[sq] & ^ownPieces`
  - Bitscan `targets` into `Move` structs; fill `Captured` from `p.Board[to]` (mailbox — one read per capture, only for actual moves not 64 scans)
  - Use `types.FlagNormal` (no special knight flags)
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green (perft validates exact move counts)
- [ ] **Commit**: "Migrate knight move generation to bitboards"

#### Step 2.3: Migrate king move generation
- [ ] `pkg/engine/move_king.go`:
  - Replace the 8-offset loop with: `targets := kingAttacks[sq] & ^ownPieces`
  - Bitscan into `Move` structs; `Captured` from mailbox
  - Castling generation stays in `castling.go` unchanged (data-driven, works fine, low frequency)
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green
- [ ] **Commit**: "Migrate king move generation to bitboards"

#### Step 2.4: Migrate pawn move generation
- [ ] `pkg/engine/move_pawn.go` + pawn section of `pkg/engine/move_captures.go`:
  - White pushes: `(whitePawns << 8) & empty`
  - White double pushes: `single := (whitePawns << 8) & empty; double := (single << 8) & empty & rankMask(3)` — only pawns originally on rank 2 reach rank 4 via two empties (the `& rankMask(3)` filters the result to rank 4, which only rank-2 pawns can reach)
  - White captures: `((whitePawns & notA) << 7) & blackPieces` (NW), `((whitePawns & notH) << 9) & blackPieces` (NE)
  - White en passant: same capture shifts, AND with `Bitboard(1) << p.EnPassantTarget` (one specific square)
  - Promotions: when target square is on rank 8 (sq 56..63), emit 4 `Move` structs with `Promotion: Queen|Rook|Bishop|Knight` and `Flag: FlagPromotion`
  - Black: mirror with `>>` instead of `<<`, rank 1 promotions, rank 4 double-push filter
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green — **en passant and promotions are the trickiest; perft catches errors**
- [ ] **Commit**: "Migrate pawn move generation to bitboards"

#### Step 2.5: Migrate sliders (magic bitboards) — the big speedup
- [ ] `pkg/engine/move_bishop.go`:
  - `targets := bishopAttacks(sq, p.Occupied) & ^ownPieces`
  - Bitscan into `Move` structs; `Captured` from mailbox
- [ ] `pkg/engine/move_rook.go`:
  - `targets := rookAttacks(sq, p.Occupied) & ^ownPieces`
- [ ] Queen: no new file — queen moves = `bishopAttacks(sq, occ) | rookAttacks(sq, occ)`, handled in `moves.go` dispatcher (Step 2.6) or inline in a new `move_queen.go` if cleaner
- [ ] `pkg/engine/move_captures.go` slider sections:
  - Same generators but AND with `enemyPieces` instead of `^ownPieces` (captures only — for quiescence)
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green
- [ ] **Commit**: "Migrate slider move generation to magic bitboards" — **the 10x moment**

#### Step 2.6: Migrate PseudoLegalMoves dispatcher
- [ ] `pkg/engine/moves.go`:
  - Replace `for i, piece := range p.Board` (64-square scan) with per-piece-type bitscan loops:
    - `for p.WhitePawns != 0 { sq := bitscan(p.WhitePawns); p.WhitePawns &= p.WhitePawns - 1; ... }`
    - Same for each of the 12 piece bitboards (or 6 per color, dispatching by piece type within each)
  - Alternative cleaner layout: one loop per piece type per color, calling the migrated generators from Steps 2.2-2.5
- [ ] `go test ./pkg/engine/ ./pkg/ai/ -short` green
- [ ] **Commit**: "Migrate PseudoLegalMoves dispatcher to bitscan loops"

### Phase 3 — Cleanup (optional, low priority)

#### Step 3.1: Simplify legacy free functions
- [ ] `pkg/engine/helpers.go`: `KingCheck()` can delegate to `attackersTo` (already done in Step 2.1, but review for leftover code)
- [ ] `pkg/engine/attacks.go`: remove the old reverse-scan code if fully unused

#### Step 3.2: evaluate.go PST iteration
- [ ] `pkg/engine/evaluate.go`: PST initialization in `LoadFen` can iterate set bits of each piece bitboard instead of 64-square scan. Small startup speedup, no runtime impact (eval is incremental).

#### Step 3.3: draw.go insufficient material
- [ ] `pkg/engine/draw.go`: `IsInsufficientMaterial` can check `popcount(whiteBishops|whiteKnights)`, etc., instead of scanning the mailbox. Trivial.

#### Step 3.4 (DO NOT DO): SAN / FEN / PV / WASM output
- These keep reading the mailbox. They are cold paths (called once per move, not per node). Migrating them adds complexity for no measurable speedup. Leave them alone.

## What does NOT change

- `pkg/types/types.go` — `Piece`, `Move`, `MoveFlag`, `Board` type all stay. The mailbox and bitboards coexist.
- `pkg/engine/san.go` — SAN generation reads `p.Board[sq]` (mailbox). Unchanged.
- `pkg/engine/fen.go` parsing — reads the FEN string, fills mailbox. Only change: calls `updateBitboards()` at the end. FEN output reads mailbox. Unchanged.
- `pkg/engine/zobrist.go` — hash deltas computed from `piece` (read from mailbox in Make). Unchanged.
- `pkg/engine/tt.go` — transposition table. Unchanged.
- `pkg/engine/castling.go` — castling rights logic. Unchanged (castling generation stays data-driven).
- `pkg/engine/status.go`, `pkg/engine/draw.go` — game status and draw rules. Unchanged (draw.go IsInsufficientMaterial optional cleanup in 3.3).
- `pkg/engine/perft.go` — perft test harness. Unchanged (it's the validator, not the validated).
- `pkg/engine/legal.go` — legal move filter (Make/Unmake + IsInCheck). Unchanged — `IsInCheck` now uses bitboards internally (Step 2.1) but the filter logic stays.
- `pkg/ai/*` — AI search. Reads `p.Board[sq]` for move ordering and PV display (cold paths). Unchanged.
- `cmd/wasm/main.go` — WASM bridge. Reads `p.Board` for output. Unchanged.
- `cmd/uci/*` — UCI engine. Unchanged.
- `front/*` — frontend. Unchanged (WASM contract is the same).

## Expected outcome

- Phase 0: ~100ms slower startup (magic table generation), no runtime change.
- Phase 1: no runtime change, perft validates bitboards stay in sync.
- Phase 2.1-2.4: incremental speedups, hard to measure individually.
- Phase 2.5: **the 10x moment** — slider move gen drops from ~8-iteration loops to one multiply+shift+lookup. Perft NPS should jump significantly.
- Phase 2.6: removes the last 64-square scan; move gen iterates only over actual pieces (~16 per side). Another ~2x on top.
- Net: depth at 1s should go from 12 to ~15-16 on the starting position. Nodes/sec from ~2.1M to ~15-30M.

## Reference

- `cmd/playground/main.go` — proven reference for knight/king tables, pawn shifts, classical ray-trim. Port logic from there.
- https://www.chessprogramming.org/Magic_Bitboards — theory and magic-search algorithm.
- Stockfish `bitboards.cpp` / `magic.cpp` — reference implementation (fancy magic).