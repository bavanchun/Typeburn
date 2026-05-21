package update

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func withTempCache(t *testing.T) func() {
	t.Helper()
	dir := t.TempDir()
	orig := cacheFilePath
	cacheFilePath = filepath.Join(dir, "update-check.json")
	return func() { cacheFilePath = orig }
}

func TestCacheSaveLoad_RoundTrip(t *testing.T) {
	defer withTempCache(t)()

	r := &Result{
		SchemaVersion:    cacheSchemaVersion,
		Current:          "v2.0.0",
		Latest:           "v2.1.0",
		UpgradeAvailable: true,
		ReleaseURL:       "https://github.com/bavanchun/Typeburn/releases/tag/v2.1.0",
		CheckedAt:        time.Now().UTC(),
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if !fresh {
		t.Fatal("expected fresh cache after save")
	}
	if got.Latest != "v2.1.0" {
		t.Errorf("Latest: want v2.1.0, got %q", got.Latest)
	}
	if !got.UpgradeAvailable {
		t.Error("UpgradeAvailable: want true")
	}
}

func TestCacheLoad_MissingFile(t *testing.T) {
	defer withTempCache(t)()
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected no fresh cache for missing file")
	}
}

func TestCacheLoad_CorruptFile(t *testing.T) {
	defer withTempCache(t)()
	path, _ := getCachePath()
	_ = os.MkdirAll(filepath.Dir(path), 0700)
	_ = os.WriteFile(path, []byte(`{not json`), 0600)
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected no fresh cache for corrupt file")
	}
}

func TestCacheLoad_Expired(t *testing.T) {
	defer withTempCache(t)()
	r := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "v2.1.0",
		CheckedAt:     time.Now().UTC().Add(-25 * time.Hour), // beyond 24h TTL
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected stale cache to be rejected")
	}
}

func TestCacheLoad_FutureDated(t *testing.T) {
	defer withTempCache(t)()
	r := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "v2.1.0",
		CheckedAt:     time.Now().UTC().Add(1 * time.Hour), // future
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected future-dated cache to be rejected")
	}
}

func TestCacheLoad_WrongSchema(t *testing.T) {
	defer withTempCache(t)()
	r := &Result{
		SchemaVersion: 99,
		Current:       "v2.0.0",
		Latest:        "v2.1.0",
		CheckedAt:     time.Now().UTC(),
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected wrong schema version to be rejected")
	}
}

func TestCacheLoad_InvalidLatest(t *testing.T) {
	defer withTempCache(t)()
	r := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "\x1b[31mevil\x1b[0m", // ANSI injection attempt
		CheckedAt:     time.Now().UTC(),
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected invalid Latest to be rejected")
	}
}

func TestCacheLoad_InvalidReleaseURL(t *testing.T) {
	defer withTempCache(t)()
	r := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "v2.1.0",
		ReleaseURL:    "https://evil.example.com/",
		CheckedAt:     time.Now().UTC(),
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave: %v", err)
	}
	got, fresh := cacheLoad()
	if fresh || got != nil {
		t.Error("expected invalid ReleaseURL to be rejected")
	}
}

func TestCacheSave_CreatesParentDir(t *testing.T) {
	dir := t.TempDir()
	orig := cacheFilePath
	// Use a nested path whose parent doesn't exist yet.
	cacheFilePath = filepath.Join(dir, "nested", "deep", "update-check.json")
	defer func() { cacheFilePath = orig }()

	r := &Result{
		SchemaVersion: cacheSchemaVersion,
		Current:       "v2.0.0",
		Latest:        "v2.0.0",
		CheckedAt:     time.Now().UTC(),
	}
	if err := cacheSave(r); err != nil {
		t.Fatalf("cacheSave should create parent dirs: %v", err)
	}
	if _, err := os.Stat(cacheFilePath); err != nil {
		t.Errorf("cache file not created: %v", err)
	}
}
