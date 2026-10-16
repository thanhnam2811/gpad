package search

import (
	"bytes"
	"time"
)

// Match represents a single search match in the text.
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
}

// Engine defines the search contract.
type Engine interface {
	FindAll(data []byte, query []byte) Result
}

// SequentialEngine uses bytes.Index (Go SIMD optimized).
type SequentialEngine struct{}

func (s *SequentialEngine) FindAll(data []byte, query []byte) Result {
	start := time.Now()
	if len(query) == 0 || len(data) < len(query) {
		return Result{Duration: time.Since(start)}
	}

	var matches []Match
	qLen := len(query)
	offset := 0

	for {
		idx := bytes.Index(data[offset:], query)
		if idx == -1 {
			break
		}
		absoluteOffset := offset + idx
		matches = append(matches, Match{
			Offset: absoluteOffset,
			Length: qLen,
		})
		offset = absoluteOffset + qLen
	}

	return Result{
		Matches:  matches,
		Duration: time.Since(start),
		Count:    len(matches),
	}
}
