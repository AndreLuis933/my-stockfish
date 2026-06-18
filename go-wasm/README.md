# Go WASM — Chess Engine

Go 1.25 chess engine compiled to WebAssembly. Powers the Xadrez (Chess) game in the React frontend.

---

## What it does

The engine handles all chess logic for the browser app:

- **Board representation**: `[64]Piece` array (mailbox indexing, a1=0, h8=63)
- **FEN loading**: piece placement, side to move, castling rights (`KQkq`)
- **Move generation**: all piece types — pawn (forward, double, captures, en passant, promotion), knight, bishop, rook, queen, king (one-step + castling)
- **Castling**: kingside & queenside for both colors; all 6 FIDE conditions checked (rights, empty path, king not in check, king path not attacked, king destination not attacked); rook moves with the king; rights cleared on king/rook moves and rook captures
- **Legal move filtering**: pseudo-legal moves filtered by snapshot/make/restore (state stack for recursion) — pins, en-passant discovered checks, and king-moves-into-check handled automatically
- **Check detection**: `IsSquareAttacked` (reverse-scan from a square) + `IsInCheck` (king square attacked) + `KingCheck()` (returns checked king's square index or -1)
- **Game status**: `CurrentStatus()` returns `playing | white-wins | black-wins | draw` (checkmate = no legal moves + in check, stalemate = no legal moves + not in check)
- **Perft validation**: `Perft()` recursive move enumeration; validated against all 6 standard positions from chessprogramming.org/Perft_Results

---

## Project structure

```
go-wasm/
├── cmd/
│   ├── wasm/main.go           # WASM entry: registers goWasmEngine JS functions
│   └── command/main.go        # CLI debug: loads FEN, runs Perft
├── pkg/
│   ├── types/types.go         # Piece uint8, CastlingRights uint8, Move struct, Piece methods
│   └── engine/                # Chess logic (pure Go, no JS deps)
│       ├── board.go           # Board [64]Piece, state vars, KingCheck(), Perft()
│       ├── anotation.go       # LoadFen(): FEN → Board, turn, castling rights
│       ├── print.go           # PrintBoard(): ASCII debug
│       ├── moves.go           # getPseudoLegalMoves(): iterate board, dispatch per piece type
│       ├── move_pawn.go       # Pawn moves: forward, double, captures, en passant, promotion
│       ├── move_knight.go     # Knight L-jumps with row/col diff validation
│       ├── move_bishop.go     # Bishop diagonal slides
│       ├── move_rook.go       # Rook orthogonal slides
│       ├── move_king.go       # King one-step + castling (kingside/queenside, attack checks)
│       ├── move_apply.go      # MakeMove(): applies move, en passant, castling rook, rights clearing
│       ├── attacks.go         # FindKing, IsSquareAttacked, IsInCheck
│       ├── legal.go           # GetValidMoves(): pseudo-legal → legal filter (state stack)
│       ├── status.go          # GameStatus enum + CurrentStatus()
│       ├── legal_test.go      # Tests: FEN, castling rights, legal moves, pins, en passant
│       ├── status_test.go     # Tests: checkmate, stalemate, game status
│       └── perft_test.go      # Tests: Perft on all 6 chessprogramming.org positions
├── tools/main.go              # Type generator: Go AST → wasm-contract.ts (optional)
└── bin/gen-types.exe          # Compiled type generator
```

---

## Getting started

### Prerequisites

- [Go 1.25+](https://go.dev/dl/)

### Run tests

```bash
go test ./pkg/engine/ -v
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

`Move.Promotion` is a `*Piece` (color bits | type bits) — only set for pawn promotion moves. The engine emits 4 separate moves per promotable pawn push (Q, N, B, R), each with a different `Promotion` byte.

## Castling rights encoding

```
CastlingRights uint8 bitmask:
  bit 0: CastleWhiteK  (white kingside,  e1→g1,  rook h1→f1)
  bit 1: CastleWhiteQ  (white queenside, e1→c1,  rook a1→d1)
  bit 2: CastleBlackK  (black kingside,  e8→g8,  rook h8→f8)
  bit 3: CastleBlackQ  (black queenside, e8→c8,  rook a8→d8)
```

Parsed from FEN field 2 (`KQkq` or `-`) by `LoadFen`. Cleared in `MakeMove` on king moves, rook moves from corners, and rook captures on corners.

---

## Registered JS functions (cmd/wasm/main.go)

| JS name | Go bridge | Pure function | Args | Return |
|---|---|---|---|---|
| `validMovesChess` | `getValidMovesJS` | `engine.GetValidMoves()` | — | JSON string of `Move[]` |
| `initBoard` | `initBoardJs` | `engine.LoadFen("rnbqkbnr/...")` | — | `number[]` (64 board bytes) |
| `makeMove` | `makeMoveJS` | `engine.MakeMove(from, to, promotion)` | `number, number, number?` | `number[]` (64 board bytes) |
| `isCheckJS` | `isCheckJS` | `engine.KingCheck()` | — | `number` (checked king's square index, or -1) |
| `gameStatus` | `gameStatusJS` | `engine.CurrentStatus().String()` | — | `string` (`"playing" \| "white-wins" \| "black-wins" \| "draw"`) |

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