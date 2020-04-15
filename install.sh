#!/bin/sh

case "$(uname -s)" in
   Darwin)
      system="darwin"
     ;;
   *)
      system="linux"
     ;;
esac

echo "Installing latest 'mob' release from GitHub"
url=$(curl -s https://api.github.com/repos/remotemobprogramming/mob/releases/latest \
| grep "browser_download_url.*mob_.*${system}_amd64\.tar\.gz" \
| cut -d ":" -f 2,3 \
| tr -d \")
# echo "$url"
tarball="${url##*/}"
curl -sSL $url | tar xz -C /usr/local/bin/ mob

# echo "Installing 'mob' in '/usr/local/bin'"
chmod +x /usr/local/bin/mob

location="$(which mob)"
echo "Mob binary location: $location"

version="$(mob version)"
echo "Mob binary version: $version"