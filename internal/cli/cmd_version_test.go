package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/bavanchun/Typeburn/internal/update"
)

func stubCheck(result *update.Result, err error) func(ctx context.Context, ver string, force bool) (*update.Result, error) {
	return func(_ context.Context, _ string, _ bool) (*update.Result, error) {
		return result, err
	}
}

func versionRoot(t *testing.T, out, errOut *bytes.Buffer, args ...string) error {
	t.Helper()
	root := NewRoot(
		WithWriters(out, errOut),
		WithHomeRunner(func(context.Context, string, string) error { return nil }),
	)
	root.SetArgs(args)
	return root.Execute()
}

func TestVersionJSON_Unchanged(t *testing.T) {
	// --json without --check-update must produce the same shape as v2.0.0.
	var out bytes.Buffer
	if err := versionRoot(t, &out, &bytes.Buffer{}, "version", "--json"); err != nil {
		t.Fatalf("version --json: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
	}
	for _, key := range []string{"version", "commit", "date", "go_version", "os", "arch"} {
		if _, ok := got[key]; !ok {
			t.Errorf("missing key %q in --json output", key)
		}
	}
	if _, hasUpdate := got["update_check"]; hasUpdate {
		t.Error("update_check must NOT appear in plain --json output (backwards-compat)")
	}
}

func TestVersionCheckUpdate_UpgradeAvailable(t *testing.T) {
	orig := checkFn
	checkFn = stubCheck(&update.Result{
		Current:          "v2.0.0",
		Latest:           "v2.1.0",
		UpgradeAvailable: true,
		ReleaseURL:       "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
		CheckedAt:        time.Now().UTC(),
	}, nil)
	defer func() { checkFn = orig }()

	var out bytes.Buffer
	if err := versionRoot(t, &out, &bytes.Buffer{}, "version", "--check-update"); err != nil {
		t.Fatalf("version --check-update: %v", err)
	}
	body := out.String()
	if !strings.Contains(body, "v2.1.0") {
		t.Errorf("expected v2.1.0 in output, got:\n%s", body)
	}
	if !strings.Contains(body, "brew upgrade typeburn") {
		t.Errorf("expected upgrade hint, got:\n%s", body)
	}
}

func TestVersionCheckUpdate_UpToDate(t *testing.T) {
	orig := checkFn
	checkFn = stubCheck(&update.Result{
		Current:   "v2.0.0",
		Latest:    "v2.0.0",
		CheckedAt: time.Now().UTC(),
	}, nil)
	defer func() { checkFn = orig }()

	var out bytes.Buffer
	if err := versionRoot(t, &out, &bytes.Buffer{}, "version", "--check-update"); err != nil {
		t.Fatalf("version --check-update: %v", err)
	}
	if !strings.Contains(out.String(), "latest version") {
		t.Errorf("expected 'latest version' in output, got:\n%s", out.String())
	}
}

func TestVersionCheckUpdate_DevSkip(t *testing.T) {
	orig := checkFn
	checkFn = stubCheck(nil, nil) // nil,nil = dev skip
	defer func() { checkFn = orig }()

	var out bytes.Buffer
	if err := versionRoot(t, &out, &bytes.Buffer{}, "version", "--check-update"); err != nil {
		t.Fatalf("version --check-update: %v", err)
	}
	if !strings.Contains(out.String(), "skipped") {
		t.Errorf("expected 'skipped' in output, got:\n%s", out.String())
	}
}

func TestVersionCheckUpdate_Error(t *testing.T) {
	orig := checkFn
	checkFn = stubCheck(nil, errors.New("network unreachable"))
	defer func() { checkFn = orig }()

	var out, errOut bytes.Buffer
	// Human mode: error goes to stderr, exit code 0.
	if err := versionRoot(t, &out, &errOut, "version", "--check-update"); err != nil {
		t.Fatalf("version --check-update with error should exit 0, got: %v", err)
	}
	if !strings.Contains(errOut.String(), "could not check for updates") {
		t.Errorf("expected error on stderr, got:\n%s", errOut.String())
	}
}

func TestVersionCheckUpdate_JSONError(t *testing.T) {
	// --json --check-update with a check error must emit valid JSON only,
	// not JSON followed by the raw error string (the old double-emit bug).
	orig := checkFn
	checkFn = stubCheck(nil, errors.New("network unreachable"))
	defer func() { checkFn = orig }()

	var out, errOut bytes.Buffer
	if err := versionRoot(t, &out, &errOut, "version", "--json", "--check-update"); err != nil {
		t.Fatalf("version --json --check-update with error should exit 0, got: %v", err)
	}
	if errOut.Len() > 0 {
		t.Errorf("no output on stderr expected in --json mode, got: %s", errOut.String())
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("output must be valid JSON, got parse error: %v\noutput: %s", err, out.String())
	}
	for _, key := range []string{"version", "commit", "date", "go_version", "os", "arch"} {
		v, ok := got["version"].(map[string]any)
		if !ok {
			t.Fatalf("version key must be an object, got: %T", got["version"])
		}
		if _, ok := v[key]; !ok {
			t.Errorf("version object missing key %q", key)
		}
	}
	uc, ok := got["update_check"].(map[string]any)
	if !ok {
		t.Fatalf("expected update_check object, got: %v", got["update_check"])
	}
	if uc["error"] != "network unreachable" {
		t.Errorf("update_check.error = %q, want %q", uc["error"], "network unreachable")
	}
}

func TestVersionCheckUpdate_JSONWrapper(t *testing.T) {
	orig := checkFn
	checkFn = stubCheck(&update.Result{
		SchemaVersion:    1,
		Current:          "v2.0.0",
		Latest:           "v2.1.0",
		UpgradeAvailable: true,
		ReleaseURL:       "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
		CheckedAt:        time.Now().UTC(),
	}, nil)
	defer func() { checkFn = orig }()

	var out bytes.Buffer
	if err := versionRoot(t, &out, &bytes.Buffer{}, "version", "--json", "--check-update"); err != nil {
		t.Fatalf("version --json --check-update: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out.Bytes(), &got); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out.String())
	}
	if _, ok := got["version"]; !ok {
		t.Error("wrapper missing 'version' key")
	}
	if _, ok := got["update_check"]; !ok {
		t.Error("wrapper missing 'update_check' key")
	}
}
