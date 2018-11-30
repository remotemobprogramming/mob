# Tool for Remote Mob Programming

Swift handover for remote mobs using git. 
`mob` is a CLI tool written in GO.
It keeps your master branch clean and creates WIP commits on `mob-session` branch.

The timer only works on OS X.

## How to use it?

```bash
# simon begins the mob session as typist
simon$ mob start 10
# WORK
# after 10 minutes...
simon$ mob next
# carola takes over as the second typist
carola$ mob start 10
# WORK
# after 10 minutes...
carola$ mob next
simon$ mob start 10
# WORK
# After 6 minutes the work is done.
simon$ mob done
simon$ git commit --message "describe what the mob session was all about"
```

## How does it work?

- `mob start 10` creates branch `mob-session` and pulls from `origin/mob-session`, and creates a ten minute timer
- `mob next` pushes all changes to `origin/mob-session`in a `WIP [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session` 

- `mob status` display the mob session status and all the created WIP commits
- `mob reset` deletes `mob-session` and `origin/mob-session`

## How does it really work?

```bash
$ MOB_DEBUG=true mob start
```

Prints out any git commands and their results.

## How to install

```bash
$ brew install golang
$ git clone https://github.com/simonharrer/mob
$ cd mob
$ ./install
# Now, you can use the mob tool from anywhere directory in the terminal
```

## How to contribute

Create a pull request.
