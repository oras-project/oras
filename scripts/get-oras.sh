#!/bin/sh

# This script find the release by tag, downloads it to the user's home directory
# extracts the binary to a folder in the user's home directory
# Then it creates an install-oras.sh file that you can call be . to alias the oras binary
# To put it all together you can execute it in one line as: 
# get-oras.sh;. install-oras.sh
# or if you are installing from curl: 
# curl $host/get-oras.sh | sh;. install-oras.sh

ORAS_DOWNLOAD_LINK="$1"

# Logging
log() {
    echo "[$(date --rfc-3339=seconds)]: $*"
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
ORAS_INSTALLER_NAME=${ORAS_INSTALLER_NAME:-'install-oras.sh'}

callAPI() {
    local auth=$1
    local url=$2

    if [ -z $(hash | grep curl=) ]; then
        curl $auth -s $url
    else
        failure "missing `curl`, install curl and wget"
        exit 1
    fi
}

downloadFile() {
    local url=$1
    local output=$2
    notice "Downloading file: $url"
    notice "Writing to file: $output"

    # Do we have wget?
    if [ -z $(hash | grep wget=) ]; then
        wget --no-verbose --https-only --wait=15 --limit-rate=500k $url -O $output
    else
        failure "missing `wget`, install wget (and curl)"
        exit 1
    fi
}

getReleaseDownloadLink() {
    local repo=$1
    local tag=$2
    local os=$3
    local arch=$4
    local release=$os"_"$arch
    local search="\"name\": \"(.*)$release.tar.gz\""
    local assetid=$(callAPI $BASIC_AUTH https://api.github.com/repos/$repo/releases/tags/$tag | grep -C 2 -E "\"name\": \"(.*)$release.tar.gz\"" | awk -F '"id": ' '{print $2}' | awk -F ',' '{print $1}')

    # If we couldn't find anything then return 1 to exit the script
    if [ -z $assetid ]; then
        failure "Could not find release asset for $release"
        return 1
    fi

    ORAS_DOWNLOAD_LINK=$(callAPI $BASIC_AUTH "https://api.github.com/repos/$repo/releases/assets/$assetid" | grep -E '"browser_download_url": "(.*)"' | awk -F ' ' '{print $2}' | awk -F '"' '{print $2}' | awk -F '"' '{print $1}')
}

# Create an installer so that you can do `./get-oras | sh & . ./install-oras.sh`
createInstaller() {
cat <<EOF > $ORAS_INSTALLER_NAME
#!/bin/sh

alias oras="$ORAS_RELEASE_LOCATION"
oras
EOF
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

  log "Extracting to: $ORAS_RELEASE_DIR"
  mkdir -p $ORAS_RELEASE_DIR
  tar -xvzf $ARCHIVE_NAME -C $ORAS_RELEASE_DIR

  log "Creating installer for: $ORAS_RELEASE_DIR -> $ORAS_INSTALLER_NAME"
  createInstaller
  chmod +x $ORAS_INSTALLER_NAME

  notice 'To complete installation execute: `. install-oras.sh`'
}

main

exit $MAIN_EXIT_CODE
