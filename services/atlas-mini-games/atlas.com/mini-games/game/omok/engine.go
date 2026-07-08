package omok

const BoardSize = 15
const Cells = BoardSize * BoardSize

// Semantics mirror Cosmic MiniGame.searchCombo/searchCombo2 (<cosmic>/src/main/java/server/maps/MiniGame.java:431-516):
// only rule is empty-cell; five or more consecutive wins; no forbidden moves.
func Place(board [Cells]byte, x uint32, y uint32, stone byte) ([Cells]byte, bool) {
	if x >= BoardSize || y >= BoardSize || stone == 0 {
		return board, false
	}
	idx := int(y)*BoardSize + int(x)
	if board[idx] != 0 {
		return board, false
	}
	board[idx] = stone
	return board, true
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
