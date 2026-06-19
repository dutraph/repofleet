package api

import (
	"context"
	"fmt"
	"strings"
)

// githubClient talks to github.com or a GitHub Enterprise instance.
type githubClient struct {
	pat     string
	baseURL string // GHE base (e.g. https://ghe.corp); blank = github.com
}

func (c *githubClient) ProviderName() string { return "GitHub" }

func (c *githubClient) apiBase() string {
	if c.baseURL == "" {
		return "https://api.github.com"
	}
	return strings.TrimRight(c.baseURL, "/") + "/api/v3"
}

type ghRepo struct {
	FullName    string `json:"full_name"`
	Name        string `json:"name"`
	CloneURL    string `json:"clone_url"`
	SSHURL      string `json:"ssh_url"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
}

func (c *githubClient) List(ctx context.Context) ([]RemoteRepo, error) {
	headers := map[string]string{
		"Authorization": "Bearer " + c.pat,
		"Accept":        "application/vnd.github+json",
		"User-Agent":    "repofleet",
	}
	var out []RemoteRepo
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/user/repos?per_page=100&page=%d&affiliation=owner,collaborator,organization_member&sort=updated",
			c.apiBase(), page)
		var batch []ghRepo
		if err := getJSON(ctx, url, headers, &batch); err != nil {
			return nil, err
		}
		for _, r := range batch {
			out = append(out, RemoteRepo{
				FullName: r.FullName, Name: r.Name,
				CloneURL: r.CloneURL, SSHURL: r.SSHURL,
				Description: r.Description, Private: r.Private,
			})
		}
		if len(batch) < 100 {
			break
		}
	}
	return out, nil
}
