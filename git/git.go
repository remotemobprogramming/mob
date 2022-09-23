package git

import (
	"bufio"
	"github.com/remotemobprogramming/mob/v4/say"
	"os"
	"os/exec"
	"strings"
)

var (
	GitPassthroughStderrStdout = false // hack to get git hooks to print to stdout/stderr
	workingDir                 = ""
)

func DotGitDir() string {
	return SilentGit("rev-parse", "--absolute-git-dir")
}

func RootDir() string {
	return strings.TrimSuffix(DotGitDir(), "/.git")
}

func IsNothingToCommit() bool {
	output := SilentGit("status", "--short")
	return len(output) == 0
}

func HasUncommittedChanges() bool {
	return !IsNothingToCommit()
}

func Branches() []string {
	return strings.Split(SilentGit("branch", "--format=%(refname:short)"), "\n")
}

func RemoteBranches() []string {
	return strings.Split(SilentGit("branch", "--remotes", "--format=%(refname:short)"), "\n")
}

func DoBranchesDiverge(ancestor string, successor string) bool {
	_, _, err := RunCommandSilent("git", "merge-base", "--is-ancestor", ancestor, successor)
	if err == nil {
		return false
	}
	return true
}

func UserName() string {
	return SilentGitIgnoreFailure("config", "--get", "user.name")
}

func UserEmail() string {
	return SilentGit("config", "--get", "user.email")
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

func GitWithoutEmptyStrings(args ...string) {
	argsWithoutEmptyStrings := deleteEmptyStrings(args)
	Git(argsWithoutEmptyStrings...)
}

func SetWorkingDir(dir string) {
	workingDir = dir
}

func SilentGit(args ...string) string {
	commandString, output, err := RunCommandSilent("git", args...)

	if err != nil {
		if !IsRepository() {
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			say.Error(commandString)
			say.Error(output)
			say.Error(err.Error())
		}
		os.Exit(1)
	}
	return strings.TrimSpace(output)
}

func Git(args ...string) {
	commandString, output, err := "", "", error(nil)
	if GitPassthroughStderrStdout {
		commandString, output, err = RunCommand("git", args...)
	} else {
		commandString, output, err = RunCommandSilent("git", args...)
	}

	if err != nil {
		if !IsRepository() {
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			say.Error(commandString)
			say.Error(output)
			say.Error(err.Error())
		}
		os.Exit(1)
	} else {
		say.Indented(commandString)
	}
}

func CommitHash() string {
	return SilentGitIgnoreFailure(workingDir, "rev-parse", "HEAD")
}

func GitIgnoreFailure(args ...string) error {
	commandString, output, err := "", "", error(nil)
	if GitPassthroughStderrStdout {
		commandString, output, err = RunCommand("git", args...)
	} else {
		commandString, output, err = RunCommandSilent("git", args...)
	}

	say.Indented(commandString)

	if err != nil {
		if !IsRepository() {
			say.Error("expecting the current working directory to be a git repository.")
			os.Exit(1)
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

func SilentGitIgnoreFailure(args ...string) string {
	_, output, err := RunCommandSilent("git", args...)

	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

func IsInstalled() bool {
	_, _, err := RunCommandSilent("", "git", "--version")
	if err != nil {
		say.Debug("isGitInstalled encountered an error: " + err.Error())
	}
	return err == nil
}

func IsRepository() bool {
	_, _, err := RunCommandSilent("git", "rev-parse")
	return err == nil
}

func RunCommandSilent(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Running command <" + commandString + "> in silent mode, capturing combined output")
	outputBytes, err := command.CombinedOutput()
	output := string(outputBytes)
	say.Debug(output)
	return commandString, output, err
}

func RunCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
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
