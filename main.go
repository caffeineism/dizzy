package main

import (
	"math/rand"
	"time"
)

func main() {
	initRender()
	getTestAgent().run()
}

func getTestAgent() agent {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return agent{
		signal:   signal{pos: defaultPos(r.Intn(numPieces)), summit: slab},
		strategy: getTestStrategy(),
		random:   r,
	}
}

func getTestStrategy() strategy {
	return strategy{
		weights:  []float64{-1.5, -.05, -2, -1, -8},
		features: []feature{landingHeight, filledCells, rowTransitions, colTransitions, rowsWithHoles},
	}
}
