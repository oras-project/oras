#!/usr/bin/env bash
# Copyright The ORAS Authors.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

# ── Configuration ────────────────────────────────────────────────────────────
REPO="oras-project/oras"
REMOTE="${ORAS_REMOTE:-upstream}"
DRY_RUN=false
VERSION_FILE="internal/version/version.go"

# ── Colors ───────────────────────────────────────────────────────────────────
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m' # No Color

# ── Helpers ──────────────────────────────────────────────────────────────────
info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[WARN]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }
fatal()   { error "$@"; exit 1; }

confirm() {
    local msg="$1"
    if [[ "$DRY_RUN" == "true" ]]; then
        info "[dry-run] Would prompt: $msg"
        return 0
    fi
    echo -en "${BOLD}$msg [y/N]${NC} "
    read -r answer
    [[ "$answer" =~ ^[Yy]$ ]] || { info "Aborted."; exit 0; }
}

run() {
    if [[ "$DRY_RUN" == "true" ]]; then
        info "[dry-run] $*"
    else
        "$@"
    fi
}

# ── Validation ───────────────────────────────────────────────────────────────
validate_semver() {
    local version="$1"
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-(alpha|beta|rc)\.[0-9]+)?$ ]]; then
        fatal "Invalid semver: '$version'. Expected format: X.Y.Z or X.Y.Z-(alpha|beta|rc).N"
    fi
}

check_prerequisites() {
    info "Checking prerequisites..."

    if ! command -v gh &>/dev/null; then
        fatal "'gh' (GitHub CLI) is not installed. Install it from https://cli.github.com/"
    fi

    if ! gh auth status &>/dev/null; then
        fatal "'gh' is not authenticated. Run 'gh auth login' first."
    fi

    if ! command -v gpg &>/dev/null; then
        fatal "'gpg' is not installed. Install GnuPG first."
    fi

    if ! gpg --list-secret-keys --keyid-format=long 2>/dev/null | grep -q sec; then
        fatal "No GPG secret keys found. Generate one with 'gpg --full-generate-key'."
    fi

    if ! git remote get-url "$REMOTE" &>/dev/null; then
        fatal "Git remote '$REMOTE' not found. Set ORAS_REMOTE or add the remote."
    fi

    success "All prerequisites satisfied."
}

parse_version_parts() {
    local version="$1"
    local base="${version%%-*}"
    MAJOR="${base%%.*}"
    local rest="${base#*.}"
    MINOR="${rest%%.*}"
    PATCH="${rest#*.}"
    PRERELEASE=""
    if [[ "$version" == *-* ]]; then
        PRERELEASE="${version#*-}"
    fi
}

# ── Phase: prep ──────────────────────────────────────────────────────────────
phase_prep() {
    local version="$1"
    validate_semver "$version"
    check_prerequisites

    parse_version_parts "$version"

    # Determine base branch
    local base_branch="main"
    if [[ "$PATCH" -gt 0 ]]; then
        base_branch="release-${MAJOR}.${MINOR}"
        info "Patch release detected, targeting branch '$base_branch'."
    fi

    info "Preparing release v${version}..."

    # Ensure we are up-to-date
    info "Fetching latest from ${REMOTE}..."
    run git fetch "$REMOTE"

    # Create release branch
    local branch="chore/release-v${version}"
    info "Creating branch '${branch}' from '${REMOTE}/${base_branch}'..."
    run git checkout -b "$branch" "${REMOTE}/${base_branch}"

    # Update version file
    info "Updating ${VERSION_FILE}..."
    if [[ "$DRY_RUN" != "true" ]]; then
        sed -i.bak -E "s/(Version = \").*(\")/\1${version}\2/" "$VERSION_FILE"
        sed -i.bak -E "s/(BuildMetadata = \").*(\")/\1\2/" "$VERSION_FILE"
        rm -f "${VERSION_FILE}.bak"
    else
        info "[dry-run] Would set Version = \"${version}\" and BuildMetadata = \"\""
    fi

    # Commit and push
    info "Committing version bump..."
    run git add "$VERSION_FILE"
    run git commit -s -m "chore: bump version to ${version}"
    info "Pushing branch '${branch}' to origin..."
    run git push origin "$branch"

    # Create PR
    local pr_title="bump: tag and release ORAS CLI v${version}"
    info "Creating pull request..."
    if [[ "$DRY_RUN" != "true" ]]; then
        gh pr create \
            --repo "$REPO" \
            --title "$pr_title" \
            --base "$base_branch" \
            --body "$(cat <<EOF
## Release ORAS CLI v${version}

This PR bumps the version to **${version}** and clears the build metadata in preparation for the release.

### Changes
- Set \`Version\` to \`${version}\`
- Set \`BuildMetadata\` to \`""\`

### Release Checklist
- [ ] Version bump reviewed
- [ ] CI passing
- [ ] Maintainer approval received
EOF
)"
    else
        info "[dry-run] Would create PR: '$pr_title' targeting '$base_branch'"
    fi

    # Print commit SHA
    local sha
    sha=$(git rev-parse HEAD)
    success "Version bump committed."
    echo ""
    echo -e "${BOLD}Commit SHA:${NC} ${sha}"
    echo ""

    # Slack vote template
    echo -e "${BOLD}Slack Vote Template:${NC}"
    echo "────────────────────────────────────────"
    cat <<EOF
:oras: *ORAS CLI v${version} Release Vote*

Hi team, I'd like to propose the release of ORAS CLI v${version}.

*Release PR:* (paste PR URL here)
*Commit SHA:* \`${sha}\`

Please vote:
:+1: Approve
:-1: Object (please provide reason)

Vote closes in 48 hours. Requires at least 2 approvals from maintainers.
EOF
    echo "────────────────────────────────────────"
}

# ── Phase: tag ───────────────────────────────────────────────────────────────
phase_tag() {
    local version="$1"
    local sha="$2"
    validate_semver "$version"

    info "Tagging release v${version} at ${sha}..."

    # Validate the commit has the correct version
    info "Validating version at commit ${sha}..."
    local committed_version
    committed_version=$(git show "${sha}:${VERSION_FILE}" | grep 'Version = ' | sed -E 's/.*"(.*)".*/\1/')
    if [[ "$committed_version" != "$version" ]]; then
        fatal "Version mismatch: commit has '${committed_version}', expected '${version}'."
    fi

    local committed_metadata
    committed_metadata=$(git show "${sha}:${VERSION_FILE}" | grep 'BuildMetadata = ' | sed -E 's/.*"(.*)".*/\1/')
    if [[ -n "$committed_metadata" ]]; then
        fatal "BuildMetadata is not empty at ${sha}: '${committed_metadata}'. Expected empty string."
    fi

    success "Version validated: ${version} (BuildMetadata is clear)."

    # Create signed tag
    confirm "Create signed tag v${version} at ${sha}?"
    info "Creating signed tag v${version}..."
    run git tag -s "v${version}" "$sha" -m "Release v${version}"
    info "Pushing tag v${version} to ${REMOTE}..."
    run git push "$REMOTE" "v${version}"

    success "Tag v${version} pushed to ${REMOTE}."

    # Create release branch for new minor versions (patch==0, no pre-release)
    parse_version_parts "$version"
    if [[ "$PATCH" -eq 0 && -z "$PRERELEASE" ]]; then
        local release_branch="release-${MAJOR}.${MINOR}"
        info "New minor release detected. Creating release branch '${release_branch}'..."
        confirm "Create and push release branch '${release_branch}'?"
        run git branch "$release_branch" "$sha"
        run git push "$REMOTE" "$release_branch"
        success "Release branch '${release_branch}' created and pushed."
    fi
}

# ── Phase: validate ──────────────────────────────────────────────────────────
phase_validate() {
    local version="$1"
    validate_semver "$version"

    info "Validating release v${version}..."

    # Poll GitHub Actions workflows
    local workflows=("release-ghcr" "release-github")
    for wf in "${workflows[@]}"; do
        info "Polling workflow '${wf}' for tag v${version}..."
        if [[ "$DRY_RUN" == "true" ]]; then
            info "[dry-run] Would poll workflow '${wf}' until completion."
            continue
        fi

        local max_attempts=60
        local attempt=0
        while [[ $attempt -lt $max_attempts ]]; do
            local status
            status=$(gh run list \
                --repo "$REPO" \
                --workflow "${wf}.yml" \
                --branch "v${version}" \
                --limit 1 \
                --json status,conclusion \
                --jq '.[0] | "\(.status) \(.conclusion)"' 2>/dev/null || echo "not_found")

            if [[ "$status" == "not_found" || "$status" == " " ]]; then
                info "Workflow '${wf}' not found yet, waiting..."
            elif [[ "$status" == "completed success" ]]; then
                success "Workflow '${wf}' completed successfully."
                break
            elif [[ "$status" == completed* ]]; then
                fatal "Workflow '${wf}' failed: ${status}"
            else
                info "Workflow '${wf}' status: ${status}. Waiting..."
            fi

            attempt=$((attempt + 1))
            sleep 30
        done

        if [[ $attempt -ge $max_attempts ]]; then
            fatal "Timed out waiting for workflow '${wf}' to complete."
        fi
    done

    # Fetch distribution artifacts
    info "Fetching distribution artifacts..."
    run make fetch-dist VERSION="$version"

    # Verify checksums
    info "Verifying checksums..."
    if [[ "$DRY_RUN" != "true" ]]; then
        cd _dist
        if ! shasum -a 256 -c "oras_${version}_checksums.txt" --ignore-missing; then
            fatal "Checksum verification failed!"
        fi
        success "Checksums verified."
        cd ..
    fi

    # Test linux/amd64 binary
    info "Testing linux/amd64 binary version..."
    if [[ "$DRY_RUN" != "true" ]]; then
        local tarball="_dist/oras_${version}_linux_amd64.tar.gz"
        if [[ ! -f "$tarball" ]]; then
            fatal "Binary archive not found: ${tarball}"
        fi
        local tmpdir
        tmpdir=$(mktemp -d)
        tar -xzf "$tarball" -C "$tmpdir"
        local binary_version
        binary_version=$("${tmpdir}/oras" version 2>/dev/null | grep 'Version:' | awk '{print $2}' || echo "unknown")
        rm -rf "$tmpdir"

        if [[ "$binary_version" != "$version" ]]; then
            fatal "Binary version mismatch: got '${binary_version}', expected '${version}'."
        fi
        success "Binary version verified: ${binary_version}"
    fi

    success "Release v${version} validation complete."
}

# ── Phase: publish ───────────────────────────────────────────────────────────
phase_publish() {
    local version="$1"
    validate_semver "$version"

    info "Publishing release v${version}..."

    # Sign artifacts
    info "Signing artifacts..."
    run make sign

    # Verify GPG signatures
    info "Verifying GPG signatures..."
    if [[ "$DRY_RUN" != "true" ]]; then
        local failed=false
        for sig in _dist/*.asc; do
            local original="${sig%.asc}"
            if ! gpg --verify "$sig" "$original" 2>/dev/null; then
                error "GPG verification failed for: ${original}"
                failed=true
            fi
        done
        if [[ "$failed" == "true" ]]; then
            fatal "One or more GPG signature verifications failed."
        fi
        success "All GPG signatures verified."
    fi

    # Upload .asc files to GitHub release
    info "Uploading signature files to GitHub release..."
    if [[ "$DRY_RUN" != "true" ]]; then
        for sig in _dist/*.asc; do
            info "Uploading $(basename "$sig")..."
            gh release upload "v${version}" "$sig" --repo "$REPO" --clobber
        done
        success "Signature files uploaded."
    fi

    # Add signing key info to release notes
    info "Adding signing key information to release notes..."
    if [[ "$DRY_RUN" != "true" ]]; then
        local gpg_key_id
        gpg_key_id=$(gpg --list-secret-keys --keyid-format=long 2>/dev/null \
            | grep sec | head -1 | awk '{print $2}' | cut -d'/' -f2)

        local existing_notes
        existing_notes=$(gh release view "v${version}" --repo "$REPO" --json body --jq '.body')

        local signing_note
        signing_note="$(cat <<EOF

---

## Verification

The release artifacts are signed with GPG key \`${gpg_key_id}\`.

To verify a downloaded artifact:
\`\`\`bash
# Import the signing key (if not already imported)
gpg --keyserver keyserver.ubuntu.com --recv-keys ${gpg_key_id}

# Verify the signature
gpg --verify oras_${version}_linux_amd64.tar.gz.asc oras_${version}_linux_amd64.tar.gz
\`\`\`
EOF
)"
        gh release edit "v${version}" --repo "$REPO" \
            --notes "${existing_notes}${signing_note}"
        success "Release notes updated with signing key info."
    fi

    # Publish release (remove draft status)
    confirm "Publish release v${version} (remove draft status)?"
    info "Publishing release..."
    run gh release edit "v${version}" --repo "$REPO" --draft=false

    # Trigger snap workflow
    info "Triggering snap workflow..."
    if [[ "$DRY_RUN" != "true" ]]; then
        gh workflow run snap.yml --repo "$REPO" --ref "v${version}" 2>/dev/null || \
            warn "Could not trigger snap workflow. You may need to trigger it manually."
    fi

    # Clean up
    info "Cleaning up _dist/ directory..."
    run rm -rf _dist/

    success "Release v${version} published successfully!"
    echo ""
    echo -e "${BOLD}Post-release Slack Template:${NC}"
    echo "────────────────────────────────────────"
    cat <<EOF
:tada: *ORAS CLI v${version} has been released!*

*Release page:* https://github.com/${REPO}/releases/tag/v${version}
*GHCR:* \`ghcr.io/oras-project/oras:v${version}\`

Thanks to all contributors! :heart:
EOF
    echo "────────────────────────────────────────"
}

# ── Main ─────────────────────────────────────────────────────────────────────
usage() {
    cat <<EOF
Usage: $(basename "$0") [--dry-run] <phase> <version> [sha]

Phases:
  prep <version>        Prepare a release: bump version, create PR
  tag <version> <sha>   Tag a release: create and push signed tag
  validate <version>    Validate a release: verify CI, artifacts, checksums
  publish <version>     Publish a release: sign, upload, publish

Options:
  --dry-run    Print commands without executing them

Environment:
  ORAS_REMOTE  Git remote for upstream (default: upstream)

Examples:
  $(basename "$0") prep 1.3.0
  $(basename "$0") tag 1.3.0 abc1234
  $(basename "$0") validate 1.3.0
  $(basename "$0") publish 1.3.0
  $(basename "$0") --dry-run prep 1.3.0-rc.1
EOF
}

main() {
    # Parse global flags
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --dry-run)
                DRY_RUN=true
                info "Dry-run mode enabled."
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                break
                ;;
        esac
    done

    if [[ $# -lt 1 ]]; then
        usage
        exit 1
    fi

    local phase="$1"
    shift

    case "$phase" in
        prep)
            [[ $# -ge 1 ]] || { error "Usage: $0 prep <version>"; exit 1; }
            phase_prep "$1"
            ;;
        tag)
            [[ $# -ge 2 ]] || { error "Usage: $0 tag <version> <sha>"; exit 1; }
            phase_tag "$1" "$2"
            ;;
        validate)
            [[ $# -ge 1 ]] || { error "Usage: $0 validate <version>"; exit 1; }
            phase_validate "$1"
            ;;
        publish)
            [[ $# -ge 1 ]] || { error "Usage: $0 publish <version>"; exit 1; }
            phase_publish "$1"
            ;;
        *)
            error "Unknown phase: '${phase}'"
            usage
            exit 1
            ;;
    esac
}

main "$@"
