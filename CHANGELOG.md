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
 