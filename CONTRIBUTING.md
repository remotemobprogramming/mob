# Contributing

Contributions are very welcome! Below are some helpful guidelines.

## How to build the project

**mob** requires at least [Go](https://go.dev/) 1.15 to build:

```
$ cd /path/to/git/clone/of/remotemobprogramming/mob
$ go build
```

Running single test files during development is probably easiest in your IDE.
To check if all tests are passing, simply run

```
$ go test -v
```

To do some manual testing, you can install the new binary to `/usr/local/bin/`:

```
$ ./install
```

Afterwards, you can check if everything works as you expect.
If it does not, you might want to add the `--debug` option to your call:

```
$ mob config --debug
```

## How to contribute

If you want to tackle an existing issue please add a comment on GitHub to make sure the issue is
sufficiently discussed and that no two contributors collide by working on the same issue. 
To submit a contribution, please follow the following workflow:

- Fork the project
- Create a feature branch
- Add your contribution
- Test your changes locally, i.e. do an `./install` and try your new version of `mob`
- Run all the tests via `go test -v`, and if they pass:
- Create a Pull Request

By creating a pull request you certify that your code conforms to the
[Developer Certificate of Origin (DCO)](https://developercertificate.org/).
This essentially means that you have the right to contribute the code under
this project's [license](LICENSE) and that you agree to this license.

### Commits

Commit messages should be clear and fully elaborate the context and the reason of a change.
If your commit refers to an issue, please post-fix it with the issue number, e.g.

```
Issue: #123
```

You can sign-off your commits with `git commit -m "..." -s`, but you're not required to do so.

### Pull Requests

If your Pull Request resolves an issue, please add a respective line to the end, like

```
Resolves #123
```

That's it! Happy contributing 😃
