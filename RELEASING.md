# Releasing ORAS CLI

ORAS CLI releases are managed by [`scripts/release.sh`](scripts/release.sh), which automates preparation, tagging, validation, and publishing.

## Prerequisites

Before starting a release:

- Install the GitHub CLI (`gh`) and authenticate it.
- Ensure the `GPG_PRIVATE_KEY` and `GPG_PASSPHRASE` GitHub Actions secrets are configured for release signing.
- Configure Git commit and tag signing (GPG or SSH).
- Install `gpg`; `publish` uses it to verify artifact signatures even if you sign with SSH.
- Ensure the ORAS project release public key is available in [`KEYS`](KEYS) for artifact signature verification.
- Before releasing, update the Go version in `go.mod` and the relevant GitHub Actions workflows when the project moves to a new Go version.

## Release process

Run commands from the repository root. Replace `VERSION` and `COMMIT_SHA` with the intended release version and commit SHA.

### 1. Prepare the release

```sh
scripts/release.sh prep VERSION
```

The script validates the version and release branch, updates `internal/version/version.go`, creates a signed-off (DCO) release-preparation commit, pushes a branch, and opens a pull request.

It also prints a release-vote template for Slack `#oras-maintainers`. The pull request must pass CI and receive the required maintainer super-majority approval before merging.

Releases for the current series and new minor versions are prepared from `main`. Patch releases for older series must use their corresponding `release-X.Y` branch. Set `ORAS_MAIN_RELEASE_VERSION` to override the release series detected for `main` when necessary.

### 2. Tag the release

After the release pull request is approved and merged, tag the merged commit:

```sh
scripts/release.sh tag VERSION COMMIT_SHA
```

The script verifies that the commit records the requested version, creates and pushes a signed `vVERSION` tag, and creates a `release-X.Y` branch for a new minor release.

The tag triggers the `release-github` and `release-ghcr` workflows.

### 3. Validate the release

Run `validate` right after tagging; it waits up to 30 minutes for the tag-triggered workflows to finish:

```sh
scripts/release.sh validate VERSION
```

The script downloads release artifacts, checks their SHA-256 checksums, and verifies the binary version and available commit information.

Review the release notes and verify that the GitHub Release is associated with the intended `vVERSION` tag and commit, not a GoReleaser placeholder tag such as `untagged-<hash>`.

### 4. Publish the release

After validation succeeds, run:

```sh
scripts/release.sh publish VERSION
```

The script verifies artifact signatures against the ORAS project release key, adds verification information to the release notes, publishes the GitHub Release, and triggers the Snap publishing workflow (with `isStable=false` for pre-releases).

## Backport releases

If an older minor series needs a fix after a newer minor has shipped, release it from a `release-X.Y` maintenance branch instead of `main`:

1. If `release-X.Y` does not exist yet, cut it from that series' last tag and push it, for example `git branch release-1.2 v1.2.3` followed by `git push <remote> release-1.2`.
2. Cherry-pick the fix onto `release-X.Y` through a pull request against that branch.
3. Check out `release-X.Y` and run `scripts/release.sh prep VERSION`. The release pull request targets `release-X.Y`.
4. Continue with the tag, validate, and publish steps as usual, tagging the merged commit on `release-X.Y`.

## Dry runs

Use `--dry-run` to preview the `prep` and `tag` phases without changing the repository:

```sh
scripts/release.sh --dry-run prep VERSION
scripts/release.sh --dry-run tag VERSION COMMIT_SHA
```

`validate` and `publish` need a real tag and the `_dist/` directory downloaded by `validate`, so they cannot be dry-run before a release.

## Post-release tasks

- Update the [oras-www](https://github.com/oras-project/oras-www) documentation as needed.
- Announce the release in Slack `#oras`.
- Publish applicable security advisories after the release has shipped.

## Implementation references

- [`scripts/release.sh`](scripts/release.sh)
- [`.goreleaser.yml`](.goreleaser.yml)
- [Release workflows](.github/workflows)