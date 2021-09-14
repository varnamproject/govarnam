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
arch=$(arch)

confirm() {
  [[ "$1" == [yY] || "$1" == [yY][eE][sS] ]]
}

init_version() {
  version=$(get_latest_release "govarnam")
  if [[ -z $version ]]; then
    echo "Couldn't find latest version. Possible reason: GitHub API Rate Limit"
    exit 1
  fi
  versionNumber=${version/v/}
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
  ./install.sh
}

step1="Step 1: Install GoVarnam"
step2="Step 2: Install your language"
step3="Step 3: Install Varnam IBus Engine"

echo "Welcome to Varnam Installer. This installation is a 3-step process."
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

list_schemes() {
  assetsURL=$(curl --silent 'https://api.github.com/repos/varnamproject/schemes/releases/latest' |
    grep '"assets_url":' |
    sed -E 's/.*"([^"]+)".*/\1/')
  langs=$(curl --silent $assetsURL |
    grep -E 'name(.*?).zip' |
    sed -E 's/.*"([^"]+)".*/\1/' |
    sed s/.zip//)
  echo $"$langs"
}

install_scheme() {
  cd $workDir
  schemeID="$1"
  releaseName="$schemeID"
  url="https://github.com/varnamproject/schemes/releases/download/$version/$releaseName.zip"
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

for lang in ${answer//,/ }; do
  # Trim whitespaces
  lang=`echo $lang | sed 's/ *$//g'`

  echo "Setup $lang"
  install_scheme "$lang"
done

install_govarnam_ibus_engine() {
  cd $workDir
  releaseName="varnam-ibus-engine-$versionNumber-$arch"
  url="https://github.com/varnamproject/govarnam-ibus/releases/download/$version/$releaseName.zip"
  echo "Downloading $releaseName from $url"
  curl -L -o govarnam.zip "$url"

  unzip govarnam.zip
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
echo "Telegram Group: https://t.me/varnamproject"
echo "Matrix Group: https://matrix.to/#/#varnamproject:poddery.com"
echo ""
echo "https://varnamproject.github.io"
echo "-----------------------------"