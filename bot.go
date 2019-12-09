package main

import (
	"math"
	"math/bits"
	"time"
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
		if a.speed > 0 {
			a.print()
			time.Sleep(time.Duration(a.speed) * time.Millisecond)
		}
		a = a.lockAndNewPiece()
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
		if c.gameOver {
			continue
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
		factor = factor * (factor + 1) / 2
		weightedRows -= factor
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
		// directly below it. A 2-deep well looks like this: [filled][empty][filled]
		//                                                  				 [empty]
		// This feature was inspired by Dellacherie's original feature of cumulative
		// wells, which punishes deeper wells by their triangle number where n =
		// depth. I have found that cumulative wells tends to overpunish deeper
		// wells and that simply measuing this way will do the trick.
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

		// holeQuota helps the bot see how "bad" its holes are, allowing it to make
		// intelligent downstacking decisions. The two main principles are:
		// * The number of pieces required to uncover a hole is a function of how
		//   many empty cells are on the rows above that cover it.
		// * Stacking over higher up holes is more damaging than stacking over ones
		//   near the bottom. When we stack over a hole near the bottom, we may
		//   actually clear this anyway through the course of normal play before
		//   even having time to clear the bottom hole.
		//
		// As a result of these observations, holeQuota adds up empty cells on the
		// rows that directly cover a hole. It also discounts these by how far away
		// from the hole they are. If a row's empties have already been counted,
		// it will skip to the next to avoid double counting.
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

		// stableSurface asks "if we received S, Z, and O all simultaneously, could
		// we fit fit them without creating a hole and without overlapping one
		// other?" Horizontal S and Z placements are ignored. The intuition behind
		// this is that if S, Z, and O all fit at once, then the board is resilient
		// against floods of these pieces. The shapes self-perpetuate, always
		// allowing future S, Z, and O's so long as the board height permits.
		var stableSurface int
		var oMap, sMap, zMap uint
		for i := 0; i < len(c.colHeights)-1; i++ {
			switch c.colHeights[i] - c.colHeights[i+1] {
			case 0:
				oMap |= 1 << i
			case 1:
				zMap |= 1 << i
			case -1:
				sMap |= 1 << i
			}
		}
		width := uint(3)
	stableLoop:
		for i, oMask := 0, width; oMask < width<<len(c.colHeights)-1; i, oMask = i+1, oMask<<1 {
			if 1<<i&oMap != 0 {
				for j, zMask := 0, width; zMask < width<<len(c.colHeights)-1; j, zMask = j+1, zMask<<1 {
					if 1<<j&zMap != 0 {
						for k, sMask := 0, width; sMask < width<<len(c.colHeights)-1; k, sMask = k+1, sMask<<1 {
							if 1<<k&sMap != 0 {
								if oMask&zMask|oMask&sMask|zMask&sMask == 0 {
									stableSurface = 1
									break stableLoop
								}
							}
						}
					}
				}
			}
		}
		score += strat[7] * float64(stableSurface)

		// ********** END FEATURES *************************************************

		if score > bestScore {
			bestScore = score
			bestPlacement = placements[i]
		}
	}
	return bestPlacement
}
