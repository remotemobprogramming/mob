# 2.7.0
- Add a Smiley face to Happy Collaborating message to make your day :)

# 2.6.0
- **NEW** Allow keeping manual commits while squashing all wip commits by finishing a mob-session with `mob done --squash-wip`.
- Removed experimental command `mob squash-wip` in favor of new `mob done --squash-wip`.  
- Added missing configuration option `MOB_WIP_BRANCH_PREFIX` for `.mob` file.

Thanks to @gregorriegler and @hollesse making this release possible!

# 2.5.0
- Enable git hooks with `MOB_GIT_HOOKS_ENABLED=true`. By default, this option is false and no git hooks such as `pre-commit` or `pre-push` are triggered via mob itself.

# 2.4.0
- As an alternative to the environment variables, you can configure the mob tool with a `.mob` file in your home directory. For an example have a look at `mob-configuration-example` file. 
- As an alternative to the environment variables, you can configure the mob tool with a `.mob` file in your git project root directory. The configuration options `MOB_VOICE_COMMAND`, `MOB_VOICE_MESSAGE`, `MOB_NOTIFY_COMMAND`, and `MOB_NOTIFY_MESSAGE` are disabled for the project specific configuration to prevent bash injection attacks.
Thanks to @vrpntngr & @hollesse making this release possible as part of the @INNOQ Hands-On Event, February 2022.

# 2.3.0
- With `export MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER=true` the room name is automatically derived from the value you passed in via the `mob start --branch <branch>` parameter.

# 2.2.0
- When mob encounters unpushed commits in the base branch, the tool now provides help for the user to fix this immediately.
- Improves console output when using `mob break 5` in combination with https://timer.mob.sh
- Gives the user the chance to name their final commit after `mob done --no-squash`

# 2.1.0
- When having set `MOB_TIMER_ROOM` the local timer keeps on working. To disable the local timer altogether, please disable it via `export MOB_TIMER_LOCAL=false`.
- Improved message when nothing to commit (Thanks to @seanpoulter for https://github.com/remotemobprogramming/mob/pull/202)

# 2.0.0
- **NEW** create a team room on https://timer.mob.sh to have a team timer. Set `MOB_TIMER_ROOM` to the name of your team room and mob will automatically send timer requests to your team room. Mob now even supports breaks with `mob break <minutes>`. 

# 1.12.0
- If you renamed the executable or use a symlink to use a different name, `mob` will detect the new name and use that in its console output.
- Improves error handling when using `mob start -i`. When the working directory is a subdirectory that would be removed due to `git stash` the mob tool will tell the user about this and aborts with an error. 

# 1.11.1
- Fixes a bug which let `mob version` fail when run outside of a git repository.

# 1.11.0
- Allow to override the text in the notification and the voice via environment variables `MOB_NOTIFY_MESSAGE` and `MOB_VOICE_MESSAGE`.
- Allow to override the stash name used for stashing uncommitted changes via the environment variable `MOB_STASH_NAME`.
- Allow to override the cli name of the tool via `MOB_CLI_NAME` so you can use `pair`, `ensemble`, `team`, or whatever you like best, instead of `mob`. Just install the `mob` tool, set an alias in your cli and set the environment variable `MOB_CLI_NAME` to the name of your alias.

# 1.10.0
- Print current time after mob start. This helps when scrolling through the terminal to distinguish the mob start calls.

# 1.9.0
- Show commit hash of WIP commits made by the `mob` tool on the console.
- `mob start --include-uncommitted-changes` now fails fast. That means, if `mob` can detect any issue preventing it to succeed, it will exit BEFORE calling `git stash`. This will make error recovery much easier as one doesn't have to think about applying any stashes by themselves. 

# 1.8.0
- `mob next` does not show the same committer multiple times in the list of previous committers.
- `mob next` does not suggest the current Git user to be the next typist as long as there were other persons involved in the mob.
- `mob next` performs a simple lookahead to also suggest persons who might have been absent only during the last mob round.
- When user.name is not set in the git config, mob no longer shows an error but a warning with a help how to fix it.

# 1.7.0
- Allows creating parallel mob sessions on the same repository more easily.
- `mob branch` shows all remote mob branches currently available.
- `mob fetch` fetches the remote state, so you have everything up to date locally. Helpful to combine with `mob status` and `mob branch` who don't fetch by themselves.

# 1.6.0
- When `mob start` fails, the timer no longer starts to run.

# 1.5.0
- Less noisy output: Only show number of unpushed commits in output if there are more than 0.
- Add experimental command `mob squash-wip` to squash any WIP commits in the wip branch into a following manual commit using `git rebase --interactive` with `mob` as the temporary `GIT_EDITOR`.
- The order of the latest commit is now reversed, the latest one is shown last.
- Add experimental configuration option `MOB_WIP_BRANCH_PREFIX` to configure the `mob/` prefix to some other value. 

# 1.4.0
- The list of commits included in a mob branch is now truncated to a maximum of 5 entries to prevent the need for scrolling up in order to see the latest included changes.
- Show more informative error message when `mob <cmd>` is run outside of a git repository.
- Add environment variable MOB_TIMER which allows setting a default timer duration for `mob start` and `mob timer` commands.
- Add automatic co-author attribution. Mob will collect all committers from a WIP branch and add them as co-authors in the final WIP commit.
- added support for preventing `mob start` if there are unpushed commits
- better output if one passes a negative number for the timer

# 1.3.0
- The default `MOB_COMMIT_MESSAGE` now includes `[ci skip]` and `[skip ci]` so that all the typical CI systems (including Azure DevOps) will skip this commit.
- Add `--squash` option to `mob done` that corresponds to `--no-squash`.
- Fixes the default text to speech command on linux and osx.
- Removed `MOB_DEBUG` environment variable (has been deprecated for some time).

# 1.2.0
- Add environment variable `MOB_REQUIRE_COMMIT_MESSAGE` which you could set to true to require a commit message other than the default one.
- Fixes a bug where you could not run `mob start --branch feature-1` because feature-1 contained a dash.
- Fixes a bug which prevented the sound output of 'mob next' and 'moo' on windows

# 1.1.0
- Add optional `--no-squash` for `mob done` to keep commits from wip branch.
- Add environment variable `MOB_DONE_SQUASH` to configure the `mob done` behaviour. `MOB_DONE_SQUASH=false` is equal to passing flag `--no-squash`.
- Special thanks to @jbrains, @koeberlue, @gregor_riegler for making this release happen, obviously, in a remote mob session.

# 1.0.0
- BREAKING: `MOB_WIP_BRANCH_QUALIFIER_SEPARATOR` now defaults to '-'.
- BREAKING: `MOB_NEXT_STAY` now defaults to 'true'.
- Proposed cli commands like `mob start --include-uncommitted-changes` are now shown on a separate line to allow double clicking to copy in the terminal.

# 0.0.27
- Add way to configure `MOB_WIP_BRANCH_QUALIFIER` via an environment variable instead of `--branch` parameter. Helpful if multiple teams work on the same repository.
- Add way to configure `MOB_WIP_BRANCH_QUALIFIER_SEPARATOR` via an environment variable. Defaults to '/'. Will change to '-' in future versions to prevent branch naming conflicts (one cannot have a branch named `mob/main` and a branch named `mob/main/green` because `mob/main` cannot be a file and a directory at the same time).

# 0.0.26
- Adds way to configure the voice command via the environment variable `MOB_VOICE_COMMAND`.
- Allow disabling voice or notification by setting the environment variables `MOB_VOICE_COMMAND` or `MOB_NOTIFY_COMMAND` to an empty string.
- Fixes a bug where a failure in executing the voice command would lead to omitting the notification.
- `mob config` now shows the currently used `MOB_VOICE_COMMAND` and `MOB_NOTIFY_COMMAND`.
- Add `mob next --message "custom commit message"` as an option to override the commit message during `mob next`.

# 0.0.25
- Adds flag `--return-to-base-branch` (with shorthand `-r`) to return to base branch on `mob next`. Because 'mob' will change the default behavior from returning to the base branch to staying on the wip branch on `mob next`, this flag provides the inverse operation of `--stay`. If both are provided, the latter one wins.
- Adds flag `-i` as a shorthand notation for `--include-uncommitted-changes`.
- Fixes a bug that prevented `mob start` to work when on an outdated the WIP branch 
- `mob next` push if there are commits but no changes.

# 0.0.24
- Fixes a bug where mob couldn't handle branch names with the '/' character 

# 0.0.23
- Commit message of wip commits is no longer quoted (see #52)

# 0.0.22
- Adds `mob start --branch <branch>` to allow multiple wip branches in the form of 'mob/<base-branch>/<branch>' for a base branch. For example, when being on branch 'main' a `mob start --branch green` would switch to a wip branch named 'mob/main/green'.
- Adds `mob moo` (Thanks Niko for the idea)
- Deprecated `MOB_DEBUG` in favor of the parameter `--debug`
- Deprecated `MOB_START_INCLUDE_UNCOMMITTED_CHANGES` in favor of the parameter `--include-uncommitted-changes` instead
- Show warning if removed configuration option `MOB_BASE_BRANCH` or `MOB_WIP_BRANCH` is used.

# 0.0.20
- `mob start` on a branch named `feature1` will switch to the branch `mob/feature1` and will merge the changes back to `feature1` after `mob done`. For the `master` branch, the `mob-session` branch will still work (but this may change in the future, switching to `mob/master` at some point).
- Removes configuration options for base branch and wip branch. These are no longer necessary.
- `mob status` added. Thanks to Jeff Langr for that contribution! 

# 0.0.19
- Removes zoom screen share integration.
- Less git commands necessary for 'mob start'
- Mob automatically provides sound output on windows without any installation

# 0.0.18
- Fixes a bug where boolean environment variables such as `MOB_NEXT_STAY` set to any value (including empty value) falsely activated their respective option.
- Simplified `mob start` when joining a mob session. It uses `git checkout -B mob-session origin/mob-session` to override any local `mob-session` in the process. It reduces the amount of commands necessary and makes the mob tool more predictable: the `origin/mob-session` always contains the truth.
- Removes `mob share` command. You can still enable the zoom integration via `mob start 10 share` although this is now DEPRECATED and will eventually be removed in the future.

# 0.0.16
- `mob start` prints out untracked files as well 
- `mob start --include-uncommitted-changes` now includes untracked files in the stash 'n' pop as well 
- keying in an unknown command like `mob conf` will internally call `mob help` to print out the usage options instead of calling `mob status`
- fixed a bug where overriding `MOB_START_INCLUDE_UNCOMMITTED_CHANGES` via an environment variable could print out a wrong value (didn't affect any logic, just wrong console output)

# 0.0.15
- Any `git push` command now uses the `--no-verify` flag

# 0.0.14
- New homepage available at https://mob.sh
- `mob config` prints configuration using the environment variable names which allow overriding the values

# 0.0.13
- Fixes bug that prevented users wih git versions below 2.21 to be able to use 'mob'.
