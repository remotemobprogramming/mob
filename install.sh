#!/bin/sh
target=/usr/local/bin
if [ "$(whoami)" != "root" ] 
then
  target=~/.local/bin
  echo "you don't have root rights. will install mob in the user space. please make sure that \"$target\" is part of your PATH!"
fi

echo "Installing latest 'mob' release from GitHub to $target"
case "$(uname -s)" in
   Darwin)
      system="darwin"
     ;;
   *)
      system="linux"
     ;;
esac
url=$(curl -s https://api.github.com/repos/remotemobprogramming/mob/releases/latest \
| grep "browser_download_url.*mob_.*${system}_amd64\.tar\.gz" \
| cut -d ":" -f 2,3 \
| tr -d \")
# echo "$url"
tarball="${url##*/}"

curl -sSL $url | tar xz -C $target mob && chmod +x $target/mob

location="$(which mob)"
echo "Mob binary location: $location"

version="$(mob version)"
echo "Mob binary version: $version"