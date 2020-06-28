# Swift git handover with mob

![mob Logo](logo.svg)
<p align="center">
    <a href="https://github.com/remotemobprogramming/mob/graphs/contributors" alt="Contributors">
        <img src="https://img.shields.io/github/contributors/remotemobprogramming/mob" /></a>
     <a href="https://github.com/remotemobprogramming/mob/releases" alt="Downloads">
        <img src="https://img.shields.io/github/downloads/remotemobprogramming/mob/total" /></a>
    <a href="https://github.com/remotemobprogramming/mob/releases" alt="Downloads">
        <img src="https://img.shields.io/github/downloads/remotemobprogramming/mob/latest/total" /></a>
</p>

Swift [git handover](https://www.remotemobprogramming.org/#git-handover) with 'mob'.

- **mob** is [an open source command line tool written in go](https://github.com/remotemobprogramming/mob)
- **mob** is the fastest way to [hand over code via git](https://www.remotemobprogramming.org/#git-handover)
- **mob** keeps your branches clean and only creates WIP commits on temporary wip branches
- **mob** supports multiple wip branches per base branch
- **mob** notifies you when it's time to handover
- **mob** can moo

## How to install

```
curl -sL install.mob.sh | sh
```
If you are under Windows, you can use `git bash` to install `mob`. 

You can also install it on macOS via homebrew: 

```
brew install remotemobprogramming/brew/mob
```

## How to use

You only need three commands: `mob start`, `mob next`, and `mob done`. 
Switch to a separate branch with `mob start` and handover to the next person with `mob next`.
Continue with `mob start` and handover to the next person with `mob next`.
Continue with `mob start` and handover to the next person with `mob next`.
Continue with `mob start` and handover to the next person with `mob next`.
...
When you're done, get your changes into the staging area of the `master` branch with `mob done` and commit them.  

[![asciicast](https://asciinema.org/a/321885.svg)](https://asciinema.org/a/321885)

```
USAGE
mob start [<minutes>] [--include-uncommitted-changes] [--branch|-b <branch>]	# start mob session in wip branch
mob next [-s|--stay] 		# handover to next person and switch back to base branch
mob done 			# finish mob session by squashing all changes in wip branch to index in base branch
mob reset [--branch|-b <branch>]# removes local and remote wip branch
mob status 			# show status
mob timer <minutes>		# start a <minutes> timer
mob config 			# print configuration
mob moo 			# moo!
mob version 			# print version number
mob help 			# print usage
EXAMPLES
mob start 10 			# start 10 min session in wip branch 'mob-session'
mob start --branch green 	# start session in wip branch 'mob/<base-branch>/green'
mob next --stay			# handover code and stay on wip branch
mob done 			# get changes back to base branch
mob moo 			# be amazed
```

## How does it work

- `mob start` creates branch `mob-session` and pulls from `origin/mob-session`
- `mob next` pushes all changes to `origin/mob-session`in a `mob next [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session`
- `mob timer 10` start a ten minute timer
- `mob start 10` combines mob start and mob timer 10
- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`
- `mob config` print configuration

### Screen Share Integration

Mob no longer supports starting the screen share on 'mob start'.
Why? 
At first, this feature sounds awesome. 
In practice, however, that feature wasn't much help.
It only simulated keying in a keyboard shortcut and had several shortcomings: a) it needed to be configured correctly, b) only toggled screen share, so one had to keep in mind whether one was already screen sharing or not, and c) solely supported Zoom on macOS and Linux.
The feature promised too much, and hold very little of it.

Still, that keyboard shortcut to toggle screen sharing in Zoom is still very helpful. Just press the hotkey yourself. It is, however, essential to make the shortcut globally available (Zoom > Preferences > Keyboard Shortcuts). Otherwise, the hotkey won't work. [More tips on setting up Zoom for effective screen sharing.](https://effectivehomeoffice.com/setup-zoom-for-effective-screen-sharing/)

## More on Installation

### Scoop

On Windows, if you use [Scoop](https://scoop.netlify.app/) then just download the mob.json file, open a PowerShell terminal in that directory and:
```
scoop install mob
```

### Linux Timer

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
MOB_WIP_COMMIT_MESSAGE=mob next [ci-skip]
MOB_VOICE_COMMAND=say
MOB_NEXT_STAY=false
MOB_START_INCLUDE_UNCOMMITTED_CHANGES=false
MOB_DEBUG=false
```

Override default value permanently via environment variables:

```bash
export MOB_DEBUG=true
```

Override default value just for a single call:

```bash
MOB_DEBUG=true mob next
```


## How to contribute

[Propose your change in an issue](https://github.com/remotemobprogramming/mob/issues) or [directly create a pull request with your improvements](https://github.com/remotemobprogramming/mob/pulls).

```bash
git version # >= 2.17
go version # >= 1.14.4 

go build # builds 'mob'

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
- **mob** relies on information accessiable via git.
- **mob** provides only a few environment variables for configuration. 
- **mob** only uses the Go standard library and no 3rd party plugins.

## Credits

Developed and maintained by [Dr. Simon Harrer](https://twitter.com/simonharrer).

Contributions and testing by Jochen Christ, Martin Huber, Franziska Dessart, Nikolas Hermann
and Christoph Welcz. Thank you!

Logo designed by [Sonja Scheungrab](https://twitter.com/multebaerr).

<script async defer src="https://cdn.simpleanalytics.io/hello.js"></script>
<noscript><img src="https://api.simpleanalytics.io/hello.gif" alt=""></noscript>

<a href="https://github.com/remotemobprogramming/mob/" class="github-corner" aria-label="View source on GitHub"><svg width="80" height="80" viewBox="0 0 250 250" style="fill:#151513; color:#fff; position: absolute; top: 0; border: 0; right: 0;" aria-hidden="true"><path d="M0,0 L115,115 L130,115 L142,142 L250,250 L250,0 Z"></path><path d="M128.3,109.0 C113.8,99.7 119.0,89.6 119.0,89.6 C122.0,82.7 120.5,78.6 120.5,78.6 C119.2,72.0 123.4,76.3 123.4,76.3 C127.3,80.9 125.5,87.3 125.5,87.3 C122.9,97.6 130.6,101.9 134.4,103.2" fill="currentColor" style="transform-origin: 130px 106px;" class="octo-arm"></path><path d="M115.0,115.0 C114.9,115.1 118.7,116.5 119.8,115.4 L133.7,101.6 C136.9,99.2 139.9,98.4 142.2,98.6 C133.8,88.0 127.5,74.4 143.8,58.0 C148.5,53.4 154.0,51.2 159.7,51.0 C160.3,49.4 163.2,43.6 171.4,40.1 C171.4,40.1 176.1,42.5 178.8,56.2 C183.1,58.6 187.2,61.8 190.9,65.4 C194.5,69.0 197.7,73.2 200.1,77.6 C213.8,80.2 216.3,84.9 216.3,84.9 C212.7,93.1 206.9,96.0 205.4,96.6 C205.1,102.4 203.0,107.8 198.3,112.5 C181.9,128.9 168.3,122.5 157.7,114.1 C157.9,116.9 156.7,120.9 152.7,124.9 L141.0,136.5 C139.8,137.7 141.6,141.9 141.8,141.8 Z" fill="currentColor" class="octo-body"></path></svg></a><style>.github-corner:hover .octo-arm{animation:octocat-wave 560ms ease-in-out}@keyframes octocat-wave{0%,100%{transform:rotate(0)}20%,60%{transform:rotate(-25deg)}40%,80%{transform:rotate(10deg)}}@media (max-width:500px){.github-corner:hover .octo-arm{animation:none}.github-corner .octo-arm{animation:octocat-wave 560ms ease-in-out}}</style>
