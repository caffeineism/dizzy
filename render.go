package main

import (
	"image"
	"image/color"
	"log"
	"sync"
	"time"

	"golang.org/x/exp/shiny/driver"
	"golang.org/x/exp/shiny/screen"
	"golang.org/x/mobile/event/key"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
)

const (
	yellow = iota // Color values correspond with piece type indexes
	cyan
	purple
	blue
	orange
	green
	red
	gray
	white
	black
	screenRatio = 1.777777777
)

var (
	screenHeight = 720
	screenWidth  = int(float64(screenHeight) * screenRatio)
	colors       = [10]color.RGBA{color.RGBA{183, 149, 11, 1.0},
		color.RGBA{20, 143, 119, 1.0}, color.RGBA{118, 68, 138, 1.0},
		color.RGBA{31, 97, 141, 1.0}, color.RGBA{185, 119, 14, 1.0},
		color.RGBA{30, 132, 73, 1.0}, color.RGBA{176, 58, 46, 1.0},
		color.RGBA{64, 64, 64, 1.0}, color.RGBA{255, 255, 255, 1.0},
		color.RGBA{0, 0, 0, 1.0}}
)

func initRender() {
	keySet := getKeys()
	cb := makeColorBoard()
	size := image.Point{int(screenWidth), int(screenHeight)}
	driver.Main(func(scre screen.Screen) {
		win, err := scre.NewWindow(&screen.NewWindowOptions{
			Title:  "Dizzy",
			Width:  size.X,
			Height: size.Y,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer win.Release()
		buf, err := scre.NewBuffer(size)
		if err != nil {
			log.Fatal(err)
		}
		defer buf.Release()
		renderBoard(&cb, win, buf)
		for {
			e := win.NextEvent()
			switch e := e.(type) {

			case lifecycle.Event:
				if e.To == lifecycle.StageDead {
					return
				}

			case key.Event:
				if e.Direction == 2 { // Release
					switch e.Code {
					case keySet.left:
						cb.keyStamps[keySet.left] = time.Time{} // Zero value

					case keySet.right:
						cb.keyStamps[keySet.right] = time.Time{} // Check with .IsZero()

					case keySet.up:
						cb.keyStamps[keySet.up] = time.Time{}

					case keySet.down:
						cb.keyStamps[keySet.down] = time.Time{}
					}
				} else if e.Direction == 1 { // Initial press
					// log.Print("pressed key: ", e.Code)
					switch e.Code {
					case key.CodeEscape:
						// Exit
						return

					case keySet.left:
						go moveAction(&cb, -1, buf, win, keySet.left)

					case keySet.right:
						go moveAction(&cb, 1, buf, win, keySet.right)

					case keySet.down:
						go descendAction(&cb, 1, buf, win, keySet.down)

					case keySet.up:
						go descendAction(&cb, -1, buf, win, keySet.up)

					case keySet.lock:
						cb.colorLock()

					case keySet.cw:
						p := cb.rotate(1)
						if cb.allows(p) {
							cb.pos = p
						}

					case keySet.ccw:
						p := cb.rotate(-1)
						if cb.allows(p) {
							cb.pos = p
						}
					}
				}
				renderBoard(&cb, win, buf)

			case paint.Event:
				renderBoard(&cb, win, buf)

			case error:
				log.Print(e)
			}
		}
	})
}

func moveAction(cb *colorBoard, delta int, buf screen.Buffer, win screen.Window, key key.Code) {
	cb.mu.Lock()
	stamp := time.Now()
	cb.keyStamps[key] = stamp
	cb.mu.Unlock()
	p := cb.move(delta)
	if cb.allows(p) {
		cb.pos = p
	}
	renderBoard(cb, win, buf)
	time.Sleep(delaySideInit * time.Millisecond)
	if stamp == cb.keyStamps[key] {
		cb.pos = cb.instantMove(delta, cb.board)
	}
	renderBoard(cb, win, buf)
}

func descendAction(cb *colorBoard, delta int, buf screen.Buffer, win screen.Window, key key.Code) {
	cb.mu.Lock()
	stamp := time.Now()
	cb.keyStamps[key] = stamp
	cb.mu.Unlock()
	p := cb.descend(delta)
	if cb.allows(p) {
		cb.pos = p
	}
	renderBoard(cb, win, buf)
	time.Sleep(delayVertInit * time.Millisecond)
	if stamp == cb.keyStamps[key] {
		cb.pos = cb.instantDescend(delta, cb.board)
	}
	renderBoard(cb, win, buf)
}

func drawToBuffer(img *image.RGBA, b ...*colorBoard) {
	d := img.Bounds()
	bufWidth := d.Dx()
	bufHeight := d.Dy()
	//	Each half screen's width is made up of the following parts:
	// * Board width (10 cells wide)
	// * 2 border side cells on each side of board
	// * 1/2 empty cell on right. Then 4 cells for next preview, then another 1/2 empty cell.
	// * empty 1/2  cell on left, then 4 for hold display, then empty 1/2 cell.
	// Width = 44x, where x is the width and height of each cell.
	size := int(float64(bufWidth) / 44) // Cell width and height
	sideCells := 5
	padding := (bufHeight - size*(bHeight+2)) / 2
	startX := d.Min.X + (sideCells+1)*size // Left-most pixel on board
	startY := d.Min.Y + padding + size     // Top-most pixel on board
	drawBoard(img, b[0], bufWidth, size, startX, startY)
	if len(b) > 1 {
		drawBoard(img, b[1], bufWidth, size, startX+bufWidth/2, startY)
	}
}

func drawBoard(img *image.RGBA, b *colorBoard, bufWidth, size, startX, startY int) {
	for x := startX - size; x < startX+(bWidth+1)*size; x++ {
		for y := startY - size; y < startY; y++ {
			// Top borders
			img.SetRGBA(x, y, colors[gray])
			// Bottom borders
			img.SetRGBA(x, y+(bHeight+1)*size, colors[gray])
		}
	}
	for x := startX - size; x < startX; x++ {
		for y := startY; y < startY+size*(bHeight+1); y++ {
			// Left border
			img.SetRGBA(x, y, colors[gray])
			// Right border
			img.SetRGBA(x+size*(bWidth+1), y, colors[gray])
		}
	}
	// Board
	for i := slab; i <= roof; i++ {
		for j, cell := range b.cells[i] {
			for x := startX + size*(bWidth-j-1); x < startX+size*(bWidth-j); x++ {
				r := numRows - rowsAbove - i - 1
				for y := startY + size*r; y < startY+size*(r+1); y++ {
					img.SetRGBA(x, y, colors[cell])
				}
			}
		}
	}
	// Active piece
	for i := 0; i < pieceRows; i++ {
		row := b.pieceBits(i)
		if row == 0 {
			continue
		}
		for j := bWidth - 1; j >= 0; j-- {
			if 1<<uint64(j)&row != 0 {
				for x := startX + size*(bWidth-j-1); x < startX+size*(bWidth-j); x++ {
					r := numRows - rowsAbove - (i + b.y) - 1
					for y := startY + size*r; y < startY+size*(r+1); y++ {
						img.SetRGBA(x, y, colors[b.piece])
					}
				}
			}
		}
	}
}

// colorBoard is a layer maintained outside of core game logic used for keeping
// track of the rendering/key interface separate from the bot logic.
type colorBoard struct {
	cells [][]int
	agent
	keyStamps map[key.Code]time.Time // timeStamp of last move
	mu        sync.Mutex
}

func makeColorBoard() colorBoard {
	b := make([][]int, numRows)
	for i := range b {
		b[i] = make([]int, bWidth)
		for j := range b[i] {
			b[i][j] = black
		}
	}
	return colorBoard{
		agent:     getTestAgent(0, 0),
		cells:     b,
		keyStamps: make(map[key.Code]time.Time),
	}
}

// colorMerge merges current piece into color board.
func (cb *colorBoard) colorMerge() {
	for i := 0; i < pieceRows; i++ {
		cells := cb.pieceBits(i)
		for j := 0; j < bWidth; j++ {
			if cells>>j&1 != 0 {
				cb.cells[cb.y+i][j] = cb.piece
			}
		}
	}
	// Clear lines, slow algorithm only used for colors, not internal bot logic.
	for i := len(cb.cells) - 1; i >= 0; i-- {
		var filled int
		for _, cell := range cb.cells[i] {
			if cell != black {
				filled++
			} else {
				break
			}
		}
		if filled == bWidth {
			for j := i; j < len(cb.cells)-1; j++ {
				cb.cells[j] = cb.cells[j+1]
			}
			cb.cells[len(cb.cells)-1] = make([]int, bWidth)
			for j := 0; j < bWidth; j++ {
				cb.cells[len(cb.cells)-1][j] = black
			}
		}
	}
	cb.agent = cb.lockAndNewPiece()
}

func (cb *colorBoard) colorLock() {
	cb.pos = cb.instantDescend(1, cb.board)
	cb.colorMerge()
}

type keySet struct {
	left, right, up, down, lock, cw, ccw key.Code
}

func getKeys() keySet {
	return keySet{
		left:  key.CodeD,
		right: key.CodeF,
		up:    key.CodeU,
		down:  key.CodeN,
		lock:  key.CodeJ,
		cw:    key.CodeSemicolon,
		ccw:   key.CodeK,
	}
}

func renderBoard(cb *colorBoard, win screen.Window, buf screen.Buffer) {
	cb.print()
	cb.mu.Lock()
	drawToBuffer(buf.RGBA(), cb)
	win.Upload(image.Point{}, buf, buf.Bounds())
	cb.mu.Unlock()
}
