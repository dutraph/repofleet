// Package provider classifies a git remote URL into a known hosting
// provider and exposes the Nerd Font icon + accent used to render it
// (oh-my-zsh style). Everything here is pure string work — no network.
//
// Icons require a patched Nerd Font (https://www.nerdfonts.com). They
// are intentionally kept in one map so they are trivial to swap.
package provider

import (
	"regexp"
	"strings"
)

// Kind is the set of providers we recognize.
type Kind int

const (
	Local Kind = iota // no remote at all — a purely local repo
	GitHub
	GitLab
	AzureDevOps
	Bitbucket
	Unknown // has a remote, but host is not one we recognize
)

// Provider bundles the display metadata for a Kind.
type Provider struct {
	Kind  Kind
	Name  string // short label, e.g. "GitHub"
	Icon  string // Nerd Font glyph
	Color string // hex accent used by the theme
}

var providers = map[Kind]Provider{
	Local:       {Local, "local", "", "#7d8590"},        //  git
	GitHub:      {GitHub, "GitHub", "", "#e6edf3"},      //  github
	GitLab:      {GitLab, "GitLab", "", "#fc6d26"},      //  gitlab
	AzureDevOps: {AzureDevOps, "Azure", "", "#0078d4"},  //  azure devops
	Bitbucket:   {Bitbucket, "Bitbucket", "", "#2684ff"}, //  bitbucket
	Unknown:     {Unknown, "git", "", "#7d8590"},        //  generic
}

// Meta returns the display metadata for a Kind.
func Meta(k Kind) Provider {
	if p, ok := providers[k]; ok {
		return p
	}
	return providers[Unknown]
}

// scpLike matches the scp-style remote syntax: git@host:owner/repo.git
var scpLike = regexp.MustCompile(`^[\w.-]+@([\w.-]+):`)

// Detect classifies a remote URL. An empty URL is treated as Local.
func Detect(remote string) Kind {
	r := strings.TrimSpace(strings.ToLower(remote))
	if r == "" {
		return Local
	}
	host := hostOf(r)
	switch {
	case strings.Contains(host, "github"):
		return GitHub
	case strings.Contains(host, "gitlab"):
		return GitLab
	case strings.Contains(host, "dev.azure.com"),
		strings.Contains(host, "visualstudio.com"),
		strings.Contains(host, "azure"):
		return AzureDevOps
	case strings.Contains(host, "bitbucket"):
		return Bitbucket
	default:
		return Unknown
	}
}

// hostOf extracts the host portion from any of the git remote forms:
//
//	https://host/owner/repo.git
//	ssh://git@host:22/owner/repo.git
//	git@host:owner/repo.git   (scp-like)
func hostOf(r string) string {
	if m := scpLike.FindStringSubmatch(r); m != nil {
		return m[1]
	}
	// strip scheme
	if i := strings.Index(r, "://"); i >= 0 {
		r = r[i+3:]
	}
	// strip userinfo
	if i := strings.Index(r, "@"); i >= 0 {
		r = r[i+1:]
	}
	// cut at first / or :
	if i := strings.IndexAny(r, "/:"); i >= 0 {
		r = r[:i]
	}
	return r
}

// Normalize reduces a remote URL to a stable identity so the same repo
// cloned from https vs ssh (or with/without a trailing .git) collapses
// to one key. Used for duplicate detection.
func Normalize(remote string) string {
	r := strings.TrimSpace(strings.ToLower(remote))
	if r == "" {
		return ""
	}
	host := hostOf(r)
	// path after host
	path := r
	if m := scpLike.FindStringSubmatch(r); m != nil {
		path = r[len(m[0]):]
	} else {
		if i := strings.Index(path, "://"); i >= 0 {
			path = path[i+3:]
		}
		if i := strings.Index(path, "@"); i >= 0 {
			path = path[i+1:]
		}
		if i := strings.IndexAny(path, "/"); i >= 0 {
			path = path[i+1:]
		} else {
			path = ""
		}
	}
	path = strings.TrimSuffix(path, ".git")
	path = strings.Trim(path, "/")
	return host + "/" + path
}
