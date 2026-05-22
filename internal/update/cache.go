package update

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/bavanchun/Typeburn/internal/config"
)

const (
	cacheSchemaVersion = 1
	cacheTTL           = 24 * time.Hour
	cacheMaxAge        = 7 * 24 * time.Hour
)

// validSemverRe validates cached version strings before rendering into the TUI.
var validSemverRe = regexp.MustCompile(`^v?\d+\.\d+\.\d+([-+.][\w.-]+)?$`)

const releaseURLPrefix = "https://github.com/bavanchun/Typeburn/"

// cacheFilePath is a var so tests can override it via setCacheFilePath.
var (
	cacheFileMu   sync.Mutex
	cacheFilePath = ""
)

func getCacheFilePath() string {
	cacheFileMu.Lock()
	defer cacheFileMu.Unlock()
	return cacheFilePath
}

func setCacheFilePath(p string) {
	cacheFileMu.Lock()
	defer cacheFileMu.Unlock()
	cacheFilePath = p
}

func getCachePath() (string, error) {
	if p := getCacheFilePath(); p != "" {
		return p, nil
	}
	dir, err := config.StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "update-check.json"), nil
}

// cacheLoad reads and validates the on-disk cache.
// Returns (result, true) if the cache is present, valid, and within TTL.
// Returns (nil, false) on any error, schema mismatch, validation failure, or expiry.
func cacheLoad() (*Result, bool) {
	path, err := getCachePath()
	if err != nil {
		return nil, false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	var r Result
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, false
	}
	if r.SchemaVersion != cacheSchemaVersion {
		return nil, false
	}
	// Re-validate values before they can reach the TUI footer (injection guard).
	if r.Latest != "" && !validSemverRe.MatchString(r.Latest) {
		return nil, false
	}
	if r.ReleaseURL != "" && !strings.HasPrefix(r.ReleaseURL, releaseURLPrefix) {
		return nil, false
	}
	// Clock-skew-aware TTL: reject future-dated and wildly stale entries.
	now := time.Now().UTC()
	age := now.Sub(r.CheckedAt)
	if age < 0 || age > cacheMaxAge {
		return nil, false
	}
	if age < cacheTTL {
		return &r, true
	}
	return nil, false
}

// cacheSave atomically persists r to the update-check cache file.
// Creates the parent directory if needed. Silent-degrade: callers ignore errors.
func cacheSave(r *Result) error {
	path, err := getCachePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("update: mkdir cache dir: %w", err)
	}
	data, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("update: marshal cache: %w", err)
	}
	return atomicWriteCache(path, data)
}

// atomicWriteCache writes data to path atomically. Uses O_EXCL to prevent
// symlink-race attacks; PID suffix avoids temp-name collision on concurrent saves.
func atomicWriteCache(path string, data []byte) error {
	tmp := fmt.Sprintf("%s.%d.tmp", path, os.Getpid())

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		if os.IsExist(err) {
			_ = os.Remove(tmp)
			f, err = os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
		}
		if err != nil {
			return fmt.Errorf("update: create cache temp: %w", err)
		}
	}

	if _, err := f.Write(data); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("update: write cache temp: %w", err)
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("update: sync cache temp: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("update: close cache temp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("update: rename cache: %w", err)
	}
	return nil
}
