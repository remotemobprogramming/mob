# Fast git handover with mob

![mob Logo](logo.svg)
<p align="center">
  <a href="https://github.com/remotemobprogramming/mob/actions?query=workflow%3ATest">
    <img alt="Test Workflow" src="https://img.shields.io/github/workflow/status/remotemobprogramming/mob/Test" /></a>
  <a href="https://github.com/remotemobprogramming/mob/graphs/contributors">
    <img alt="Contributors" src="https://img.shields.io/github/contributors/remotemobprogramming/mob" /></a>
  <a href="https://github.com/remotemobprogramming/mob/releases">
    <img alt="Downloads" src="https://img.shields.io/github/downloads/remotemobprogramming/mob/total" /></a>
  <a href="https://github.com/remotemobprogramming/mob/releases">
    <img  alt="Downloads of latest" src="https://img.shields.io/github/downloads/remotemobprogramming/mob/latest/total" /></a>
  <a href="https://github.com/remotemobprogramming/mob/releases/latest">
    <img alt="Version" src="https://img.shields.io/github/v/release/remotemobprogramming/mob?sort=semver" /></a>
  <a href="https://img.shields.io/github/stars/remotemobprogramming/mob">
    <img alt="Stars" src="https://img.shields.io/github/stars/remotemobprogramming/mob" /></a>
</p>

Smooth [git handover](https://www.remotemobprogramming.org/#git-handover) with 'mob'.

- **mob** is [an open source command line tool written in go](https://github.com/remotemobprogramming/mob)
- **mob** is the fastest way to [hand over code via git](https://www.remotemobprogramming.org/#git-handover) and feels [ubersmooth](https://twitter.com/holgerGP/status/1277653842444902400?s=20)
- **mob** supports remote mob/ensemble or pair programming using screen sharing
- **mob** works on every platform, even [ Apple Silicon](https://twitter.com/simonharrer/status/1332236430429581312?s=20)
- **mob** keeps your branches clean and only creates WIP commits on temporary wip branches
- **mob** supports multiple wip branches per base branch
- **mob** notifies you when it's time ⏱️ to handover
- **mob** can moo 🐄
- **mob** is even better when you follow its [best practices](#best-practices)

## What people say about 'mob'

> "Mob has allowed us to run fast-paced, engaging, and effective sessions by enabling sub-10-second handover times and otherwise getting out of the way. A simple but great tool!" &mdash; [Jeff Langr, developer](https://twitter.com/jlangr)

> "I love it, it is a quantum leap in our collaboration." &mdash; Vasiliy Sivovolov, Senior Software Engineer

>"What a great tool to organise remote working." &mdash; [Jennifer Gommans, IT Consultant](https://twitter.com/missjennbo)

> "I was recently introduced to [mob.sh](https://mob.sh) for remote pairing/mobbing collaboration and I absolutely love it. The timer feature is really a selling point for me. Kudos" &mdash; [Fabien Illert, IT Consultant](https://twitter.com/fabienillert)

> "Really enjoying working with http://mob.sh. Whole team added it to the "Glad" column during yesterday's retro ;-)" &mdash; [twitter.com/miljar](https://twitter.com/miljar/status/1392040059105382401)

## How to install

The preferred way to install mob is as a binary via the provided install script:
```
# works for macOS, linux, and even on windows in git bash
curl -sL install.mob.sh | sh
```

On macOS via homebrew: 

```
brew install remotemobprogramming/brew/mob

# upgrade to latest version
brew upgrade remotemobprogramming/brew/mob
```

On windows via [Scoop](https://scoop.sh/):

```
scoop install mob
``` 

On [Nix](http://nixos.org) through the [mob.nix](./mob.nix) expression like this `mob = callPackage ./mob.nix {};`. To install and configure espeak-ng for text-to-speech support, pass `withSpeech = true;`.

On Arch Linux via yay:

```bash
yay -S mobsh-bin
```

On Ubuntu via [snap](https://snapcraft.io/mob-sh):

```bash
sudo snap install mob-sh
sudo snap connect mob-sh:ssh-keys
```


### Using go tools

When you already have a working go environment with a defined GOPATH you can install latest via `go install`:

With go &lt; 1.16
```
go get github.com/remotemobprogramming/mob
go install github.com/remotemobprogramming/mob
```

go 1.16 introduced support for package@version syntax, so you can install directly with:
```
go install github.com/remotemobprogramming/mob@latest
```

or pick a specific version:
```
go install github.com/remotemobprogramming/mob@v1.2.0
```

## How to use

You only need three commands: `mob start`, `mob next`, and `mob done`. 
Switch to a separate branch with `mob start` and handover to the next person with `mob next`.
Repeat.
When you're done, get your changes into the staging area of the `master` branch with `mob done` and commit them.  

[![asciicast](https://asciinema.org/a/321885.svg)](https://asciinema.org/a/321885)

```
mob enables a fast Git handover

Basic Commands:
  start              start mob session from base branch in wip branch
  next               handover changes in wip branch to next person
  done               squashes all changes in wip branch to index in base branch
  reset              removes local and remote wip branch

Basic Commands(Options):
  start [<minutes>]                      Start a <minutes> timer
    [--include-uncommitted-changes|-i]   Move uncommitted changes to wip branch
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>/<branch-postfix>'
  next
    [--stay|-s]                          Stay on wip branch (default)
    [--return-to-base-branch|-r]         Return to base branch
    [--message|-m <commit-message>]      Override commit message
  done
    [--no-squash]                        Do not squash commits from wip branch
    [--squash]                           Squash commits from wip branch
  reset
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>/<branch-postfix>'

Experimental Commands:
  squash-wip                             Combines wip commits in wip branch with subsequent manual commits to leave only manual commits.
                                         ! Works only if all wip commits have the same wip commit message !
    [--git-editor]                       Not intended for manual use. Used as a non-interactive editor (GIT_EDITOR) for git.
    [--git-sequence-editor]              Not intended for manual use. Used as a non-interactive sequence editor (GIT_SEQUENCE_EDITOR) for git.

Timer Commands:
  timer <minutes>    start a <minutes> timer
  start <minutes>    start mob session in wip branch and a timer

Get more information:
  status             show the status of the current mob session
  config             show all configuration options
  version            show the version of mob
  help               show help

Other
  moo                moo!


Add --debug to any option to enable verbose logging


Examples:
  # start 10 min session in wip branch 'mob-session'
  mob start 10

  # start session in wip branch 'mob/<base-branch>/green'
  mob start --branch green

  # handover code and return to base branch
  mob next --return-to-base-branch

  # squashes all commits and puts changes in index of base branch
  mob done

  # make a sound check
  mob moo
```

## Best Practices

- **Say out loud**
  - Whenever you key in `mob next` at the end of your turn or `mob start` at the beginning of your turn say the command out loud. 
  - *Why?* Everybody sees and also hears whose turn is ending and whose turn has started. But even more important, the person whose turn is about to start needs to know when the previous person entered `mob next` so they get the latest commit via their `mob start`.
- **Steal the screenshare**
  - After your turn, don't disable the screenshare. Let the next person steal the screenshare. (Requires a setting in Zoom)
  - *Why?* This provides more calm (and less diversion) for the rest of the mob as the video conference layout doesn't change, allowing the rest of the mob to keep discussing the problem and finding the best solution, even during a Git handover.
- **Share audio**
  - Share your audio when you share your screen.
  - *Why?* Sharing audio means everybody will hear when the timer is up. So everybody will help you to rotate, even if you have missed it coincidentally or deliberately.
- **Use a timer**
  - Always specify a timer when using `mob start` (for a 5 minute timer use `mob start 5`)
  - *Why?* Rotation is key to good pair and mob programming. Just build the habit right from the start. Try to set a timer so everybody can have a turn at least once every 30 minutes.
- **Set up a global shortcut for screensharing**
  - Set up a global keyboard shortcut to start sharing your screen. In Zoom, you can do this via Zoom > Preferences > Keyboard Shortcuts. [More tips on setting up Zoom for effective screen sharing.](https://effectivehomeoffice.com/setup-zoom-for-effective-screen-sharing/)
  - *Why?* This is just much faster than using the mouse.
- **Set your editor to autosave**
  - Have your editor save your files on every keystroke automatically. IntelliJ products do this automatically. VS Code, however, needs to be configured via "File > Auto Save toggle".
  - *Why?* Sometimes people forget to save their files. With autosave, any change will be handed over via `mob next`.

## More on Installation

### Arch Linux

There are two Arch packages in the AUR:

- [mobsh-bin](https://aur.archlinux.org/packages/mobsh-bin/): uses the binary from the upstream release.
- [mobsh](https://aur.archlinux.org/packages/mobsh/): compiles sources from scratch (and runs tests) locally.

Example installation using AUR helper `yay`:

```bash
yay -S mobsh-bin

# OR
yay -S mobsh
```

### Linux Timer 

(This is not needed when installing via snap.)

To get the timer to play "mob next" on your speakers when your time is up, you'll need an installed speech engine. 
Install that on Debian/Ubuntu/Mint as follows:

```bash
sudo apt-get install espeak-ng-espeak mbrola-us1
```

Create a little script in your `$PATH` called `say` with the following content:

```bash
#!/bin/sh
espeak -v us-mbrola-1 "$@"
```

## How to configure

Show your current configuration with `mob config`:

```
MOB_REMOTE_NAME=origin
MOB_WIP_COMMIT_MESSAGE=mob next [ci-skip] [ci skip] [skip ci]
MOB_REQUIRE_COMMIT_MESSAGE=false
MOB_VOICE_COMMAND=say "%s"
MOB_NOTIFY_COMMAND=/usr/bin/osascript -e 'display notification "%s"'
MOB_NEXT_STAY=true
MOB_START_INCLUDE_UNCOMMITTED_CHANGES=false
MOB_WIP_BRANCH_QUALIFIER=
MOB_WIP_BRANCH_QUALIFIER_SEPARATOR=-
MOB_DONE_SQUASH=true
MOB_TIMER=
```

Override default value permanently via environment variables:

```bash
export MOB_NEXT_STAY=true
```

Override default value just for a single call:

```bash
MOB_NEXT_STAY=true mob next
```

## How to contribute

[Propose your change in an issue](https://github.com/remotemobprogramming/mob/issues) or [directly create a pull request with your improvements](https://github.com/remotemobprogramming/mob/pulls).

```bash
# PROJECT_ROOT is the root of the project/repository

cd $PROJECT_ROOT

git version # >= 2.17
go version # >= 1.14

go build # builds 'mob'

./create-testbed    # creates test assets

go test # runs all tests
go test -run TestDetermineBranches # runs the single test named 'TestDetermineBranches'

# run tests and show test coverage in browser
go test -coverprofile=cover.out && go tool cover -html=cover.out
```

## Design Concepts

- **mob** is a thin wrapper around git.
- **mob** is not interactive.
- **mob** owns its wip branches. It will create wip branches, make commits, push them, but also delete them.
- **mob** requires the user to do changes in non-wip branches.
- **mob** provides a copy'n'paste solution if it encounters an error.
- **mob** relies on information accessible via git.
- **mob** provides only a few environment variables for configuration. 
- **mob** only uses the Go standard library and no 3rd party plugins.

## Who is using 'mob'?

- [INNOQ](https://www.innoq.com)
- [BLUME2000](https://twitter.com/slashBene/status/1337329356637687811?s=20)
- [REWE Digital](https://www.rewe-digital.com/)
- And probably many others who shall not be named.

## Credits

Developed and maintained by [Dr. Simon Harrer](https://twitter.com/simonharrer).

Contributions and testing by Jochen Christ, Martin Huber, Franziska Dessart, Nikolas Hermann
and Christoph Welcz. Thank you!

Logo designed by [Sonja Scheungrab](https://twitter.com/multebaerr).

<script async defer src="https://cdn.simpleanalytics.io/hello.js"></script>
<noscript><img src="https://api.simpleanalytics.io/hello.gif" alt=""></noscript>

<a href="https://github.com/remotemobprogramming/mob/" class="github-corner" aria-label="View source on GitHub"><svg width="80" height="80" viewBox="0 0 250 250" style="fill:#151513; color:#fff; position: absolute; top: 0; border: 0; right: 0;" aria-hidden="true"><path d="M0,0 L115,115 L130,115 L142,142 L250,250 L250,0 Z"></path><path d="M128.3,109.0 C113.8,99.7 119.0,89.6 119.0,89.6 C122.0,82.7 120.5,78.6 120.5,78.6 C119.2,72.0 123.4,76.3 123.4,76.3 C127.3,80.9 125.5,87.3 125.5,87.3 C122.9,97.6 130.6,101.9 134.4,103.2" fill="currentColor" style="transform-origin: 130px 106px;" class="octo-arm"></path><path d="M115.0,115.0 C114.9,115.1 118.7,116.5 119.8,115.4 L133.7,101.6 C136.9,99.2 139.9,98.4 142.2,98.6 C133.8,88.0 127.5,74.4 143.8,58.0 C148.5,53.4 154.0,51.2 159.7,51.0 C160.3,49.4 163.2,43.6 171.4,40.1 C171.4,40.1 176.1,42.5 178.8,56.2 C183.1,58.6 187.2,61.8 190.9,65.4 C194.5,69.0 197.7,73.2 200.1,77.6 C213.8,80.2 216.3,84.9 216.3,84.9 C212.7,93.1 206.9,96.0 205.4,96.6 C205.1,102.4 203.0,107.8 198.3,112.5 C181.9,128.9 168.3,122.5 157.7,114.1 C157.9,116.9 156.7,120.9 152.7,124.9 L141.0,136.5 C139.8,137.7 141.6,141.9 141.8,141.8 Z" fill="currentColor" class="octo-body"></path></svg></a><style>.github-corner:hover .octo-arm{animation:octocat-wave 560ms ease-in-out}@keyframes octocat-wave{0%,100%{transform:rotate(0)}20%,60%{transform:rotate(-25deg)}40%,80%{transform:rotate(10deg)}}@media (max-width:500px){.github-corner:hover .octo-arm{animation:none}.github-corner .octo-arm{animation:octocat-wave 560ms ease-in-out}}</style>
