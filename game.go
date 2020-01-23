package main

import (
	"math/bits"
	"math/rand"
)

// agent stores all of the player's information.
type agent struct {
	signal
	strategy
	random   *rand.Rand
	gameOver bool
	speed    int
}

// signal stores things to be considered for evaluation. The variable summit is
// the highest, non-empty row.
type signal struct {
	board
	pos
	colHeights                             [bWidth]int
	summit, lines, totalLines, totalPieces int
	gameOver                               bool
}

type strategy []float64

// pos stores the type of piece as well as it's orientation and x, y
// coordinates.
// y corresponds with which board row overlaps with a piece's bottom row.
// x corresponds with the position of which board column overlaps a piece's
// rightmost column of its four-column frame. The x coordinate is set to 0 when
// the piece's rightmost column sits on the unseen column directly to the left
// of the playfield.
// Note: x increases from left to right while column indexes (such as those in
// colHeights) start at 0 and increase from right to left.
type pos struct {
	piece, form, y, x int
}

func defaultPos(piece int) pos {
	return pos{piece, 0, initRow, initCol}
}

const formCols = 4
const formCells = 16
const pieceMask = 0b1111

// pieceBits returns the filled cells of one row of a piece, given its p position.
func (p pos) pieceBits(row int) uint64 {
	return pieces[p.piece] >> (p.form*formCells + row*formCols) & pieceMask << bWidth >> p.x
}

const filledRow = 1<<bWidth - 1
const pieceFilledCells = 4

// inBounds checks if the piece is inside the borders.
func (p pos) inBounds() bool {
	var filled int
	for i := 0; i < pieceRows; i++ {
		filled += bits.OnesCount64(p.pieceBits(i) & filledRow)
	}
	if filled != pieceFilledCells {
		// Piece escapes side boundary.
		return false
	}
	if p.y >= slab && p.y+pieceRows < roof {
		// Fast path for when piece is definitely inside top/bottom boundary.
		return true
	}
	for i := 0; i < pieceRows; i++ {
		if p.pieceBits(i) != 0 && (p.y+i > roof || p.y+i < slab) {
			// Piece escapes top or bottom.
			return false
		}
	}
	return true
}

func (p pos) move(delta int) pos {
	p.x += delta
	return p
}

func (p pos) descend(delta int) pos {
	p.y -= delta
	return p
}

const numForms = 4

func (p pos) rotate(delta int) pos {
	p.form = ((p.form+delta)%numForms + numForms) %	numForms
	return p
}

func (p pos) instantMove(delta int, b board) pos {
	for {
		p2 := p.move(delta)
		if b.allows(p2) {
			p = p2
		} else {
			return p
		}
	}
}

func (p pos) instantDescend(delta int, b board) pos {
	for {
		p2 := p.descend(delta)
		if b.allows(p2) {
			p = p2
		} else {
			return p
		}
	}
}

// updateColHeights checks and updates the highest filled row for each column.
// This method saves work by reducing the row index's upper bound.
func updateColHeights(b board, colHeights [bWidth]int, p pos, lines int) [bWidth]int {
	// Upper bound for column overlapped by piece is the topHeight or
	// old colHeight, whichever is higher (old colHeight can be higher when
	// softdropping and sliding piece underneath an overhang).
	topHeight := p.y + pieceRows - tableUpperEmptyRows[p.piece][p.form] - slab
	for i := 0; i < formCols; i++ {
		if depths[p.piece][p.form][i] != 0 && colHeights[bWidth-p.x+i] < topHeight {
			colHeights[bWidth-p.x+i] = topHeight
		}
	}
	for col := 0; col < bWidth; col++ {
		var height int
		for row := colHeights[col] + slab - lines; row >= slab; row-- {
			if b[row]>>col&1 != 0 {
				height = row - slab + 1
				break
			}
		}
		colHeights[col] = height
	}
	return colHeights
}

// lock merges the piece and updates important information
func (s signal) lock(p pos) signal {
	s.pos = p
	s.board = s.merge(s.pos)
	s.board, s.summit, s.lines = s.clearLines(s.pos, s.summit)
	s.colHeights = updateColHeights(s.board, s.colHeights, s.pos, s.lines)
	s.totalLines += s.lines
	s.totalPieces++
	s.gameOver = s.isGameOver()
	return s
}

func (a agent) lockAndNewPiece() agent {
	a.signal = a.lock(a.pos)
	a.pos = defaultPos(a.random.Intn(numPieces))
	return a
}
func makeAgent(strat strategy, seed int64, speed int) agent {
	r := rand.New(rand.NewSource(seed))
	return agent{
		signal:   signal{pos: defaultPos(r.Intn(numPieces)), summit: slab},
		strategy: strat,
		random:   r,
		speed:    speed,
	}
}
