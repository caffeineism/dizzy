package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	// initRender()
	now := time.Now()
	fmt.Println(getTestAgent().run(), "pieces")
	fmt.Println(time.Since(now))
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
		weights:  []float64{-1.5, -.05, -2, -1, -8, -10},
		features: []feature{landingHeight, filledCells, rowTransitions, colTransitions, rowsWithHoles, wells3Deep},
	}
}
