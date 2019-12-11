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

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	botGo      = flag.Int("b", -1, "run bot with speed in ms. 0 plays without rendering.")
)

// 6.3934767s 900343 140822.1289052324

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
	if *botGo >= 0 {
		var totalPieces int
		now := time.Now()
		for i := 0; i < 1; i++ {
			p := getTestAgent(int64(i), *botGo).run()
			fmt.Println(p, "pieces")
			totalPieces += p
		}
		elapsed := time.Since(now)
		fmt.Println(elapsed, totalPieces, float64(totalPieces)/elapsed.Seconds())
	} else {
		initRender()
	}
}

func getTestAgent(seed int64, speed int) agent {
	r := rand.New(rand.NewSource(seed))
	return agent{
		signal:   signal{pos: defaultPos(r.Intn(numPieces)), summit: slab},
		strategy: []float64{-4, -1, -1, -10, -0.25, -5, -0.1, 1.5, -1},
		random:   r,
		speed:    speed,
	}
}
