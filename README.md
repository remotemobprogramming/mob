# Tool for Remote Mob Programming

CLI tool written in GO that allows to switch the typist in a remote mob programming session quickly with git and a remote branch.

## How to use it?

```bash
# simon starts as first typist
simon$ mob start
# after x minutes simon ends his mobbing interval
simon$ mob next
# carola takes over as the second typist
carola$ mob start
# after x minutes carola ends her mobbing interval
carola$ mob next
# our feature is already done
carola$ mob done
carola$ git commit --message "describe what the mob session was all about"
```

## How does it work?

- `mob start` creates branch `mob-session` and pulls from `origin/mob-session`
- `mob next` pushes all changes to `origin/mob-session`in a `WIP [ci-skip]` commit
- `mob done` squashes all changes in `mob-session` into staging of `master` and removes `mob-session` and `origin/mob-session` 

## How does it really work?

```bash
$ MOB_DEBUG=true mob start
```

Prints out any git commands and their results.
