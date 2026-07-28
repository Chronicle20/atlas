package omok

import "testing"

// set writes stone directly into a copy of board at each (x,y), bypassing the
// placement rules. It is used to construct board fixtures (e.g. the surrounding
// stones of a double-three shape) without tripping the double-three rejection on
// the setup moves themselves.
func set(board [Cells]byte, stone byte, coords ...[2]uint32) [Cells]byte {
	for _, c := range coords {
		board[int(c[1])*BoardSize+int(c[0])] = stone
	}
	return board
}

// place is a small test helper: place a sequence of (x,y) stones of a given
// color onto a board as the SECOND mover (double-three rule off), failing the
// test if any placement is rejected.
func place(t *testing.T, board [Cells]byte, stone byte, coords [][2]uint32) [Cells]byte {
	t.Helper()
	for _, c := range coords {
		var res PlaceResult
		board, res = Place(board, c[0], c[1], stone, false)
		if res != Placed {
			t.Fatalf("Place(%d,%d,%d) = %d, want Placed", c[0], c[1], stone, res)
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
		if _, res := Place(board, c[0], c[1], 1, false); res != RejectedOccupied {
			t.Errorf("Place(%d,%d,1) = %d, want RejectedOccupied (out of bounds)", c[0], c[1], res)
		}
	}
}

func TestPlace_ZeroStoneRejected(t *testing.T) {
	var board [Cells]byte
	if _, res := Place(board, 7, 7, 0, false); res != RejectedOccupied {
		t.Errorf("Place(7,7,0) = %d, want RejectedOccupied (stone==0)", res)
	}
}

func TestPlace_OccupiedCellRejected(t *testing.T) {
	var board [Cells]byte
	board, res := Place(board, 3, 3, 1, false)
	if res != Placed {
		t.Fatalf("first placement = %d, want Placed", res)
	}
	if _, res := Place(board, 3, 3, 2, false); res != RejectedOccupied {
		t.Errorf("Place(3,3,2) on occupied cell = %d, want RejectedOccupied", res)
	}
}

func TestPlace_ValidPlacementSetsStone(t *testing.T) {
	var board [Cells]byte
	board, res := Place(board, 5, 6, 1, false)
	if res != Placed {
		t.Fatalf("Place(5,6,1) = %d, want Placed", res)
	}
	idx := 6*BoardSize + 5
	if board[idx] != 1 {
		t.Errorf("board[%d] = %d, want 1", idx, board[idx])
	}
}

func TestPlace_NormalMoveOnEmptyBoard(t *testing.T) {
	var board [Cells]byte
	if _, res := Place(board, 7, 7, 1, true); res != Placed {
		t.Errorf("Place(7,7,1,firstMover) on empty board = %d, want Placed", res)
	}
}

// doubleThreeBoard sets up two crossing open threes meeting at the empty center
// (7,7): a horizontal gap-three O_O at (6,7)/(8,7) and a vertical O_O at
// (7,6)/(7,8). Placing color-1 at (7,7) completes both into open threes.
func doubleThreeBoard(stone byte) [Cells]byte {
	var board [Cells]byte
	return set(board, stone,
		[2]uint32{6, 7}, [2]uint32{8, 7}, // horizontal, gap at (7,7)
		[2]uint32{7, 6}, [2]uint32{7, 8}, // vertical, gap at (7,7)
	)
}

func TestPlace_DoubleThreeRejectedForFirstMover(t *testing.T) {
	board := doubleThreeBoard(1)
	if _, res := Place(board, 7, 7, 1, true); res != RejectedDoubleThree {
		t.Errorf("Place(7,7,1,firstMover) on double-three shape = %d, want RejectedDoubleThree", res)
	}
	// The rejected stone must not be kept.
	newBoard, _ := Place(board, 7, 7, 1, true)
	if newBoard[7*BoardSize+7] != 0 {
		t.Errorf("rejected double-three left a stone at (7,7): %d, want 0", newBoard[7*BoardSize+7])
	}
}

func TestPlace_DoubleThreeAllowedForSecondMover(t *testing.T) {
	// Identical shape, but the second mover (color 2, isFirstMover=false) is not
	// bound by the black-only double-three rule.
	board := doubleThreeBoard(2)
	if _, res := Place(board, 7, 7, 2, false); res != Placed {
		t.Errorf("Place(7,7,2,secondMover) on double-three shape = %d, want Placed", res)
	}
}

func TestPlace_WinningFiveExemptFromDoubleThree(t *testing.T) {
	// Placing color-1 at (7,7) completes a horizontal FIVE (3..7,7) — a win — and
	// simultaneously forms a vertical open three (7,5)/(7,6) and a diagonal open
	// three (5,5)/(6,6). A winning move is never forbidden, even for black.
	var board [Cells]byte
	board = set(board, 1,
		[2]uint32{3, 7}, [2]uint32{4, 7}, [2]uint32{5, 7}, [2]uint32{6, 7}, // horizontal four
		[2]uint32{7, 5}, [2]uint32{7, 6}, // vertical two
		[2]uint32{5, 5}, [2]uint32{6, 6}, // down-diagonal two
	)
	nb, res := Place(board, 7, 7, 1, true)
	if res != Placed {
		t.Fatalf("Place(7,7,1,firstMover) completing a five = %d, want Placed (win exempt)", res)
	}
	if !Wins(nb, 7, 7) {
		t.Errorf("Wins(7,7) = false, want true (the completed five)")
	}
}

func TestPlace_SingleOpenThreeAllowedForFirstMover(t *testing.T) {
	// Only a single horizontal open three: placing (7,7) between (6,7)/(8,7) with
	// both far cells empty is ONE open three, below the double-three threshold.
	var board [Cells]byte
	board = set(board, 1, [2]uint32{6, 7}, [2]uint32{8, 7})
	if _, res := Place(board, 7, 7, 1, true); res != Placed {
		t.Errorf("Place(7,7,1,firstMover) on single open three = %d, want Placed", res)
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
	// No forbidden-move / overline restriction: a run of 6+ still wins, and
	// every stone in the run wins.
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
