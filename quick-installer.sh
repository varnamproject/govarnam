#!/usr/bin/env bash

set -e

# Varnam Installer

# Make a temp dir
workDir=`mktemp -d -t "varnam-installerXXXX"`

# Credits: https://gist.github.com/lukechilds/a83e1d7127b78fef38c2914c4ececc3c
get_latest_release() {
  curl --silent "https://api.github.com/repos/varnamproject/$1/releases/latest" |
    grep '"tag_name":' |
    sed -E 's/.*"([^"]+)".*/\1/'
}

version=0
versionNumber=0
schemesVersion=0
imeVersion=0
imeVersionNumber=0
arch=$(uname -m)

confirm() {
  [[ "$1" == [yY] || "$1" == [yY][eE][sS] ]]
}

init_version() {
  version=$(get_latest_release "govarnam")
  schemesVersion=$(get_latest_release "schemes")
  imeVersion=$(get_latest_release "govarnam-ibus")
  if [ -z $version ] || [ -z $schemesVersion ] || [ -z $imeVersion ]; then
    echo "Couldn't find latest Varnam version. Possible reasons:"
    echo "1. No internet connection"
    echo "2. GitHub API Rate Limit (wait an hour for the rate limit to expire)"
    exit 1
  fi
  versionNumber=${version/v/}
  imeVersionNumber=${imeVersion/v/}
}

install_govarnam() {
  cd $workDir
  releaseName="govarnam-$versionNumber-$arch"
  url="https://github.com/varnamproject/govarnam/releases/download/$version/$releaseName.zip"
  echo "Downloading $releaseName from $url"
  curl -L -o govarnam.zip "$url"

  unzip govarnam.zip
  echo "Installing $releaseName"
  cd $releaseName

  if [ -f /usr/local/lib/libgovarnam.so ]; then
    read -p "Found an existing GoVarnam installation. Replace it ? (yes/NO): " answer2
    if confirm "$answer2"; then
      sudo rm /usr/local/lib/libgovarnam.so*
    else
      echo "Not installing $releaseName"
    fi
  fi

  ./install.sh
}

step1="Step 1: Install GoVarnam"
step2="Step 2: Install your language support for GoVarnam"
step3="Step 3: Install Varnam IBus Engine"

echo "Welcome to Varnam Installer. https://varnamproject.github.io/"
echo ""
echo "This installation is a 3-step process."
echo ""
echo $step1
echo $step2
echo $step3
echo ""
read -p "Start Step 1 ? (yes/NO): " answer

init_version
if confirm "$answer"; then
  install_govarnam
fi

langs=""
list_schemes() {
  assetsURL=$(curl --silent 'https://api.github.com/repos/varnamproject/schemes/releases/latest' |
    grep '"assets_url":' |
    sed -E 's/.*"([^"]+)".*/\1/')
  langs=$(curl --silent $assetsURL |
    grep -E 'name(.*?).zip' |
    sed -E 's/.*"([^"]+)".*/\1/' |
    sed s/.zip//)
  echo $"$langs"
  echo "---"
  echo "all"
}

install_scheme() {
  cd $workDir
  schemeID="$1"
  releaseName="$schemeID"
  url="https://github.com/varnamproject/schemes/releases/download/$schemesVersion/$releaseName.zip"
  echo "Downloading $releaseName from $url"
  curl -L -o "$releaseName.zip" "$url"

  unzip "$releaseName.zip"
  echo "Installing $releaseName"

  cd $releaseName
  ./install.sh

  if ls */*.vlf >/dev/null 2>&1; then
    # At least 1 file
    read -p "Found Varnam Learnings File (.vlf) to import words from. Import for '$schemeID' ? (yes/no): " answer2
    if confirm "$answer2"; then
      ./import.sh
    fi
  fi
}

echo ""
echo $step2
echo ""
list_schemes
echo ""
read -p "Which language would you like to install ? (Separate by comma if there are multiple): " answer

if [[ "$answer" == "all" ]]; then
  for lang in $langs; do
    # Trim whitespaces
    lang=`echo $lang | sed 's/ *$//g'`

    echo "Setup $lang"
    install_scheme "$lang"
  done
else
  for lang in ${answer//,/ }; do
    # Trim whitespaces
    lang=`echo $lang | sed 's/ *$//g'`

    echo "Setup $lang"
    install_scheme "$lang"
  done
fi

install_govarnam_ibus_engine() {
  cd $workDir
  releaseName="varnam-ibus-engine-$imeVersionNumber-$arch"
  url="https://github.com/varnamproject/govarnam-ibus/releases/download/$imeVersion/$releaseName.zip"
  echo "Downloading $releaseName from $url"
  curl -L -o "$releaseName.zip" "$url"

  unzip "$releaseName.zip"
  echo "Installing $releaseName"
  cd $releaseName
  ./install.sh
}

echo ""
echo $step3
echo ""
read -p "Start Step 3 ? (yes/NO): " answer

if confirm "$answer"; then
  install_govarnam_ibus_engine
fi

echo ""
echo "-----------------------------"
echo "Varnam Installation Finished!"
echo ""
echo "Log Out & Log In again for changes to take effect !!"
echo ""
echo "Getting Started: https://varnamproject.github.io/docs/getting-started/"
echo ""
echo "For help contact:"
echo "Telegram Group: https://t.me/varnamproject"
echo "Matrix Group: https://matrix.to/#/#varnamproject:poddery.com"
echo ""
echo "Website: https://varnamproject.github.io"
echo "-----------------------------"
