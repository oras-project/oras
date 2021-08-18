#!/bin/bash

# This script find the release by tag, downloads it to the user's home directory
# extracts the binary to a folder in the user's home directory
# then tries to link that binary to /usr/local/bin, if it fails to do that
# it will try to link to $GOPATH/bin instead

# Logging
LOGFILE=get-oras.sh.log
RETAIN_NUM_LINES=10

logsetup() {
    TMP=$(tail -n $RETAIN_NUM_LINES $LOGFILE 2>/dev/null) && echo "${TMP}" >$LOGFILE
    exec > >(tee -a $LOGFILE)
    exec 2>&1
}

log() {
    echo -e "[$(date --rfc-3339=seconds)]: $*"
}

failure() {
    local RED='\033[0;31m'
    local NC='\033[0m' # No Color
    log "${RED}$*${NC}"
}

notice() {
    local CYAN='\033[0;36m'
    local NC='\033[0m' # No Color
    log "${CYAN}$*${NC}"
}

# Setup logging for troubleshooting
logsetup

# Search Parameters
REPO='oras-project/oras'
TAG='v0.11.0' # TODO: This tag will need to change later
OS='linux'    # TODO: Can add OS/Architecture detection later
ARCH='amd64'
GITHUB_USERNAME=''
GITHUB_PASSWORD=''
BASIC_AUTH="-u $GITHUB_USERNAME:"
BASIC_AUTH_WGET="--http-user=$GITHUB_USERNAME --http-password="

# NOTE:
# Github has rate-limiting set on their API's so to get around that you need to set your github username
# when using their API. In most cases this is fine, but when working on the script this is important to have
if [[ -z $GITHUB_USERNAME ]]; then
    BASIC_AUTH=''
    BASIC_AUTH_WGET=''

    log "GITHUB_USERNAME is not set, removing basic authentication parameters"
    log "(Note: If you are rate-limited, add a value to that variable before launching the script.)"
fi

# Script Paramters
# TODO We can check if this variables are already set before setting them
ARCHIVE_NAME="oras_$TAG_$OS_$ARCH.tar.gz"
ORAS_RELEASE_DIR="$HOME/oras-releases/$TAG"
ORAS_INSTALL_DIR="/usr/local/bin"
ORAS_FILE_NAME="oras"
ORAS_RELEASE_LOCATION="$ORAS_RELEASE_DIR/$ORAS_FILE_NAME"

getReleaseDownloadLink() {
    local repo=$1
    local tag=$2
    local os=$3
    local arch=$4
    local releases=$(curl $BASIC_AUTH -s https://api.github.com/repos/$repo/releases/tags/$tag | grep -B 3 -E '"name":.*gz"')

    # Notes:
    # Architectures and Assets will alternate between field and value, so match the name you're looking for
    # then get the asset id from the assets array
    local architectures=($(echo "$releases" | grep -E '"name": "(.*)"'))
    local assets=($(echo "$releases" | grep -E '"id":(.*)'))

    for i in "${!architectures[@]}"; do
        if [[ "${architectures[$i]}" = *$os* && "${architectures[$i]}" = *$arch* ]]; then
            if [[ "${assets[i]}" =~ ([0-9]+) ]]; then
                local assetid="${BASH_REMATCH[0]}"
                notice "Asset id found: $assetid"
            fi
        fi
    done

    # If we couldn't find anything then return 1 to exit the script
    if [[ -z $assetid ]]; then
        failure "Could not find release asset for $os_$arch"
        return 1
    fi

    local downloadurl=($(curl $BASIC_AUTH -s "https://api.github.com/repos/$repo/releases/assets/$assetid" | grep -E '"browser_download_url": "(.*)"'))

    ORAS_DOWNLOAD_LINK=$(echo "${downloadurl[1]}" | awk -F '"' '{print $2}')
}

# Search for the download link from the github repo
if [[ -z $ORAS_DOWNLOAD_LINK ]]; then
    log "Getting download link for $REPO release ($TAG $OS $ARCH)"
    getReleaseDownloadLink $REPO $TAG $OS $ARCH
fi

log "Downloading oras from: $ORAS_DOWNLOAD_LINK"

notice "This will take a moment to begin"
wget --https-only --wait=15 --limit-rate=500k $ORAS_DOWNLOAD_LINK -O $ARCHIVE_NAME

log "Extracting to $ORAS_RELEASE_DIR"
mkdir -p $ORAS_RELEASE_DIR
tar -xvzf $ARCHIVE_NAME -C $ORAS_RELEASE_DIR

log "Linking $ORAS_RELEASE_LOCATION to $ORAS_INSTALL_DIR"
ln -s $ORAS_RELEASE_LOCATION $ORAS_INSTALL_DIR

# Error handling
EXIT_CODE=0
if [[ -z $(ls $ORAS_INSTALL_DIR | grep oras) ]]; then
    failure "Failed to link oras to $ORAS_INSTALL_DIR, trying fallback"

    # If we have go installed, we could try to put it in the go bin folder instead
    GOPATH="$(echo "$(go env)" | grep GOPATH | awk -F '=' '{print $2}' | awk -F '"' '{print $2}')"
    ORAS_SECONDARY_INSTALL_DIR="$GOPATH/bin"

    if [[ -n $GOPATH ]]; then
        log "Trying to link to: $ORAS_SECONDARY_INSTALL_DIR instead"
        ln -s $ORAS_RELEASE_LOCATION $ORAS_SECONDARY_INSTALL_DIR
        if [[ -z $(ls $ORAS_SECONDARY_INSTALL_DIR| grep oras) ]]; then
            failure "Failed to link oras to secondary install directory $ORAS_SECONDARY_INSTALL_DIR"
            EXIT_CODE=1
        fi
        log "Linked to $ORAS_SECONDARY_INSTALL_DIR/oras"
    fi

    if [[ $EXIT_CODE -eq 1 ]]; then 
        failure "Could not install oras this is probably due to permission issues, to complete installation manually link/copy $ORAS_RELEASE_LOCATION to $ORAS_INSTALL_DIR"
    fi
fi

notice "If you've experienced any issues with this script, please attach $LOGFILE in your github issue."

exit $EXIT_CODE
