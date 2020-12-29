# Inbox

These items are _not_ in priority order.

- Remove the duplicate pattern in tests of `Printf` then `FailNow()` by introducing a "fail with failure function.
- Add `--squash` option to `mob done` that corresponds to `--no-squash`.
- When parsing environment variables, rename the variable `changed` to `overridden`.
- Add `sayWarning()` that corresponds to `sayError()`.
- Try introducing parameterized tests?
- Improve the message at the end of `mob done`:
  - The message currently directs the user to do `git commit`, even when there are no uncommitted changes.
  - Toyota solution: Change the message to say "If you have uncommitted changes, then commit them now."
  - Lexus solution: Detect whether there are uncommitted changes, then display an appropriate message.

