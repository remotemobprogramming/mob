package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
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
		status()
	} else if argument == "d" || argument == "done" || argument == "e" || argument == "end" {
		done()
		status()
	} else if argument == "r" || argument == "reset" {
		reset()
		status()
	} else if argument == "t" || argument == "timer" {
		if len(os.Args) > 2 {
			timer := os.Args[2]
			startTimer(timer)
		} else {
			fmt.Println("provide the number of minutes for the timer")
			fmt.Println("try 'mob timer 10'")
		}
	} else if argument == "h" || argument == "help" {
		help()
	} else if argument == "status" {
		status()
	} else {
		status()
		help()
	}
}

func isDebug() bool {
	_, isSet := os.LookupEnv("MOB_DEBUG")
	return isSet
}

func isInfo() bool {
	return !isDebug()
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
	err := command.Start()
	if err != nil {
		say("timer couldn't be started... (timer only works on OSX)")
	}
}

func reset() {
	git("fetch")
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
		say("uncommitted changes, aborting 'mob start'")
		return
	}

	git("fetch") // abort if didn't work

	if hasMobbingBranch() && hasMobbingBranchOrigin() {
		say("rejoining mob session")
		git("checkout", branch)
		git("merge", "origin/"+branch, "--ff-only")
		git("branch", "--set-upstream-to=origin/"+branch, branch)
	} else if !hasMobbingBranch() && !hasMobbingBranchOrigin() {
		say("create " + branch + " from master")
		git("checkout", master)
		git("merge", "origin/master", "--ff-only")
		git("branch", branch)
		git("checkout", branch)
		git("push", "--set-upstream", "origin", branch)
	} else if !hasMobbingBranch() && hasMobbingBranchOrigin() {
		say("joining mob session")
		git("checkout", branch)
	} else {
		say("purging local branch and start new " + branch + " branch from " + master)
		git("branch", "-D", branch) // check if unmerged commits

		git("checkout", master)
		git("merge", "origin/master", "--ff-only")
		git("branch", branch)
		git("checkout", branch)
		git("push", "--set-upstream", "origin", branch)
	}

	say("start hacking")

	if len(os.Args) > 2 {
		timer := os.Args[2]
		startTimer(timer)
	} else {
		fmt.Println("provide the number of minutes for the timer")
		fmt.Println("try 'mob start 10'")
	}
}

func next() {
	if !isMobbing() {
		say("nothing was done, because you aren't mobbing")
		say("try 'mob start 10' to start the next mob session with a ten-minute timer")
		return
	}

	if isNothingToCommit() {
		say("nothing was done, so nothing to commit")
	} else {
		git("add", "--all")
		git("commit", "--message", "\"WIP in Mob Session [ci-skip]\"")
		git("push", "origin", branch)
	}

	git("checkout", master)
	say("join the 'rest of the mob'")
}

func done() {
	if !isMobbing() {
		say("nothing was done, because you aren't mobbing")
		say("try 'mob start 10' to start the next mob session with a ten-minute timer")
		return
	}

	git("fetch")

	if hasMobbingBranchOrigin() {
		if !isNothingToCommit() {
			git("add", "--all")
			git("commit", "--message", message)
		}
		git("push", "origin", branch)

		git("checkout", master)
		git("merge", "--squash", branch)

		git("branch", "-D", branch)
		git("push", "origin", "--delete", branch)

		say("lean back, you survived your mob session :-)")
		say("execute 'git commit' to describe what the mob achieved")
	} else {
		git("checkout", master)
		git("branch", "-D", branch)
		say("someone else already ended your mob session")
	}
}

func status() {
	if isMobbing() {
		say("mobbing in progress")

		output := silentgit("--no-pager", "log", master+".."+branch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
		fmt.Println(output)
	} else {
		say("you aren't mobbing right now")
		say("try 'mob start 10' to start the next mob session with a ten-minute timer")
	}

	if !hasSay() {
		say("text-to-speech disabled because 'say' not found")
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
	if isDebug() || isInfo() {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if isDebug() {
		fmt.Println(output)
	}
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
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
