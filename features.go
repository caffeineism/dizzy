package main

import (
	"math/bits"
)

type feature func(s signal) float64

type strategy struct {
	features []feature
	weights  []float64
}

// landingHeight finds the row where the landing piece's top row overlaps.
// Simplified from Dellacherie's feature, which uses (top+bottom)/2.
func landingHeight(s signal) float64 {
	return float64(s.y - tableUpperEmptyRows[s.piece][s.form])
}

// coveredCells counts the number of cells a piece cannot normally access. This
// is a simplified algorithm for what is analogous to "buried holes" by
// C. Fahey: buried holes = coveredCells - filled cells.
func coveredCells(s signal) float64 {
	var sum int
	for i := 0; i < len(s.colHeights); i++ {
		sum += s.colHeights[i]
	}
	return float64(sum)
}

// filledCells is analogous to cleared lines.
func filledCells(s signal) float64 {
	var sum int
	for i := s.summit; i >= slab; i-- {
		sum += bits.OnesCount64(s.board[i])
	}
	return float64(sum)
}

const (
	walledRow     = uint64(1<<(bWidth+1) | 1) // 100000000001
	leftBorderRow = uint64(3 << bWidth)       // 110000000000
)

// rowTransitions counts how many times a filled cell neighbors an empty cell to
// its left or right. The left and right wall count as filled cells.
// Adapted from Dellacherie's original feature.
func rowTransitions(s signal) float64 {
	var sum int
	// Empty rows always contain two transitions (where the walls neighbor the
	// open playfield).
	sum += 2 * (roof - s.summit)
	// We will shift the row left once and surround it with filled wall bits.
	// Then, we can xor this with the original that has two filled bits on the
	// left border. What is left is a row with set bits in place of transitions.
	for i := s.summit; i >= slab; i-- {
		row := s.board[i]
		sum += bits.OnesCount64(((row << 1) | walledRow) ^ (row | leftBorderRow))
	}
	return float64(sum)
}

// colTransitions counts how many times a filled cell neighbors an empty cell
// above or below it. Adapted from Dellacherie's original feature.
func colTransitions(s signal) float64 {
	var sum int
	for i := s.summit; i >= slab; i-- {
		// xor neighboring rows. Set bits are where transitions occurred.
		sum += bits.OnesCount64(s.board[i+1] ^ s.board[i])
	}
	sum += bits.OnesCount64(s.board[slab] ^ filledRow) // Bottom row and floor
	return float64(sum)
}

// rowsWithHoles counts the number of rows that have at least one covered empty
// cell. Adapted from Thiery and Scherrer's original feature.
func rowsWithHoles(s signal) float64 {
	var sum int
	var rowHoles uint64
	last := s.board[s.summit+1]
	for i := s.summit; i >= slab; i-- {
		row := s.board[i]
		rowHoles = ^row & (last | rowHoles)
		if rowHoles != 0 {
			sum++
		}
		last = row
	}
	return float64(sum)
}

// wells3Deep counts the number of wells with at least one two empty cells
// directly below it. A 3-deep well looks like this: [filled][empty][filled]
//                                                  				 [empty]
//                                                  				 [empty]
// This feature was inspired by Dellacherie's original feature of cumulative
// wells, which punishes deeper wells by their triangle number where n = depth.
// I have found that cumulative wells tends to overpunish deeper wells and that
// simply measuring 3-deep wells does the trick.
func wells3Deep(s signal) float64 {
	var sum int
	for i := s.summit; i >= slab+2; i-- {
		r := walledRow | s.board[i]<<1
		wells := (r >> 1) & (r << 1) &^ r &^ (s.board[i-1] << 1) &^ (s.board[i-2] << 1)
		sum += bits.OnesCount64(wells)
	}
	return float64(sum)
}
