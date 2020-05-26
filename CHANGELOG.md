# 0.0.16
- fixed a bug where overriding `MOB_START_INCLUDE_UNCOMMITTED_CHANGES` via an environment variable could print out a wrong value (didn't affect any logic, just wrong console output)

# 0.0.15
- Any `git push` command now uses the `--no-verify` flag

# 0.0.14
- New homepage available at https://mob.sh
- `mob config` prints configuration using the environment variable names which allow overriding the values

# 0.0.13
- Fixes bug that prevented users wih git versions below 2.21 to be able to use 'mob'.
 