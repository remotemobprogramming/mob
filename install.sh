#!/bin/sh
target=/usr/local/bin
user_arg=$1
stream_cmd="curl -sL install.mob.sh"
readme="https://mob.sh"

determine_os() {
  case "$(uname -s)" in
  Darwin)
    echo "darwin"
    ;;
  MINGW64*)
    echo "windows"
    ;;
  *)
    echo "linux"
    ;;
  esac
}

determine_user_install() {
  case "$(determine_os)" in
  windows)
    echo "--user"
    ;;
  *)
    $user_arg
    ;;
  esac
}

determine_local_target() {
  case "$(determine_os)" in
  windows)
    # shellcheck disable=SC1003
    echo "$USERPROFILE/bin" | tr '\\' '/'
    ;;
  linux)
    systemd-path user-binaries
    ;;
  esac
}

determine_mob_binary() {
 case "$(determine_os)" in
  windows)
    echo "mob.exe"
    ;;
  *)
    echo "mob"
    ;;
  esac
}

determine_ending() {
 case "$(determine_os)" in
  windows)
    echo "tar.gz"
    ;;
  *)
    echo "tar.gz"
    ;;
  esac
}

handle_user_installation() {
  user_install=$(determine_user_install)
  if [ "$user_install" = "--user" ]; then
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
    if [ "$(command -v sudo)" != "" ]; then
      echo "calling the installation with sudo might help."
      echo
      echo "  $stream_cmd | sudo sh"
      echo
    fi
    exit 1
  fi
}

install_remote_binary() {
  echo "installing latest 'mob' release from GitHub to $target..."
  url=$(curl -s https://api.github.com/repos/remotemobprogramming/mob/releases/latest |
    grep "browser_download_url.*mob_.*$(determine_os)_amd64\.$(determine_ending)" |
    cut -d ":" -f 2,3 |
    tr -d ' \"')
  curl -sSL "$url" | tar xz -C "$target" "$(determine_mob_binary)" && chmod +x "$target"/mob
}

add_to_path() {
  case "$(determine_os)" in
  windows)
    powershell -command "[System.Environment]::SetEnvironmentVariable('Path', [System.Environment]::GetEnvironmentVariable('Path', [System.EnvironmentVariableTarget]::User)+';$target', [System.EnvironmentVariableTarget]::User)"
    ;;
  esac
}

display_success() {
  location="$(command -v mob)"
  echo "Mob binary location: $location"

  version="$(mob version)"
  echo "Mob binary version: $version"
}

check_say() {
  case "$(determine_os)" in
  linux)
    say=$(command -v say)
    if [ ! -e "$say" ]; then
      echo
      echo "Couldn't find a 'say' command on your system."
      echo "While 'mob' will still work, you won't get any spoken indication that your time is up."
      echo "Please refer to the documentation how to setup text to speech on a *NIX system."
      echo
      echo "$readme#$(determine_os)-timer"
      echo
    fi
    ;;
  esac
}

main() {
  handle_user_installation
  check_access_rights
  install_remote_binary
  add_to_path
  display_success
  check_say
}

main
