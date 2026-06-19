// Package scanner walks one or more root directories looking for git
// repositories. For each repo it reads the origin remote straight out
// of .git/config (fast, no subprocess) and classifies the provider.
// Duplicate detection groups repos that share the same normalized
// remote — i.e. the same upstream cloned into more than one path.
package scanner

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dutraph/repofleet/internal/provider"
)

// Repo is a single git working copy discovered on disk. Git-status
// fields (Branch/Dirty/Ahead/Behind) are filled in lazily by the
// gitops package — the scanner leaves them zero so the listing appears
// instantly even across hundreds of repos.
type Repo struct {
	Path      string        // absolute path to the working tree
	Name      string        // base directory name
	RemoteURL string        // origin URL ("" when none)
	Provider  provider.Kind // classified from RemoteURL
	Key       string        // normalized remote, "" when local-only

	// Duplicate grouping: DupCount is how many repos share this Key
	// (1 = unique). DupIndex is this repo's position within the group.
	DupCount int
	DupIndex int
}

// IsDuplicate reports whether this repo's remote is cloned elsewhere too.
func (r Repo) IsDuplicate() bool { return r.DupCount > 1 }

// skipDirs are never descended into — they are large and never contain
// repos we care about (and vendored .git copies would create noise).
var skipDirs = map[string]bool{
	"node_modules": true,
	".Trash":       true,
	"Library":      true,
	"vendor":       true,
	".cache":       true,
}

// Scan walks each root up to maxDepth levels deep and returns every git
// repo found, sorted by path. maxDepth <= 0 means unlimited.
func Scan(roots []string, maxDepth int) ([]Repo, error) {
	var repos []Repo
	seen := map[string]bool{}

	for _, root := range roots {
		root = expand(root)
		base, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		baseDepth := strings.Count(base, string(os.PathSeparator))

		_ = filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // unreadable dir — skip, don't abort the whole walk
			}
			if !d.IsDir() {
				return nil
			}
			name := d.Name()
			if path != base && skipDirs[name] {
				return filepath.SkipDir
			}
			if maxDepth > 0 {
				if strings.Count(path, string(os.PathSeparator))-baseDepth > maxDepth {
					return filepath.SkipDir
				}
			}
			// A directory containing .git is a repo root.
			if isGitDir(path) {
				if !seen[path] {
					seen[path] = true
					repos = append(repos, newRepo(path))
				}
				return filepath.SkipDir // don't descend into the repo
			}
			return nil
		})
	}

	markDuplicates(repos)
	sort.Slice(repos, func(i, j int) bool { return repos[i].Path < repos[j].Path })
	return repos, nil
}

func newRepo(path string) Repo {
	remote := readOriginRemote(path)
	return Repo{
		Path:      path,
		Name:      filepath.Base(path),
		RemoteURL: remote,
		Provider:  provider.Detect(remote),
		Key:       provider.Normalize(remote),
	}
}

// isGitDir reports whether path/.git exists (a directory for normal
// repos, or a file for worktrees/submodules).
func isGitDir(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil && (info.IsDir() || info.Mode().IsRegular())
}

// readOriginRemote parses .git/config for the origin remote URL without
// shelling out to git. Falls back to the first remote it finds.
func readOriginRemote(repoPath string) string {
	f, err := os.Open(filepath.Join(repoPath, ".git", "config"))
	if err != nil {
		return ""
	}
	defer f.Close()

	var current string
	var origin, first string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "[remote ") {
			current = strings.Trim(strings.TrimPrefix(line, "[remote "), "\"]")
			continue
		}
		if strings.HasPrefix(line, "[") {
			current = ""
			continue
		}
		if current != "" && strings.HasPrefix(line, "url") {
			if i := strings.Index(line, "="); i >= 0 {
				url := strings.TrimSpace(line[i+1:])
				if first == "" {
					first = url
				}
				if current == "origin" {
					origin = url
				}
			}
		}
	}
	if origin != "" {
		return origin
	}
	return first
}

// markDuplicates fills DupCount/DupIndex by grouping on Key.
func markDuplicates(repos []Repo) {
	groups := map[string][]int{}
	for i := range repos {
		if repos[i].Key == "" {
			continue
		}
		groups[repos[i].Key] = append(groups[repos[i].Key], i)
	}
	for _, idxs := range groups {
		for pos, i := range idxs {
			repos[i].DupCount = len(idxs)
			repos[i].DupIndex = pos + 1
		}
	}
}

// FindByKey returns the paths of every scanned repo that shares the
// given normalized remote key. Used before cloning to warn the user
// that the repo already exists locally.
func FindByKey(repos []Repo, key string) []string {
	var out []string
	for _, r := range repos {
		if key != "" && r.Key == key {
			out = append(out, r.Path)
		}
	}
	return out
}

func expand(p string) string {
	if p == "~" || strings.HasPrefix(p, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
