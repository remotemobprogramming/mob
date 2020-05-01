# Tool for Remote Mob Programming

![mob Logo](logo.svg)

Swift handover for [remote mobs](https://remotemobprogramming.org) using git.
`mob` is a CLI tool written in GO.
It keeps your master branch clean and creates WIP commits on `mob-session` branch.

## How to install

```bash
curl -s https://raw.githubusercontent.com/remotemobprogramming/mob/master/install.sh | sh
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

The `mob share` feature uses the zoom keyboard shortcut "Start/Stop Screen Sharing". This only works if you
- make the shortcut globally available (Zoom > Preferences > Keyboard Shortcuts), and
- keep the default shortcut at CMD+SHIFT+S (macOS)/ ALT+S (Linux).

## More on Installation

### Linux Timer

To get the timer to play "mob next" when your time is up, you'll need an installed speech engine. 
Install that on Debian/Ubuntu/Mint as follows:

```bash
sudo apt-get install espeak-ng-espeak mbrola-us1
```

Create a little script in your $PATH called `say` with the following content:

```bash
#!/bin/sh
# please install espeak-ng-espeak and mbrola-us-1 (multiverse) for this to work!
# sudo apt install espeak-nq-espeak mbrola-us1
# you might also try out different speakers as well ;-)
espeak -v us-mbrola-1 "$@"
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
