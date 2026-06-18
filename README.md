# Damas Brasileiras + Xadrez

Two board games in one React app:

- **Damas Brasileiras** (Brazilian Checkers) — fully playable, AI runs in TypeScript
- **Xadrez** (Chess) — playable, move generation and board state run in Go compiled to WebAssembly

The Go WASM integration is **live**: the worker loads the engine, the React hook calls it for valid moves and move application, and pawn promotion is wired end-to-end.

---

## Repository Layout

```
my-stockfish/
├── front/        # React 19 + TypeScript (strict) + Vite (Bun) — see front/README.md
├── go-wasm/      # Go 1.25 chess engine compiled to WASM — see go-wasm/README.md
├── AGENTS.md     # Project state, architecture, and contribution rules
├── CLAUDE.md     # Global coding preferences (symlink)
└── .gitignore
```

Each subproject has its own README with detailed architecture and instructions:

- [`front/README.md`](front/README.md) — React app, Vite plugin, WASM worker bridge, components
- [`go-wasm/README.md`](go-wasm/README.md) — Go chess engine, move generation, perft validation, WASM build

---

## Stack

| Layer | Tech |
|---|---|
| Frontend | React 19 + TypeScript (strict) + Vite + CSS Modules |
| Routing | react-router-dom v7 |
| Package manager | Bun (frontend) / Go modules (engine) |
| Checkers AI | TypeScript — Minimax + Alpha-Beta + IDDFS (depth 8) |
| Chess engine | Go 1.25 → WebAssembly (board state, move gen, castling, check/checkmate) |

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
go test ./pkg/engine/ -v

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

### Xadrez (Chess) — playable

- Game runs in the browser via Go WASM: human-vs-human
- Go engine handles: board representation, FEN loading (with turn + castling rights), move generation for all piece types, captures, en passant, pawn promotion, **castling**
- **Castling**: kingside & queenside for both colors; all 6 FIDE conditions checked; rook moves with the king; castling rights cleared on king/rook moves and rook captures
- **Legal move filtering**: pseudo-legal moves filtered by snapshot/make/restore — pins, en-passant discovered checks, king-moves-into-check handled automatically
- **Check detection & game status**: checkmate, stalemate, and draw detection; king square glows red when in check; "Xeque!" badge; result overlay ("Brancas vencem!" / "Pretas vencem!" / "Empate!")
- **Perft validation**: all 6 standard positions from chessprogramming.org pass (initial position through depth 5 = 4,865,609 nodes; Kiwipete; Positions 3-6)
- Pawn promotion: picker modal (Q/N/R/B) wired end-to-end
- Board flip, turn banner, result overlay

### What is missing

- **Chess AI**: no minimax/alpha-beta in Go yet; `human-vs-ai` mode is defined but black doesn't move automatically
- **Type generator auto-run**: the Vite plugin does not run the type generator; `wasm-contract.ts` is hand-maintained

See `AGENTS.md` for the full architecture, call flow, encoding details, and contribution rules.

---

## License

Personal project — no license specified.