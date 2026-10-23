package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
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

func TestCommandLineInformationAndLiteralPaths(t *testing.T) {
	for _, args := range [][]string{{"--version"}, {"--help"}} {
		var out bytes.Buffer
		_, done, err := parseCommandLine(args, &out)
		if err != nil || !done || out.Len() == 0 {
			t.Fatalf("%v: done=%v error=%v output=%q", args, done, err, out.String())
		}
	}
	files, done, err := parseCommandLine([]string{"--", "--version", "file with spaces"}, io.Discard)
	if err != nil || done || !reflect.DeepEqual(files, []string{"--version", "file with spaces"}) {
		t.Fatalf("literal paths: %v %v %v", files, done, err)
	}
}

func TestInformationModesDoNotWriteHome(t *testing.T) {
	if os.Getenv("GPAD_CLI_TEST_HELPER") == "1" {
		os.Args = []string{"gpad", os.Args[len(os.Args)-1]}
		main()
		return
	}
	for _, mode := range []string{"--version", "--help"} {
		home := t.TempDir()
		command := exec.Command(os.Args[0], "-test.run=^TestInformationModesDoNotWriteHome$", "--", mode)
		command.Env = append(os.Environ(), "GPAD_CLI_TEST_HELPER=1", "HOME="+home, "XDG_DATA_HOME="+home, "XDG_RUNTIME_DIR="+home)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("%s: %v %s", mode, err, output)
		}
		if !strings.Contains(string(output), "Gopad++") && !strings.Contains(string(output), "Usage: gpad") {
			t.Fatalf("missing output: %s", output)
		}
		entries, err := os.ReadDir(home)
		if err != nil || len(entries) != 0 {
			t.Fatalf("%s wrote HOME: %v %v", mode, entries, err)
		}
	}
}

func TestCommandLineFailuresExitNonzeroWithoutSessionWrites(t *testing.T) {
	for _, mode := range []string{"--not-a-real-flag", "--install-desktop"} {
		home := t.TempDir()
		blocker := filepath.Join(t.TempDir(), "not-directory")
		if err := os.WriteFile(blocker, nil, 0600); err != nil {
			t.Fatal(err)
		}
		command := exec.Command(os.Args[0], "-test.run=^TestInformationModesDoNotWriteHome$", "--", mode)
		command.Env = append(os.Environ(), "GPAD_CLI_TEST_HELPER=1", "HOME="+home, "XDG_DATA_HOME="+blocker, "XDG_RUNTIME_DIR="+home)
		output, err := command.CombinedOutput()
		exit, ok := err.(*exec.ExitError)
		if !ok || exit.ExitCode() != 2 {
			t.Fatalf("%s: expected exit 2, got %v %s", mode, err, output)
		}
		entries, err := os.ReadDir(home)
		if err != nil || len(entries) != 0 {
			t.Fatalf("%s touched session HOME: %v %v", mode, entries, err)
		}
	}
}
