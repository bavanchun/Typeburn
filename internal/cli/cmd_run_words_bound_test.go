package cli

import (
	"strings"
	"testing"

	"github.com/bavanchun/Typeburn/v2/internal/config"
	"github.com/bavanchun/Typeburn/v2/internal/words"
)

// TestBuildRunRequest_RejectsAnUntypableWordCount: the count sizes the target
// buffer, so an unbounded one is an out-of-memory kill dressed up as a flag.
// It is refused rather than clamped — quietly starting a 10 000-word test for
// someone who asked for two billion is not what they asked for either.
func TestBuildRunRequest_RejectsAnUntypableWordCount(t *testing.T) {
	f := runFlags{mode: "words", words: 2_000_000_000}

	_, err := buildRunRequest(testRunCmd(f, "words"), f, config.Defaults())

	if err == nil {
		t.Fatal("--words 2000000000 was accepted")
	}
	if !strings.Contains(err.Error(), "at most") {
		t.Errorf("error %q does not say what the limit is", err)
	}
	if ExitCode(err) != ExitUsage {
		t.Errorf("want usage exit, got %d", ExitCode(err))
	}
}

// TestBuildRunRequest_AcceptsTheLargestUsableWordCount pins the other side of
// the boundary, so the limit cannot drift down into counts a person might type.
func TestBuildRunRequest_AcceptsTheLargestUsableWordCount(t *testing.T) {
	f := runFlags{mode: "words", words: words.MaxWords}

	got, err := buildRunRequest(testRunCmd(f, "words"), f, config.Defaults())

	if err != nil {
		t.Fatalf("--words %d was rejected: %v", words.MaxWords, err)
	}
	if got.length != words.MaxWords {
		t.Errorf("length %d, want %d", got.length, words.MaxWords)
	}
}
