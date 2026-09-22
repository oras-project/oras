# ORAS CLI

[![Build Status](https://github.com/oras-project/oras/actions/workflows/build.yml/badge.svg?event=push)](https://github.com/oras-project/oras/actions/workflows/build.yml?query=workflow%3Abuild+event%3Apush)
[![codecov](https://codecov.io/gh/oras-project/oras/branch/main/graph/badge.svg)](https://codecov.io/gh/oras-project/oras)
[![Go Report Card](https://goreportcard.com/badge/oras.land/oras)](https://goreportcard.com/report/oras.land/oras)


<p style="text-align: left;">
<a href="https://oras.land/"><img src="https://oras.land/img/oras.svg" alt="banner" width="100px"></a>
</p>

## Docs

Documentation for the ORAS CLI is located on
the project website: [oras.land/cli](https://oras.land/docs/category/oras-commands)

## Container images

Release images are published for multiple architectures to
[GitHub Container Registry](https://ghcr.io/oras-project/oras).

- `:main` tracks the latest merge on the default branch.
- `:vX.Y.Z` is the immutable version tag for a release.
- `:vX.Y` tracks the latest stable patch in a minor release line.
- `:vX` tracks the latest stable release in a major release line.
- `:latest` tracks the highest stable release across all major versions.

Prereleases never move rolling tags. A maintenance-branch release only moves
the rolling tags for its own minor and major lines when it is newer than the
current tag in that line. This keeps `:latest` and the bounded version tags
from moving backwards after a backport.

## Development Environment Setup

Refer to the [development guide](https://oras.land/community/developer_guide) to get started [contributing to ORAS](https://oras.land/community/contributing_guide).

## Code of Conduct

This project has adopted the [CNCF Code of Conduct](https://github.com/cncf/foundation/blob/master/code-of-conduct.md). See [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) for further details.

