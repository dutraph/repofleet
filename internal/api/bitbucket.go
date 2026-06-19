package api

import (
	"context"
	"encoding/base64"
)

// bitbucketClient talks to Bitbucket Cloud. Auth is Basic with the
// username + an app password (Bitbucket's PAT equivalent).
type bitbucketClient struct {
	user string
	pat  string
}

func (c *bitbucketClient) ProviderName() string { return "Bitbucket" }

type bbRepo struct {
	FullName    string `json:"full_name"`
	Name        string `json:"name"`
	Description string `json:"description"`
	IsPrivate   bool   `json:"is_private"`
	Links       struct {
		Clone []struct {
			Name string `json:"name"`
			Href string `json:"href"`
		} `json:"clone"`
	} `json:"links"`
}

type bbPage struct {
	Values []bbRepo `json:"values"`
	Next   string   `json:"next"`
}

func (c *bitbucketClient) List(ctx context.Context) ([]RemoteRepo, error) {
	auth := base64.StdEncoding.EncodeToString([]byte(c.user + ":" + c.pat))
	headers := map[string]string{
		"Authorization": "Basic " + auth,
		"User-Agent":    "repofleet",
	}
	url := "https://api.bitbucket.org/2.0/repositories?role=member&pagelen=100"
	var out []RemoteRepo
	for i := 0; i < maxPages && url != ""; i++ {
		var page bbPage
		if err := getJSON(ctx, url, headers, &page); err != nil {
			return nil, err
		}
		for _, r := range page.Values {
			rr := RemoteRepo{
				FullName: r.FullName, Name: r.Name,
				Description: r.Description, Private: r.IsPrivate,
			}
			for _, cl := range r.Links.Clone {
				switch cl.Name {
				case "https":
					rr.CloneURL = cl.Href
				case "ssh":
					rr.SSHURL = cl.Href
				}
			}
			out = append(out, rr)
		}
		url = page.Next
	}
	return out, nil
}
