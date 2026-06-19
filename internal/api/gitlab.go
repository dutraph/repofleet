package api

import (
	"context"
	"fmt"
	"strings"
)

// gitlabClient talks to gitlab.com or a self-hosted GitLab instance.
type gitlabClient struct {
	pat     string
	baseURL string // blank = gitlab.com
}

func (c *gitlabClient) ProviderName() string { return "GitLab" }

func (c *gitlabClient) apiBase() string {
	base := c.baseURL
	if base == "" {
		base = "https://gitlab.com"
	}
	return strings.TrimRight(base, "/") + "/api/v4"
}

type glRepo struct {
	PathWithNamespace string `json:"path_with_namespace"`
	Name              string `json:"name"`
	HTTPURL           string `json:"http_url_to_repo"`
	SSHURL            string `json:"ssh_url_to_repo"`
	Description       string `json:"description"`
	Visibility        string `json:"visibility"`
}

func (c *gitlabClient) List(ctx context.Context) ([]RemoteRepo, error) {
	headers := map[string]string{
		"PRIVATE-TOKEN": c.pat,
		"User-Agent":    "repofleet",
	}
	var out []RemoteRepo
	for page := 1; page <= maxPages; page++ {
		url := fmt.Sprintf("%s/projects?membership=true&per_page=100&page=%d&order_by=last_activity_at",
			c.apiBase(), page)
		var batch []glRepo
		if err := getJSON(ctx, url, headers, &batch); err != nil {
			return nil, err
		}
		for _, r := range batch {
			out = append(out, RemoteRepo{
				FullName: r.PathWithNamespace, Name: r.Name,
				CloneURL: r.HTTPURL, SSHURL: r.SSHURL,
				Description: r.Description, Private: r.Visibility != "public",
			})
		}
		if len(batch) < 100 {
			break
		}
	}
	return out, nil
}
