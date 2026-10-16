package search

import (
	"bytes"
	"context"
	"regexp"
	"runtime"
	"sync"
	"time"
)

// Match represents a found substring in the text.
type Match struct {
	Offset int
	Line   int
	Col    int
	Length int
}

// Result encapsulates search matches and execution metrics.
type Result struct {
	Matches  []Match
	Duration time.Duration
	Count    int
	Worker   string
}

// Engine defines search interface.
type Engine interface {
	FindAll(data []byte, query []byte) Result
	FindAllContext(ctx context.Context, data []byte, query []byte) Result
	FindRegex(data []byte, re *regexp.Regexp) Result
}

// AdaptiveSearchDispatcher dynamically routes search based on buffer size and CPU topology.
type AdaptiveSearchDispatcher struct {
	crossoverThreshold int
	workers            int
	maxStoredMatches   int
}

// NewAdaptiveSearchDispatcher creates an engine configured for this host's CPU cores,
// reserving at least 1 core for the UI rendering thread.
func NewAdaptiveSearchDispatcher() *AdaptiveSearchDispatcher {
	workers := runtime.NumCPU() - 1
	if workers < 1 {
		workers = 1
	}
	return &AdaptiveSearchDispatcher{
		crossoverThreshold: 256 * 1024, // 256 KB empirical crossover point
		workers:            workers,    // Reserve 1 CPU core for UI rendering
		maxStoredMatches:   10000,     // Cap stored match structs to prevent multi-megabyte heap spikes
	}
}

// FindAll automatically selects Sequential SIMD or Parallel Worker Pool with Background context.
func (a *AdaptiveSearchDispatcher) FindAll(data []byte, query []byte) Result {
	return a.FindAllContext(context.Background(), data, query)
}

// FindAllContext executes search respecting context cancellation and UI core reservation.
func (a *AdaptiveSearchDispatcher) FindAllContext(ctx context.Context, data []byte, query []byte) Result {
	start := time.Now()
	if len(query) == 0 || len(data) < len(query) {
		return Result{Duration: time.Since(start)}
	}

	var matches []Match
	var totalCount int
	var workerName string

	if len(data) < a.crossoverThreshold || a.workers <= 1 {
		matches, totalCount = a.sequentialSIMD(ctx, data, query)
		workerName = "SIMD-Sequential"
	} else {
		matches, totalCount = a.parallelWorkerPool(ctx, data, query)
		workerName = "Parallel-WorkerPool"
	}

	return Result{
		Matches:  matches,
		Duration: time.Since(start),
		Count:    totalCount,
		Worker:   workerName,
	}
}

// sequentialSIMD uses Go stdlib bytes.Index (AVX2/BMI2 optimized on x86_64).
func (a *AdaptiveSearchDispatcher) sequentialSIMD(ctx context.Context, data []byte, query []byte) ([]Match, int) {
	var matches []Match
	qLen := len(query)
	offset := 0
	count := 0
	loop := 0

	for {
		loop++
		if loop&0xff == 0 && ctx.Err() != nil {
			return nil, 0
		}

		idx := bytes.Index(data[offset:], query)
		if idx == -1 {
			break
		}
		abs := offset + idx
		count++
		if len(matches) < a.maxStoredMatches {
			matches = append(matches, Match{Offset: abs, Length: qLen})
		}
		offset = abs + qLen
	}
	return matches, count
}

// parallelWorkerPool distributes search across reserved CPU cores with overlap handling and cancellation.
func (a *AdaptiveSearchDispatcher) parallelWorkerPool(ctx context.Context, data []byte, query []byte) ([]Match, int) {
	numWorkers := a.workers
	dataLen := len(data)
	qLen := len(query)
	chunkSize := (dataLen + numWorkers - 1) / numWorkers
	perWorkerCap := (a.maxStoredMatches + numWorkers - 1) / numWorkers

	type workerResult struct {
		matches []Match
		count   int
	}
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup

	for w := 0; w < numWorkers; w++ {
		start := w * chunkSize
		if start >= dataLen {
			break
		}
		end := start + chunkSize
		if end > dataLen {
			end = dataLen
		}

		searchEnd := end + qLen - 1
		if searchEnd > dataLen {
			searchEnd = dataLen
		}

		workerIdx := w
		chunk := data[start:searchEnd]

		wg.Add(1)
		go func(wIdx int, chunkData []byte, baseOffset int, limit int) {
			defer wg.Done()
			m := make([]Match, 0, min(perWorkerCap, 1024))
			count := 0
			off := 0
			loopCount := 0

			for {
				loopCount++
				if loopCount&0xff == 0 && ctx.Err() != nil {
					return
				}

				idx := bytes.Index(chunkData[off:], query)
				if idx == -1 {
					break
				}
				abs := off + idx
				actualOffset := baseOffset + abs
				if actualOffset >= limit {
					break
				}

				count++
				if len(m) < perWorkerCap {
					m = append(m, Match{Offset: actualOffset, Length: qLen})
				}
				off = abs + qLen
			}
			results[wIdx] = workerResult{matches: m, count: count}
		}(workerIdx, chunk, start, end)
	}

	wg.Wait()
	if ctx.Err() != nil {
		return nil, 0
	}

	totalCount := 0
	for _, r := range results {
		totalCount += r.count
	}

	merged := make([]Match, 0, min(totalCount, a.maxStoredMatches))
	for _, r := range results {
		for _, m := range r.matches {
			if len(merged) < a.maxStoredMatches {
				merged = append(merged, m)
			}
		}
	}
	return merged, totalCount
}

// FindRegex handles regex searches.
func (a *AdaptiveSearchDispatcher) FindRegex(data []byte, re *regexp.Regexp) Result {
	start := time.Now()
	if re == nil || len(data) == 0 {
		return Result{Duration: time.Since(start)}
	}

	indices := re.FindAllIndex(data, -1)
	if indices == nil {
		return Result{Duration: time.Since(start), Worker: "Regexp"}
	}

	matches := make([]Match, len(indices))
	for i, pair := range indices {
		matches[i] = Match{
			Offset: pair[0],
			Length: pair[1] - pair[0],
		}
	}

	return Result{
		Matches:  matches,
		Duration: time.Since(start),
		Count:    len(matches),
		Worker:   "Regexp",
	}
}
