#!/bin/sh   
target=/usr/local/bin
local_target=$(systemd-path user-binaries)
user_arg=$1

handle_user_installation() {
  if [ "$user_arg" = "--user" ]
  then
    if [ $local_target != "" ] && [ ! -d $local_target ]
    then
      mkdir $local_target
    fi
      
    if [ -d $local_target ]
    then
      target=$local_target
    else
      echo "unfortunately, there is no user-binaries path on your system. aborting installation."
      exit 1
    fi
  fi
}

check_access_rights() {
  if [ ! -w $target ]
  then
    echo "you do not seem to have access rights to $target."
    echo "calling the installation with sudo might help"
    if [ "$local_target" != "" ]
    then
      echo "alternatively, you may also use the --user flag"
      echo "to install the app into your user binary path $local_target"
      echo "  ./install.sh --user"
    fi
    exit 1
  fi
}

download_binary() {
  echo "downloading latest 'mob' release from GitHub..."
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
}

display_success() {
  location="$(which mob)"
  echo "Mob binary location: $location"

  version="$(mob version)"
  echo "Mob binary version: $version"
}

main() {
  handle_user_installation
  check_access_rights
  download_binary
  install_binary
  display_success
}

main