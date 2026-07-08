package omok

import "testing"

// place is a small test helper: place a sequence of (x,y) stones of a given
// color onto a board, failing the test if any placement is rejected.
func place(t *testing.T, board [Cells]byte, stone byte, coords [][2]uint32) [Cells]byte {
	t.Helper()
	for _, c := range coords {
		var ok bool
		board, ok = Place(board, c[0], c[1], stone)
		if !ok {
			t.Fatalf("Place(%d,%d,%d) rejected unexpectedly", c[0], c[1], stone)
		}
	}
	return board
}

func TestPlace_OutOfBounds(t *testing.T) {
	var board [Cells]byte
	cases := [][2]uint32{
		{BoardSize, 0},
		{0, BoardSize},
		{BoardSize, BoardSize},
		{100, 0},
		{0, 100},
	}
	for _, c := range cases {
		if _, ok := Place(board, c[0], c[1], 1); ok {
			t.Errorf("Place(%d,%d,1) = ok, want rejected (out of bounds)", c[0], c[1])
		}
	}
}

func TestPlace_ZeroStoneRejected(t *testing.T) {
	var board [Cells]byte
	if _, ok := Place(board, 7, 7, 0); ok {
		t.Errorf("Place(7,7,0) = ok, want rejected (stone==0)")
	}
}

func TestPlace_OccupiedCellRejected(t *testing.T) {
	var board [Cells]byte
	board, ok := Place(board, 3, 3, 1)
	if !ok {
		t.Fatalf("first placement unexpectedly rejected")
	}
	if _, ok := Place(board, 3, 3, 2); ok {
		t.Errorf("Place(3,3,2) on occupied cell = ok, want rejected")
	}
}

func TestPlace_ValidPlacementSetsStone(t *testing.T) {
	var board [Cells]byte
	board, ok := Place(board, 5, 6, 1)
	if !ok {
		t.Fatalf("Place(5,6,1) rejected, want accepted")
	}
	idx := 6*BoardSize + 5
	if board[idx] != 1 {
		t.Errorf("board[%d] = %d, want 1", idx, board[idx])
	}
}

func TestWins_Horizontal(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{3, 7}, {4, 7}, {5, 7}, {6, 7}, {7, 7}})
	if !Wins(board, 5, 7) {
		t.Errorf("Wins(5,7) = false, want true (horizontal 5-run)")
	}
}

func TestWins_Vertical(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{7, 3}, {7, 4}, {7, 5}, {7, 6}, {7, 7}})
	if !Wins(board, 7, 5) {
		t.Errorf("Wins(7,5) = false, want true (vertical 5-run)")
	}
}

func TestWins_DiagonalDown(t *testing.T) {
	// (x,y) increasing together: top-left to bottom-right diagonal.
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{2, 2}, {3, 3}, {4, 4}, {5, 5}, {6, 6}})
	if !Wins(board, 4, 4) {
		t.Errorf("Wins(4,4) = false, want true (down-diagonal 5-run)")
	}
}

func TestWins_DiagonalUp(t *testing.T) {
	// x increasing, y decreasing: bottom-left to top-right diagonal.
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{2, 8}, {3, 7}, {4, 6}, {5, 5}, {6, 4}})
	if !Wins(board, 4, 6) {
		t.Errorf("Wins(4,6) = false, want true (up-diagonal 5-run)")
	}
}

func TestWins_CornerTopLeft(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{0, 0}, {1, 0}, {2, 0}, {3, 0}, {4, 0}})
	if !Wins(board, 0, 0) {
		t.Errorf("Wins(0,0) = false, want true (horizontal run starting at corner)")
	}
}

func TestWins_CornerBottomRight(t *testing.T) {
	last := uint32(BoardSize - 1)
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{
		{last, last}, {last - 1, last}, {last - 2, last}, {last - 3, last}, {last - 4, last},
	})
	if !Wins(board, last, last) {
		t.Errorf("Wins(%d,%d) = false, want true (horizontal run ending at bottom-right corner)", last, last)
	}
}

func TestWins_EdgeTop(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{5, 0}, {6, 0}, {7, 0}, {8, 0}, {9, 0}})
	if !Wins(board, 7, 0) {
		t.Errorf("Wins(7,0) = false, want true (run along top edge)")
	}
}

func TestWins_EdgeBottom(t *testing.T) {
	last := uint32(BoardSize - 1)
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{5, last}, {6, last}, {7, last}, {8, last}, {9, last}})
	if !Wins(board, 7, last) {
		t.Errorf("Wins(7,%d) = false, want true (run along bottom edge)", last)
	}
}

func TestWins_EdgeLeft(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{0, 5}, {0, 6}, {0, 7}, {0, 8}, {0, 9}})
	if !Wins(board, 0, 7) {
		t.Errorf("Wins(0,7) = false, want true (run along left edge)")
	}
}

func TestWins_EdgeRight(t *testing.T) {
	last := uint32(BoardSize - 1)
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{last, 5}, {last, 6}, {last, 7}, {last, 8}, {last, 9}})
	if !Wins(board, last, 7) {
		t.Errorf("Wins(%d,7) = false, want true (run along right edge)", last)
	}
}

func TestWins_RunOfSixOverlineAllowed(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{2, 7}, {3, 7}, {4, 7}, {5, 7}, {6, 7}, {7, 7}})
	// Cosmic's searchCombo/searchCombo2 has no forbidden-move / overline
	// restriction: a run of 6+ still wins, and every stone in the run wins.
	if !Wins(board, 2, 7) {
		t.Errorf("Wins(2,7) = false, want true (overline run of 6, checked at one end)")
	}
	if !Wins(board, 7, 7) {
		t.Errorf("Wins(7,7) = false, want true (overline run of 6, checked at other end)")
	}
	if !Wins(board, 4, 7) {
		t.Errorf("Wins(4,7) = false, want true (overline run of 6, checked in the middle)")
	}
}

func TestWins_FourInARowDoesNotWin(t *testing.T) {
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{3, 7}, {4, 7}, {5, 7}, {6, 7}})
	if Wins(board, 5, 7) {
		t.Errorf("Wins(5,7) = true, want false (only 4 in a row)")
	}
}

func TestWins_BrokenRunDoesNotWin(t *testing.T) {
	// A gap at x=5 breaks what would otherwise be a 5-run into 2+3.
	var board [Cells]byte
	board = place(t, board, 1, [][2]uint32{{2, 7}, {3, 7}, {4, 7}, {6, 7}, {7, 7}})
	if Wins(board, 4, 7) {
		t.Errorf("Wins(4,7) = true, want false (run broken by gap at x=5)")
	}
	if Wins(board, 6, 7) {
		t.Errorf("Wins(6,7) = true, want false (run broken by gap at x=5)")
	}
}

func TestWins_EmptyCellDoesNotWin(t *testing.T) {
	var board [Cells]byte
	if Wins(board, 7, 7) {
		t.Errorf("Wins(7,7) on empty board = true, want false")
	}
}

func TestWins_DoesNotCountOpponentStones(t *testing.T) {
	var board [Cells]byte
	// Four of stone 1, flanked by stone 2 on both sides — must not count
	// through a differently-colored stone.
	board = place(t, board, 2, [][2]uint32{{2, 7}, {7, 7}})
	board = place(t, board, 1, [][2]uint32{{3, 7}, {4, 7}, {5, 7}, {6, 7}})
	if Wins(board, 4, 7) {
		t.Errorf("Wins(4,7) = true, want false (only 4-run of stone 1, flanked by opponent stones)")
	}
}
