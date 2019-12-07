package main

import (
	"math"
)

func (a agent) run() int {
	// a.print()
	for false == a.gameOver {
		a.pos = a.findBestPlacement(a.findPlacements(a.piece, a.colHeights), a.signal)
		if a.pos == (pos{}) { // No placement found
			a.gameOver = true
			return a.totalPieces
		}
		// a.print()
		a = a.lockAndNewPiece()
		// time.Sleep(1 * time.Millisecond)
	}
	return a.totalPieces
}

// findPlacements returns a slice of pos of placements that can be gotten to
// without softdropping and sliding or rotating under overhangs. This method
// uses simple height subtraction in order to avoid need for collision checks.
func (b board) findPlacements(piece int, colHeights [bWidth]int) []pos {
	var placements []pos
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

func (strat strategy) findBestPlacement(placements []pos, sig signal) pos {
	bestScore := math.Inf(-1)
	var bestPlacement pos
	for _, p := range placements {
		var score float64
		candidate := sig.lock(p)
		if candidate.isGameOver() {
			continue
		}
		for i := 0; i < len(strat.features); i++ {
			score += strat.weights[i] * strat.features[i](candidate)
		}
		if score > bestScore {
			bestScore = score
			bestPlacement = p
		}
	}
	return bestPlacement
}
