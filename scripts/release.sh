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

# ORAS Release Helper Script
# Usage: scripts/release.sh <phase> <version> [args...]
# Phases: prep, tag, validate, publish

REPO="oras-project/oras"
VERSION_FILE="internal/version/version.go"
REMOTE="${ORAS_REMOTE:-upstream}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info()    { echo -e "${CYAN}[INFO]${NC} $*"; }
success() { echo -e "${GREEN}[OK]${NC} $*"; }
warn()    { echo -e "${YELLOW}[MANUAL]${NC} $*"; }
error()   { echo -e "${RED}[ERROR]${NC} $*" >&2; }

DRY_RUN=false

# Parse global flags
parse_global_flags() {
    local args=()
    for arg in "$@"; do
        case "$arg" in
            --dry-run) DRY_RUN=true ;;
            *) args+=("$arg") ;;
        esac
    done
    echo "${args[@]:-}"
}

run() {
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] $*"
        return 0
    fi
    "$@"
}

confirm() {
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would confirm: $1"
        return 0
    fi
    echo -en "${YELLOW}$1 [y/N]: ${NC}"
    read -r response
    case "$response" in
        [yY][eE][sS]|[yY]) return 0 ;;
        *) error "Aborted."; exit 1 ;;
    esac
}

# Validate semver format
validate_version() {
    local version="$1"
    if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$ ]]; then
        error "Invalid version format: $version (expected semver like 1.2.3 or 1.2.3-rc.1)"
        exit 1
    fi
}

# Check if this is a new minor version (patch == 0, no pre-release)
is_new_minor() {
    local version="$1"
    local patch
    patch=$(echo "$version" | cut -d. -f3 | cut -d- -f1)
    if [[ "$patch" == "0" && ! "$version" =~ - ]]; then
        return 0
    fi
    return 1
}

# Get major.minor from version
get_major_minor() {
    local version="$1"
    echo "$version" | cut -d. -f1-2
}

###############################################################################
# Check prerequisites (gh, gpg, remote)
###############################################################################
check_prerequisites() {
    info "Checking prerequisites..."
    if ! command -v gh &>/dev/null; then
        error "'gh' CLI not found. Install: https://cli.github.com"
        exit 1
    fi
    if ! gh auth status &>/dev/null; then
        error "'gh' CLI not authenticated. Run: gh auth login"
        exit 1
    fi
    if ! command -v gpg &>/dev/null; then
        error "'gpg' not found. Install GPG."
        exit 1
    fi
    if ! gpg --list-secret-keys --keyid-format LONG 2>/dev/null | grep -q sec; then
        error "No GPG secret keys found. Import or generate a key."
        exit 1
    fi
    if ! git remote get-url "$REMOTE" &>/dev/null; then
        error "Git remote '${REMOTE}' not found. Set ORAS_REMOTE or add the remote."
        exit 1
    fi
}

###############################################################################
# Phase 1: Prep
###############################################################################
do_prep() {
    local version="$1"
    validate_version "$version"
    info "Preparing release v${version}"

    check_prerequisites
    success "Prerequisites OK"

    # Check we're on main and up to date
    local current_branch
    current_branch=$(git branch --show-current)
    if [ "$current_branch" != "main" ]; then
        warn "Currently on branch '$current_branch', not 'main'."
        confirm "Continue anyway?"
    fi

    # Ensure working tree is clean
    if ! git diff --quiet || ! git diff --cached --quiet; then
        error "Working tree is not clean. Commit or stash changes before running prep."
        exit 1
    fi

    # Update version file
    info "Updating ${VERSION_FILE}..."
    sed -i.bak -E "s/Version = \"[^\"]+\"/Version = \"${version}\"/" "$VERSION_FILE"
    sed -i.bak -E "s/BuildMetadata = \"[^\"]*\"/BuildMetadata = \"\"/" "$VERSION_FILE"
    rm -f "${VERSION_FILE}.bak"
    success "Version set to ${version}"

    # Show the diff
    git diff "$VERSION_FILE"

    # Create branch, commit, push, PR
    local branch="chore/release-v${version}"
    info "Creating branch ${branch}..."
    run git checkout -b "$branch"
    run git add "$VERSION_FILE"
    run git commit -m "bump: tag and release ORAS CLI v${version}"
    run git push "${REMOTE}" "$branch"

    info "Creating pull request..."
    local pr_url
    if [ "$DRY_RUN" = true ]; then
        pr_url="https://github.com/${REPO}/pull/DRY-RUN"
        info "[DRY-RUN] Would create PR"
    else
        pr_url=$(gh pr create \
            --repo "$REPO" \
            --title "bump: tag and release ORAS CLI v${version}" \
            --body "$(cat <<EOF
## Release v${version}

This PR bumps the version to v${version} for the upcoming release.

### Checklist
- [ ] Version updated in \`internal/version/version.go\`
- [ ] CI checks pass
- [ ] Vote called in Slack
- [ ] Vote passed (3+ binding votes, no vetoes)
EOF
)" \
            --head "$branch" \
            --base main)
    fi
    success "PR created: ${pr_url}"

    # Get the commit SHA
    local sha
    sha=$(git rev-parse HEAD)
    echo ""
    success "Release commit SHA: ${sha}"

    # Print Slack vote template
    echo ""
    warn "Copy the following message to Slack #oras to call for a vote:"
    echo ""
    echo -e "${CYAN}---${NC}"
    cat <<EOF
:ballot_box: *[VOTE] Release ORAS CLI v${version}*

Hi everyone, I'd like to call a vote for the release of ORAS CLI v${version}.

*Release PR:* ${pr_url}
*Release commit:* \`${sha}\`

Please review the PR and vote:
  :+1: (binding) — approve
  :-1: (binding) — veto (please provide reason)

The vote will remain open for at least 72 hours.
A minimum of 3 binding +1 votes and no vetoes is required.
EOF
    echo -e "${CYAN}---${NC}"
    echo ""
    warn "After the vote passes and the PR is merged, run:"
    echo "  scripts/release.sh tag ${version} <merged-commit-sha>"
}

###############################################################################
# Phase 2: Tag
###############################################################################
do_tag() {
    local version="$1"
    local sha="${2:-}"
    validate_version "$version"

    if [ -z "$sha" ]; then
        error "Usage: scripts/release.sh tag <version> <commit-sha>"
        exit 1
    fi

    info "Tagging v${version} at ${sha}"

    check_prerequisites
    success "Prerequisites OK"

    # Verify the commit exists and is on main
    if ! git cat-file -t "$sha" &>/dev/null; then
        error "Commit ${sha} not found. Did you fetch latest?"
        exit 1
    fi

    # Check that the version file at that commit has the right version
    local file_version
    file_version=$(git show "${sha}:${VERSION_FILE}" | grep 'Version = ' | sed -E 's/.*"([^"]+)".*/\1/')
    if [ "$file_version" != "$version" ]; then
        error "Version in ${VERSION_FILE} at ${sha} is '${file_version}', expected '${version}'"
        exit 1
    fi

    # Create signed tag
    confirm "Create signed tag v${version} at ${sha}?"
    run git tag -s "v${version}" "$sha" -m "Release v${version}"
    success "Tag v${version} created"

    # Push tag
    confirm "Push tag v${version} to ${REMOTE}?"
    run git push "${REMOTE}" "v${version}"
    success "Tag pushed"

    # Create release branch for new minor versions
    if is_new_minor "$version"; then
        local release_branch="release-$(get_major_minor "$version")"
        info "New minor version detected. Creating release branch ${release_branch}..."
        run git branch "$release_branch" "$sha"
        confirm "Push release branch ${release_branch} to ${REMOTE}?"
        run git push "${REMOTE}" "$release_branch"
        success "Release branch ${release_branch} pushed"
    fi

    echo ""
    info "CI workflows triggered. Monitor at:"
    echo "  https://github.com/${REPO}/actions?query=event%3Apush+branch%3Av${version}"
    echo ""
    warn "Once CI completes, run:"
    echo "  scripts/release.sh validate ${version}"
}

###############################################################################
# Phase 3: Validate
###############################################################################
do_validate() {
    local version="$1"
    validate_version "$version"
    info "Validating release v${version}"

    # Wait for CI workflows
    info "Checking CI workflow status..."
    local workflows=("release-ghcr" "release-github")
    local all_done=false
    local max_attempts=60  # 30 minutes at 30s intervals
    local attempt=0

    while [ "$all_done" != "true" ] && [ "$attempt" -lt "$max_attempts" ]; do
        all_done=true
        for wf in "${workflows[@]}"; do
            local status
            status=$(gh run list \
                --repo "$REPO" \
                --workflow "$wf" \
                --branch "v${version}" \
                --limit 1 \
                --json status,conclusion \
                --jq '.[0] | "\(.status) \(.conclusion)"' 2>/dev/null || echo "not_found")

            if [[ "$status" == *"completed success"* ]]; then
                success "Workflow ${wf}: completed successfully"
            elif [[ "$status" == *"completed"* ]]; then
                error "Workflow ${wf}: ${status}"
                exit 1
            elif [[ "$status" == "not_found" ]]; then
                warn "Workflow ${wf}: not found yet"
                all_done=false
            else
                info "Workflow ${wf}: ${status} (waiting...)"
                all_done=false
            fi
        done

        if [ "$all_done" != "true" ]; then
            ((attempt++))
            if [ "$attempt" -lt "$max_attempts" ]; then
                info "Waiting 30s for workflows... (attempt ${attempt}/${max_attempts})"
                sleep 30
            fi
        fi
    done

    if [ "$all_done" != "true" ]; then
        error "Timed out waiting for workflows"
        exit 1
    fi
    success "All CI workflows completed"

    # Fetch distribution artifacts via gh to support draft releases
    info "Fetching distribution artifacts..."
    run mkdir -p _dist
    run gh release download "v${version}" \
        --repo "$REPO" \
        --dir _dist/ \
        --pattern "oras_${version}_*" \
        --clobber
    success "Artifacts downloaded to _dist/"

    # Verify checksums
    info "Verifying checksums..."
    if command -v sha256sum >/dev/null 2>&1; then
        (cd _dist && sha256sum -c "oras_${version}_checksums.txt" --ignore-missing)
    elif command -v shasum >/dev/null 2>&1; then
        (cd _dist && shasum -a 256 -c "oras_${version}_checksums.txt" --ignore-missing)
    else
        error "Neither sha256sum nor shasum is available; cannot verify checksums."
        exit 1
    fi
    success "Checksums verified"

    # Test binary for the current OS/arch
    local os arch artifact
    case "$(uname -s)" in
        Darwin) os="darwin" ;;
        Linux)  os="linux" ;;
        *)      os="linux" ;;
    esac
    case "$(uname -m)" in
        x86_64)  arch="amd64" ;;
        arm64|aarch64) arch="arm64" ;;
        *)       arch="amd64" ;;
    esac
    artifact="${os}_${arch}"
    info "Testing ${os}/${arch} binary..."
    local tmpdir
    tmpdir=$(mktemp -d)
    tar -xzf "_dist/oras_${version}_${artifact}.tar.gz" -C "$tmpdir"
    local bin_version
    bin_version=$("$tmpdir/oras" version 2>/dev/null | head -1 || echo "unknown")
    rm -rf "$tmpdir"

    if echo "$bin_version" | grep -q "$version"; then
        success "Binary version matches: ${bin_version}"
    else
        error "Binary version mismatch. Expected '${version}', got: ${bin_version}"
        exit 1
    fi

    # Check git commit in binary
    local tag_sha
    tag_sha=$(git rev-list -n 1 "v${version}" 2>/dev/null || echo "unknown")
    if echo "$bin_version" | grep -q "${tag_sha:0:7}"; then
        success "Binary git commit matches tag SHA"
    else
        warn "Could not verify git commit in binary output (non-fatal)"
    fi

    echo ""
    success "All validation checks passed!"
    echo ""
    warn "Review the release notes at:"
    echo "  https://github.com/${REPO}/releases/tag/v${version}"
    echo ""
    warn "When ready, run:"
    echo "  scripts/release.sh publish ${version}"
}

###############################################################################
# Phase 4: Sign & Publish
###############################################################################
do_publish() {
    local version="$1"
    validate_version "$version"
    info "Publishing release v${version}"

    # Check artifacts exist
    if [ ! -d "_dist" ]; then
        error "_dist/ directory not found. Run 'scripts/release.sh validate ${version}' first."
        exit 1
    fi

    # Sign artifacts
    info "Signing artifacts with GPG..."
    run make SHELL=/bin/bash sign
    success "Artifacts signed"

    # Verify signatures
    info "Verifying GPG signatures..."
    local sig_count=0
    for asc in _dist/*.asc; do
        [ -f "$asc" ] || continue
        local orig="${asc%.asc}"
        if gpg --verify "$asc" "$orig" 2>/dev/null; then
            success "Verified: $(basename "$asc")"
            ((sig_count++))
        else
            error "Signature verification failed: $(basename "$asc")"
            exit 1
        fi
    done
    if [ "$sig_count" -eq 0 ]; then
        error "No .asc signature files found in _dist/"
        exit 1
    fi
    success "All ${sig_count} signatures verified"

    # Upload signatures to GitHub release
    info "Uploading signatures to GitHub release..."
    confirm "Upload ${sig_count} .asc files to release v${version}?"
    if [[ "$DRY_RUN" == "true" ]]; then
        info "[DRY-RUN] Would upload ${sig_count} .asc files to release v${version}"
    else
        gh release upload "v${version}" _dist/*.asc --repo "$REPO"
    fi
    success "Signatures uploaded"

    # Get GPG key fingerprint for release notes
    local fingerprint
    fingerprint=$(gpg --list-secret-keys --keyid-format LONG 2>/dev/null | grep sec | head -1 | awk '{print $2}' | cut -d/ -f2)
    if [ -n "$fingerprint" ]; then
        info "Adding signing key info to release notes..."
        local note="## Verification\n\nAll release artifacts are signed with GPG key \`${fingerprint}\`. Signatures (\`.asc\` files) are attached to this release.\n\nPublic keys are available in the [KEYS](https://github.com/${REPO}/blob/main/KEYS) file."
        if [ "$DRY_RUN" = true ]; then
            info "[DRY-RUN] Would append signing info to release notes"
        else
            local existing_notes
            existing_notes=$(gh release view "v${version}" --repo "$REPO" --json body --jq '.body')
            gh release edit "v${version}" --repo "$REPO" --notes "$(printf '%s\n\n%b' "$existing_notes" "$note")"
        fi
        success "Release notes updated with signing key info"
    fi

    # Publish release
    confirm "Publish release v${version} (remove draft status)?"
    run gh release edit "v${version}" --repo "$REPO" --draft=false
    success "Release v${version} published!"

    # Trigger snap workflow
    info "Triggering snap workflow..."
    local is_stable="true"
    if [[ "$version" == *"-"* ]]; then
        is_stable="false"
    fi
    if [ "$DRY_RUN" = true ]; then
        info "[DRY-RUN] Would trigger release-snap.yml with version=v${version} isStable=${is_stable}"
    else
        gh workflow run release-snap.yml \
            --repo "$REPO" \
            --ref "v${version}" \
            --field version="v${version}" \
            --field isStable="${is_stable}" 2>/dev/null \
        && success "Snap workflow triggered" \
        || warn "Could not trigger snap workflow. You may need to trigger it manually."
    fi

    # Clean up
    confirm "Remove _dist/ directory?"
    run rm -rf _dist
    success "Cleaned up _dist/"

    echo ""
    success "Release v${version} is live!"
    echo "  https://github.com/${REPO}/releases/tag/v${version}"
    echo ""
    warn "Post-release steps (manual):"
    echo "  1. Update oras-www docs if needed"
    echo "  2. Announce in Slack #oras:"
    echo ""
    echo -e "${CYAN}---${NC}"
    cat <<EOF
:tada: *ORAS CLI v${version} Released!*

We're excited to announce the release of ORAS CLI v${version}!

*Release:* https://github.com/${REPO}/releases/tag/v${version}
*Changelog:* https://github.com/${REPO}/releases/tag/v${version}

Thank you to all contributors!
EOF
    echo -e "${CYAN}---${NC}"
}

###############################################################################
# Main
###############################################################################
usage() {
    cat <<EOF
Usage: scripts/release.sh [--dry-run] <phase> <version> [args...]

Phases:
  prep <version>              Bump version, create PR, print vote template
  tag <version> <commit-sha>  Create and push signed tag
  validate <version>          Wait for CI, download and verify artifacts
  publish <version>           Sign, upload signatures, publish release

Flags:
  --dry-run                   Print actions without executing them

Environment:
  ORAS_REMOTE                 Git remote name (default: upstream)

Examples:
  scripts/release.sh prep 1.3.0
  scripts/release.sh tag 1.3.0 abc1234
  scripts/release.sh validate 1.3.0
  scripts/release.sh publish 1.3.0
  scripts/release.sh --dry-run prep 1.3.0-rc.1
EOF
}

main() {
    local args
    args=$(parse_global_flags "$@")
    # Re-split args
    set -- $args

    local phase="${1:-}"
    local version="${2:-}"

    if [ -z "$phase" ]; then
        usage
        exit 1
    fi

    # Ensure we're in the repo root
    cd "$(git rev-parse --show-toplevel)"

    case "$phase" in
        prep)
            [ -z "$version" ] && { error "Usage: scripts/release.sh prep <version>"; exit 1; }
            do_prep "$version"
            ;;
        tag)
            [ -z "$version" ] && { error "Usage: scripts/release.sh tag <version> <sha>"; exit 1; }
            do_tag "$version" "${3:-}"
            ;;
        validate)
            [ -z "$version" ] && { error "Usage: scripts/release.sh validate <version>"; exit 1; }
            do_validate "$version"
            ;;
        publish)
            [ -z "$version" ] && { error "Usage: scripts/release.sh publish <version>"; exit 1; }
            do_publish "$version"
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            error "Unknown phase: ${phase}"
            usage
            exit 1
            ;;
    esac
}

main "$@"
