#!/bin/sh
target=/usr/local/bin
user_arg=$1
stream_cmd="curl -sL install.mob.sh"
readme="https://mob.sh"

determine_arch() {
  case "$(uname -s)" in
  Darwin)
    echo "universal"
    ;;
  *)
    echo "amd64"
    ;;
  esac
}

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
    echo "$user_arg"
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
    grep "browser_download_url.*mob_.*$(determine_os)_$(determine_arch)\.$(determine_ending)" |
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

check_command() {
  location="$(command -v mob)"

  if [ $location = "" ]; then
    echo 
    echo "(!) 'mob' could not be found after install!" 

    case "$(determine_os)" in
    linux)
      echo "    If you installed using --user it should be found when you login next time." 
      echo "    If it does not, you might need to manually add it to your .profile or equivalent like so:"
      echo 
      echo "    echo \"export PATH=$target:\\\$PATH\" >> ~/.profile" 
      ;;
    *)
      echo "    Make sure that $target is in your PATH"
    esac
    return
  fi
    
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
      echo "Please refer to the documentation how to setup text to speech on a *NIX system:"
      echo
      echo "     $readme#$(determine_os)-timer"
      echo
    fi
    ;;
  esac
}

check_installation_path() {
  location="$(command -v mob)"
  if [ "$(determine_os)" = "windows" ]; then
    location=$(echo $location | sed -E 's|^/([a-zA-Z])|\U\1:|')
  fi
  if [ "$location" != "$target/mob" ] && [ "$location" != "" ]; then
    echo "(!) The installation location doesn't match the location of the mob binary."
    echo "    This means that the binary that's used is not the binary that has just been installed"
    echo "    You probably want to delete the binary at $location"
  fi
}

main() {
  handle_user_installation
  check_access_rights
  install_remote_binary
  add_to_path
  check_command
  check_say
  check_installation_path
}

main
