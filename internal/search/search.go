package search

import (
	"bytes"
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
	FindRegex(data []byte, re *regexp.Regexp) Result
}

// AdaptiveSearchDispatcher dynamically routes search based on buffer size and CPU topology.
type AdaptiveSearchDispatcher struct {
	crossoverThreshold int
	workers            int
}

// NewAdaptiveSearchDispatcher creates an engine configured for this host's CPU cores.
func NewAdaptiveSearchDispatcher() *AdaptiveSearchDispatcher {
	return &AdaptiveSearchDispatcher{
		crossoverThreshold: 256 * 1024, // 256 KB empirical crossover point
		workers:            runtime.NumCPU(),
	}
}

// FindAll automatically selects Sequential SIMD or Parallel Worker Pool.
func (a *AdaptiveSearchDispatcher) FindAll(data []byte, query []byte) Result {
	start := time.Now()
	if len(query) == 0 || len(data) < len(query) {
		return Result{Duration: time.Since(start)}
	}

	var matches []Match
	var workerName string

	if len(data) < a.crossoverThreshold || a.workers <= 1 {
		matches = a.sequentialSIMD(data, query)
		workerName = "SIMD-Sequential"
	} else {
		matches = a.parallelWorkerPool(data, query)
		workerName = "Parallel-WorkerPool"
	}

	return Result{
		Matches:  matches,
		Duration: time.Since(start),
		Count:    len(matches),
		Worker:   workerName,
	}
}

// sequentialSIMD uses Go stdlib bytes.Index (AVX2/BMI2 optimized on x86_64).
func (a *AdaptiveSearchDispatcher) sequentialSIMD(data []byte, query []byte) []Match {
	var matches []Match
	qLen := len(query)
	offset := 0

	for {
		idx := bytes.Index(data[offset:], query)
		if idx == -1 {
			break
		}
		abs := offset + idx
		matches = append(matches, Match{Offset: abs, Length: qLen})
		offset = abs + qLen
	}
	return matches
}

// parallelWorkerPool distributes search across CPU cores with overlap handling.
func (a *AdaptiveSearchDispatcher) parallelWorkerPool(data []byte, query []byte) []Match {
	numWorkers := a.workers
	dataLen := len(data)
	qLen := len(query)
	chunkSize := (dataLen + numWorkers - 1) / numWorkers

	var wg sync.WaitGroup
	results := make([][]Match, numWorkers)

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
			var m []Match
			off := 0
			for {
				idx := bytes.Index(chunkData[off:], query)
				if idx == -1 {
					break
				}
				abs := off + idx
				actualOffset := baseOffset + abs
				if actualOffset >= limit {
					break
				}
				m = append(m, Match{Offset: actualOffset, Length: qLen})
				off = abs + qLen
			}
			results[wIdx] = m
		}(workerIdx, chunk, start, end)
	}

	wg.Wait()

	total := 0
	for _, r := range results {
		total += len(r)
	}
	merged := make([]Match, 0, total)
	for _, r := range results {
		merged = append(merged, r...)
	}
	return merged
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
