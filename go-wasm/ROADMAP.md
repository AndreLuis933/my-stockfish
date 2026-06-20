# AI Improvement Roadmap

## Current state (baseline)
- ~6.5M nodes/sec
- Depth 6 at 500ms, depth 7 at 1s
- No TT, no quiescence, no killers, no bitboards

## Phase 1 — Search quality (biggest elo gain)
1. **Transposition table** — Zobrist hash → {depth, score, move}. 2-4× node reduction. Prerequisite for aspiration windows, parallel search, pondering.
2. **Quiescence search** — extend search past depth 0 with captures only. Fixes horizon effect (AI thinks it's up a queen because capture is past cutoff). Makes eval accurate → alpha-beta prunes correctly.
3. **Killer moves + history heuristic** — two [maxPly]Move arrays. If a quiet move caused a cutoff at ply N, try it first at ply N in siblings. 20-30% node reduction. Trivial to add.

Expected result: depth 8-9 at 1s, dramatically better tactical play.

## Phase 2 — Search speed (node reduction)
4. **Null move pruning** — "let opponent move twice, if they still can't beat my score, prune." 20-40% node reduction. Risk: zugzwang (skip if ≤2 pieces or endgame).
5. **Late move reductions (LMR)** — after 3-4 full-depth moves, reduce rest by 1 ply. If reduced search improves alpha, re-search full. 20-40% in midgame. Needs killers + history first.
6. **Aspiration windows** — search depth N+1 with [score-50, score+50] instead of [-inf, +inf]. 90% success → massive pruning. If fail, re-search full (uses TT).

Expected result: depth 9-10 at 1s, 200-400 elo gain.

## Phase 3 — Engine rewrite
7. **Bitboards + magic bitboards** — replace [64]Piece mailbox with 12 uint64 bitmasks. Move gen + attack detection become bitwise ops, 10-50× faster. Full rewrite of pkg/engine. Raw NPS: 30-50M.

Expected result: depth 11-12 at 1s.

## Phase 4 — Parallelism
8. **Parallel search (goroutines)** — Young Brothers Wait Concept: search PV move sequentially, rest in parallel sharing TT. ~1.8× with 4 cores, ~2.5× with 8. Needs thread-safe TT.

Expected result: depth 12-13 at 1s with 4 cores.

## Other improvements (not ranked by phase)
- **Opening book** — hash of FEN → move. Instant move for first 5-10 moves. Trivial, low cost.
- **Pondering** — think on opponent's time. 2× effective time. Needs TT + goroutines.