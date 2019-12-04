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
	for _, c := range s.colHeights {
		sum += c
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
	// We take the popcount of that to get the total transitions.
	for i := s.summit; i >= 0; i-- {
		row := s.board[i]
		bits.OnesCount64(((row << 1) | walledRow) ^ (row | leftBorderRow))
	}
	return float64(sum)
}

// TODO TEST ROWTRANSITIONS
