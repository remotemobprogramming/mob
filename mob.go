package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const branch = "mob-session"
const message = "\"Mob Session DONE [ci-skip]\""
const master = "master"

func main() {
	argument := getCommand()
	if argument == "s" || argument == "start" {
		start()
		status()
	} else if argument == "n" || argument == "next" {
		next()
	} else if argument == "d" || argument == "done" || argument == "e" || argument == "end" {
		done()
	} else if argument == "r" || argument == "reset" {
		reset()
	} else if argument == "t" || argument == "timer" {
		if len(os.Args) > 2 {
			timer := os.Args[2]
			startTimer(timer)
		}
	} else if argument == "h" || argument == "help" || argument == "--help" || argument == "-h" {
		help()
	} else {
		status()
	}
}

func isDebug() bool {
	_, isSet := os.LookupEnv("MOB_DEBUG")
	return isSet
}

func startTimer(timerInMinutes string) {
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)

	command := exec.Command("sh", "-c", "( sleep "+timerInSeconds+" && say \"time's up\" & )")
	if isDebug() {
		fmt.Println(command.Args)
	}
	err := command.Start()
	if err != nil {
		sayError("timer couldn't be started... (timer only works on OSX)")
		sayError(err)
	} else {
		timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
		sayOkay(timerInMinutes + " minutes timer started (finishes at approx. " + timeOfTimeout + ")")
	}
}

func reset() {
	git("fetch", "--prune")
	git("checkout", master)
	if hasMobbingBranch() {
		git("branch", "-D", branch)
	}
	if hasMobbingBranchOrigin() {
		git("push", "origin", "--delete", branch)
	}
}

func start() {
	if !isNothingToCommit() {
		sayNote("uncommitted changes")
		return
	}

	git("fetch", "--prune")

	if hasMobbingBranch() && hasMobbingBranchOrigin() {
		sayInfo("rejoining mob session")
		git("branch", "-D", branch)
		git("checkout", branch)
		git("branch", "--set-upstream-to=origin/"+branch, branch)
	} else if !hasMobbingBranch() && !hasMobbingBranchOrigin() {
		sayInfo("create " + branch + " from master")
		git("checkout", master)
		git("merge", "origin/master", "--ff-only")
		git("branch", branch)
		git("checkout", branch)
		git("push", "--set-upstream", "origin", branch)
	} else if !hasMobbingBranch() && hasMobbingBranchOrigin() {
		sayInfo("joining mob session")
		git("checkout", branch)
		git("branch", "--set-upstream-to=origin/"+branch, branch)
	} else {
		sayInfo("purging local branch and start new " + branch + " branch from " + master)
		git("branch", "-D", branch) // check if unmerged commits

		git("checkout", master)
		git("merge", "origin/master", "--ff-only")
		git("branch", branch)
		git("checkout", branch)
		git("push", "--set-upstream", "origin", branch)
	}

	if len(os.Args) > 2 {
		timer := os.Args[2]
		startTimer(timer)
	}
}

func next() {
	if !isMobbing() {
		sayError("you aren't mobbing")
		return
	}

	if isNothingToCommit() {
		sayInfo("nothing was done, so nothing to commit")
	} else {
		git("add", "--all")
		git("commit", "--message", "\"WIP in Mob Session [ci-skip]\"")
		changes := getChangesOfLastCommit()
		git("push", "origin", branch)
		say(changes)
	}

	git("checkout", master)
}

func getChangesOfLastCommit() string {
	return strings.TrimSpace(silentgit("diff", "HEAD^1", "--stat"))
}

func getCachedChanges() string {
	return strings.TrimSpace(silentgit("diff", "--cached", "--stat"))
}

func done() {
	if !isMobbing() {
		sayError("you aren't mobbing")
		return
	}

	git("fetch", "--prune")

	if hasMobbingBranchOrigin() {
		if !isNothingToCommit() {
			git("add", "--all")
			git("commit", "--message", message)
		}
		git("push", "origin", branch)

		git("checkout", master)
		git("merge", "origin/"+master, "--ff-only")
		git("merge", "--squash", branch)

		git("branch", "-D", branch)
		git("push", "origin", "--delete", branch)
		say(getCachedChanges())
		sayTodo("git commit -m 'describe the changes'")
	} else {
		git("checkout", master)
		git("branch", "-D", branch)
		sayInfo("someone else already ended your mob session")
	}
}

func status() {
	if isMobbing() {
		sayInfo("mobbing in progress")

		output := silentgit("--no-pager", "log", master+".."+branch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
		say(output)
	} else {
		sayInfo("you aren't mobbing right now")
	}

	if !hasSay() {
		sayNote("text-to-speech disabled because 'say' not found")
	}
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	isMobbing := len(strings.TrimSpace(output)) == 0
	return isMobbing
}

func isMobbing() bool {
	output := silentgit("branch")
	return strings.Contains(output, "* "+branch)
}

func hasMobbingBranch() bool {
	output := silentgit("branch")
	return strings.Contains(output, "  "+branch) || strings.Contains(output, "* "+branch)
}

func hasMobbingBranchOrigin() bool {
	output := silentgit("branch", "--remotes")
	return strings.Contains(output, "  origin/"+branch)
}

func getGitUserName() string {
	return silentgit("config", "--get", "user.name")
}

func showNext() {
	// output := silentgit("--no-pager", "log", master+".."+branch, "--pretty=format:%an", "--abbrev-commit")
	// to lines; find own name and ignore very last line; try to determine next
}

func help() {
	say("usage")
	say("\tmob [s]tart \t# start mobbing as typist")
	say("\tmob [n]ext \t# hand over to next typist")
	say("\tmob [d]one \t# finish mob session")
	say("\tmob [r]eset \t# resets any unfinished mob session")
	say("\tmob status \t# show status of mob session")
	say("\tmob [h]elp \t# prints this help")
}

func silentgit(args ...string) string {
	command := exec.Command("git", args...)
	if isDebug() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	if err != nil {
		fmt.Println(output)
		fmt.Println(err)
		os.Exit(1)
	}
	return output
}

func hasSay() bool {
	command := exec.Command("which", "say")
	if isDebug() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	return err == nil
}

func git(args ...string) string {
	command := exec.Command("git", args...)
	if isDebug() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	if err != nil {
		sayError(command.Args)
		sayError(err)
		os.Exit(1)
	} else {
		sayOkay(command.Args)
	}
	return output
}

func say(s string) {
	fmt.Println(s)
}

func sayError(s interface{}) {
	fmt.Print(" ⚡ ")
	fmt.Print(s)
	fmt.Print("\n")
}

func sayOkay(s interface{}) {
	fmt.Print(" ✓ ")
	fmt.Print(s)
	fmt.Print("\n")
}

func sayNote(s interface{}) {
	fmt.Print(" ❗ ")
	fmt.Print(s)
	fmt.Print("\n")
}

func sayTodo(s interface{}) {
	fmt.Print(" ☐ ")
	fmt.Print(s)
	fmt.Print("\n")
}

func sayInfo(s string) {
	fmt.Print(" > ")
	fmt.Print(s)
	fmt.Print("\n")
}

func getCommand() string {
	args := os.Args
	if len(args) <= 1 {
		return ""
	}
	return args[1]
}
