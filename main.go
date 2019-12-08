package main

import (
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"runtime/pprof"
	"time"
)

var cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

// 57.0507935s 7775038 136282.731983386

func main() {
	flag.Parse()
	if *cpuprofile != "" {
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	// initRender()
	var totalPieces int
	now := time.Now()
	for i := 0; i < 1; i++ {
		p := getTestAgent(int64(i)).run()
		fmt.Println(p, "pieces")
		totalPieces += p
	}
	elapsed := time.Since(now)
	fmt.Println(elapsed, totalPieces, float64(totalPieces)/elapsed.Seconds())
}

func getTestAgent(seed int64) agent {
	r := rand.New(rand.NewSource(seed))
	// r := rand.New(rand.NewSource(time.Now().UnixNano()))
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
