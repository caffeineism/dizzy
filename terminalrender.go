package main

import (
	"fmt"
	"reflect"
	"runtime"
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
	debugInserted := insertDebugInfo(pieceInserted, a.signal, a.strategy)
	sb.Reset()
	sb.WriteString(" " + strings.Repeat("__", bWidth) + "\n") // Top border
	sb.WriteString(debugInserted)
	sb.WriteString(" " + strings.Repeat("‾‾", bWidth) + "\n ") // Bottom border
	for i := 0; i < bWidth; i++ {
		sb.WriteString(strconv.Itoa(i+1) + " ") // Column labels
	}
	fmt.Printf(sb.String() + "\n")
}

func insertDebugInfo(str string, s signal, strat strategy) string {
	rows := strings.Split(str, "\n")
	rows = rows[:len(rows)-1]
	for i := 0; i < len(rows); i++ {
		rows[i] = rows[i] + " " + strconv.Itoa(roof-i) // Row label
	}
	var index int
	rows[index] = rows[index] + fmt.Sprintf("\t%vy %vx", s.y, s.x) // X, Y coordinate
	index += 3
	var score float64
	for i := 0; i < len(strat.features); i++ {
		name := string(runtime.FuncForPC(reflect.ValueOf(strat.features[i]).Pointer()).Name())[5:]
		value := strat.features[i](s.lock(s.pos))
		weightedValue := strat.weights[i] * value
		score += weightedValue
		rows[i+index] = rows[i+index] + "\t" + fmt.Sprintf("%-9.1f", weightedValue) + " " + fmt.Sprintf("%2.0f", value) + " " + name
	}
	rows[index-1] = rows[index-1] + fmt.Sprintf("\t%-12.1f score", score)
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
