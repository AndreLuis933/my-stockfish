"""Build a quality-filtered Polyglot opening book from a PGN database.

Pipeline (each step is a separate subcommand so you can inspect between phases):

  parse    PGN  -> raw.json        (Phase 1: collect position/move frequencies)
  filter   raw  -> popular.json    (Phase 2: drop rare moves, keep top-N per position)
  analyze  pop  -> filtered.json   (Phase 3: Stockfish quality filter + blended weights)
  write    filt -> book.bin        (Phase 4: write sorted polyglot .bin)
  all      PGN  -> book.bin        (run all 4 phases in one go)

Usage:
  pip install chess
  zstd -d lichess_db_standard_rated_2015-08.pgn.zst -o games.pgn

  # Step-by-step (recommended — inspect between phases):
  python scripts/build-book.py parse games.pgn -o raw.json --min-elo 2000
  python scripts/build-book.py filter raw.json -o popular.json --min-move-games 5 --max-moves-per-position 5
  python scripts/build-book.py analyze popular.json -o filtered.json --depth 12 --eval-range 100 --max-time 3600
  python scripts/build-book.py write filtered.json -o go-wasm/books/book.bin

  # Or all at once:
  python scripts/build-book.py all games.pgn -o go-wasm/books/book.bin
"""


import argparse
import json
import os
import struct
import time

import chess
from tqdm import tqdm

ENTRY_FORMAT = ">QHHI"  # key(8) move(2) weight(2) learn(4)
ENTRY_SIZE = struct.calcsize(ENTRY_FORMAT)

PROMO_CODES = {
    chess.KNIGHT: 1,
    chess.BISHOP: 2,
    chess.ROOK: 3,
    chess.QUEEN: 4,
}

PROMO_DECODE = {1: chess.KNIGHT, 2: chess.BISHOP, 3: chess.ROOK, 4: chess.QUEEN}


# ─── helpers ─────────────────────────────────────────────────────────────────


def encode_move(move: chess.Move) -> int:
    """Encode a chess.Move into the 16-bit Polyglot move format."""
    promo = 0
    if move.promotion:
        promo = PROMO_CODES[move.promotion]
    return move.to_square | (move.from_square << 6) | (promo << 12)


def decode_move(raw_move: int, board: chess.Board) -> chess.Move | None:
    """Decode a 16-bit Polyglot move into a chess.Move, validating legality."""
    to_square = raw_move & 0x3F
    from_square = (raw_move >> 6) & 0x3F
    promo_part = (raw_move >> 12) & 0x7

    if from_square == to_square:
        return None

    promotion = PROMO_DECODE.get(promo_part) if promo_part else None

    move = chess.Move(from_square, to_square, promotion)
    if move in board.legal_moves:
        return move
    return None


def parse_elo(value: str | None) -> int:
    """Parse an Elo header value, returning 0 for unknown/unrated/invalid."""
    if not value or value == "?":
        return 0
    try:
        return int(value)
    except ValueError:
        return 0


def score_to_cp(score: chess.engine.Score) -> int:
    """Convert a Score to centipawns. Mate scores map to large +/- values."""
    if score.is_mate():
        mate = score.mate()
        if mate is None:
            return 0
        if mate > 0:
            return 10000 - mate
        return -(10000 - abs(mate))
    return score.score()


def auto_detect_stockfish() -> str:
    """Find stockfish executable in the project root or PATH."""
    for candidate in ["stockfish.exe", "stockfish", os.path.join(os.getcwd(), "stockfish.exe")]:
        if os.path.isfile(candidate):
            return candidate
    return "stockfish"


class ProgressFile:
    """File wrapper that updates a tqdm progress bar on every read based on byte offset."""

    def __init__(self, file, bar: tqdm):
        self._file = file
        self._bar = bar
        self._offset = 0

    def read(self, size: int = -1) -> bytes:
        data = self._file.read(size)
        if data:
            advance = len(data)
            self._bar.update(advance)
            self._offset += advance
        return data

    def readline(self) -> bytes:
        data = self._file.readline()
        if data:
            advance = len(data)
            self._bar.update(advance)
            self._offset += advance
        return data

    def tell(self) -> int:
        return self._file.tell()

    def seek(self, offset: int, whence: int = 0) -> int:
        new_pos = self._file.seek(offset, whence)
        self._bar.update(new_pos - self._offset)
        self._offset = new_pos
        return new_pos

    def close(self) -> None:
        self._file.close()


# ─── Phase 1: parse ──────────────────────────────────────────────────────────


def cmd_parse(args: argparse.Namespace) -> None:
    """Phase 1: Parse PGN, collect (hash, move) frequencies + FENs."""
    min_elo = args.min_elo
    max_plies = args.max_plies
    input_path = args.input
    output_path = args.output
    file_size = os.path.getsize(input_path)

    print(f"[Phase 1] Parsing PGN: {input_path}")
    print(f"          file size: {file_size / 1024 / 1024 / 1024:.2f} GB")
    print(f"          min_elo={min_elo}, max_plies={max_plies}")
    print(f"          output: {output_path}")
    print()

    entries: dict[str, int] = {}  # "hash:raw_move" -> weight (string keys for JSON)
    fens: dict[str, str] = {}  # "hash" -> FEN
    games_processed = 0
    games_skipped_elo = 0
    games_skipped_no_moves = 0
    last_report_games = 0
    last_report_time = time.time()
    start_time = time.time()

    raw_file = open(input_path, encoding="utf-8", errors="ignore")
    bar = tqdm(
        total=file_size,
        unit="B",
        unit_scale=True,
        unit_divisor=1024,
        desc="Parsing PGN",
        mininterval=0.5,
        dynamic_ncols=True,
    )
    pgn_file = ProgressFile(raw_file, bar)

    try:
        while True:
            game = chess.pgn.read_game(pgn_file)
            if game is None:
                break

            white_elo = parse_elo(game.headers.get("WhiteElo"))
            black_elo = parse_elo(game.headers.get("BlackElo"))
            if white_elo < min_elo or black_elo < min_elo:
                games_skipped_elo += 1
                continue

            moves = list(game.mainline_moves())
            if not moves:
                games_skipped_no_moves += 1
                continue

            games_processed += 1
            board = game.board()
            for ply, move in enumerate(moves):
                if ply >= max_plies:
                    break
                key = chess.polyglot.zobrist_hash(board)
                key_str = str(key)
                if key_str not in fens:
                    fens[key_str] = board.fen()
                raw_move = encode_move(move)
                entry_key = f"{key_str}:{raw_move}"
                entries[entry_key] = entries.get(entry_key, 0) + 1
                board.push(move)

            # Periodically print stats alongside the bar
            if games_processed % 50000 == 0:
                now = time.time()
                game_rate = (games_processed - last_report_games) / (now - last_report_time)
                bar.write(
                    f"  {games_processed:,} games ({game_rate:.0f} games/s) | "
                    f"{games_skipped_elo:,} skipped | "
                    f"{len(entries):,} entries | "
                    f"{len(fens):,} positions"
                )
                last_report_games = games_processed
                last_report_time = now

    finally:
        bar.close()
        raw_file.close()

    elapsed = time.time() - start_time
    print()
    print(
        f"[Phase 1] Done in {elapsed:.0f}s:\n"
        f"          {games_processed:,} games parsed\n"
        f"          {games_skipped_elo:,} skipped by Elo filter\n"
        f"          {games_skipped_no_moves:,} skipped (no moves)\n"
        f"          {len(entries):,} raw entries\n"
        f"          {len(fens):,} unique positions"
    )

    data = {"entries": entries, "fens": fens}
    tmp = output_path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(data, f)
    os.replace(tmp, output_path)
    file_mb = os.path.getsize(output_path) / 1024 / 1024
    print(f"          Saved: {output_path} ({file_mb:.1f} MB)")


# ─── Phase 2: filter ─────────────────────────────────────────────────────────


def cmd_filter(args: argparse.Namespace) -> None:
    """Phase 2: Drop moves below the frequency threshold, keep top-N per position."""
    input_path = args.input
    output_path = args.output
    min_move_games = args.min_move_games
    max_moves = args.max_moves_per_position

    print(f"[Phase 2] Popularity filter: min_move_games={min_move_games}, max_moves_per_position={max_moves}")
    print(f"          input: {input_path}")
    print(f"          output: {output_path}")

    file_size = os.path.getsize(input_path)
    with tqdm(
        total=file_size,
        unit="B",
        unit_scale=True,
        unit_divisor=1024,
        desc="Loading JSON",
        mininterval=0.5,
        dynamic_ncols=True,
    ) as bar:
        raw_file = open(input_path, encoding="utf-8")
        pgn_file = ProgressFile(raw_file, bar)
        try:
            data = json.load(pgn_file)
        finally:
            raw_file.close()

    raw_entries: dict[str, int] = data["entries"]
    fens: dict[str, str] = data["fens"]
    print(f"          Loaded {len(raw_entries):,} entries, {len(fens):,} positions")

    # Group by position, apply min_move_games filter
    positions: dict[str, dict[str, int]] = {}
    dropped_by_frequency = 0

    for entry_key, weight in tqdm(
        raw_entries.items(),
        total=len(raw_entries),
        unit="move",
        desc="Filtering",
        mininterval=0.5,
        dynamic_ncols=True,
    ):
        if weight < min_move_games:
            dropped_by_frequency += 1
            continue
        key_str, raw_str = entry_key.split(":", 1)
        pos = positions.setdefault(key_str, {})
        pos[raw_str] = weight

    # Keep only top-N moves per position (sorted by weight descending)
    dropped_by_topn = 0
    if max_moves > 0:
        for pos_hash, moves in positions.items():
            if len(moves) <= max_moves:
                continue
            sorted_moves = sorted(moves.items(), key=lambda x: x[1], reverse=True)
            kept = dict(sorted_moves[:max_moves])
            dropped_by_topn += len(moves) - len(kept)
            positions[pos_hash] = kept

    fens_filtered = {h: fens[h] for h in positions if h in fens}
    total_moves = sum(len(m) for m in positions.values())

    print(
        f"[Phase 2] Done:\n"
        f"          {len(positions):,} unique positions\n"
        f"          {total_moves:,} moves kept\n"
        f"          {dropped_by_frequency:,} moves dropped by frequency filter\n"
        f"          {dropped_by_topn:,} moves dropped by top-{max_moves} limit"
    )

    out = {"positions": positions, "fens": fens_filtered}
    tmp = output_path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(out, f)
    os.replace(tmp, output_path)
    file_mb = os.path.getsize(output_path) / 1024 / 1024
    print(f"          Saved: {output_path} ({file_mb:.1f} MB)")


# ─── Phase 3: analyze (Stockfish) ────────────────────────────────────────────


def cmd_analyze(args: argparse.Namespace) -> None:
    """Phase 3: Filter moves using Stockfish analysis + blended weights.

    For each position:
      1. Unrestricted analysis at depth -> true best eval (benchmark).
      2. Restricted analysis (root_moves = book moves, multipv) -> eval per book move.
      3. Hard gate: keep only moves where -eval_range <= move_cp <= +eval_range (absolute).
      4. quality_score = max(0, 1 - (true_best_cp - move_cp) / eval_range)  (relative to best)
      5. popularity_score = original_weight / max_weight_in_position
      6. final_weight = round(raw_blend / sum(raw_blends) * 100)  (50/50 blend, sums to ~100)
    """
    input_path = args.input
    output_path = args.output
    depth = args.depth
    eval_range = args.eval_range
    max_time = args.max_time
    engine_path = args.stockfish or auto_detect_stockfish()
    checkpoint_path = args.checkpoint or (output_path + ".checkpoint.json")
    resume = args.resume

    print(f"[Phase 3] Stockfish quality filter + blended weights")
    print(f"          input: {input_path}")
    print(f"          output: {output_path}")
    print(f"          depth={depth}, eval_range=±{eval_range}cp")
    if max_time:
        print(f"          max_time={max_time}s ({max_time/60:.0f} min)")
    print(f"          engine: {engine_path}")

    with open(input_path, encoding="utf-8") as f:
        data = json.load(f)

    positions: dict[str, dict[str, int]] = data["positions"]
    fens: dict[str, str] = data["fens"]
    print(f"          Loaded {len(positions):,} positions, {len(fens):,} FENs")

    # Load checkpoint if resuming
    evaluated: dict[str, dict[str, int]] = {}
    if resume and os.path.exists(checkpoint_path):
        with open(checkpoint_path, encoding="utf-8") as f:
            ckpt = json.load(f)
        evaluated = ckpt["evaluated"]
        print(f"          Resumed: {len(evaluated):,} positions already evaluated")

    # Sort for deterministic order; skip already-evaluated
    pos_items = sorted(positions.items())
    pending = [(h, moves) for h, moves in pos_items if h not in evaluated]
    total_pending = len(pending)
    print(f"          {total_pending:,} positions to evaluate")
    print()

    start_time = time.time()
    checkpoint_interval = 5000
    time_limit_hit = False

    bar = tqdm(
        total=total_pending,
        unit="pos",
        desc="Stockfish",
        mininterval=0.5,
        dynamic_ncols=True,
    )

    engine = chess.engine.SimpleEngine.popen_uci(engine_path)
    try:
        for i, (pos_hash_str, book_moves) in enumerate(pending):
            # Check time limit
            if max_time and (time.time() - start_time) >= max_time:
                bar.write(
                    f"  Time limit reached: {i:,}/{total_pending:,} positions evaluated"
                )
                time_limit_hit = True
                break

            fen = fens.get(pos_hash_str)
            if fen is None:
                bar.update(1)
                continue

            board = chess.Board(fen)

            # 1) Unrestricted analysis -> true best eval (benchmark)
            try:
                info_best = engine.analyse(board, chess.engine.Limit(depth=depth))
                true_best_cp = score_to_cp(info_best["score"].pov(board.turn))
            except (chess.engine.EngineError, KeyError):
                bar.update(1)
                continue

            # 2) Restricted analysis -> eval per book move
            move_list = list(book_moves.keys())
            restricted_moves = [decode_move(int(m), board) for m in move_list]
            restricted_moves = [m for m in restricted_moves if m is not None]
            if not restricted_moves:
                bar.update(1)
                continue

            try:
                infos = engine.analyse(
                    board,
                    chess.engine.Limit(depth=depth),
                    root_moves=restricted_moves,
                    multipv=len(restricted_moves),
                )
            except chess.engine.EngineError:
                bar.update(1)
                continue

            # 3) Hard gate: keep only moves where -eval_range <= move_cp <= +eval_range
            # 4) Compute quality_score relative to best
            # 5) Compute popularity_score
            # 6) Blend 50/50 and normalize to sum=100 per position
            max_weight = max(book_moves.values()) if book_moves else 1
            candidates: list[tuple[str, float]] = []  # (raw_str, raw_blend)

            for info in infos:
                pv = info.get("pv", [])
                if not pv:
                    continue
                move = pv[0]
                raw = encode_move(move)
                raw_str = str(raw)
                if raw_str not in book_moves:
                    continue

                move_cp = score_to_cp(info["score"].pov(board.turn))

                # Hard gate: absolute eval must be within ±eval_range
                if move_cp < -eval_range or move_cp > eval_range:
                    continue

                # quality_score: 1.0 if equal to best, 0.0 if eval_range worse
                gap = true_best_cp - move_cp
                quality_score = max(0.0, 1.0 - gap / eval_range)

                # popularity_score: 1.0 for most popular move, scaled for others
                popularity_score = book_moves[raw_str] / max_weight

                # 50/50 blend
                raw_blend = (popularity_score + quality_score) / 2.0
                candidates.append((raw_str, raw_blend))

            # Normalize to sum=100 per position
            surviving: dict[str, int] = {}
            if candidates:
                total_blend = sum(b for _, b in candidates)
                if total_blend > 0:
                    for raw_str, blend in candidates:
                        weight = round(blend / total_blend * 100)
                        if weight > 0:
                            surviving[raw_str] = weight

            evaluated[pos_hash_str] = surviving

            processed = i + 1
            if processed % checkpoint_interval == 0:
                _save_checkpoint(checkpoint_path, evaluated)
                kept = sum(len(v) for v in evaluated.values())
                bar.write(
                    f"  Checkpoint: {len(evaluated):,} evaluated, "
                    f"{kept:,} moves kept"
                )

            bar.update(1)

    finally:
        engine.quit()
        bar.close()

    _save_checkpoint(checkpoint_path, evaluated)

    # Build filtered output
    filtered: dict[str, dict[str, int]] = {}
    total_moves = 0
    for pos_hash_str, moves in evaluated.items():
        if moves:
            filtered[pos_hash_str] = moves
            total_moves += len(moves)

    elapsed = time.time() - start_time
    print()
    status = " (time limit reached)" if time_limit_hit else ""
    print(
        f"[Phase 3] Done in {elapsed:.0f}s{status}:\n"
        f"          {len(filtered):,} positions\n"
        f"          {total_moves:,} moves kept\n"
        f"          {len(evaluated) - len(filtered):,} positions dropped (no surviving moves)"
    )

    fens_filtered = {h: fens[h] for h in filtered if h in fens}
    out = {"positions": filtered, "fens": fens_filtered}
    tmp = output_path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump(out, f)
    os.replace(tmp, output_path)
    file_size = os.path.getsize(output_path) / 1024 / 1024
    print(f"          Saved: {output_path} ({file_size:.1f} MB)")


def _save_checkpoint(path: str, evaluated: dict[str, dict[str, int]]) -> None:
    tmp = path + ".tmp"
    with open(tmp, "w", encoding="utf-8") as f:
        json.dump({"evaluated": evaluated}, f)
    os.replace(tmp, path)


# ─── Phase 4: write ──────────────────────────────────────────────────────────


def cmd_write(args: argparse.Namespace) -> None:
    """Phase 4: Write a sorted polyglot .bin file."""
    input_path = args.input
    output_path = args.output

    print(f"[Phase 4] Writing book: {output_path}")
    print(f"          input: {input_path}")

    with open(input_path, encoding="utf-8") as f:
        data = json.load(f)

    positions: dict[str, dict[str, int]] = data["positions"]
    print(f"          Loaded {len(positions):,} positions")

    entries: list[tuple[int, int, int]] = []
    for pos_hash_str, moves in positions.items():
        key = int(pos_hash_str)
        for raw_str, weight in moves.items():
            entries.append((key, int(raw_str), weight))

    entries.sort(key=lambda e: (e[0], e[1]))

    with open(output_path, "wb") as f:
        for key, raw_move, weight in entries:
            f.write(struct.pack(ENTRY_FORMAT, key, raw_move, weight, 0))

    size = len(entries) * ENTRY_SIZE
    print(
        f"[Phase 4] Done:\n"
        f"          {len(entries):,} entries\n"
        f"          {size / 1024 / 1024:.1f} MB\n"
        f"          {output_path}"
    )
    print(f"\n  Copy to engine:  Copy-Item {output_path} go-wasm/books/book.bin")
    print(f"  Copy to browser: Copy-Item {output_path} front/public/books/book.bin")


# ─── all-in-one ──────────────────────────────────────────────────────────────


def cmd_all(args: argparse.Namespace) -> None:
    """Run all 4 phases in sequence."""
    raw_path = args.output + ".raw.json"
    popular_path = args.output + ".popular.json"
    filtered_path = args.output + ".filtered.json"

    engine_path = args.stockfish or auto_detect_stockfish()

    print("=" * 60)
    print("Polyglot Opening Book Builder — all phases")
    print("=" * 60)
    print(f"Input:       {args.input}")
    print(f"Output:      {args.output}")
    print(f"Min Elo:     {args.min_elo}")
    print(f"Max plies:   {args.max_plies}")
    print(f"Min games:   {args.min_move_games}")
    print(f"Max moves:   {args.max_moves_per_position}")
    print(f"Eval range:  ±{args.eval_range}cp")
    print(f"Depth:       {args.depth}")
    if args.max_time:
        print(f"Max time:    {args.max_time}s ({args.max_time/60:.0f} min)")
    print(f"Engine:      {engine_path}")
    print()

    # Phase 1
    parse_ns = argparse.Namespace(
        input=args.input, output=raw_path, min_elo=args.min_elo, max_plies=args.max_plies
    )
    cmd_parse(parse_ns)
    print()

    # Phase 2
    filter_ns = argparse.Namespace(
        input=raw_path,
        output=popular_path,
        min_move_games=args.min_move_games,
        max_moves_per_position=args.max_moves_per_position,
    )
    cmd_filter(filter_ns)
    print()

    # Phase 3
    analyze_ns = argparse.Namespace(
        input=popular_path,
        output=filtered_path,
        depth=args.depth,
        eval_range=args.eval_range,
        max_time=args.max_time,
        stockfish=engine_path,
        checkpoint=filtered_path + ".checkpoint.json",
        resume=args.resume,
    )
    cmd_analyze(analyze_ns)
    print()

    # Phase 4
    write_ns = argparse.Namespace(input=filtered_path, output=args.output)
    cmd_write(write_ns)

    # Clean up intermediates
    for p in [raw_path, popular_path, filtered_path, filtered_path + ".checkpoint.json"]:
        if os.path.exists(p):
            os.remove(p)


# ─── CLI ─────────────────────────────────────────────────────────────────────


def build_parser() -> argparse.ArgumentParser:
    parser = argparse.ArgumentParser(
        description="Build a quality-filtered Polyglot opening book from PGN."
    )
    sub = parser.add_subparsers(dest="command", required=True)

    # parse
    p = sub.add_parser("parse", help="Phase 1: PGN -> raw entries JSON")
    p.add_argument("input", help="Input PGN file")
    p.add_argument("-o", "--output", required=True, help="Output JSON file")
    p.add_argument("--min-elo", type=int, default=2000, help="Min Elo for both players (default: 2000)")
    p.add_argument("--max-plies", type=int, default=20, help="Max plies per game (default: 20)")
    p.set_defaults(func=cmd_parse)

    # filter
    p = sub.add_parser("filter", help="Phase 2: raw entries -> popular positions JSON")
    p.add_argument("input", help="Input JSON from 'parse'")
    p.add_argument("-o", "--output", required=True, help="Output JSON file")
    p.add_argument("--min-move-games", type=int, default=5, help="Min games per move (default: 5)")
    p.add_argument("--max-moves-per-position", type=int, default=5, help="Top-N moves by popularity per position (default: 5, 0=unlimited)")
    p.set_defaults(func=cmd_filter)

    # analyze
    p = sub.add_parser("analyze", help="Phase 3: Stockfish quality filter + blended weights")
    p.add_argument("input", help="Input JSON from 'filter'")
    p.add_argument("-o", "--output", required=True, help="Output JSON file")
    p.add_argument("--depth", type=int, default=12, help="Stockfish depth (default: 12)")
    p.add_argument("--eval-range", type=int, default=100, help="Absolute eval gate ±cp (default: 100 = ±1 pawn)")
    p.add_argument("--max-time", type=int, default=0, help="Max time in seconds (default: 0 = unlimited). Stops and writes partial book when reached.")
    p.add_argument("--stockfish", help="Path to Stockfish (auto-detected if omitted)")
    p.add_argument("--checkpoint", help="Checkpoint file (default: <output>.checkpoint.json)")
    p.add_argument("--resume", action="store_true", help="Resume from checkpoint")
    p.set_defaults(func=cmd_analyze)

    # write
    p = sub.add_parser("write", help="Phase 4: filtered positions -> .bin")
    p.add_argument("input", help="Input JSON from 'analyze'")
    p.add_argument("-o", "--output", required=True, help="Output .bin file")
    p.set_defaults(func=cmd_write)

    # all
    p = sub.add_parser("all", help="Run all 4 phases")
    p.add_argument("input", help="Input PGN file")
    p.add_argument("-o", "--output", required=True, help="Output .bin file")
    p.add_argument("--min-elo", type=int, default=2000, help="Min Elo for both players (default: 2000)")
    p.add_argument("--max-plies", type=int, default=20, help="Max plies per game (default: 20)")
    p.add_argument("--min-move-games", type=int, default=5, help="Min games per move (default: 5)")
    p.add_argument("--max-moves-per-position", type=int, default=5, help="Top-N moves per position (default: 5)")
    p.add_argument("--eval-range", type=int, default=100, help="Absolute eval gate ±cp (default: 100)")
    p.add_argument("--depth", type=int, default=12, help="Stockfish depth (default: 12)")
    p.add_argument("--max-time", type=int, default=0, help="Max time in seconds for Phase 3 (default: 0 = unlimited)")
    p.add_argument("--stockfish", help="Path to Stockfish (auto-detected if omitted)")
    p.add_argument("--resume", action="store_true", help="Resume Phase 3 from checkpoint")
    p.set_defaults(func=cmd_all)

    return parser


def main() -> None:
    parser = build_parser()
    args = parser.parse_args()
    args.func(args)


if __name__ == "__main__":
    main()