package omok

const (
	BoardSize = 15
	Cells     = BoardSize * BoardSize
)

// PlaceResult is the outcome of an attempted Omok stone placement.
type PlaceResult int

const (
	// Placed: the stone was accepted and written to the returned board.
	Placed PlaceResult = iota
	// RejectedOccupied: the target cell is out of bounds or already occupied
	// (also the defensive stone==0 guard). The board is returned unchanged.
	RejectedOccupied
	// RejectedDoubleThree: the move is a renju double-three (two or more open
	// threes) by the first mover (black). The board is returned unchanged.
	RejectedDoubleThree
)

// Place attempts to set stone at (x,y). Five or more consecutive wins. The only
// forbidden move is the renju double-three, and only for the first mover (black,
// color 1): after tentatively placing, if the stone forms two or more open
// threes AND does not itself win (a five is never forbidden), the placement is
// rejected. isFirstMover selects whether the double-three rule applies. On any
// rejection the (unchanged) input board is returned.
func Place(board [Cells]byte, x uint32, y uint32, stone byte, isFirstMover bool) ([Cells]byte, PlaceResult) {
	if x >= BoardSize || y >= BoardSize || stone == 0 {
		return board, RejectedOccupied
	}
	idx := int(y)*BoardSize + int(x)
	if board[idx] != 0 {
		return board, RejectedOccupied
	}
	board[idx] = stone
	// A winning five is never forbidden; the double-three rule is black-only.
	if isFirstMover && !Wins(board, x, y) && countOpenThrees(board, x, y, stone) >= 2 {
		board[idx] = 0
		return board, RejectedDoubleThree
	}
	return board, Placed
}

func Wins(board [Cells]byte, x uint32, y uint32) bool {
	stone := board[int(y)*BoardSize+int(x)]
	if stone == 0 {
		return false
	}
	dirs := [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	for _, d := range dirs {
		run := 1
		for _, sign := range [2]int{1, -1} {
			cx, cy := int(x), int(y)
			for {
				cx += d[0] * sign
				cy += d[1] * sign
				if cx < 0 || cx >= BoardSize || cy < 0 || cy >= BoardSize || board[cy*BoardSize+cx] != stone {
					break
				}
				run++
			}
		}
		if run >= 5 {
			return true
		}
	}
	return false
}

// inBounds reports whether the cell (x,y) lies on the board.
func inBounds(x, y int) bool {
	return x >= 0 && x < BoardSize && y >= 0 && y < BoardSize
}

// countOpenThrees counts how many of the four line directions the stone at
// (x,y) participates in an open three (see isOpenThree). The stone must already
// be written to board.
func countOpenThrees(board [Cells]byte, x uint32, y uint32, stone byte) int {
	dirs := [4][2]int{{1, 0}, {0, 1}, {1, 1}, {1, -1}}
	count := 0
	for _, d := range dirs {
		if isOpenThree(board, x, y, stone, d) {
			count++
		}
	}
	return count
}

// isOpenThree reports whether the stone at (x,y) forms an open three along
// direction d: a shape where a single additional mover stone placed on some
// empty cell along d would create an open four (see isOpenFour). It scans the
// empty cells within +/-4 of (x,y) along d, hypothetically places the mover
// stone on each, and tests whether (x,y) then sits in an open four. The stone
// at (x,y) must already be written to board.
func isOpenThree(board [Cells]byte, x uint32, y uint32, stone byte, d [2]int) bool {
	for k := -4; k <= 4; k++ {
		if k == 0 {
			continue
		}
		ex := int(x) + d[0]*k
		ey := int(y) + d[1]*k
		if !inBounds(ex, ey) {
			continue
		}
		eidx := ey*BoardSize + ex
		if board[eidx] != 0 {
			continue
		}
		board[eidx] = stone // hypothetical mover stone (board is a value copy)
		open := isOpenFour(board, x, y, stone, d)
		board[eidx] = 0 // undo before trying the next empty cell
		if open {
			return true
		}
	}
	return false
}

// isOpenFour reports whether (x,y) lies in a run of EXACTLY four consecutive
// mover stones along direction d whose two bounding cells are both empty
// (in bounds and == 0). Off-board bounds count as blocking, so a run touching
// an edge is not open. The stone at (x,y) must already be written to board.
func isOpenFour(board [Cells]byte, x uint32, y uint32, stone byte, d [2]int) bool {
	run := 1
	// Forward until a non-stone cell; (fx,fy) ends on the forward bounding cell.
	fx, fy := int(x)+d[0], int(y)+d[1]
	for inBounds(fx, fy) && board[fy*BoardSize+fx] == stone {
		run++
		fx += d[0]
		fy += d[1]
	}
	// Backward until a non-stone cell; (bx,by) ends on the backward bounding cell.
	bx, by := int(x)-d[0], int(y)-d[1]
	for inBounds(bx, by) && board[by*BoardSize+bx] == stone {
		run++
		bx -= d[0]
		by -= d[1]
	}
	if run != 4 {
		return false
	}
	forwardOpen := inBounds(fx, fy) && board[fy*BoardSize+fx] == 0
	backwardOpen := inBounds(bx, by) && board[by*BoardSize+bx] == 0
	return forwardOpen && backwardOpen
}
