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

// 42.1695072s 7775038 184375.83259213425

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
	return agent{
		signal:   signal{pos: defaultPos(r.Intn(numPieces)), summit: slab},
		strategy: []float64{-1.5, -.05, -2, -1, -8, -10},
		random:   r,
	}
}
