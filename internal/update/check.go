package update

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Check returns the update status for currentVer.
// Returns (nil, nil) when currentVer is a dev/pseudo-version — no check is performed.
// When force=true the 24h cache is bypassed and a fresh fetch is made.
// On any network or parse error the caller receives a non-nil error; silent-degrade
// is the caller's responsibility (both the opportunistic startup check and the
// `--check-update` command treat errors as no-op).
func Check(ctx context.Context, currentVer string, force bool) (*Result, error) {
	v := strings.ToLower(strings.TrimSpace(currentVer))
	if v == "" || v == "dev" || v == "unknown" || strings.HasPrefix(v, "v0.0.0-") {
		return nil, nil
	}

	if !force {
		if cached, fresh := cacheLoad(); fresh {
			return cached, nil
		}
	}

	rel, err := FetchLatest(ctx, currentVer)
	if err != nil {
		return nil, err
	}

	// The tag reaches Apply, where it is interpolated into a download URL and
	// into the asset filename that is then joined onto the install directory.
	// Nothing between here and there constrains its shape, and the same guard
	// already runs on the cache-read path — apply it to the live result too so
	// the "a tag is always v1.2.3" invariant is stated rather than inferred
	// from parseSemver happening to reject everything else.
	if !validSemverRe.MatchString(rel.TagName) {
		return nil, fmt.Errorf("update: release tag %q is not a version", rel.TagName)
	}

	// Treat draft/prerelease as "no stable upgrade available".
	if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
		r := &Result{
			SchemaVersion:    cacheSchemaVersion,
			Current:          currentVer,
			Latest:           currentVer,
			UpgradeAvailable: false,
			CheckedAt:        time.Now().UTC(),
		}
		_ = cacheSave(r) // best-effort; TTL suppresses repeat network hits during prerelease windows
		return r, nil
	}

	upgrade := Compare(currentVer, rel.TagName) < 0

	// Mirror the guard that cacheLoad applies on read: only carry URLs that
	// belong to this repo, so forced/live results cannot surface arbitrary URLs
	// through --check-update output.
	url := rel.HTMLURL
	if !strings.HasPrefix(url, releaseURLPrefix) {
		url = ""
	}

	r := &Result{
		SchemaVersion:    cacheSchemaVersion,
		Current:          currentVer,
		Latest:           rel.TagName,
		UpgradeAvailable: upgrade,
		ReleaseURL:       url,
		CheckedAt:        time.Now().UTC(),
	}
	_ = cacheSave(r) // best-effort; errors are non-fatal
	return r, nil
}
