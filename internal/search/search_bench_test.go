package search

import (
	"bytes"
	"context"
	"os"
	"regexp"
	"runtime"
	"sync"
	"testing"
)

// Candidate 1: Sequential bytes.Index (Go stdlib AVX2/SIMD)
func searchSequential(data []byte, query []byte) []Match {
	if len(query) == 0 || len(data) < len(query) {
		return nil
	}
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

// Candidate 2: Parallel Worker Pool (based on runtime.NumCPU())
func searchParallelWorkerPool(data []byte, query []byte, numWorkers int) []Match {
	if len(query) == 0 || len(data) < len(query) {
		return nil
	}
	if numWorkers <= 1 || len(data) < 128*1024 {
		return searchSequential(data, query)
	}

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

		// Add overlap so matches spanning chunks are not missed
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
				// Only keep matches whose start belongs to this chunk
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

// Candidate 3: Precompiled Regexp
func searchPrecompiledRegexp(data []byte, re *regexp.Regexp) []Match {
	indices := re.FindAllIndex(data, -1)
	if indices == nil {
		return nil
	}
	matches := make([]Match, len(indices))
	for i, pair := range indices {
		matches[i] = Match{
			Offset: pair[0],
			Length: pair[1] - pair[0],
		}
	}
	return matches
}

// Benchmark fixtures
var (
	benchData64K []byte
	benchData1M  []byte
	benchData10M []byte
	benchQuery   = []byte("GopadSearchTarget")
	benchRegexp  *regexp.Regexp
)

func TestMain(m *testing.M) {
	benchRegexp = regexp.MustCompile("GopadSearchTarget")

	// Generate realistic text with matches spread throughout
	pattern := []byte("The quick brown fox jumps over the lazy dog. Gopad++ fast editor. ")
	target := []byte("Special keyword GopadSearchTarget inserted here for search. ")

	genData := func(size int) []byte {
		buf := make([]byte, 0, size)
		counter := 0
		for len(buf) < size {
			counter++
			if counter%50 == 0 {
				buf = append(buf, target...)
			} else {
				buf = append(buf, pattern...)
			}
		}
		return buf[:size]
	}

	benchData64K = genData(64 * 1024)
	benchData1M = genData(1 * 1024 * 1024)
	benchData10M = genData(10 * 1024 * 1024)

	code := m.Run()
	os.Exit(code)
}

func BenchmarkSearch_64K(b *testing.B) {
	b.SetBytes(int64(len(benchData64K)))

	b.Run("Sequential_SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchSequential(benchData64K, benchQuery)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Parallel_Workers_4", func(b *testing.B) {
		workers := runtime.NumCPU()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchParallelWorkerPool(benchData64K, benchQuery, workers)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Precompiled_Regexp", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchPrecompiledRegexp(benchData64K, benchRegexp)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})
}

func BenchmarkSearch_1M(b *testing.B) {
	b.SetBytes(int64(len(benchData1M)))

	b.Run("Sequential_SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchSequential(benchData1M, benchQuery)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Parallel_Workers_4", func(b *testing.B) {
		workers := runtime.NumCPU()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchParallelWorkerPool(benchData1M, benchQuery, workers)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Precompiled_Regexp", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchPrecompiledRegexp(benchData1M, benchRegexp)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})
}

func BenchmarkSearch_10M(b *testing.B) {
	b.SetBytes(int64(len(benchData10M)))

	b.Run("Sequential_SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchSequential(benchData10M, benchQuery)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Parallel_Workers_4", func(b *testing.B) {
		workers := runtime.NumCPU()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchParallelWorkerPool(benchData10M, benchQuery, workers)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Precompiled_Regexp", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchPrecompiledRegexp(benchData10M, benchRegexp)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})
}

// Candidate 2B: Bounded Parallel Worker Pool with UI-reserve cores & Context cancellation
func searchBoundedParallelPool(ctx context.Context, data []byte, query []byte, numWorkers int, maxStored int) ([]Match, int) {
	dataLen := len(data)
	qLen := len(query)
	if qLen == 0 || dataLen < qLen {
		return nil, 0
	}

	chunkSize := (dataLen + numWorkers - 1) / numWorkers
	type workerResult struct {
		matches []Match
		count   int
	}
	results := make([]workerResult, numWorkers)
	var wg sync.WaitGroup
	perWorkerCap := (maxStored + numWorkers - 1) / numWorkers

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
				if loopCount&0xff == 0 {
					if ctx.Err() != nil {
						return
					}
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

	merged := make([]Match, 0, min(totalCount, maxStored))
	for _, r := range results {
		for _, m := range r.matches {
			if len(merged) < maxStored {
				merged = append(merged, m)
			}
		}
	}
	return merged, totalCount
}

func BenchmarkSearch_500M(b *testing.B) {
	path := "../../scratch/test_500m.log"
	data, err := os.ReadFile(path)
	if err != nil {
		b.Skipf("skipping 500M bench; file %s not present: %v", path, err)
	}
	b.SetBytes(int64(len(data)))
	query := []byte("transactions") // ~2.45 million occurrences

	b.Run("Candidate_A_Unbounded_4Workers", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchParallelWorkerPool(data, query, 4)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Candidate_B_Bounded10K_3Workers_UIReserve", func(b *testing.B) {
		ctx := context.Background()
		workers := max(1, runtime.NumCPU()-1)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			matches, count := searchBoundedParallelPool(ctx, data, query, workers, 10000)
			if count == 0 || len(matches) == 0 {
				b.Fatal("no match")
			}
		}
	})

	b.Run("Candidate_C_Sequential_SIMD", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			res := searchSequential(data, query)
			if len(res) == 0 {
				b.Fatal("no match")
			}
		}
	})
}

