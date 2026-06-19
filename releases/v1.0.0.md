# repofleet v1.0.0

First stable release. **repofleet** is a fast, keyboard-driven TUI (k9s-style) to manage every git repository you have — both the ones already on your machine and the ones living on GitHub, GitLab, Azure DevOps and Bitbucket.

## Highlights

**Discover & inspect**
- Scans your directories and lists every git repo, with the provider icon next to each (oh-my-zsh style):  GitHub ·  GitLab ·  Azure DevOps ·  Bitbucket ·  local.
- Live status per repo: current branch, dirty/clean, ahead/behind.
- Search (`/`) and filter by provider type (`t`) — pick a provider and see only its repos.
- Details pane (`enter`) with the repo's remote, path and `git status`.

**Find duplicates**
- Detects repositories cloned into more than one path (same remote) and tags them `⧉ i/n`.
- Dedicated duplicates screen (`D`) that groups copies by repo and lists the path of each one.

**Bulk & per-repo git actions**
- Multi-select (`space` / `a`) then `pull --ff-only` (`p`), `pull --prune` (`P`), `fetch` (`f`).
- `fetch all` (`F`) to sync every repo at once.
- Switch branch (`b`) — local and remote branches, remote ones become tracking branches automatically.
- Remove a local copy (`d`) with a confirmation prompt.

**`:` command bar**
- Run any git command on the selected repo, from any screen (list, duplicate group, details).
- Backed by a real subprocess, so interactive commands work: `commit`, `rebase -i`, `add -p`, your aliases, pager and `$EDITOR`.

**Clone from your git server**
- Connect accounts via PAT (`fleet login`) for GitHub, GitLab, Azure DevOps and Bitbucket.
- Browse the repositories your token can see, filter, and pick one to clone.
- Choose **HTTPS or SSH** (`tab`), and a **filesystem browser** to pick the destination — with type-to-filter and **create-folder** (`ctrl+n`).
- Warns before cloning if the repo is already cloned elsewhere on your machine.

## Install

```bash
# go install
go install github.com/dutraph/repofleet/cmd/fleet@latest

# install script
curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash

# or grab a prebuilt binary from the assets below (verify against SHA256SUMS)
```

Prebuilt binaries: `darwin`/`linux` × `amd64`/`arm64`. Verify with the attached `SHA256SUMS`.

## Requirements

- `git` on your `PATH`
- a [Nerd Font](https://www.nerdfonts.com) in your terminal for the provider icons

## Keyboard reference

`space` select · `a` all · `p` pull · `P` pull --prune · `f` fetch · `F` fetch all · `b` branch · `d` remove · `D` duplicates · `t` filter by type · `/` search · `:` git command · `c` clone from server · `enter` details · `r` rescan · `?` help · `q` quit

---

Built with Bubble Tea + Lipgloss. Feedback and issues welcome.
