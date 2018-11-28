package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const branch = "mob-session"

func isDebug() bool {
	_, isSet := os.LookupEnv("MOB_DEBUG")
	return isSet
}

func isInfo() bool {
	return !isDebug()
}

func main() {
	argument := getCommand()
	if argument == "s" || argument == "start" {
		start()
		status()
	} else if argument == "n" || argument == "next" {
		next()
		status()
	} else if argument == "d" || argument == "done" {
		done()
		status()
	} else if argument == "r" || argument == "reset" {
		reset()
		status()
	} else if argument == "t" || argument == "timer" {
		if len(os.Args) > 2 {
			timer := os.Args[2]
			startTimer(timer)
		}
	} else if argument == "h" || argument == "help" {
		help()
	} else {
		status()
	}
}

func startTimer(timerInMinutes string) {
	fmt.Println("starting " + timerInMinutes + " minutes timer")

	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)

	command := exec.Command("sh", "-c", "( sleep "+timerInSeconds+" && say \"time's up\" & )")
	if isDebug() {
		fmt.Println(command.Args)
	}
	command.Start()
}

func reset() {
	git("checkout", "master")
	if hasMobbingBranch() {
		git("branch", "-D", branch)
	}
	if hasMobbingBranchOrigin() {
		git("push", "origin", "--delete", branch)
	}
}

func start() {
	git("checkout", "master")
	if !isNothingToCommit() {
		say("uncommitted changes, aborting 'mob start'")
		return
	}
	git("pull")
	git("fetch")
	if !hasMobbingBranch() {
		git("branch", branch)
	}
	git("checkout", branch)
	if hasMobbingBranchOrigin() {
		git("merge", "origin/"+branch)
		git("branch", "--set-upstream-to=origin/"+branch, branch)
	}
	say("start hacking")

	if len(os.Args) > 2 {
		timer := os.Args[2]
		startTimer(timer)
	}
}

func next() {
	if isNothingToCommit() {
		say("nothing was done, so nothing to commit")
		return
	}
	git("add", ".", "--all")
	git("commit", "--message", "\"WIP in Mob Session [ci-skip]\"")
	git("push", "origin", branch)
	say("join the 'rest of the mob'")
}

func done() {
	git("checkout", "master")
	git("merge", "--squash", branch)
	git("branch", "-D", branch)
	git("push", "origin", "--delete", branch)
	say("lean back, you survived your mob session :-)")
	say("execute 'git commit' to describe what the mob achieved")
}

func status() {
	if isMobbing() {
		say("mobbing in progress")

		output := silentgit("--no-pager", "log", "master.."+branch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
		fmt.Println(output)
	} else {
		say("you aren't mobbing right now")
	}
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	isMobbing := len(strings.TrimSpace(output)) == 0
	return isMobbing
}

func isMobbing() bool {
	output := silentgit("branch")
	isMobbing := strings.Contains(output, "* "+branch)
	return isMobbing
}

func hasMobbingBranch() bool {
	output := silentgit("branch")
	return strings.Contains(output, "  "+branch) || strings.Contains(output, "* "+branch)
}

func hasMobbingBranchOrigin() bool {
	silentgit("fetch")
	output := silentgit("branch", "--remotes")
	return strings.Contains(output, "  origin/"+branch)
}

func help() {
	say("usage")
	say("\tmob [s]tart \t# start mobbing as typist")
	say("\tmob [n]ext \t# hand over to next typist")
	say("\tmob [d]one \t# finish mob session")
	say("\tmob [r]eset \t# resets any unfinished mob session")
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
	if err != nil && isDebug() {
		fmt.Println(err)
	}
	return output
}

func git(args ...string) string {
	command := exec.Command("git", args...)
	if isDebug() || isInfo() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	if err != nil && (isDebug() || isInfo()) {
		fmt.Println(err)
	}
	return output
}

func say(s string) {
	fmt.Println(s)
}

func getCommand() string {
	args := os.Args
	if len(args) <= 1 {
		return ""
	}
	return args[1]
}
