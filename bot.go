package main

import (
	"math"
	"math/bits"
)

type strategy struct {
	weights []float64
}

func (a agent) run() int {
	placements := make([]pos, 0, bWidth*3)
	for false == a.gameOver {
		placements = placements[:0] // Reuse slice to save allocation time
		a.pos = findBestPlacement(a.signal, a.strategy, a.findPlacements(a.piece, a.colHeights, placements))
		if a.pos == (pos{}) { // No placement found
			a.gameOver = true
			return a.totalPieces
		}
		// a.print()
		a = a.lockAndNewPiece()
		// time.Sleep(500 * time.Millisecond)
	}
	return a.totalPieces
}

// findPlacements returns a slice of pos of placements that can be gotten to
// without softdropping and sliding or rotating under overhangs. This method
// uses simple height subtraction in order to avoid need for collision checks.
func (b board) findPlacements(piece int, colHeights [bWidth]int, placements []pos) []pos {
	for form := 0; form < tableUsedForms[piece]; form++ {
		for x := tableXStart[piece][form]; x <= tableXStop[piece][form]; x++ {
			var landingRow int
			// Piece column pCol = 0 for rightmost column.
			for pCol := 0; pCol < formCols; pCol++ {
				depth := depths[piece][form][pCol]
				if depth == 0 {
					continue // No piece content on this column.
				}
				currentLandingRow := colHeights[bWidth-x+pCol] - depth + slab + 1
				if currentLandingRow > landingRow {
					landingRow = currentLandingRow
				}
			}
			placements = append(placements, pos{piece, form, landingRow, x})
		}
	}
	return placements
}

const (
	walledRow     = uint64(1<<(bWidth+1) | 1) // 100000000001
	leftBorderRow = uint64(3 << bWidth)       // 110000000000
)

func findBestPlacement(sig signal, strat []float64, placements []pos) pos {
	bestScore := math.Inf(-1)
	var bestPlacement pos
	for i := 0; i < len(placements); i++ {
		var score float64
		c := sig.lock(placements[i])
		if c.isGameOver() {
			continue
		}

		// Features. Originally had these neatly encapsulated as a different
		// function for each feature, allowing findBestPlacement to iterate over
		// them. Unfortunately, this resulted in much of the execution tied up in
		// runtime.duffcopy. Rolling out the features this way, while messier,
		// avoids having to copy over the signal struct redundantly and speeds
		// up execution by 26%.
		// (Tried passing the signal as a pointer as well, which slowed down the
		// program by 9%).
		//

		// landingHeight finds the row where the landing piece's top row overlaps.
		// Simplified from Dellacherie's feature, which uses (top+bottom)/2.
		score += strat[0] * float64(c.y-tableUpperEmptyRows[c.piece][c.form])

		// filledCells is analogous to cleared lines.
		var sum int
		for i := c.summit; i >= slab; i-- {
			sum += bits.OnesCount64(c.board[i])
		}
		score += strat[1] * float64(sum)

		// rowTransitions counts how many times a filled cell neighbors an empty cell to
		// its left or right. The left and right wall count as filled cells.
		// Adapted from Dellacherie's original feature.
		sum = 0
		// Empty rows always contain two transitions (where the walls neighbor the
		// open playfield).
		sum += 2 * (roof - c.summit)
		// We will shift the row left once and surround it with filled wall bits.
		// Then, we can xor this with the original that has two filled bits on the
		// left border. What is left is a row with set bits in place of transitions.
		for i := c.summit; i >= slab; i-- {
			row := c.board[i]
			sum += bits.OnesCount64(((row << 1) | walledRow) ^ (row | leftBorderRow))
		}
		score += strat[2] * float64(sum)

		// colTransitions counts how many times a filled cell neighbors an empty cell
		// above or below it. Adapted from Dellacherie's original feature.
		sum = 0
		for i := c.summit; i >= slab; i-- {
			// xor neighboring rows. Set bits are where transitions occurred.
			sum += bits.OnesCount64(c.board[i+1] ^ c.board[i])
		}
		sum += bits.OnesCount64(c.board[slab] ^ filledRow) // Bottom row and floor
		score += strat[3] * float64(sum)

		// rowsWithHoles counts the number of rows that have at least one covered empty
		// cell. Adapted from Thiery and Scherrer's original feature.
		sum = 0
		var rowHoles uint64
		last := c.board[c.summit+1]
		for i := c.summit; i >= slab; i-- {
			row := c.board[i]
			rowHoles = ^row & (last | rowHoles)
			if rowHoles != 0 {
				sum++
			}
			last = row
		}
		score += strat[4] * float64(sum)

		// wells3Deep counts the number of wells with at least one two empty cells
		// directly below it. A 3-deep well looks like this: [filled][empty][filled]
		//                                                  				 [empty]
		//                                                  				 [empty]
		// This feature was inspired by Dellacherie's original feature of cumulative
		// wells, which punishes deeper wells by their triangle number where n = depth.
		// I have found that cumulative wells tends to overpunish deeper wells and that
		// simply measuring 3-deep wells does the trick.
		sum = 0
		for i := c.summit; i >= slab+2; i-- {
			r := walledRow | c.board[i]<<1
			wells := (r >> 1) & (r << 1) &^ r &^ (c.board[i-1] << 1) &^ (c.board[i-2] << 1)
			sum += bits.OnesCount64(wells)
		}
		score += strat[5] * float64(sum)

		if score > bestScore {
			bestScore = score
			bestPlacement = placements[i]
		}
	}
	return bestPlacement
}
