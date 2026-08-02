package storage

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

// Two typeburn instances are two OS processes sharing one XDG data directory.
// They share no memory, so the race detector cannot observe this class of bug:
// the only way to assert it is to spawn real processes and count what survived.
const (
	childIDEnv  = "TYPEBURN_HISTORY_APPEND_ID"
	childNumEnv = "TYPEBURN_HISTORY_APPEND_N"

	concurrentWriters = 2
	appendsPerWriter  = 60
	appendCadence     = 2 * time.Millisecond
)

// TestHistoryAppendChild is the body of a spawned writer, not a standalone
// test. It is skipped unless the parent sets childIDEnv. The child blocks on a
// one-byte start signal from stdin so every writer begins contending at the
// same instant without the parent guessing a sleep duration.
func TestHistoryAppendChild(t *testing.T) {
	rawID := os.Getenv(childIDEnv)
	if rawID == "" {
		t.Skip("writer helper; spawned by the concurrent append test")
	}
	id, err := strconv.Atoi(rawID)
	if err != nil {
		t.Fatalf("bad %s=%q: %v", childIDEnv, rawID, err)
	}
	count, err := strconv.Atoi(os.Getenv(childNumEnv))
	if err != nil {
		t.Fatalf("bad %s: %v", childNumEnv, err)
	}

	var signal [1]byte
	if _, err := os.Stdin.Read(signal[:]); err != nil {
		t.Fatalf("writer %d: read start signal: %v", id, err)
	}

	// Keep appending after a failed write so the parent can measure how many
	// records survived overall, not just how far the first failure got.
	for i := range count {
		if _, err := AppendHistory(childRecord(id, i)); err != nil {
			t.Errorf("writer %d: append %d: %v", id, i, err)
		}
		time.Sleep(appendCadence)
	}
}

// childRecord builds a record uniquely identifying its writer (Length) and its
// position in that writer's sequence (WPM), with a distinct timestamp so no
// two records can be collapsed by sorting or capping.
func childRecord(id, seq int) Record {
	return Record{
		Time:        baseTime.Add(time.Duration(id*1000+seq) * time.Second),
		Mode:        "time",
		Length:      id,
		WPM:         seq,
		NetWPM:      float64(seq),
		RawWPM:      float64(seq),
		Accuracy:    100,
		Consistency: 100,
	}
}

// TestConcurrentAppends_NoRecordsLost asserts that every record written by
// every concurrent writer reaches disk. It asserts on the final count only —
// never on interleaving — so it stays deterministic regardless of scheduling.
func TestConcurrentAppends_NoRecordsLost(t *testing.T) {
	dir := t.TempDir()
	want := concurrentWriters * appendsPerWriter

	cmds := make([]*exec.Cmd, 0, concurrentWriters)
	starts := make([]io.WriteCloser, 0, concurrentWriters)
	logs := make([]*bytes.Buffer, 0, concurrentWriters)

	for id := 1; id <= concurrentWriters; id++ {
		cmd := exec.Command(os.Args[0], "-test.run=^TestHistoryAppendChild$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			"XDG_DATA_HOME="+dir,
			childIDEnv+"="+strconv.Itoa(id),
			childNumEnv+"="+strconv.Itoa(appendsPerWriter),
		)
		out := &bytes.Buffer{}
		cmd.Stdout = out
		cmd.Stderr = out
		start, err := cmd.StdinPipe()
		if err != nil {
			t.Fatalf("writer %d: stdin pipe: %v", id, err)
		}
		if err := cmd.Start(); err != nil {
			t.Fatalf("writer %d: start: %v", id, err)
		}
		cmds = append(cmds, cmd)
		starts = append(starts, start)
		logs = append(logs, out)
	}

	// Release every writer before waiting on any of them.
	for i, start := range starts {
		if _, err := start.Write([]byte("g")); err != nil {
			t.Fatalf("writer %d: send start signal: %v", i+1, err)
		}
		_ = start.Close()
	}
	// Count what survived before reporting write failures: the persisted total
	// is the measurement that matters, and a failing writer must not hide it.
	waitErrs := make([]error, len(cmds))
	for i, cmd := range cmds {
		waitErrs[i] = cmd.Wait()
	}

	t.Setenv("XDG_DATA_HOME", dir)
	got := LoadHistory()
	perWriter := map[int]int{}
	for _, rec := range got {
		perWriter[rec.Length]++
	}
	if len(got) != want {
		t.Errorf("persisted %d records, want %d; per writer: %v", len(got), want, perWriter)
	}
	for i, err := range waitErrs {
		if err != nil {
			t.Errorf("writer %d exited with %v:\n%s", i+1, err, logs[i].String())
		}
	}
	for id := 1; id <= concurrentWriters; id++ {
		if perWriter[id] != appendsPerWriter {
			t.Errorf("writer %d persisted %d records, want %d", id, perWriter[id], appendsPerWriter)
		}
	}
}
