package engine

import (
	"testing"

	"webassemble/pkg/types"
)

// verifyBitboards checks that every piece on the mailbox has its bit set
// in the corresponding bitboard, and every empty square has no bit set.
// Also checks derived occupancy bitboards match.
func verifyBitboards(t *testing.T, p *Position, context string) {
	t.Helper()
	for sq, piece := range p.Board {
		if piece == 0 {
			if p.WhitePieces&(1<<sq) != 0 {
				t.Errorf("%s: sq %d empty on mailbox but set in WhitePieces", context, sq)
			}
			if p.BlackPieces&(1<<sq) != 0 {
				t.Errorf("%s: sq %d empty on mailbox but set in BlackPieces", context, sq)
			}
			continue
		}
		bb := p.pieceBitboardFor(piece)
		if bb == nil {
			t.Errorf("%s: sq %d has piece %d but no bitboard found", context, sq, piece)
			continue
		}
		if *bb&(1<<sq) == 0 {
			t.Errorf("%s: sq %d has piece %d but bit not set in its bitboard", context, sq, piece)
		}
	}
	// Verify derived occupancy matches the union of piece bitboards.
	wantWhite := p.WhitePawns | p.WhiteKnights | p.WhiteBishops |
		p.WhiteRooks | p.WhiteQueens | p.WhiteKing
	wantBlack := p.BlackPawns | p.BlackKnights | p.BlackBishops |
		p.BlackRooks | p.BlackQueens | p.BlackKing
	if p.WhitePieces != wantWhite {
		t.Errorf("%s: WhitePieces mismatch: got %016x, want %016x", context, uint64(p.WhitePieces), uint64(wantWhite))
	}
	if p.BlackPieces != wantBlack {
		t.Errorf("%s: BlackPieces mismatch: got %016x, want %016x", context, uint64(p.BlackPieces), uint64(wantBlack))
	}
	if p.Occupied != (wantWhite|wantBlack) {
		t.Errorf("%s: Occupied mismatch", context)
	}
	if p.Empty != ^(wantWhite | wantBlack) {
		t.Errorf("%s: Empty mismatch", context)
	}
}

func TestBitboardMakeUnmakeStartingPosition(t *testing.T) {
	LoadFen(StartingFEN)
	verifyBitboards(t, Game, "after LoadFen")

	// Generate all legal moves, make each one, verify bitboards, unmake, verify.
	var ml MoveList
	Game.LegalMoves(&ml)

	for i := range ml.Len() {
		move := ml.Get(i)
		Game.Make(move)
		verifyBitboards(t, Game, "after Make")
		Game.Unmake(move)
		verifyBitboards(t, Game, "after Unmake")
	}
}

func TestBitboardMakeUnmakeTacticalPosition(t *testing.T) {
	fens := []string{
		"r3k2r/p1ppqpb1/bn2Qnp1/2qPN3/1p2P3/2N5/PPPBBPPP/R3K2R b KQkq - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
	}

	for _, fen := range fens {
		LoadFen(fen)
		verifyBitboards(t, Game, "LoadFen: "+fen)

		var ml MoveList
		Game.LegalMoves(&ml)

		for i := range ml.Len() {
			move := ml.Get(i)
			Game.Make(move)
			verifyBitboards(t, Game, "Make: "+fen)

			// Go one ply deeper to stress test Make/Unmake stacking.
			var ml2 MoveList
			Game.LegalMoves(&ml2)
			for j := range ml2.Len() {
				move2 := ml2.Get(j)
				Game.Make(move2)
				verifyBitboards(t, Game, "Make2: "+fen)
				Game.Unmake(move2)
				verifyBitboards(t, Game, "Unmake2: "+fen)
			}

			Game.Unmake(move)
			verifyBitboards(t, Game, "Unmake: "+fen)
		}
	}
}

func TestBitboardMakeUnmakeAllMoveTypes(t *testing.T) {
	// A position that exercises every move flag type.
	LoadFen("r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1")
	verifyBitboards(t, Game, "castling setup")

	var ml, ml2 MoveList
	Game.LegalMoves(&ml)

	seenFlags := map[types.MoveFlag]bool{}
	for i := range ml.Len() {
		move := ml.Get(i)
		seenFlags[move.Flag] = true
		Game.Make(move)
		verifyBitboards(t, Game, "castling Make")
		Game.Unmake(move)
		verifyBitboards(t, Game, "castling Unmake")
	}

	// Promotion + en passant
	LoadFen("8/P7/8/8/8/8/8/4k2K w - - 0 1")
	verifyBitboards(t, Game, "promotion setup")
	Game.LegalMoves(&ml)
	for i := range ml.Len() {
		move := ml.Get(i)
		seenFlags[move.Flag] = true
		Game.Make(move)
		verifyBitboards(t, Game, "promotion Make")
		Game.Unmake(move)
		verifyBitboards(t, Game, "promotion Unmake")
	}

	// En passant via 2-ply search: white pawn already on e5, black plays d7-d5, white e.p. captures.
	// Using a FEN without e.p. target (avoids pre-existing squareToIndex bug).
	LoadFen("4k3/3p4/8/4P3/8/8/8/4K3 w - - 0 1")
	verifyBitboards(t, Game, "e.p. 2-ply setup")
	ml.Clear()
	Game.LegalMoves(&ml)
	epFound := false
	for i := range ml.Len() {
		move1 := ml.Get(i)
		Game.Make(move1)
		ml2.Clear()
		Game.LegalMoves(&ml2)
		for j := range ml2.Len() {
			move2 := ml2.Get(j)
			if move2.Flag != types.FlagDoublePush {
				continue
			}
			Game.Make(move2)
			var ml3 MoveList
			Game.LegalMoves(&ml3)
			for k := range ml3.Len() {
				move3 := ml3.Get(k)
				seenFlags[move3.Flag] = true
				if move3.Flag == types.FlagEnPassant {
					epFound = true
				}
				Game.Make(move3)
				verifyBitboards(t, Game, "e.p. Make3")
				Game.Unmake(move3)
				verifyBitboards(t, Game, "e.p. Unmake3")
			}
			Game.Unmake(move2)
		}
		Game.Unmake(move1)
	}
	if !epFound {
		t.Log("warning: no en passant move found")
	}

	// Double push: white pawn on a2 can push to a3 or double-push to a4.
	LoadFen("4k3/8/8/8/8/8/P7/4K3 w - - 0 1")
	verifyBitboards(t, Game, "double push setup")
	ml.Clear()
	Game.LegalMoves(&ml)
	for i := range ml.Len() {
		move := ml.Get(i)
		seenFlags[move.Flag] = true
		Game.Make(move)
		verifyBitboards(t, Game, "double push Make")
		Game.Unmake(move)
		verifyBitboards(t, Game, "double push Unmake")
	}

	// Verify we exercised all flag types.
	for _, flag := range []types.MoveFlag{
		types.FlagNormal, types.FlagDoublePush, types.FlagEnPassant,
		types.FlagCastleK, types.FlagCastleQ, types.FlagPromotion,
	} {
		if !seenFlags[flag] {
			t.Errorf("did not test move flag %d", flag)
		}
	}
}