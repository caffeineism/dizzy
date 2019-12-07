package main

// The length of a board slice is equal to height + 5 above + 3 below
// Example indexes of a board 10 row height:
// 17
// 16
// 15
// 14
// 13
// 12 <- Top-most reachable row (roof)
// 11
// 10
// 9
// 8
// 7
// 6
// 5
// 4
// 3 <- Bottom-most reachable row (slab)
// 2
// 1
// 0

// Some definitions useful for dealing with and iterating through a board.
// Some unused rows above and below the reachable playfield let the piece
// avoid going into out of bounds indexes.
const (
	rowsAbove = 5                          // Number of hidden rows above board.
	slab      = 3                          // Hidden rows below and index of bottom-most reachable row.
	numRows   = bHeight + slab + rowsAbove // Total board rows
	roof      = slab + bHeight - 1         // Top-most reachable row
)

type board [numRows]uint64

const pieceRows = 4

// merge inserts piece content at pos coordinates and returns new board.
func (b board) merge(p pos) board {
	for i := 0; i < pieceRows; i++ {
		b[p.y+i] |= p.pieceBits(i)
	}
	return b
}

// collides checks if a piece position overlaps filled cells on the board.
func (b board) collides(p pos) bool {
	return p.pieceBits(0)&b[p.y]|
		p.pieceBits(1)&b[p.y+1]|
		p.pieceBits(2)&b[p.y+2]|
		p.pieceBits(3)&b[p.y+3] != 0
}

// allows checks if position does not collide and is not out of bounds.
func (b board) allows(p pos) bool {
	return p.inBounds() && !b.collides(p)
}

// clearLines removes filled rows from the board as well as updates summit.
func (b board) clearLines(p pos, summit int) (board, int, int) {
	landingTopRow := p.y + pieceRows - tableUpperEmptyRows[p.piece][p.form] - 1
	if landingTopRow > summit {
		summit = landingTopRow
	}
	var lines int
	// If we iterate over the landing piece's rows and don't see any filled rows,
	// then we're done. Otherwise, we iterate to the highest non-empty row,
	// bringing its contents down as we go along.
	// Start at piece's bottom-most filled row.
	row := p.y + tableLowerEmptyRows[p.piece][p.form]
	for row <= landingTopRow || (lines > 0 && row <= summit) {
		// if the row we want to copy is filled, then we'll skip over it by using
		// the variable "lines" as an offset.
		for b[row+lines] == filledRow {
			lines++
			// Each line cleared means one less row at the top to work on.
			summit--
		}
		b[row] = b[row+lines]
		row++
	}
	// Clear the leftover rows at the top of the stack.
	for i := 0; i < lines; i++ {
		b[summit+i+1] = 0
	}
	return b, summit, lines
}

func (b board) isGameOver() bool {
	return b[roof+1] != 0
}
