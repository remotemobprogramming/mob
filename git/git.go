package git

import (
	"bufio"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/workdir"
)

type Client struct {
	PassthroughStderrStdout bool
}

func (g *Client) runCommandSilent(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workdir.Path) > 0 {
		command.Dir = workdir.Path
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Running command <" + commandString + "> in silent mode, capturing combined output")
	outputBytes, err := command.CombinedOutput()
	output := string(outputBytes)
	say.Debug(output)
	return commandString, output, err
}

func (g *Client) runCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workdir.Path) > 0 {
		command.Dir = workdir.Path
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Running command <" + commandString + "> passing output through")

	stdout, _ := command.StdoutPipe()
	command.Stderr = command.Stdout
	errStart := command.Start()
	if errStart != nil {
		return commandString, "", errStart
	}

	output := ""

	stdoutscanner := bufio.NewScanner(stdout)
	lineEnded := true
	stdoutscanner.Split(bufio.ScanBytes)
	for stdoutscanner.Scan() {
		character := stdoutscanner.Text()
		if character == "\n" {
			lineEnded = true
		} else {
			if lineEnded {
				say.PrintToConsole("  ")
				lineEnded = false
			}
		}
		say.PrintToConsole(character)
		output += character
	}

	errWait := command.Wait()
	if errWait != nil {
		say.Debug(output)
		return commandString, output, errWait
	}

	say.Debug(output)
	return commandString, output, nil
}

func (g *Client) Run(args ...string) {
	say.Indented("git " + strings.Join(args, " "))
	commandString, output, err := "", "", error(nil)
	if g.PassthroughStderrStdout {
		commandString, output, err = g.runCommand("git", args...)
	} else {
		commandString, output, err = g.runCommandSilent("git", args...)
	}

	if err != nil {
		if !g.IsRepo() {
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			if strings.Contains(output, "does not support push options") {
				say.Error("The receiving end does not support push options")
				say.Fix("Disable the push option ci.skip in your .mob file or set the expected environment variable", "export MOB_SKIP_CI_PUSH_OPTION_ENABLED=false")
			} else {
				say.Error(commandString)
				say.Error(output)
				say.Error(err.Error())
			}
		}
		exit.Exit(1)
	}
}

func (g *Client) Silent(args ...string) string {
	commandString, output, err := g.runCommandSilent("git", args...)

	if err != nil {
		if !g.IsRepo() {
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			say.Error(commandString)
			say.Error(output)
			say.Error(err.Error())
		}
		exit.Exit(1)
	}
	return strings.TrimSpace(output)
}

func (g *Client) SilentIgnoreFailure(args ...string) (string, error) {
	_, output, err := g.runCommandSilent("git", args...)

	if err != nil {
		return "", err
	}
	return strings.TrimSpace(output), nil
}

func (g *Client) RunWithoutEmptyStrings(args ...string) {
	argsWithoutEmptyStrings := deleteEmptyStrings(args)
	g.Run(argsWithoutEmptyStrings...)
}

func (g *Client) RunIgnoreFailure(args ...string) error {
	commandString, output, err := "", "", error(nil)
	if g.PassthroughStderrStdout {
		commandString, output, err = g.runCommand("git", args...)
	} else {
		commandString, output, err = g.runCommandSilent("git", args...)
	}

	if err != nil {
		if !g.IsRepo() {
			say.Error("expecting the current working directory to be a git repository.")
			exit.Exit(1)
		} else {
			say.Warning(commandString)
			say.Warning(output)
			say.Warning(err.Error())
			return err
		}
	}

	say.Indented(commandString)
	return nil
}

func HooksOption(c config.Configuration) string {
	if c.GitHooksEnabled {
		return ""
	} else {
		return "--no-verify"
	}
}

func (g *Client) CurrentBranch() string {
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return g.Silent("rev-parse", "--abbrev-ref", "HEAD")
}

func (g *Client) Branches() []string {
	return strings.Split(g.Silent("branch", "--format=%(refname:short)"), "\n")
}

func (g *Client) RemoteBranches() []string {
	return strings.Split(g.Silent("branch", "--remotes", "--format=%(refname:short)"), "\n")
}

func (g *Client) UserName() string {
	output, _ := g.SilentIgnoreFailure("config", "--get", "user.name")
	return output
}

func (g *Client) UserEmail() string {
	return g.Silent("config", "--get", "user.email")
}

func (g *Client) IsRepo() bool {
	_, _, err := g.runCommandSilent("git", "rev-parse")
	return err == nil
}

func (g *Client) RootDir() string {
	return g.Silent("rev-parse", "--show-toplevel")
}

func (g *Client) Dir() string {
	return g.Silent("rev-parse", "--absolute-git-dir")
}

func (g *Client) HasCommits() bool {
	commitCount := g.Silent("rev-list", "--all", "--count")
	return commitCount != "0"
}

func (g *Client) DoBranchesDiverge(ancestor string, successor string) bool {
	_, _, err := g.runCommandSilent("git", "merge-base", "--is-ancestor", ancestor, successor)
	if err == nil {
		return false
	}
	return true
}

func (g *Client) Version() string {
	_, output, err := g.runCommandSilent("git", "--version")
	if err != nil {
		say.Debug("gitVersion encountered an error: " + err.Error())
		return ""
	}
	return strings.TrimSpace(output)
}

func (g *Client) IsNothingToCommit() bool {
	output := g.Silent("status", "--porcelain")
	return len(output) == 0
}

func (g *Client) HasUncommittedChanges() bool {
	return !g.IsNothingToCommit()
}

func (g *Client) CommitHash() string {
	output, _ := g.SilentIgnoreFailure("rev-parse", "HEAD")
	return output
}

type GitVersion struct {
	Major int
	Minor int
	Patch int
}

func ParseVersion(version string) GitVersion {
	// The git version string can be customized, so we need a more complex regex, for example: git version 2.38.1.windows.1
	// "git" and "version" are optional, and the version number can be x, x.y or x.y.z
	r := regexp.MustCompile(`(?:git)?(?: version )?(?P<major>\d+)(?:\.(?P<minor>\d+)(?:\.(?P<patch>\d+))?)?`)
	matches := r.FindStringSubmatch(version)
	var v GitVersion
	var err error
	if len(matches) > r.SubexpIndex("major") {
		v.Major, err = strconv.Atoi(matches[r.SubexpIndex("major")])
		if err != nil {
			v.Major = 0
			return v
		}
	}
	if len(matches) > r.SubexpIndex("minor") {
		v.Minor, err = strconv.Atoi(matches[r.SubexpIndex("minor")])
		if err != nil {
			v.Minor = 0
			return v
		}
	}
	if len(matches) > r.SubexpIndex("patch") {
		v.Patch, err = strconv.Atoi(matches[r.SubexpIndex("patch")])
		if err != nil {
			v.Patch = 0
		}
	}
	return v
}

func (v GitVersion) Less(rhs GitVersion) bool {
	return v.Major < rhs.Major ||
		(v.Major == rhs.Major && v.Minor < rhs.Minor) ||
		(v.Major == rhs.Major && v.Minor == rhs.Minor && v.Patch < rhs.Patch)
}

func deleteEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}
