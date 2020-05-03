# mob: Swift Handover using git

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

You only need three commands.

- `mob start` creates branch `mob-session` and pulls from `origin/mob-session`
- `mob next` pushes all changes to `origin/mob-session`in a `mob next [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session`

There are a few more commands and options for even more convenience.

- `mob start 10` also creates a ten minute timer
- `mob start 10 share` also start screen sharing in Zoom (requires Zoom configuration)
- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`
- `mob share` start screen sharing in Zoom (requires Zoom configuration)

### Zoom Screenshare

The `mob share` feature uses the zoom keyboard shortcut "Start/Stop Screen Sharing". This only works if you
- make the shortcut globally available (Zoom > Preferences > Keyboard Shortcuts), and
- keep the default shortcut at CMD+SHIFT+S (macOS)/ ALT+S (Linux).

[More tips on setting up Zoom for effective screen sharing.](https://effectivehomeoffice.com/setup-zoom-for-effective-screen-sharing/)

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

## How can one configure it?

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

Or override default value just for a single call:

```bash
$ MOB_NEXT_STAY=true mob next
```

## How to contribute

Open an issue or create a pull request.

## Credits

- Developed and maintained by [Simon Harrer](https://twitter.com/simonharrer).
- Contributions and testing by Jochen Christ, Martin Huber, Franziska Dessart, and Nikolas Hermann. Thank you!
- Logo designed by [Sonja Scheungrab](https://twitter.com/multebaerr).

<script async defer src="https://cdn.simpleanalytics.io/hello.js"></script>
<noscript><img src="https://api.simpleanalytics.io/hello.gif" alt=""></noscript>