package search

import (
	"bytes"
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
