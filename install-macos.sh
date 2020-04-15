#!/bin/bash

pushd /tmp/ > /dev/null

echo "Installing latest 'mob' release from GitHub"
url=$(curl -s https://api.github.com/repos/remotemobprogramming/mob/releases/latest \
| grep "browser_download_url.*mob_.*darwin_amd64\.tar\.gz" \
| cut -d ":" -f 2,3 \
| tr -d \")
echo "$url"
tarball="${url##*/}"
curl -sSL $url > $tarball

echo "Extracting $tarball"
tar -xzf $tarball

# echo "Installing 'mob' in '/usr/local/bin'"
chmod +x mob
mv mob /usr/local/bin/

popd > /dev/null

location="$(which mob)"
echo "Mob binary location: $location"

version="$(mob version)"
echo "Mob binary version: $version"