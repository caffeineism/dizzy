package main

import (
	"fmt"
	"strconv"
	"strings"
)

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

const strEmptyCell = "  "
const strFilledCell = "▓▓"

// print writes the board contents to stdout.
// Right-most board column corresponds with 1s bit.
func (b board) print(p pos, strat ...strategy) {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("%vy %vx", p.y, p.x))
	sb.WriteString("\n")
	sb.WriteString(" ")
	for i := 0; i < bWidth; i++ {
		sb.WriteString("__") // Top border
	}
	sb.WriteString("\n")
	for i := roof - 1; i >= slab; i-- {
		row := stringRow(b[i])
		var onPieceRow bool
		if i >= p.y && i < p.y+pieceRows {
			if pBits := p.pieceBits(i - p.y); pBits != 0 {
				onPieceRow = true
				sb.WriteString("|")
				for k := bWidth - 1; k >= 0; k-- {
					if 1<<uint64(k)&pBits != 0 {
						sb.WriteString("[]")
					} else {
						if 1<<uint64(k)&b[i] != 0 {
							sb.WriteString(strFilledCell)
						} else {
							sb.WriteString(strEmptyCell)
						}
					}
				}
				sb.WriteString("|")
			}
		}
		if !onPieceRow {
			sb.WriteString(row)
		}
		// Row labels
		sb.WriteString(" " + strconv.Itoa(i) + "\n")
	}
	sb.WriteString(" ")
	for i := 0; i < bWidth; i++ {
		sb.WriteString("‾‾") // Bottom border
	}
	sb.WriteString("\n ")
	for i := 0; i < bWidth; i++ {
		sb.WriteString(strconv.Itoa(i+1) + " ") // Column labels
	}
	sb.WriteString("\n")
	fmt.Printf(sb.String())
}

func stringRow(r uint64) string {
	var sb strings.Builder
	sb.WriteString("|") // Left side border
	for j := bWidth - 1; j >= 0; j-- {
		if 1<<uint64(j)&r != 0 {
			sb.WriteString(strFilledCell)
		} else {
			sb.WriteString(strEmptyCell)
		}
	}
	// Right side border and row labels
	sb.WriteString("|")
	return sb.String()
}

func stringPiece(r uint64) string {
	var sb strings.Builder
	emptyCell := "0"
	filledCell := "1"
	for j := 63; j >= 0; j-- {
		if 1<<uint64(j)&r != 0 {
			sb.WriteString(filledCell)
		} else {
			sb.WriteString(emptyCell)
		}
		if j%4 == 0 {
			sb.WriteString("\n")
		}
		if j%16 == 0 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
