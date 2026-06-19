package engine

import (
	"testing"

	"webassemble/pkg/types"
)

func loadFEN(t *testing.T, fen string) {
	t.Helper()
	Game.reset()
	LoadFen(fen)
}

func TestLoadFenStartingPosition(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")

	if !Game.WhiteToMove {
		t.Fatal("expected white to move")
	}
	if Game.CastlingRights != types.CastleAll {
		t.Fatalf("expected castling rights KQkq, got %d", Game.CastlingRights)
	}
	if Game.Board[0] != types.Rook|types.ColorWhite {
		t.Errorf("expected white rook on a1, got %d", Game.Board[0])
	}
	if Game.Board[4] != types.King|types.ColorWhite {
		t.Errorf("expected white king on e1, got %d", Game.Board[4])
	}
	if Game.Board[60] != types.King|types.ColorBlack {
		t.Errorf("expected black king on e8, got %d", Game.Board[60])
	}
	if Game.Board[63] != types.Rook|types.ColorBlack {
		t.Errorf("expected black rook on h8, got %d", Game.Board[63])
	}
}

func TestLoadFenCastlingRights(t *testing.T) {
	cases := []struct {
		fen    string
		rights types.CastlingRights
	}{
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1", types.CastleAll},
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQ - 0 1", types.CastleWhiteAll},
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR b kq - 0 1", types.CastleBlackAll},
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w - - 0 1", 0},
		{"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w K - 0 1", types.CastleWhiteK},
	}
	for _, c := range cases {
		t.Run(c.fen, func(t *testing.T) {
			loadFEN(t, c.fen)
			if Game.CastlingRights != c.rights {
				t.Fatalf("expected rights %d, got %d", c.rights, Game.CastlingRights)
			}
		})
	}
}

func TestLegalMoveCountStartingPosition(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	moves := Game.LegalMovesSlice()
	if len(moves) != 20 {
		t.Fatalf("expected 20 legal moves at start, got %d", len(moves))
	}
}

func TestLegalMoveCountEmptyBoard(t *testing.T) {
	loadFEN(t, "8/8/8/8/8/8/8/8 w - - 0 1")
	moves := Game.LegalMovesSlice()
	if len(moves) != 0 {
		t.Fatalf("expected 0 legal moves on empty board, got %d", len(moves))
	}
}

func TestLegalMovesKingInCheckFromRook(t *testing.T) {
	// White king e1, black rook e2, black king e8. White is in check from the rook.
	loadFEN(t, "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1")
	moves := Game.LegalMovesSlice()

	movesToTarget := map[int]bool{}
	for _, m := range moves {
		movesToTarget[m.To] = true
	}

	rookIdx := 12 // e2
	if !movesToTarget[rookIdx] {
		t.Errorf("expected king to be able to capture the checking rook on e2")
	}

	kingIdx := 4 // e1
	for _, m := range moves {
		if m.From != kingIdx {
			t.Errorf("when in check, only the king can move (or capture the checker); found move from %d", m.From)
		}
	}
}

func TestLegalMovesPinnedPiece(t *testing.T) {
	// White bishop e2 pinned by black rook on e8 against white king on e1.
	// Bishop must not be allowed to move off the e-file.
	loadFEN(t, "4r3/8/8/8/8/8/4B3/4K3 w - - 0 1")
	moves := Game.LegalMovesSlice()

	bishopIdx := 12 // e2
	for _, m := range moves {
		if m.From == bishopIdx {
			t.Errorf("pinned bishop should not be allowed to move off the pin line")
		}
	}
}

func TestLegalMovesKingCannotMoveIntoCheck(t *testing.T) {
	// White queen on e7 attacks d8, e8 (king square), f8, and the e-file.
	// Black king on e8 is in check and must move; d8 and f8 are attacked by the enemy queen.
	loadFEN(t, "4k3/4Q3/8/8/8/8/8/4K3 b - - 0 1")
	moves := Game.LegalMovesSlice()

	kingIdx := 60 // e8

	kingMoveTargets := map[int]bool{}
	for _, m := range moves {
		if m.From == kingIdx {
			kingMoveTargets[m.To] = true
		}
	}

	if kingMoveTargets[59] {
		t.Errorf("king must not move to d8 (attacked by enemy queen)")
	}
	if kingMoveTargets[61] {
		t.Errorf("king must not move to f8 (attacked by enemy queen)")
	}
}

func TestLegalMovesEnPassantDiscoveredCheck(t *testing.T) {
	// Famous edge case: en passant capture that exposes the king to a rook check.
	// White king on e1, white pawn on d5, black pawn just moved d7-d5 (en passant target d6).
	// Black rook on a5, white king on e1, so the e-file is not the pin. Use the classic:
	// "8/8/8/8/k2Pp2R/8/8/4K3 b - d3 0 1" — black pawn on e4, white pawn just moved d2-d4,
	// black king on a4, white rook on h4. If black plays exd3 e.p., both pawns leave rank 4,
	// exposing the king to the rook → illegal.
	loadFEN(t, "8/8/8/k2Pp2R/8/8/8/4K3 b - d6 0 1")
	moves := Game.LegalMovesSlice()

	for _, m := range moves {
		if m.From == 36 && m.To == 29 { // e5 capturing d6 e.p. (indices depend on orientation)
			t.Errorf("en passant exposing king to rook check should be illegal")
		}
	}
}

func TestIsInCheck(t *testing.T) {
	// White king e1, black rook e2, black king e8. White is in check.
	loadFEN(t, "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1")
	if !Game.IsInCheck(types.ColorWhite) {
		t.Error("expected white to be in check")
	}
	if Game.IsInCheck(types.ColorBlack) {
		t.Error("black should not be in check")
	}
}

func TestIsInCheckNoCheck(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if Game.IsInCheck(types.ColorWhite) {
		t.Error("white should not be in check at the start")
	}
	if Game.IsInCheck(types.ColorBlack) {
		t.Error("black should not be in check at the start")
	}
}

func TestFindKing(t *testing.T) {
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if Game.FindKing(types.ColorWhite) != 4 {
		t.Errorf("expected white king on e1 (idx 4), got %d", Game.FindKing(types.ColorWhite))
	}
	if Game.FindKing(types.ColorBlack) != 60 {
		t.Errorf("expected black king on e8 (idx 60), got %d", Game.FindKing(types.ColorBlack))
	}
}

func TestIsSquareAttackedByPawn(t *testing.T) {
	// White pawn on e4 (idx 28) attacks d5 (35) and f5 (37).
	loadFEN(t, "8/8/8/8/4P3/8/8/8 w - - 0 1")
	if !Game.IsSquareAttacked(35, types.ColorWhite) {
		t.Error("white pawn on e4 should attack d5")
	}
	if !Game.IsSquareAttacked(37, types.ColorWhite) {
		t.Error("white pawn on e4 should attack f5")
	}
	if Game.IsSquareAttacked(36, types.ColorWhite) {
		t.Error("white pawn on e4 should NOT attack e5 (pawns don't attack forward)")
	}
}

func TestKingCheck(t *testing.T) {
	// White in check (rook on e2 attacks king on e1) → returns white king index
	loadFEN(t, "4k3/8/8/8/8/8/4r3/4K3 w - - 0 1")
	if got := KingCheck(); got != 4 {
		t.Errorf("expected KingCheck() to return white king idx 4, got %d", got)
	}

	// No check → returns -1
	loadFEN(t, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1")
	if got := KingCheck(); got != -1 {
		t.Errorf("expected KingCheck() to return -1 when not in check, got %d", got)
	}

	// Black in check (white rook on e7 attacks king on e8), black to move
	loadFEN(t, "4k3/4R3/8/8/8/8/8/4K3 b - - 0 1")
	if got := KingCheck(); got != 60 {
		t.Errorf("expected KingCheck() to return black king idx 60, got %d", got)
	}
}