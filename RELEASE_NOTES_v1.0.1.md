# repofleet v1.0.1

Small follow-up to the first stable release.

## What's new

- **Filter duplicates by provider type** — in the duplicates screen (`D`), press `t` to cycle through the providers that actually have duplicates (`all → GitHub → GitLab → Azure DevOps → Bitbucket → local`). The active filter shows in the title.

## Housekeeping

- Stopped tracking the compiled `fleet` binary and fixed `.gitignore` (it still referenced the old binary name), so the build artifact no longer lands in the repo.

## Install

```bash
go install github.com/dutraph/repofleet/cmd/fleet@latest
# or
curl -fsSL https://raw.githubusercontent.com/dutraph/repofleet/main/install.sh | bash
# or grab a prebuilt binary from the assets below (verify against SHA256SUMS)
```

Full feature overview is in the README and the v1.0.0 notes.
