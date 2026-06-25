# AI Improvement Roadmap

## Current state (after Phase 1 + Phase 2)
- ~2.1M nodes/sec (lower than baseline due to per-node pruning overhead, but
  node count drops ~2× and depth jumps 5 plies — net win is huge)
- Depth 12 at 1s on starting position (was depth 7 at baseline)
- Depth 10 at 1s on middlegame, depth 10 at 0.5s on tactical
- TT + quiescence + killers + history + null-move pruning + LMR + aspiration

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

## Phase 3 — Engine rewrite
7. **Bitboards + magic bitboards** — replace [64]Piece mailbox with 12 uint64
   bitmasks. Move gen + attack detection become bitwise ops, 10-50× faster.
   Full rewrite of pkg/engine. Raw NPS: 30-50M.

Expected result: depth 11-12 at 1s (with current pruning, depth 14-16 likely).

## Phase 4 — Parallelism
8. **Parallel search (goroutines)** — Young Brothers Wait Concept: search PV
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