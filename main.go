package main

import (
	"math/rand"
	"time"
)

func main() {
	// initRender()
	strat := strategy{
		weights:  []float64{-2, -10, -.1, -1},
		features: []feature{landingHeight, coveredCells, filledCells, rowTransitions},
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	a := agent{
		signal:   signal{pos: defaultPos(r.Intn(numPieces)), summit: slab},
		strategy: strat,
		random:   r,
	}
	a.run()
}
