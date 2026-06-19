# Release notes

Per-version notes for repofleet. The published GitHub releases live at
<https://github.com/dutraph/repofleet/releases>; these files are the
source for each release's description.

| Version | Notes | Highlights |
| --- | --- | --- |
| v1.0.1 | [v1.0.1.md](v1.0.1.md) | Filter duplicates by provider type; stop tracking the build binary |
| v1.0.0 | [v1.0.0.md](v1.0.0.md) | First stable release — full local + remote repo management TUI |

## Cutting a new release

1. Write `releases/vX.Y.Z.md`.
2. Bump `VERSION` (or pass `VERSION=X.Y.Z` to `make install`).
3. Commit, then tag and push:

   ```bash
   git tag -a vX.Y.Z -m "repofleet vX.Y.Z"
   git push origin vX.Y.Z
   ```

4. The `release` workflow builds the binaries and creates the GitHub
   release. Attach these notes:

   ```bash
   gh release edit vX.Y.Z --notes-file releases/vX.Y.Z.md
   ```
