# Usage Scenarios

## Variant A

```
# person A
mob start
mob next

# person B
mob start
mob next

# person C
mob start
mob done (git merge --squash)
git commit --message "This is it"
git push
```

## Variant: keep manual commits

```
# person A
mob start
git commit --message "one"
git commit --message "two"
 # no unstaged changes
mob next

# person B
mob start
git commit --message "three"
# no unstaged changes
mob next

# person C
mob start
git commit --message "four"
git commit --message "five"
# no unstaged changes
mob done (git merge --no-commit)  # I want `mob done` without squashing
git push
```

## Variant: keep manual commits (what about mob next commits)

```
# person A
mob start
git commit --message "one"
git commit --message "two"
# unstaged changes
mob next (git commit --message "mob next [ci-skip]")

# person B
mob start
git commit --amend --message "three" # previous handover commit is gone
# unstaged changes
mob next (git commit --message "mob next [ci-skip]")

# person C
mob start
git commit --message "four"
git commit --message "five"
# unstaged changes
mob done (git commit --message "mob next [ci-skip]"; git push; git merge --no-commit)
# mob done prints out a warning that there might be mob next commits (with heuristic print out git commit hashes of the wip commits)
git push
```

## Abandoned variant

```
# person A
mob start
mob next --message "one"
mob next --message "two" # feels strange

# person B
mob start
mob next --message "three"

# person C
mob start
git commit --message "four"
git commit --message "five"
# no unstaged changes
mob done (git merge --no-commit)
git push
```


## Explain mob next

```
MOB_NEXT_STAY=false # old behavior

main $ mob start
mob/main $ do something
mob/main $ mob next
main $
main $ mob start


MOB_NEXT_STAY=true # new behavior
main $ mob start
mob/main $ do something
mob/main $ mob next
mob/main $
mob/main $ mob start
```