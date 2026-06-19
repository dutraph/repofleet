// Package gitops shells out to the system git binary for the handful of
// operations the TUI performs: reading status, pulling, fetching, and
// cloning. Keeping this isolated means the rest of the app never builds
// a git command line by hand.
package gitops

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Status is the lazily-computed git state of a working tree.
type Status struct {
	Branch string
	Dirty  bool // uncommitted changes in the working tree or index
	Ahead  int  // commits ahead of upstream
	Behind int  // commits behind upstream
	Err    error
}

// Symbol renders a compact, ANSI-free status word for table cells.
// (Table cells must stay plain text — see internal/ui/table.go.)
func (s Status) Symbol() string {
	if s.Err != nil {
		return "?"
	}
	parts := []string{}
	if s.Dirty {
		parts = append(parts, "✗")
	} else {
		parts = append(parts, "✓")
	}
	if s.Ahead > 0 {
		parts = append(parts, "↑"+strconv.Itoa(s.Ahead))
	}
	if s.Behind > 0 {
		parts = append(parts, "↓"+strconv.Itoa(s.Behind))
	}
	return strings.Join(parts, " ")
}

// GetStatus runs `git status --porcelain=v2 --branch` and parses it.
func GetStatus(repoPath string) Status {
	cmd := exec.Command("git", "-C", repoPath, "status", "--porcelain=v2", "--branch")
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return Status{Err: fmt.Errorf("git status: %s", strings.TrimSpace(errBuf.String()))}
	}

	var st Status
	sc := bufio.NewScanner(&out)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "# branch.head "):
			st.Branch = strings.TrimPrefix(line, "# branch.head ")
		case strings.HasPrefix(line, "# branch.ab "):
			fmt.Sscanf(strings.TrimPrefix(line, "# branch.ab "), "+%d -%d", &st.Ahead, &st.Behind)
		case strings.HasPrefix(line, "1 "), strings.HasPrefix(line, "2 "),
			strings.HasPrefix(line, "u "), strings.HasPrefix(line, "? "):
			st.Dirty = true
		}
	}
	if st.Branch == "" {
		st.Branch = "(detached)"
	}
	return st
}

// Result is the outcome of a git command run against one repo.
type Result struct {
	Path   string
	Output string
	Err    error
}

// Pull runs `git pull --ff-only` so the TUI never creates surprise
// merge commits; a non-fast-forward simply reports back as an error.
func Pull(repoPath string) Result {
	return run(repoPath, "pull", "--ff-only")
}

// Fetch runs `git fetch --all --prune`.
func Fetch(repoPath string) Result {
	return run(repoPath, "fetch", "--all", "--prune")
}

// PullPrune runs `git pull --ff-only --prune`, so it also prunes stale
// remote-tracking branches while fast-forwarding.
func PullPrune(repoPath string) Result {
	return run(repoPath, "pull", "--ff-only", "--prune")
}

// Checkout switches the working tree to branch (creating a tracking
// branch automatically when only the remote one exists).
func Checkout(repoPath, branch string) Result {
	return run(repoPath, "checkout", branch)
}

// Branch is a single ref the user can switch to.
type Branch struct {
	Name    string // display name; remotes keep their "origin/" prefix
	Remote  bool   // true when this is a remote-tracking ref
	Current bool   // true for the checked-out branch
}

// CheckoutName returns the argument to pass to `git checkout`: the short
// name for a remote ref (so git creates a tracking branch instead of a
// detached HEAD), or the name itself for a local branch.
func (b Branch) CheckoutName() string {
	if b.Remote {
		if i := strings.Index(b.Name, "/"); i >= 0 {
			return b.Name[i+1:]
		}
	}
	return b.Name
}

// Branches lists local heads first, then remote-tracking branches that
// have no local counterpart.
func Branches(repoPath string) ([]Branch, error) {
	cur := run(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	current := ""
	if cur.Err == nil {
		current = strings.TrimSpace(cur.Output)
	}

	local := run(repoPath, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if local.Err != nil {
		return nil, fmt.Errorf("list branches: %s", local.Output)
	}

	var out []Branch
	localShort := map[string]bool{}
	for _, b := range strings.Split(local.Output, "\n") {
		b = strings.TrimSpace(b)
		if b == "" {
			continue
		}
		localShort[b] = true
		out = append(out, Branch{Name: b, Current: b == current})
	}

	remote := run(repoPath, "for-each-ref", "--format=%(refname:short)", "refs/remotes")
	if remote.Err == nil {
		for _, b := range strings.Split(remote.Output, "\n") {
			b = strings.TrimSpace(b)
			if b == "" || strings.HasSuffix(b, "/HEAD") {
				continue
			}
			short := b
			if i := strings.Index(b, "/"); i >= 0 {
				short = b[i+1:]
			}
			if !localShort[short] {
				out = append(out, Branch{Name: b, Remote: true})
				localShort[short] = true // dedupe multiple remotes
			}
		}
	}
	return out, nil
}

// StatusText runs a human-readable `git status -sb` for the detail pane.
func StatusText(repoPath string) Result {
	return run(repoPath, "status", "-sb")
}

func run(repoPath string, args ...string) Result {
	full := append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", full...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return Result{Path: repoPath, Output: strings.TrimSpace(out.String()), Err: err}
}

// CloneCommand returns the *exec.Cmd for cloning url into dest. The TUI
// runs this via tea.ExecProcess so the user sees git's real-time
// progress on screen. The caller is responsible for the duplicate
// check (see scanner.FindByKey) before invoking this.
func CloneCommand(url, dest string) *exec.Cmd {
	return exec.Command("git", "clone", url, dest)
}

// DestExists reports whether dest already exists and is non-empty, which
// would make `git clone` fail.
func DestExists(dest string) bool {
	entries, err := os.ReadDir(dest)
	return err == nil && len(entries) > 0
}

// CommandLine builds an interactive `git` command for the `:` command
// bar. It runs through `sh -c` so the user's git config (aliases,
// pager, editor) and interactive commands (commit, rebase -i, add -p)
// all work, and pauses afterwards so the output stays on screen until
// the user presses Enter. rawArgs is whatever the user typed (an
// optional leading "git " is stripped by the caller).
func CommandLine(repoPath, rawArgs string) *exec.Cmd {
	q := shellSingleQuote(repoPath)
	script := "git -C " + q + " " + rawArgs +
		"; status=$?; printf '\\n\\033[2m── press Enter to return ──\\033[0m'; read _; exit $status"
	return exec.Command("sh", "-c", script)
}

// shellSingleQuote wraps s in single quotes, safely escaping any single
// quotes inside it, so it can be embedded in an sh -c script.
func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}
