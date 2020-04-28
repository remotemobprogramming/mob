#!/bin/sh   
target=/usr/local/bin
local_target=$(systemd-path user-binaries)
user_arg=$1
stream_cmd="curl -s https://raw.githubusercontent.com/remotemobprogramming/mob/master/install.sh"
readme_location="https://github.com/remotemobprogramming/mob/blob/master/README.md"

handle_user_installation() {
  if [ "$user_arg" = "--user" ]
  then
    if [ "$local_target" != "" ] && [ ! -d "$local_target" ]
    then
      mkdir $local_target
    fi
      
    if [ -d "$local_target" ]
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
    echo "you do not have access rights to $target."
    echo
    if [ "$local_target" != "" ]
    then
      echo "we recommend that you use the --user flag"
      echo "to install the app into your user binary path $local_target"
      echo
      echo "  $stream_cmd | sh -s - --user"
      echo
    fi
    echo "calling the installation with sudo might help."
    echo "please do it ONLY if you understand the implications of calling some remote script with ROOT rights."
    echo
    echo "  $stream_cmd | sudo sh"
    echo
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

check_say() {
  say=$(which say)
  if [ ! -e "$say" ]
  then
    echo
    echo "you do not have an installed 'say' command on your system."
    echo "while 'mob' will still work, you won't get any nice spoken indication that your time is up."
    echo "please refer to the README.md how to setup text to speech on a *NIX system."
    echo
    echo "$readme_location#linux-timer"
    echo
  fi
}

main() {
  handle_user_installation
  check_access_rights
  download_binary
  display_success
  check_say
}

main