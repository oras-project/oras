#!/bin/sh

# This script find the release by tag, downloads it to the user's home directory
# extracts the binary to a folder in the user's home directory
# then tries to link that binary to /usr/local/bin, if it fails to do that
# it will try to link to $GOPATH/bin instead

ORAS_DOWNLOAD_LINK="$1"

# Logging
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
set -e -o nolog
set --

# Search Parameters
REPO=${REPO:-'oras-project/oras'}
TAG=${TAG:-'v0.11.0'} # TODO: This tag will need to change later
OS=${OS:-'linux'}    # TODO: Can add OS/Architecture detection later
ARCH=${ARCH:-'amd64'}
GITHUB_USERNAME=${GITHUB_USERNAME:-''}
GITHUB_PASSWORD=${GITHUB_PASSWORD:-''}
BASIC_AUTH=${BASIC_AUTH:-"-u $GITHUB_USERNAME:"}
BASIC_AUTH_WGET=${BASIC_AUTH_WGET:-"--http-user=$GITHUB_USERNAME --http-password="}

# NOTE:
# Github has rate-limiting set on their API's so to get around that you need to set your github username
# when using their API. In most cases this is fine, but when working on the script this is important to have
if [ -z $GITHUB_USERNAME ]; then
    BASIC_AUTH=''
    BASIC_AUTH_WGET=''

    log "GITHUB_USERNAME is not set, removing basic authentication parameters"
    log "(Note: If you are rate-limited, add a value to that variable before launching the script.)"
fi

# Script Paramters
ARCHIVE_NAME=${ARCHIVE_NAME:-"oras_$TAG_$OS_$ARCH.tar.gz"}
ORAS_RELEASE_DIR=${ORAS_RELEASE_DIR:-"$HOME/oras-releases/$TAG"}
ORAS_INSTALL_DIR=${ORAS_INSTALL_DIR:-"/usr/local/bin"}
ORAS_FILE_NAME=${ORAS_FILE_NAME:-"oras"}
ORAS_RELEASE_LOCATION=${ORAS_RELEASE_LOCATION:-"$ORAS_RELEASE_DIR/$ORAS_FILE_NAME"}

callAPI() {
    local auth=$1
    local url=$2
    notice "calling api: $url"

    # Do we have curl? 
    if [ -z $(hash | grep curl=) ]; then
        curl $auth -s $url
    else
        notice "missing `curl`, installing"
        apt install curl
        callAPI $1 $2
    fi
}

downloadFile() {
    local url=$1
    local output=$2
    notice "downloading file: $url, writing to: $output"

    # Do we have wget?
    if [ -z $(hash | grep wget=) ]; then
        wget --no-verbose --https-only --wait=15 --limit-rate=500k $url -O $output
    else
        notice "missing `wget`, installing"
        apt install wget
        downloadFile $1 $2
    fi
}

getReleaseDownloadLink() {
    local repo=$1
    local tag=$2
    local os=$3
    local arch=$4
    local releases=$(callAPI $BASIC_AUTH https://api.github.com/repos/$repo/releases/tags/$tag | grep -B 3 -E '"name":.*gz"')

    # Notes:
    # Architectures and Assets will alternate between field and value, so match the name you're looking for
    # then get the asset id from the assets array
    local architectures=$(echo "$releases" | grep -E '"name": "(.*)"')
    local assets=$(echo "$releases" | grep -E '"id":(.*)')

    for i in "${!architectures[@]}"; do
        if [ "${architectures[$i]}" = *$os* && "${architectures[$i]}" = *$arch* ]; then
            if [ "${assets[i]}" =~ ([0-9]+) ]; then
                local assetid="${BASH_REMATCH[0]}"
                log "Asset id found: $assetid"
            fi
        fi
    done

    # If we couldn't find anything then return 1 to exit the script
    if [ -z $assetid ]; then
        failure "Could not find release asset for $os_$arch"
        return 1
    fi

    local downloadurl=($(callAPI $BASIC_AUTH "https://api.github.com/repos/$repo/releases/assets/$assetid" | grep -E '"browser_download_url": "(.*)"'))

    ORAS_DOWNLOAD_LINK=$(echo "${downloadurl[1]}" | awk -F '"' '{print $2}')
}

MAIN_EXIT_CODE=0
main() {
  # Search for the download link from the github repo
  if [ -z $ORAS_DOWNLOAD_LINK ]; then
    log "Getting download link for $REPO release ($TAG $OS $ARCH)"
    getReleaseDownloadLink $REPO $TAG $OS $ARCH
  fi

  notice "This will take a moment to begin"
  downloadFile $ORAS_DOWNLOAD_LINK $ARCHIVE_NAME

  log "Extracting to, $ORAS_RELEASE_DIR"
  mkdir -p $ORAS_RELEASE_DIR
  tar -xvzf $ARCHIVE_NAME -C $ORAS_RELEASE_DIR

  log "Linking, $ORAS_RELEASE_LOCATION to $ORAS_INSTALL_DIR"
  ln -s $ORAS_RELEASE_LOCATION $ORAS_INSTALL_DIR

  # Check if we were able to install oras
  if [ -z $(ls $ORAS_INSTALL_DIR/$TAG) ]; then
    MAIN_EXIT_CODE=1
    failure "Failed to link oras to $ORAS_INSTALL_DIR"
    
  fi

  notice "If you've experienced any issues with this script, please attach $LOGFILE in your github issue."
}

main

exit $MAIN_EXIT_CODE
