package main

import (
	"math"
	"math/bits"
	"time"
)

func (a agent) run() (int, int) {
	placements := make([]pos, 0, bWidth*3)
	for false == a.gameOver {
		placements = placements[:0] // Reuse slice to save allocation time
		a.pos = findBestPlacement(a.signal, a.strategy, a.findPlacements(a.piece, a.colHeights, placements))
		if a.pos == (pos{}) { // No placement found
			a.gameOver = true
			return a.totalPieces, a.totalLines
		}
		if a.speed > 0 {
			a.print()
			time.Sleep(time.Duration(a.speed) * time.Millisecond)
		}
		a = a.lockAndNewPiece()
	}
	return a.totalPieces, a.totalLines
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

func findBestPlacement(sig signal, strat strategy, placements []pos) pos {
	bestScore := math.Inf(-1)
	var bestPlacement pos
	for i := 0; i < len(placements); i++ {
		var score float64
		c := sig.lock(placements[i])
		if c.gameOver {
			continue
		}

		var heightDiffs [bWidth - 1]int
		for i := 0; i < len(c.colHeights)-1; i++ {
			heightDiffs[i] = c.colHeights[i] - c.colHeights[i+1]
		}

		// ********** START FEATURES ***********************************************
		// I originally had these neatly encapsulated as individual functions,
		// allowing findBestPlacement to cleanly iterate over them. Unfortunately,
		// this resulted in much of the execution tied up in runtime.duffcopy due
		// to having to copy over the signal struct redundantly. Passing the struct
		// to the functions as a pointer resulted in a 9% slowdown, likely because
		// it increases the work Go's garbage collector had to do. Rolling out the
		// features this way, while not ideal, speeds up execution by 26%.

		// weightedRows captures and generalizes the information contained by the
		// features "landing height" and "cleared lines." Imagine if the current
		// board's filled cells were stacked on top of all previously cleared rows.
		// Sum each row index multiplied by how many cells are filled in that row.
		weightedRows := float64(c.totalLines*(c.totalLines+1)) / 2
		for i := c.summit; i >= slab; i-- {
			filled := float64(bits.OnesCount64(c.board[i]))
			weightedRows += filled / float64(bWidth) * float64(c.totalLines+i-slab+1)
		}
		factor := float64(c.totalPieces*pieceFilledCells) / float64(bWidth)
		weightedRows -= factor * (factor + 1) / 2
		score += strat[0] * float64(weightedRows)

		// rowTransitions counts how many times a filled cell neighbors an empty cell to
		// its left or right. The left and right walls count as filled.
		// Adapted from Dellacherie's original feature.
		var rowTransitions int
		// Empty rows always contain two transitions (where the walls neighbor the
		// open playfield).
		rowTransitions += 2 * (roof - c.summit)
		// We will shift the row left once and surround it with filled wall bits.
		// Then, we can xor this with the original that has two filled bits on the
		// left border. What is left is a row with set bits in place of transitions.
		for i := c.summit; i >= slab; i-- {
			row := c.board[i]
			rowTransitions += bits.OnesCount64(((row << 1) | walledRow) ^ (row | leftBorderRow))
		}
		score += strat[1] * float64(rowTransitions)

		// colTransitions counts how many times a filled cell neighbors an empty cell
		// above or below it. Adapted from Dellacherie's original feature.
		var colTransitions int
		for i := c.summit; i >= slab; i-- {
			// xor neighboring rows. Set bits are where transitions occurred.
			colTransitions += bits.OnesCount64(c.board[i+1] ^ c.board[i])
		}
		colTransitions += bits.OnesCount64(c.board[slab] ^ filledRow) // Bottom row and floor
		score += strat[2] * float64(colTransitions)

		// rowsWithHoles counts the number of rows that have at least one covered empty
		// cell. Adapted from Thiery and Scherrer's original feature.
		var rowsWithHoles int
		var rowHoles uint64
		last := c.board[c.summit+1]
		for i := c.summit; i >= slab; i-- {
			row := c.board[i]
			rowHoles = ^row & (last | rowHoles)
			if rowHoles != 0 {
				rowsWithHoles++
			}
			last = row
		}
		score += strat[3] * float64(rowsWithHoles)

		// wells2Deep counts the number of wells with at least one empty cell
		// directly below it. A 2-deep well looks like this:
		// [filled][empty][filled]
		//         [empty]
		// This feature was inspired by Dellacherie's original feature named
		// cumulative wells, which punishes deeper wells by their triangle number
		// where n = depth. I have found that cumulative wells tends to overpunish
		// deeper wells. Counting only 2-deep and 3-deep wells seem to do the trick.
		var wells3Deep, wells2Deep int
		for i := c.summit; i >= slab+1; i-- {
			r := walledRow | c.board[i]<<1
			wells := (r >> 1) & (r << 1) &^ r &^ (c.board[i-1] << 1)
			wells2Deep += bits.OnesCount64(wells)
			if i >= slab+2 {
				wells3Deep += bits.OnesCount64(wells &^ (c.board[i-2] << 1))
			}
		}
		score += strat[4] * float64(wells2Deep)
		score += strat[5] * float64(wells3Deep)

		// holeQuota helps the bot see how "bad" its holes are, helping it to make
		// better downstacking decisions. The two main ideas are:
		// * The number of pieces required to uncover a hole is a function of how
		//   many empty cells are on the rows above that cover it.
		// * Stacking over higher up holes is more damaging than stacking over ones
		//   near the bottom. When we stack over a hole near the bottom, we may
		//   actually clear this anyway through the course of normal play before
		//   having enough pieces to get to the hole.
		//
		// holeQuota adds up empty cells on rows directly covering a hole. It
		// gives a discount for how many rows away from the hole they are. If a
		// row's empty cells have already been counted, skips them to avoid
		// double-counting.
		var quota int
		var visitedMap uint64
		for i := c.summit; i >= slab; i-- {
			// Does this row have at least one hole?
			holesOnRow := c.board[i+1] &^ c.board[i]
			if holesOnRow != 0 {
				for j := 0; j < bWidth; j++ {
					// Is there a hole on th is column?
					if holesOnRow>>j&1 != 0 {
						depth := 1
						// While row directly above hole is filled
						for c.board[i+depth]>>j&1 != 0 {
							if visitedMap>>uint64(i+depth)&1 == 0 { // If row not seen yet
								quota++ // Always punish at least one empty
								empties := bWidth - bits.OnesCount64(c.board[i+depth])
								discount := depth - 1
								if empties > discount {
									quota += empties - discount
								}
								visitedMap |= (1 << uint64(i+depth))
							}
							depth++
						}
					}
				}
			}
		}
		score += strat[6] * float64(quota)

		// safeSZ asks "if we received both S and Z simultaneously, could
		// we place them both without creating a hole and without overlapping one
		// another?" Horizontal S and Z placements are ignored. This means that the
		// surface is resilent against floods of S and Z, since the shapes
		// self-perpetuate and allow future S and Zs as long as the board's height
		// permits.
		var safeSZ int
		var sMap, zMap uint
		for i := 0; i < len(heightDiffs); i++ {
			switch heightDiffs[i] {
			case 1:
				zMap |= 1 << (i + 1)
			case -1:
				sMap |= 1 << (i + 1)
			}
		}
		if zMap != 0 && sMap != 0 {
			if (zMap<<1&^sMap == 0 && bits.OnesCount(sMap>>1&^zMap|sMap<<1&^zMap) > 2) ||
				(sMap<<1&^zMap == 0 && bits.OnesCount(zMap>>1&^sMap|zMap<<1&^sMap) > 2) ||
				(zMap<<1&^sMap != 0 && sMap<<1&^zMap != 0) ||
				(bits.OnesCount(zMap) > 1 && bits.OnesCount(sMap) > 1) {
				safeSZ = 1
			}
		}
		score += strat[7] * float64(safeSZ)

		// wellTraps counts the number of 0103 surface patterns. While these
		// patterns allow placements for S and Z, they make a 3-deep well in doing
		// so.
		var wellTraps int
		// Wall cases
		if heightDiffs[0] < 0 && heightDiffs[1] == 1 {
			wellTraps++
		}
		if heightDiffs[len(heightDiffs)-1] > 0 && heightDiffs[len(heightDiffs)-2] == -1 {
			wellTraps++
		}
		for i := 0; i < len(heightDiffs)-2; i++ {
			if heightDiffs[i] > -heightDiffs[i+1]+1 && heightDiffs[i+1] < 0 &&
				heightDiffs[i+2] == 1 {
				wellTraps++
			}
			if heightDiffs[i] == -1 && heightDiffs[i+1] > 0 &&
				heightDiffs[i+2] < -heightDiffs[i+1]-1 {
				wellTraps++
			}
		}
		score += strat[8] * float64(wellTraps)

		// ********** END FEATURES *************************************************

		if score > bestScore {
			bestScore = score
			bestPlacement = placements[i]
		}
	}
	return bestPlacement
}
