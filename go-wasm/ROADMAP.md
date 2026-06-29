# AI Improvement Roadmap

## Current state (after Phase 1 + Phase 2 + Phase 3)
- **~5.5M nodes/sec** search (native), up from ~2.1M before bitboards — raw perft
  hits 25M (Make/Unmake) / 32M (bulk-count) nodes/sec
- **Depth 13 at 1s** on starting position (was depth 12 before bitboards, depth 7
  at baseline); depth 15 at 5s
- Depth 14 at ~1.3s on starting, depth 11-12 at ~1.3s on middlegame/tactical
- TT + quiescence + killers + history + null-move pruning + LMR + aspiration,
  now on a hybrid bitboard engine with magic bitboards for sliders

## Phase 1 — Search quality (biggest elo gain) ✅ DONE
1. ✅ **Transposition table** — Zobrist hash → 32MB TT, gen-aware replacement (gen+depth priority), mate-score ply adjustment
2. ✅ **Quiescence search** — stand-pat + captures-only past depth 0
3. ✅ **Killer moves + history heuristic** — two [256]Move slots per ply +
   [4096]int history table with depth² bonus and aging

Expected result: depth 8-9 at 1s, dramatically better tactical play. ✅ Exceeded
(depth 12 at 1s).

## Phase 2 — Search speed (node reduction) ✅ DONE
4. ✅ **Null move pruning** — R=2, guarded by hasNonPawnMaterial + !inCheck +
   ply > 0 + depth > 3, inline hash/side flip (no Make call)
5. ✅ **Late move reductions (LMR)** — first 3 moves full depth, rest reduced
   by 1 ply, re-search at full depth if reduced search beats alpha
6. ✅ **Aspiration windows** — from depth 3, search [score-50, score+50],
   re-search full window on fail-low/fail-high

Expected result: depth 9-10 at 1s, 200-400 elo gain. ✅ Exceeded (depth 12).

## Phase 3 — Engine rewrite ✅ DONE
7. ✅ **Bitboards + magic bitboards** — hybrid representation: 12 `uint64` piece
   bitboards + 4 occupancy bitboards maintained alongside the `[64]Piece` mailbox.
   Move generation uses magic lookups for sliders (bishop/rook/queen), precomputed
   tables for knights/kings, and shift-based gen for pawns. `Make`/`Unmake` update
   bitboards incrementally with inline XOR deltas. The mailbox is kept for O(1)
   "what piece is on square X" queries (SAN, captures, eval).
8. ✅ **Move struct shrink** — `Move.From`/`Move.To` changed from `int` (8 bytes)
   to `uint8` (1 byte): struct dropped from 24 → 5 bytes, MoveList from 6KB → 1.3KB.
9. ✅ **Move ordering rewrite** — score each move once into a reusable scratch
   buffer (was O(n²) rescoring inside the insertion sort), and the root now passes
   its iterative-deepening `previousBest` to ordering (it was being ignored).
   `hasNonPawnMaterial` is now an O(1) bitboard test instead of a 64-square scan.

Result: depth 13 at 1s (raw perft 25-33 Mnps, search ~5.5 Mnps — ~2× the
pre-bitboard search throughput). Raw NPS is lower than the "30-50M" estimate
because this is a *hybrid* (mailbox kept), not a pure-bitboard rewrite — the
trade keeps SAN/eval/capture lookups simple.

### Next up (Phase 3 follow-on)
- **Lazy move selection** — instead of fully sorting the move list up front, pick
  the best *remaining* move on demand inside the negamax loop. Alpha-beta usually
  cuts off after 2-3 moves, so the other ~30 moves never need ordering. This
  attacks the current #1 hot spot (`orderMoves`, ~30% of search time) and the #2
  (`TranspositionTable.Probe`, ~18%, memory-latency bound).

## Phase 4 — Parallelism
10. **Parallel search (goroutines)** — Young Brothers Wait Concept: search PV
    move sequentially, rest in parallel sharing TT. ~1.8× with 4 cores, ~2.5×
    with 8. Needs thread-safe TT.

Expected result: depth 12-13 at 1s with 4 cores (depth 15-17 with pruning).

## Other improvements (not ranked by phase)
- **Opening book** — hash of FEN → move. Instant move for first 5-10 moves.
- **Pondering** — think on opponent's time. 2× effective time. Needs TT +
  goroutines.
- **Test coverage** — 9 new Phase 2 tests (deep-ply panic regression, null-move
  depth comparison, zugzwang endgame, LMR legality, aspiration window match,
  killer recording, history aging, board corruption, in-check null-move guard).