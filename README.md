# Tool for Remote Mob Programming

![mob Logo](logo.svg)

Swift handover for [remote mobs](https://remotemobprogramming.org) using git.
`mob` is a CLI tool written in GO.
It keeps your master branch clean and creates WIP commits on `mob-session` branch.

## How to install

```bash
sh -c "$(curl -s https://raw.githubusercontent.com/remotemobprogramming/mob/master/install.sh)"
```

## How to use it?

[![asciicast](https://asciinema.org/a/321885.svg)](https://asciinema.org/a/321885)

## How does it work?

- `mob start` creates branch `mob-session` and pulls from `origin/mob-session`
- `mob next` pushes all changes to `origin/mob-session`in a `mob next [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session`

- `mob start 10` and creates a ten minute timer
- `mob start 10 share` and activates screenshare in zoom (macOS or Linux with xdotool, requires zoom configuration)
- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`
- `mob share` start screenshare with zoom (macOS or Linux with xdotool, requires configuration in zoom to work)

### Zoom Screenshare

The `mob share` feature only works if you activate make the screenshare hotkey in zoom globally available, and keep the default shortcut at CMD+SHIFT+S (macOS)/ ALT+S (Linux).

## More on Installation

### Linux Timer

To get the timer to work on Linux, you need the GNUstep speech engine. Install that on ubuntu as follows:

```bash
sudo apt-get install gnustep-gui-runtime
```

### Windows

- Install [Golang](https://golang.org/): Download and execute MSI from Download page
- Open console and execute following commands

```bash
git clone https://github.com/remotemobprogramming/mob
cd mob
.\install.cmd
# Now, you can use the mob tool from anywhere directory in the terminal
```

## How can one customize it?

You can set several environment variables that will be picked up by `mob`:

```bash
# override default values if necessary
export MOB_WIP_BRANCH=mob-session
export MOB_BASE_BRANCH=master
export MOB_REMOTE_NAME=origin
export MOB_WIP_COMMIT_MESSAGE="mob next [ci-skip]"
export MOB_NEXT_STAY=false # set to true to stay in the MOB_WIP_BRANCH after 'mob next' instead of checking out MOB_BASE_BRANCH
export MOB_VOICE_COMMAND=espeak # for using alternatives to 'say'
export MOB_DEBUG=false
```

The easiest way to enable them for a single call is as follows:

```bash
$ MOB_NEXT_STAY=true mob next
```

## How to contribute

Create a pull request.

## Credits

- Developed and maintained by [Simon Harrer](https://twitter.com/simonharrer).
- Contributions and testing by Jochen Christ, Martin Huber, Franziska Dessart, and Nikolas Hermann. Thank you!
- Logo designed by [Sonja Scheungrab](https://twitter.com/multebaerr).
