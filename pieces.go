package main

import (
	"fmt"
	"strconv"
	"strings"
)

const ( // Derived from printPieceBitsValues()
	oBits = uint64(459374171485374048)
	iBits = uint64(4919057724360232704)
	tBits = uint64(343400450913207520)
	jBits = uint64(309623445149452512)
	lBits = uint64(883832423375569632)
	sBits = uint64(631630311668713152)
	zBits = uint64(344526221937478752)
)

const numPieces = 7

var pieces = [numPieces]uint64{oBits, iBits, tBits, jBits, lBits, sBits, zBits}
var tableLowerEmptyRows, tableUpperEmptyRows = getNumEmptyRows()

// getNumEmptyRows returns [piece][form]int = number of empty rows at the bottom
// or top of a piece's 4x4 bounding box.
func getNumEmptyRows() ([numPieces][numForms]int, [numPieces][numForms]int) {
	var tableLowerEmptyRows [numPieces][numForms]int
	var tableUpperEmptyRows [numPieces][numForms]int
	for i := 0; i < numPieces; i++ {
		for j := 0; j < numForms; j++ {
			p := pos{piece: i, form: j}
			var count int
			for p.pieceBits(count) == 0 {
				count++
			}
			tableLowerEmptyRows[i][j] = count
			count = 0
			index := pieceRows - 1
			for p.pieceBits(index) == 0 {
				index--
				count++
			}
			tableUpperEmptyRows[i][j] = count
		}
	}
	return tableLowerEmptyRows, tableUpperEmptyRows
}

var tableXStart, tableXStop = getXStartStop()

// getXStartStop returns [piece][form]int = leftmost and rightmost x coordinate
// a piece can reach.
func getXStartStop() ([numPieces][numForms]int, [numPieces][numForms]int) {
	var tableXStart [numPieces][numForms]int
	var tableXStop [numPieces][numForms]int
	maxX := bWidth + formCols - 1
	for i := 0; i < numPieces; i++ {
		for j := 0; j < numForms; j++ {
			for x := 0; x < maxX; x++ {
				p := pos{i, j, bHeight / 2, x}
				if p.inBounds() {
					tableXStart[i][j] = x
					break
				}
			}
			for x := maxX; x >= 0; x-- {
				p := pos{i, j, bHeight / 2, x}
				if p.inBounds() {
					tableXStop[i][j] = x
					break
				}
			}
		}
	}
	return tableXStart, tableXStop
}

// number of forms needed to consider when finding candidate placements.
var tableUsedForms = [numPieces]int{1, iszForms, numForms, numForms, numForms, iszForms, iszForms}

var depths = getDepths()

// getDepths finds how many rows up from the bottom of a piece's bounding box
// is its first filled cell. If the column is empty, its depth is 0. If the
// column's bottom-most row is filled (e.g. vertical I-piece), its depth is 1.
// The right-most column has an index of 0.
func getDepths() [numPieces][numForms][formCols]int {
	var table [numPieces][numForms][formCols]int
	for piece := 0; piece < numPieces; piece++ {
		for form := 0; form < numForms; form++ {
			for col := 0; col < formCols; col++ {
				for row := 0; row < pieceRows; row++ {
					if pieces[piece]>>(form*formCells+row*formCols)>>col&1 != 0 {
						table[piece][form][col] = row + 1
						break
					}
				}
			}
		}
	}
	return table
}

// Each piece is encoded within a 64-bit unsigned integer, 16 bits for each of
// the four orientations, within containing four rows of four bits of piece
// content. This function is never called. It is only called outside normal
// execution in order to derive the values. These are then inserted into the
// code as constant literal values.
func printPieceBitsValues() {
	var oBits uint64
	var index uint64
	// O-piece
	// 0000   row 0 (top)
	// 0110   row 1
	// 0110		row 2
	// 0000		row 3 (bottom)
	for i := 0; i < 4; i++ {
		index += 4
		oBits |= 6 << index
		index += 4
		oBits |= 6 << index
		index += 8
	}

	var iBits uint64
	index = 0
	// I-piece
	// 0000
	// 1111
	// 0000
	// 0000
	index += 8
	iBits |= 15 << index
	index += 4

	// 0010
	// 0010
	// 0010
	// 0010
	for i := 0; i < 4; i++ {
		index += 4
		iBits |= 2 << index
	}

	// 0000
	// 0000
	// 1111
	// 0000
	index += 8
	iBits |= 15 << index
	index += 8

	// 0100
	// 0100
	// 0100
	// 0100
	for i := 0; i < 4; i++ {
		index += 4
		iBits |= 4 << index
	}

	var tBits uint64
	index = 0
	// T-piece
	// 0000
	// 0100
	// 1110
	// 0000
	index += 4
	tBits |= 14 << index
	index += 4
	tBits |= 4 << index
	index += 4

	// 0000
	// 0100
	// 0110
	// 0100
	index += 4
	tBits |= 4 << index
	index += 4
	tBits |= 6 << index
	index += 4
	tBits |= 4 << index
	index += 4

	// // 0000
	// // 0000
	// // 1110
	// // 0100
	index += 4
	tBits |= 4 << index
	index += 4
	tBits |= 14 << index
	index += 8

	// // 0000
	// // 0100
	// // 1100
	// // 0100
	index += 4
	tBits |= 4 << index
	index += 4
	tBits |= 12 << index
	index += 4
	tBits |= 4 << index

	var jBits uint64
	index = 0
	// J-piece
	// 0000
	// 1000
	// 1110
	// 0000
	index += 4
	jBits |= 14 << index
	index += 4
	jBits |= 8 << index
	index += 4

	// 0000
	// 0110
	// 0100
	// 0100
	index += 4
	jBits |= 4 << index
	index += 4
	jBits |= 4 << index
	index += 4
	jBits |= 6 << index
	index += 4

	// 0000
	// 0000
	// 1110
	// 0010
	index += 4
	jBits |= 2 << index
	index += 4
	jBits |= 14 << index
	index += 8

	// 0000
	// 0100
	// 0100
	// 1100
	index += 4
	jBits |= 12 << index
	index += 4
	jBits |= 4 << index
	index += 4
	jBits |= 4 << index

	var lBits uint64
	index = 0
	// L-piece
	// 0000
	// 0010
	// 1110
	// 0000
	index += 4
	lBits |= 14 << index
	index += 4
	lBits |= 2 << index
	index += 4

	// 0000
	// 0100
	// 0100
	// 0110
	index += 4
	lBits |= 6 << index
	index += 4
	lBits |= 4 << index
	index += 4
	lBits |= 4 << index
	index += 4

	// 0000
	// 0000
	// 1110
	// 1000
	index += 4
	lBits |= 8 << index
	index += 4
	lBits |= 14 << index
	index += 8

	// 0000
	// 1100
	// 0100
	// 0100
	index += 4
	lBits |= 4 << index
	index += 4
	lBits |= 4 << index
	index += 4
	lBits |= 12 << index
	index += 4

	var sBits uint64
	index = 0
	// S-piece
	// 0000
	// 0110
	// 1100
	// 0000
	index += 4
	sBits |= 12 << index
	index += 4
	sBits |= 6 << index
	index += 8

	// 0000
	// 0100
	// 0110
	// 0010
	sBits |= 2 << index
	index += 4
	sBits |= 6 << index
	index += 4
	sBits |= 4 << index
	index += 8

	// 0000
	// 0000
	// 0110
	// 1100
	sBits |= 12 << index
	index += 4
	sBits |= 6 << index
	index += 12

	// 0000
	// 1000
	// 1100
	// 0100
	sBits |= 4 << index
	index += 4
	sBits |= 12 << index
	index += 4
	sBits |= 8 << index

	var zBits uint64
	index = 0
	// Z-piece
	// 0000
	// 1100
	// 0110
	// 0000
	index += 4
	zBits |= 6 << index
	index += 4
	zBits |= 12 << index
	index += 8

	// 0000
	// 0010
	// 0110
	// 0100
	zBits |= 4 << index
	index += 4
	zBits |= 6 << index
	index += 4
	zBits |= 2 << index
	index += 8

	// 0000
	// 0000
	// 1100
	// 0110
	zBits |= 6 << index
	index += 4
	zBits |= 12 << index
	index += 12

	// 0000
	// 0100
	// 1100
	// 1000
	zBits |= 8 << index
	index += 4
	zBits |= 12 << index
	index += 4
	zBits |= 4 << index

	var sb strings.Builder
	sb.WriteString("const (\n")
	sb.WriteString(" oBits = uint64(" + strconv.FormatUint(uint64(oBits), 10) + ")\n")
	sb.WriteString(" iBits = uint64(" + strconv.FormatUint(uint64(iBits), 10) + ")\n")
	sb.WriteString(" tBits = uint64(" + strconv.FormatUint(uint64(tBits), 10) + ")\n")
	sb.WriteString(" jBits = uint64(" + strconv.FormatUint(uint64(jBits), 10) + ")\n")
	sb.WriteString(" lBits = uint64(" + strconv.FormatUint(uint64(lBits), 10) + ")\n")
	sb.WriteString(" sBits = uint64(" + strconv.FormatUint(uint64(sBits), 10) + ")\n")
	sb.WriteString(" zBits = uint64(" + strconv.FormatUint(uint64(zBits), 10) + ")\n")
	sb.WriteString(")")
	fmt.Println(sb.String())
	// const (
	// 	oBits = uint64(459374171485374048)
	// 	iBits = uint64(4919057724360232704)
	// 	tBits = uint64(343400450913207520)
	// 	jBits = uint64(309623445149452512)
	// 	lBits = uint64(883832423375569632)
	// 	sBits = uint64(631630311668713152)
	// 	zBits = uint64(344526221937478752)
	//  )
}
