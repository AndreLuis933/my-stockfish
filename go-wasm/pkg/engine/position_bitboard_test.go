package engine

import (
	"testing"

	"webassemble/pkg/types"
)

func TestBitboardStartingPosition(t *testing.T) {
	LoadFen(StartingFEN)
	if popcount(Game.WhitePawns) != 8 {
		t.Errorf("whitePawns popcount = %d, want 8", popcount(Game.WhitePawns))
	}
	if popcount(Game.BlackPawns) != 8 {
		t.Errorf("blackPawns popcount = %d, want 8", popcount(Game.BlackPawns))
	}
	if popcount(Game.WhiteKnights) != 2 {
		t.Errorf("whiteKnights popcount = %d, want 2", popcount(Game.WhiteKnights))
	}
	if popcount(Game.BlackKnights) != 2 {
		t.Errorf("blackKnights popcount = %d, want 2", popcount(Game.BlackKnights))
	}
	if popcount(Game.WhiteBishops) != 2 {
		t.Errorf("whiteBishops popcount = %d, want 2", popcount(Game.WhiteBishops))
	}
	if popcount(Game.BlackBishops) != 2 {
		t.Errorf("blackBishops popcount = %d, want 2", popcount(Game.BlackBishops))
	}
	if popcount(Game.WhiteRooks) != 2 {
		t.Errorf("whiteRooks popcount = %d, want 2", popcount(Game.WhiteRooks))
	}
	if popcount(Game.BlackRooks) != 2 {
		t.Errorf("blackRooks popcount = %d, want 2", popcount(Game.BlackRooks))
	}
	if popcount(Game.WhiteQueens) != 1 {
		t.Errorf("whiteQueens popcount = %d, want 1", popcount(Game.WhiteQueens))
	}
	if popcount(Game.BlackQueens) != 1 {
		t.Errorf("blackQueens popcount = %d, want 1", popcount(Game.BlackQueens))
	}
	if popcount(Game.WhiteKing) != 1 {
		t.Errorf("whiteKing popcount = %d, want 1", popcount(Game.WhiteKing))
	}
	if popcount(Game.BlackKing) != 1 {
		t.Errorf("blackKing popcount = %d, want 1", popcount(Game.BlackKing))
	}
	if popcount(Game.Occupied) != 32 {
		t.Errorf("occupied popcount = %d, want 32", popcount(Game.Occupied))
	}
	if popcount(Game.Empty) != 32 {
		t.Errorf("empty popcount = %d, want 32", popcount(Game.Empty))
	}
	if popcount(Game.WhitePieces) != 16 {
		t.Errorf("whitePieces popcount = %d, want 16", popcount(Game.WhitePieces))
	}
	if popcount(Game.BlackPieces) != 16 {
		t.Errorf("blackPieces popcount = %d, want 16", popcount(Game.BlackPieces))
	}
}

func TestBitboardMailboxConsistency(t *testing.T) {
	// After LoadFen, every piece on the mailbox must have its bit set in the
	// corresponding bitboard, and every empty square must have no bit set.
	positions := []string{
		StartingFEN,
		"r3k2r/p1ppqpb1/bn2Qnp1/2qPN3/1p2P3/2N5/PPPBBPPP/R3K2R b KQkq - 0 1",
		"8/2p5/3p4/KP5r/1R3p1k/8/4P1P1/8 w - - 0 1",
		"rnbq1k1r/pp1Pbppp/2p5/8/2B5/8/PPP1NnPP/RNBQK2R w KQ - 1 8",
		"r4rk1/1pp1qppp/p1np1n2/2b1p1B1/2B1P1b1/P1NP1N2/1PP1QPPP/R4RK1 w - - 0 10",
		"8/8/8/8/8/8/8/4K2k w - - 0 1",
	}

	for _, fen := range positions {
		LoadFen(fen)
		for sq, piece := range Game.Board {
			if piece == 0 {
				// Empty square: no bitboard should have this bit set.
				if Game.WhitePieces&(1<<sq) != 0 || Game.BlackPieces&(1<<sq) != 0 {
					t.Errorf("FEN %s: sq %d is empty on mailbox but set in occupancy", fen, sq)
				}
				continue
			}
			// Piece present: the corresponding bitboard must have the bit set.
			bb := Game.pieceBitboardFor(piece)
			if bb == nil {
				t.Errorf("FEN %s: sq %d has piece %d but no bitboard found", fen, sq, piece)
				continue
			}
			if *bb&(1<<sq) == 0 {
				t.Errorf("FEN %s: sq %d has piece %d but bit not set in its bitboard", fen, sq, piece)
			}
		}
	}
}

func TestPieceBitboardFor(t *testing.T) {
	// Verify the helper returns the right pointer for each piece type/color.
	p := &Position{}
	p.buildPieceBBTable()
	tests := []struct {
		piece types.Piece
		want  *Bitboard
	}{
		{types.Pawn | types.ColorWhite, &p.WhitePawns},
		{types.Knight | types.ColorWhite, &p.WhiteKnights},
		{types.Bishop | types.ColorWhite, &p.WhiteBishops},
		{types.Rook | types.ColorWhite, &p.WhiteRooks},
		{types.Queen | types.ColorWhite, &p.WhiteQueens},
		{types.King | types.ColorWhite, &p.WhiteKing},
		{types.Pawn | types.ColorBlack, &p.BlackPawns},
		{types.Knight | types.ColorBlack, &p.BlackKnights},
		{types.Bishop | types.ColorBlack, &p.BlackBishops},
		{types.Rook | types.ColorBlack, &p.BlackRooks},
		{types.Queen | types.ColorBlack, &p.BlackQueens},
		{types.King | types.ColorBlack, &p.BlackKing},
	}
	for _, tc := range tests {
		got := p.pieceBitboardFor(tc.piece)
		if got != tc.want {
			t.Errorf("pieceBitboardFor(%d): got wrong pointer", tc.piece)
		}
	}
}