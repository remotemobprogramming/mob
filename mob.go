package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const message = "\"Mob Session DONE [ci-skip]\""

var wip_branch = "mob-session"
var base_branch = "master"
var remote_name = "origin"

func main() {
	user_base_branch, user_base_branch_set := os.LookupEnv("MOB_BASE_BRANCH")
	if (user_base_branch_set) {
		base_branch = user_base_branch
	}
	user_wip_branch, user_wip_branch_set := os.LookupEnv("MOB_WIP_BRANCH")
	if (user_wip_branch_set) {
		wip_branch = user_wip_branch
	}

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
	git("checkout", base_branch)
	if hasMobbingBranch() {
		git("branch", "-D", wip_branch)
	}
	if hasMobbingBranchOrigin() {
		git("push", remote_name, "--delete", wip_branch)
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
		git("branch", "-D", wip_branch)
		git("checkout", wip_branch)
		git("branch", "--set-upstream-to="+remote_name+"/"+wip_branch, wip_branch)
	} else if !hasMobbingBranch() && !hasMobbingBranchOrigin() {
		sayInfo("create " + wip_branch + " from " + base_branch)
		git("checkout", base_branch)
		git("merge", remote_name+"/"+base_branch, "--ff-only")
		git("branch", wip_branch)
		git("checkout", wip_branch)
		git("push", "--set-upstream", remote_name, wip_branch)
	} else if !hasMobbingBranch() && hasMobbingBranchOrigin() {
		sayInfo("joining mob session")
		git("checkout", wip_branch)
		git("branch", "--set-upstream-to="+remote_name+"/"+wip_branch, wip_branch)
	} else {
		sayInfo("purging local branch and start new " + wip_branch + " branch from " + base_branch)
		git("branch", "-D", wip_branch) // check if unmerged commits

		git("checkout", base_branch)
		git("merge", remote_name+"/"+base_branch, "--ff-only")
		git("branch", wip_branch)
		git("checkout", wip_branch)
		git("push", "--set-upstream", remote_name, wip_branch)
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
		git("push", remote_name, wip_branch)
		say(changes)
		showNext()
	}

	git("checkout", base_branch)
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
		git("push", remote_name, wip_branch)

		git("checkout", base_branch)
		git("merge", remote_name+"/"+base_branch, "--ff-only")
		git("merge", "--squash", wip_branch)

		git("branch", "-D", wip_branch)
		git("push", remote_name, "--delete", wip_branch)
		say(getCachedChanges())
		sayTodo("git commit -m 'describe the changes'")
	} else {
		git("checkout", base_branch)
		git("branch", "-D", wip_branch)
		sayInfo("someone else already ended your mob session")
	}
}

func status() {
	if isMobbing() {
		sayInfo("mobbing in progress")

		output := silentgit("--no-pager", "log", base_branch+".."+wip_branch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
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
	return strings.Contains(output, "* "+wip_branch)
}

func hasMobbingBranch() bool {
	output := silentgit("branch")
	return strings.Contains(output, "  "+wip_branch) || strings.Contains(output, "* "+wip_branch)
}

func hasMobbingBranchOrigin() bool {
	output := silentgit("branch", "--remotes")
	return strings.Contains(output, "  "+remote_name+"/"+wip_branch)
}

func getGitUserName() string {
	return silentgit("config", "--get", "user.name")
}

func showNext() {
	changes := strings.TrimSpace(silentgit("--no-pager", "log", base_branch+".."+wip_branch, "--pretty=format:%an", "--abbrev-commit"))
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	gitUserName := getGitUserName()
	if numberOfLines < 1 {
		return
	}
	for i := 0; i < len(lines); i++ {
		if lines[i] == gitUserName && i > 0 {
			sayInfo("Probably " + lines[i-1] + " is next")
		}
	}
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
