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
