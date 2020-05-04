#!/bin/sh
target=/usr/local/bin
user_arg=$1
stream_cmd="curl -sL install.mob.sh"
readme_say="https://mob.sh#linux-timer"

determine_local_target() {
  systemd-path user-binaries
}

handle_user_installation() {
  if [ "$user_arg" = "--user" ]; then
    local_target=$(determine_local_target)
    if [ "$local_target" != "" ] && [ ! -d "$local_target" ]; then
      mkdir -p "$local_target"
    fi

    if [ -d "$local_target" ]; then
      target=$local_target
    else
      echo "unfortunately, there is no user-binaries path on your system. aborting installation."
      exit 1
    fi
  fi
}

check_access_rights() {
  if [ ! -w "$target" ]; then
    echo "you do not have access rights to $target."
    echo
    local_target=$(determine_local_target)
    if [ "$local_target" != "" ]; then
      echo "we recommend that you use the --user flag"
      echo "to install the app into your user binary path $local_target"
      echo
      echo "  $stream_cmd | sh -s - --user"
      echo
    fi
    echo "calling the installation with sudo might help."
    echo
    echo "  $stream_cmd | sudo sh"
    echo
    exit 1
  fi
}

install_remote_binary() {
  echo "installing latest 'mob' release from GitHub to $target..."
  case "$(uname -s)" in
  Darwin)
    system="darwin"
    ;;
  *)
    system="linux"
    ;;
  esac
  url=$(curl -s https://api.github.com/repos/remotemobprogramming/mob/releases/latest |
    grep "browser_download_url.*mob_.*${system}_amd64\.tar\.gz" |
    cut -d ":" -f 2,3 |
    tr -d ' \"')
  curl -sSL "$url" | tar xz -C "$target" mob && chmod +x "$target"/mob
}

display_success() {
  location="$(command -v mob)"
  echo "Mob binary location: $location"

  version="$(mob version)"
  echo "Mob binary version: $version"
}

check_say() {
  say=$(command -v say)
  if [ ! -e "$say" ]; then
    echo
    echo "Couldn't find a 'say' command on your system."
    echo "While 'mob' will still work, you won't get any spoken indication that your time is up."
    echo "Please refer to the documentation how to setup text to speech on a *NIX system."
    echo
    echo "$readme_say"
    echo
  fi
}

main() {
  handle_user_installation
  check_access_rights
  install_remote_binary
  display_success
  check_say
}

main
