# Swift git handover with mob

![mob Logo](logo.svg)

Swift [git handover](https://www.remotemobprogramming.org/#git-handover) with 'mob'.

- `mob` is [an open source command line tool written in go](https://github.com/remotemobprogramming/mob)
- `mob` is the fasted way to [handover code via git](https://www.remotemobprogramming.org/#git-handover)
- `mob` keeps your `master` branch clean
- `mob` creates WIP commits on the `mob-session` branch
- `mob` notifies you when it's time to handover

## How to install

```
curl -sL install.mob.sh | sh
```

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
mob start [<minutes> [share]] [--include-uncommitted-changes]	# start mob session
mob next [-s|--stay] 	# handover to next person
mob done 		# finish mob session
mob reset 		# reset any unfinished mob session (local & remote)
mob status 		# show status of mob session
mob share 		# start screen sharing in Zoom (requires Zoom configuration)
mob timer <minutes>	# start a <minutes> timer
mob config 		# print configuration
mob help 		# print usage
mob version 		# print version number

EXAMPLES
mob start 10 		# start 10 min session
mob start 10 share 	# start 10 min session with zoom screenshare
mob next --stay		# handover code and stay on mob session branch
mob done 		# get changes back to master branch
```

## How does it work

- `mob start` creates branch `mob-session` and pulls from `origin/mob-session`
- `mob next` pushes all changes to `origin/mob-session`in a `mob next [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session`
- `mob timer 10` start a ten minute timer
- `mob share` start screen sharing in Zoom (requires Zoom configuration)
- `mob start 10` combines mob start and mob timer 10
- `mob start 10 share` combines mob start and mob timer 10 and mob share
- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`
- `mob config` print configuration

### Zoom Screen Share Integration

The `mob share` feature uses the zoom keyboard shortcut "Start/Stop Screen Sharing". This only works if you
- make the shortcut globally available (Zoom > Preferences > Keyboard Shortcuts), and
- keep the default shortcut at CMD+SHIFT+S (macOS)/ ALT+S (Linux).

[More tips on setting up Zoom for effective screen sharing.](https://effectivehomeoffice.com/setup-zoom-for-effective-screen-sharing/)

## More on Installation

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

eSpeak NG works even on a MINGW64 installation such as **Git-Bash under Windows**
(assuming you have [installed the binary through their MSI](https://github.com/espeak-ng/espeak-ng/releases)):

```bash
#!/bin/sh
# eSpeak does not set any path, so we specify its installation directory instead
"$PROGRAMFILES/eSpeak NG/espeak-ng" "$@"
```

## How to configure

Show your current configuration with `mob config`:

```
MOB_BASE_BRANCH=master
MOB_WIP_BRANCH=mob-session
MOB_REMOTE_NAME=origin
MOB_WIP_COMMIT_MESSAGE=mob next [ci-skip]
MOB_VOICE_COMMAND=say
MOB_NEXT_STAY=false
MOB_START_INCLUDE_UNCOMMITTED_CHANGES=false
MOB_DEBUG=false
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

Open an issue or create a pull request.

## Credits

Developed and maintained by [Dr. Simon Harrer](https://twitter.com/simonharrer).

Contributions and testing by Jochen Christ, Martin Huber, Franziska Dessart, and Nikolas Hermann. Thank you!

Logo designed by [Sonja Scheungrab](https://twitter.com/multebaerr).

<script async defer src="https://cdn.simpleanalytics.io/hello.js"></script>
<noscript><img src="https://api.simpleanalytics.io/hello.gif" alt=""></noscript>
