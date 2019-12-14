package main

import (
	"fmt"
	"math/bits"
	"strconv"
	"strings"
)

const (
	strEmptyCell  = "  "
	strFilledCell = "@@"
	strPieceCell  = "[]"
)

// print writes the board contents to stdout.
// Right-most board column corresponds with 1s bit.
func (a agent) print() {
	var sb strings.Builder
	for i := roof; i >= slab; i-- {
		row := stringRow(a.board[i])
		sb.WriteString(row + "\n")
	}
	pieceInserted := insertPieceInStr(sb.String(), a.pos)
	debugInserted := insertDebugInfo(pieceInserted, a.signal)
	sb.Reset()
	sb.WriteString(" " + strings.Repeat("__", bWidth) + "\n") // Top border
	sb.WriteString(debugInserted)
	sb.WriteString(" " + strings.Repeat("‾‾", bWidth) + "\n ") // Bottom border
	for i := 0; i < bWidth; i++ {
		sb.WriteString(strconv.Itoa(i+1) + " ") // Column labels
	}
	fmt.Printf(sb.String() + "\n")
}

func insertDebugInfo(str string, s signal) string {
	rows := strings.Split(str, "\n")
	rows = rows[:len(rows)-1]
	for i := 0; i < len(rows); i++ {
		rows[i] = rows[i] + " " + strconv.Itoa(roof-i) // Row label
	}
	var index int
	c := s.lock(s.pos)
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%2dy %2dx, %d pieces, %d lines",
		s.y, s.x, c.totalPieces, c.totalLines)

	var heightDiffs [bWidth - 1]int
	for i := 0; i < len(c.colHeights)-1; i++ {
		heightDiffs[i] = c.colHeights[i] - c.colHeights[i+1]
	}

	var weightedRows float64
	psuedoLines := float64(c.totalPieces*pieceFilledCells)/float64(bWidth) - float64(c.totalLines)
	for i := c.summit; i >= slab; i-- {
		filled := float64(bits.OnesCount64(c.board[i]))
		weightedRows += filled / float64(bWidth) * (float64(i-slab+1) + psuedoLines)
	}
	weightedRows -= psuedoLines * (psuedoLines + 1) / 2
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%2.2f weighted rows", weightedRows)

	var rowTransitions int
	// We will shift the row left once and surround it with filled wall bits.
	// Then, we can xor this with the original that has two filled bits on the
	// left border. What is left is a row with set bits in place of transitions.
	for i := c.summit; i >= slab; i-- {
		row := c.board[i]
		rowTransitions += bits.OnesCount64(((row<<1)|walledRow)^(row|leftBorderRow)) - 2
	}
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d row transitions", rowTransitions)

	var colTransitions int
	for i := c.summit; i >= slab; i-- {
		// xor neighboring rows. Set bits are where transitions occurred.
		colTransitions += bits.OnesCount64(c.board[i+1] ^ c.board[i])
	}
	colTransitions += bits.OnesCount64(c.board[slab] ^ filledRow) // Bottom row and floor
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d col transitions", colTransitions)

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
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d rows with holes", rowsWithHoles)

	var wells3Deep, wells2Deep int
	for i := c.summit; i >= slab+1; i-- {
		r := walledRow | c.board[i]<<1
		wells := (r >> 1) & (r << 1) &^ r &^ (c.board[i-1] << 1)
		wells2Deep += bits.OnesCount64(wells)
		if i >= slab+2 {
			wells3Deep += bits.OnesCount64(wells &^ (c.board[i-2] << 1))
		}
	}
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d wells 2-Deep", wells2Deep)
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d wells 3-Deep", wells3Deep)

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
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d hole quota", quota)

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
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d well traps", wellTraps)

	var safeSZ, safeO int
	var sMap, zMap uint
	for i := 0; i < len(heightDiffs); i++ {
		switch heightDiffs[i] {
		case 1:
			zMap |= 1 << (i + 1)
		case -1:
			sMap |= 1 << (i + 1)
		case 0:
			safeO = 1
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
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d safeSZ", safeSZ)
	index++
	rows[index%bHeight] = rows[index%bHeight] + fmt.Sprintf("\t%4d safeO", safeO)

	return strings.Join(rows, "\n") + "\n"
}

func insertPieceInStr(str string, p pos) string {
	var sb strings.Builder
	rows := strings.Split(str, "\n")
	c := len(strEmptyCell)
	for i := roof; i >= slab; i-- {
		r := rows[roof-i]
		if i >= p.y && i < p.y+pieceRows { // Are we on a row with piece?
			if pBits := p.pieceBits(i - p.y); pBits != 0 {
				for j := 0; j < bWidth; j++ {
					if 1<<uint64(bWidth-j-1)&pBits != 0 {
						r = r[:j*c+1] + strPieceCell + r[j*c+c+1:]
					}
				}
			}
		}
		sb.WriteString(r + "\n")
	}
	return sb.String()
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

func (s strategy) string() string {
	var sb strings.Builder
	for i := 0; i < len(s); i++ {
		sb.WriteString(fmt.Sprintf("% 6.2f, ", s[i]))
	}
	return sb.String()[:len(sb.String())-2]
}
