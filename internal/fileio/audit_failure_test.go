package fileio

import (
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

// A child process confines the Linux file-size limit to this synthetic save.
func TestAuditSaveFileWriteFailure(t *testing.T) {
	if dir := os.Getenv("GPAD_AUDIT_FSIZE_DIR"); dir != "" {
		signal.Ignore(syscall.SIGXFSZ)
		if err := syscall.Setrlimit(syscall.RLIMIT_FSIZE, &syscall.Rlimit{Cur: 1024, Max: 1024}); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "original")
		if err := SaveFile(path, []byte(strings.Repeat("x", 4096))); err == nil {
			t.Fatal("file-size limit failure hidden")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "checkpoint" {
			t.Fatalf("original changed on failed write: %q %v", data, err)
		}
		files, err := os.ReadDir(dir)
		if err != nil || len(files) != 1 {
			t.Fatalf("staging file leaked: %v %v", files, err)
		}
		return
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "original"), []byte("checkpoint"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestAuditSaveFileWriteFailure$")
	cmd.Env = append(os.Environ(), "GPAD_AUDIT_FSIZE_DIR="+dir)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("fault injection child: %v\n%s", err, output)
	}
}

func TestAuditSaveAcrossTempFilesystem(t *testing.T) {
	destination := t.TempDir()
	foreign, err := os.MkdirTemp("/dev/shm", "gpad-audit-")
	if err != nil {
		t.Skipf("isolated second filesystem unavailable: %v", err)
	}
	defer os.RemoveAll(foreign)
	var dstStat, tmpStat syscall.Stat_t
	if err := syscall.Stat(destination, &dstStat); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Stat(foreign, &tmpStat); err != nil {
		t.Fatal(err)
	}
	if dstStat.Dev == tmpStat.Dev {
		t.Skip("TMPDIR and target are on the same filesystem")
	}
	t.Setenv("TMPDIR", foreign)
	path := filepath.Join(destination, "saved")
	if err := SaveFile(path, []byte("cross filesystem save")); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "cross filesystem save" {
		t.Fatalf("save content %q %v", data, err)
	}
	files, err := os.ReadDir(foreign)
	if err != nil || len(files) != 0 {
		t.Fatalf("global temp files leaked: %v %v", files, err)
	}
}
