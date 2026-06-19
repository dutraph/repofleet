// Package update checks GitHub for a newer published release so the TUI
// can nudge the user when they're running an outdated binary. All calls
// are best-effort: any network/parse failure is reported as an error and
// simply ignored by the caller (no nagging when offline).
package update

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const latestURL = "https://api.github.com/repos/dutraph/repofleet/releases/latest"

type release struct {
	TagName string `json:"tag_name"`
}

// Latest returns the most recent published release version (without the
// leading "v").
func Latest(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "repofleet")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("github: HTTP %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	var r release
	if err := json.Unmarshal(body, &r); err != nil {
		return "", err
	}
	return strings.TrimPrefix(strings.TrimSpace(r.TagName), "v"), nil
}

// Newer reports whether latest is a strictly greater semver than current.
// Non-semver inputs (e.g. "dev" for an unstamped build) return false, so
// development builds are never nagged.
func Newer(latest, current string) bool {
	lp, ok1 := parse(latest)
	cp, ok2 := parse(current)
	if !ok1 || !ok2 {
		return false
	}
	for i := 0; i < 3; i++ {
		if lp[i] != cp[i] {
			return lp[i] > cp[i]
		}
	}
	return false
}

func parse(v string) ([3]int, bool) {
	v = strings.TrimPrefix(strings.TrimSpace(v), "v")
	if i := strings.IndexAny(v, "-+"); i >= 0 {
		v = v[:i] // drop pre-release / build metadata
	}
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return [3]int{}, false
	}
	var out [3]int
	for i := range parts {
		n, err := strconv.Atoi(parts[i])
		if err != nil {
			return [3]int{}, false
		}
		out[i] = n
	}
	return out, true
}
