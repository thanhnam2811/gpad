package main

import (
	"sync"
	"sync/atomic"
	"testing"
)

func TestRequestQueueStartupAndConcurrentDelivery(t *testing.T) {
	var q requestQueue
	q.add(ipcRequest{path: "startup"})
	var wakes atomic.Int64
	q.setWake(func() { wakes.Add(1) })
	if wakes.Load() != 1 {
		t.Fatal("startup request did not wake UI")
	}
	if got := q.take(); len(got) != 1 || got[0].path != "startup" {
		t.Fatalf("startup: %+v", got)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				q.add(ipcRequest{path: "file"})
			}
		}()
	}
	wg.Wait()
	if got := q.take(); len(got) != 800 {
		t.Fatalf("lost requests: %d", len(got))
	}
	if len(q.take()) != 0 {
		t.Fatal("requests delivered twice")
	}
	if wakes.Load() != 801 {
		t.Fatalf("missing wake: %d", wakes.Load())
	}
}
