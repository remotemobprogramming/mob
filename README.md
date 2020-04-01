# Tool for Remote Mob Programming

![mob Logo](logo.svg)

Swift handover for remote mobs using git.
`mob` is a CLI tool written in GO.
It keeps your master branch clean and creates WIP commits on `mob-session` branch.

## How to use it?

```bash
# Simon begins the mob session as typist
Simon $ cd secret-git-project
Simon $ mob start 10
# WORK with Simon as typist
# after 10 minutes, the timer triggers (you'll hear a 'mob next' from your speakers)
Simon $ mob next
# Carola takes over as the second typist
Carola $ mob start 10
# WORK with Carola as typist
# after 10 minutes, timer triggers...
Carola $ mob next
Maria $ mob start 10 share # share immediately activates zoom screenshare
# WORK
# After 6 minutes the work is done.
Maria $ mob done
Maria $ git commit --message "describe what the mob session was all about"
```

## How does it work?

- `mob start 10` creates branch `mob-session` and pulls from `origin/mob-session`, and creates a ten minute timer
- `mob start 10 share` also activates screenshare in zoom (macOS or Linux with xdotool, requires zoom configuration)
- `mob next` pushes all changes to `origin/mob-session`in a `mob next [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session`

- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`
- `mob share` start screenshare with zoom (macOS or Linux with xdotool, requires configuration in zoom to work)

### Zoom Screenshare

The `mob share` feature only works if you activate make the screenshare hotkey in zoom globally available, and keep the default shortcut at CMD+SHIFT+S (macOS)/ ALT+S (Linux).

## How to install

```bash
$ brew install golang
$ git clone https://github.com/remotemobprogramming/mob
$ cd mob
$ ./install
# Now, you can use the mob tool from any directory in the terminal
```

On linux systems you need the GNUstep speech engine to get the timer running.

On Ubuntu:

```bash
sudo apt-get install gnustep-gui-runtime golang
git clone https://github.com/remotemobprogramming/mob
cd mob
sudo ./install
```

### Windows

- Install [Golang](https://golang.org/): Download and execute MSI from Download page
- Open console and execute following commands

```bash
> git clone https://github.com/remotemobprogramming/mob
> cd mob
> .\install.cmd
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
