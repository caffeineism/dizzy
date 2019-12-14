package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime/pprof"
	"time"
)

var (
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")
	botGo      = flag.Int("b", -1, "run bot with specified speed in ms. 0 plays without rendering.")
	optimize   = flag.Int("o", 0, "run strategy optimization with a specified number of games per trial")
)

// 113445.006 pps

func main() {
	flag.Parse()
	switch {
	case *cpuprofile != "":
		f, err := os.Create(*cpuprofile)
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	case *botGo >= 0:
		var totalPieces int
		now := time.Now()
		for i := 0; i < 1; i++ {
			p, _ := makeAgent(testStrat, int64(i), *botGo).run()
			fmt.Println(p, "pieces")
			totalPieces += p
		}
		elapsed := time.Since(now)
		fmt.Println(elapsed, totalPieces, float64(totalPieces)/elapsed.Seconds())
	case *optimize > 0:
		ce := testStrat.newCrossEntropy(*optimize)
		ce.run()
	default:
		initRender()
	}

}

var testStrat = strategy{-1.05, -3.53, -3.69, -12.23, -5.68, -8.52, -0.84, -4.49, 4.20}
