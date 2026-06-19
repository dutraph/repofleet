// Package api lists repositories from a git host using a Personal Access
// Token. Each provider gets a hand-rolled net/http client (no heavy
// SDKs) behind a common Lister interface, so the TUI can browse and pick
// a repo to clone regardless of where it lives.
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dutraph/repofleet/internal/config"
)

// RemoteRepo is a provider-agnostic view of a remote repository.
type RemoteRepo struct {
	FullName    string // owner/name or namespace/path or project/name
	Name        string
	CloneURL    string // https clone URL
	SSHURL      string // ssh clone URL
	Description string
	Private     bool
}

// CloneURLFor returns the URL to clone with, honoring the user's
// protocol preference ("ssh" → SSHURL when available, else https).
func (r RemoteRepo) CloneURLFor(protocol string) string {
	if protocol == "ssh" && r.SSHURL != "" {
		return r.SSHURL
	}
	return r.CloneURL
}

// Lister is implemented by every provider client.
type Lister interface {
	// List returns every repository the token can see.
	List(ctx context.Context) ([]RemoteRepo, error)
	// ProviderName is a human label for status messages.
	ProviderName() string
}

// New builds the right client for an account.
func New(acct config.Account) (Lister, error) {
	switch acct.Provider {
	case config.ProviderGitHub:
		return &githubClient{pat: acct.PAT, baseURL: acct.BaseURL}, nil
	case config.ProviderGitLab:
		return &gitlabClient{pat: acct.PAT, baseURL: acct.BaseURL}, nil
	case config.ProviderAzure:
		return &azureClient{pat: acct.PAT, org: acct.Org}, nil
	case config.ProviderBitbucket:
		return &bitbucketClient{user: acct.Username, pat: acct.PAT}, nil
	default:
		return nil, fmt.Errorf("unknown provider %q", acct.Provider)
	}
}

// httpClient is shared across providers.
var httpClient = &http.Client{Timeout: 30 * time.Second}

// maxPages caps pagination so a misbehaving API can never spin forever.
const maxPages = 20

// getJSON performs an authenticated GET and decodes the JSON body into v.
// headers carries the provider-specific auth (Authorization / PRIVATE-TOKEN).
func getJSON(ctx context.Context, url string, headers map[string]string, v any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	for k, val := range headers {
		req.Header.Set(k, val)
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, snippet(body))
	}
	if v == nil {
		return nil
	}
	if err := json.Unmarshal(body, v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func snippet(b []byte) string {
	const n = 200
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}
