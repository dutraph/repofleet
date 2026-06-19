package api

import (
	"context"
	"encoding/base64"
	"fmt"
)

// azureClient talks to Azure DevOps Services for a single organization.
// The List API, called without a project, returns every repository the
// token can see across all projects in the org.
type azureClient struct {
	pat string
	org string
}

func (c *azureClient) ProviderName() string { return "Azure DevOps" }

type adoRepo struct {
	Name      string `json:"name"`
	RemoteURL string `json:"remoteUrl"`
	SSHURL    string `json:"sshUrl"`
	Project   struct {
		Name string `json:"name"`
	} `json:"project"`
}

type adoList struct {
	Value []adoRepo `json:"value"`
}

func (c *azureClient) List(ctx context.Context) ([]RemoteRepo, error) {
	// Azure DevOps uses Basic auth with an empty username and the PAT
	// as the password.
	auth := base64.StdEncoding.EncodeToString([]byte(":" + c.pat))
	headers := map[string]string{
		"Authorization": "Basic " + auth,
		"User-Agent":    "repofleet",
	}
	url := fmt.Sprintf("https://dev.azure.com/%s/_apis/git/repositories?api-version=7.1", c.org)
	var list adoList
	if err := getJSON(ctx, url, headers, &list); err != nil {
		return nil, err
	}
	out := make([]RemoteRepo, 0, len(list.Value))
	for _, r := range list.Value {
		out = append(out, RemoteRepo{
			FullName: r.Project.Name + "/" + r.Name,
			Name:     r.Name,
			CloneURL: r.RemoteURL,
			SSHURL:   r.SSHURL,
			Private:  true, // Azure DevOps repos are private by default
		})
	}
	return out, nil
}
