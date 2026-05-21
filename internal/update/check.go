package update

import (
	"context"
	"strings"
	"time"
)

// Check returns the update status for currentVer.
// Returns (nil, nil) when currentVer is a dev/pseudo-version — no check is performed.
// When force=true the 24h cache is bypassed and a fresh fetch is made.
// On any network or parse error the caller receives a non-nil error; silent-degrade
// is the caller's responsibility (both Phase 3 and Phase 4 treat errors as no-op).
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

	// Treat draft/prerelease as "no stable upgrade available".
	if rel.Draft || rel.Prerelease || IsPrerelease(rel.TagName) {
		return &Result{
			SchemaVersion:    cacheSchemaVersion,
			Current:          currentVer,
			Latest:           currentVer,
			UpgradeAvailable: false,
			CheckedAt:        time.Now().UTC(),
		}, nil
	}

	upgrade := Compare(currentVer, rel.TagName) < 0
	r := &Result{
		SchemaVersion:    cacheSchemaVersion,
		Current:          currentVer,
		Latest:           rel.TagName,
		UpgradeAvailable: upgrade,
		ReleaseURL:       rel.HTMLURL,
		CheckedAt:        time.Now().UTC(),
	}
	_ = cacheSave(r) // best-effort; errors are non-fatal
	return r, nil
}
