package main

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"
)

// crossEntropy implements the noisy cross entropy method to optimize weights
// on a set of features. The paper "Building Controllers for Tetris" was the
// primary source used for this. I have modified their method in the following
// ways:
// 1) L1 Regularization is used to put downward pressure on weights, suppressing
//    features that do not contribute to improvement.
//
// 2) This method updates variance by adding it to the noise factor multiplied
//    by the absolute value of the mean. This lets the noise match the scale
//    of the weight more accurately.
//
// 3) Noise decreases logarithmically with the number of iterations.
type crossEntropy struct {
	means, variances, bestStratSingle, bestStratMean     strategy
	population, iterations, cutoff, numOfGames           int
	rho, noise, bestResultSingle, bestResultMean, lambda float64
}

func (s strategy) newCrossEntropy(numOfGames int) crossEntropy {
	return crossEntropy{
		means:      s,
		variances:  initVariances(len(s), 10),
		population: 100,
		noise:      0.03,
		rho:        0.1,                    // Top percent of population to consider
		lambda:     0.04 / float64(len(s)), // L1 regularization constant
		numOfGames: numOfGames,
	}
}

func (ce *crossEntropy) run() {
	ce.cutoff = int(ce.rho * float64(ce.population))
	for {
		ce.iterations++
		// Get a new set of strategies
		strats := make([]strategy, ce.population)
		for i := 0; i < ce.population; i++ {
			strats[i] = ce.getStrat()
		}
		results := ce.testStrategies(strats)
		sort.Sort(sort.Reverse(results))
		ce.updateMeansAndVariances(results)
		ce.logData(results)
	}
}

// testStrats takes a slice of strategies, plays them out in parallel, and then
// returns a slice of strategy-result pairs.
func (ce *crossEntropy) testStrategies(strats []strategy) ceResultList {
	jobs := make(chan int, len(strats))
	resultChan := make(chan ceResult, len(strats))
	results := make(ceResultList, len(strats))
	for i := 0; i < len(strats); i++ {
		jobs <- i
	}
	for i := 0; i < runtime.NumCPU(); i++ {
		go ceWorker(jobs, resultChan, strats, ce.numOfGames, ce.lambda)
	}
	close(jobs)
	for i := 0; i < len(strats); i++ {
		r := <-resultChan
		results[i] = r
	}
	return results
}

// ceWorker runs with other workers, who share a pool of jobs to process games
// concurrently.
func ceWorker(jobs <-chan int, results chan<- ceResult, strats []strategy, numOfGames int, lambda float64) {
	for i := range jobs {
		var total float64
		for j := 0; j < numOfGames; j++ {
			_, lines := makeAgent(strats[i], int64(j), 0).run()
			total += float64(lines)
		}
		average := total / float64(numOfGames)
		results <- ceResult{
			strategy: strats[i],
			score:    average - l1Regularization(average, lambda, strats[i]),
			lines:    average,
		}
	}
}

// l1Regularization creates a penalty when larger values don't contribute to
// better scores. This puts downward pressure on the values and helps identify
// when a value isn't useful.
func l1Regularization(lines, lambda float64, strat strategy) float64 {
	var penalty float64
	for i := 0; i < len(strat); i++ {
		penalty += math.Abs(strat[i])
	}
	return lambda * lines * penalty
}

func (ce *crossEntropy) getStrat() strategy {
	noise := ce.noise * 1 / (math.Log10(1 + float64(ce.iterations)))
	candidate := make(strategy, len(ce.means))
	for i := 0; i < len(ce.means); i++ {
		// Generate strategy based on mean and variance and add noise
		variance := math.Abs(ce.means[i])*noise + ce.variances[i]
		candidate[i] = rand.NormFloat64()*math.Sqrt(variance) + ce.means[i]
	}
	return candidate
}

func (ce *crossEntropy) updateMeansAndVariances(results ceResultList) {
	weights := make([][]float64, len(ce.means))
	var meanLines float64
	for i := 0; i < len(ce.means); i++ {
		weights[i] = make([]float64, ce.cutoff)
		for j := 0; j < ce.cutoff; j++ {
			weights[i][j] = results[j].strategy[i]
		}
		meanLines += results[i].lines
	}
	meanLines /= float64(len(ce.means))
	for i := 0; i < len(ce.means); i++ {
		ce.means[i] = getMean(weights[i])
		ce.variances[i] = getVariance(weights[i], ce.means[i])
	}
	if meanLines > ce.bestResultMean {
		ce.bestResultMean = meanLines
		ce.bestStratMean = ce.means
	}
}

func (ce *crossEntropy) logData(results ceResultList) {
	var sb strings.Builder
	for i := 0; i < len(results); i++ {
		if results[i].lines > ce.bestResultSingle {
			ce.bestResultSingle = results[i].lines
			ce.bestStratSingle = results[i].strategy
			stars := strings.Repeat("*", 30)
			sb.WriteString(fmt.Sprintf(stars + " New Best " + stars + "\n"))
		}
	}
	strFormat := "%12.0f : "
	sb.WriteString(fmt.Sprintf(strFormat, ce.bestResultMean) + ce.bestStratMean.string() + " Best average\n")
	sb.WriteString(fmt.Sprintf(strFormat, ce.bestResultSingle) + ce.bestStratSingle.string() + " Best single\n\n")
	for i := 0; i < ce.cutoff; i++ {
		lines := results[i].lines
		sb.WriteString(fmt.Sprintf(strFormat, lines))
		sb.WriteString(results[i].strategy.string() + "\n")
	}
	t := time.Now().Format("2006-01-02 15:04:05")
	info := fmt.Sprintf("%d game(s) per trial\t %dx%d board", ce.numOfGames, bWidth, bHeight)
	sb.WriteString(fmt.Sprintf("\nIteration %d\t%s\t%s\n\n", ce.iterations, t, info))
	str := sb.String()
	fmt.Print(str)
	writeToFile(str, "ce.txt")
}

func initVariances(size int, variance float64) []float64 {
	newVari := make([]float64, size)
	for i := 0; i < len(newVari); i++ {
		newVari[i] = variance
	}
	return newVari
}

type ceResult struct {
	strategy
	score, lines float64
}

type ceResultList []ceResult

func (p ceResultList) Len() int           { return len(p) }
func (p ceResultList) Less(i, j int) bool { return p[i].score < p[j].score }
func (p ceResultList) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func getMean(data []float64) float64 {
	var sum float64
	for i := 0; i < len(data); i++ {
		sum += data[i]
	}
	return sum / float64(len(data))
}

func getVariance(data []float64, mean float64) float64 {
	var squaredDiffs float64
	for i := 0; i < len(data); i++ {
		diffs := data[i] - mean
		squaredDiffs += diffs * diffs
	}
	return squaredDiffs / float64(len(data))
}

func writeToFile(str, file string) {
	f, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE, 0600)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if _, err = f.WriteString(str); err != nil {
		panic(err)
	}
}
