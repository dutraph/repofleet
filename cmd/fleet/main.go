// Entry point for fleet. Keep this file thin — flag parsing,
// subcommand dispatch, and a single hand-off to the TUI. All business
// logic belongs in internal/.
package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dutraph/repofleet/internal/config"
	"github.com/dutraph/repofleet/internal/gitops"
	"github.com/dutraph/repofleet/internal/provider"
	"github.com/dutraph/repofleet/internal/scanner"
	"github.com/dutraph/repofleet/internal/ui"
	"github.com/dutraph/repofleet/internal/version"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "login":
			must(config.RunLoginWizard())
			return
		case "accounts", "orgs":
			must(config.ListAccounts())
			return
		case "switch":
			if len(os.Args) < 3 {
				fail("switch: missing account name (run `fleet accounts` to list)")
			}
			must(config.SwitchAccount(os.Args[2]))
			return
		case "scan":
			must(runScan())
			return
		case "version", "-v", "--version":
			fmt.Printf("fleet %s\n", version.String())
			return
		case "help", "-h", "--help":
			printHelp()
			return
		}
	}

	cfg, err := config.Load()
	if err != nil {
		fail(fmt.Sprintf("could not load config: %v", err))
	}

	p := tea.NewProgram(ui.New(cfg), tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fail(fmt.Sprintf("fatal: %v", err))
	}
}

// runScan prints all discovered repos to stdout (headless mode — handy
// for piping into other tools or a quick audit).
func runScan() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	repos, err := scanner.Scan(cfg.ScanRoots, 7)
	if err != nil {
		return err
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "PROVIDER\tDUP\tNAME\tBRANCH\tPATH")
	for _, r := range repos {
		st := gitops.GetStatus(r.Path)
		dup := ""
		if r.IsDuplicate() {
			dup = fmt.Sprintf("%d/%d", r.DupIndex, r.DupCount)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			provider.Meta(r.Provider).Name, dup, r.Name, st.Branch, r.Path)
	}
	return w.Flush()
}

func printHelp() {
	fmt.Println(`repos — manage all your local & remote git repositories

Usage:
  fleet                 launch the TUI
  fleet scan            list discovered repos (headless)
  fleet login           connect a git server (GitHub/GitLab/Azure/Bitbucket)
  fleet accounts        list configured accounts (★ = active)
  fleet switch <name>   switch the active account
  fleet version         show version
  fleet help            this help

Inside the TUI:
  space  select repo        p  pull selected      f  fetch selected
  enter  details            c  clone from server  r  rescan
  /      filter (browser)   ?  help               q  quit`)
}

func must(err error) {
	if err != nil {
		fail(err.Error())
	}
}

func fail(msg string) {
	fmt.Fprintln(os.Stderr, msg)
	os.Exit(1)
}
