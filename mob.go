package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var wip_branch = "mob-session"                        // override with MOB_WIP_BRANCH environment variable
var base_branch = "master"                            // override with MOB_BASE_BRANCH environment variable
var remote_name = "origin"                            // override with MOB_REMOTE_NAME environment variable
var wip_commit_message = "Mob Session DONE [ci-skip]" // override with MOB_WIP_COMMIT_MESSAGE environment variable
var debug = false                                     // override with MOB_DEBUG environment variable

func parseEnvironmentVariables() {
	user_base_branch, user_base_branch_set := os.LookupEnv("MOB_BASE_BRANCH")
	if user_base_branch_set {
		base_branch = user_base_branch
		say("overriding MOB_BASE_BRANCH=" + base_branch)
	}
	user_wip_branch, user_wip_branch_set := os.LookupEnv("MOB_WIP_BRANCH")
	if user_wip_branch_set {
		wip_branch = user_wip_branch
		say("overriding MOB_WIP_BRANCH=" + wip_branch)
	}
	user_remote_name, user_remote_name_set := os.LookupEnv("MOB_REMOTE_NAME")
	if user_remote_name_set {
		remote_name = user_remote_name
		say("overriding MOB_REMOTE_NAME=" + remote_name)
	}
	user_wip_commit_message, user_wip_commit_message_set := os.LookupEnv("MOB_WIP_COMMIT_MESSAGE")
	if user_wip_commit_message_set {
		wip_commit_message = user_wip_commit_message
		say("overriding MOB_WIP_COMMIT_MESSAGE=" + wip_commit_message)
	}
	_, user_mob_debug_set := os.LookupEnv("MOB_DEBUG")
	if user_mob_debug_set {
		debug = true
		say("overriding MOB_DEBUG=" + strconv.FormatBool(debug))
	}
}

func main() {
	parseEnvironmentVariables()

	argument := getCommand()
	if argument == "s" || argument == "start" {
		start()
		status()
	} else if argument == "j" || argument == "join" {
		start()
		join()
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
	} else if argument == "v" || argument == "version" || argument == "--version" || argument == "-v" {
		version()
	} else {
		status()
	}
}

func join() {
	if !isLastChangeSecondsAgo() {
		sayInfo("Actively waiting for new remote commit...")
	}
	for !isLastChangeSecondsAgo() {
		time.Sleep(time.Second)
		git("pull")
	}
}

func startTimer(timerInMinutes string) {
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)

	command := exec.Command("sh", "-c", "( sleep "+timerInSeconds+" && say \"time's up\" && (/usr/bin/osascript -e 'display notification \"time is up\"' || /usr/bin/notify-send \"time is up\")  & )")
	if debug {
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
	git("pull")

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
		git("commit", "--message", "\""+wip_commit_message+"\"")
		changes := getChangesOfLastCommit()
		git("push", remote_name, wip_branch)
		say(changes)
	}
	showNext()

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
			git("commit", "--message", "\""+wip_commit_message+"\"")
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
	return strings.TrimSpace(silentgit("config", "--get", "user.name"))
}

func isLastChangeSecondsAgo() bool {
	changes := silentgit("--no-pager", "log", base_branch+".."+wip_branch, "--pretty=format:%cr", "--abbrev-commit")
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	if numberOfLines < 1 {
		return true
	}

	return strings.Contains(lines[0], "seconds ago") || strings.Contains(lines[0], "second ago")
}

func showNext() {
	if debug {
		say("determining next person based on previous changes")
	}
	changes := strings.TrimSpace(silentgit("--no-pager", "log", base_branch+".."+wip_branch, "--pretty=format:%an", "--abbrev-commit"))
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	if debug {
		say("there have been " + strconv.Itoa(numberOfLines) + " changes")
	}
	gitUserName := getGitUserName()
	if debug {
		say("current git user.name is '" + gitUserName + "'")
	}
	if numberOfLines < 1 {
		return
	}
	var history = ""
	for i := 0; i < len(lines); i++ {
		if lines[i] == gitUserName && i > 0 {
			sayInfo("Committers after your last commit: " + history)
			sayInfo("***" + lines[i-1] + "*** is (probably) next.")
			return
		}
		if history != "" {
			history = ", " + history
		}
		history = lines[i] + history
	}
}

func help() {
	say("usage")
	say("\tmob [s]tart \t# start mobbing as typist")
	say("\tmob [j]oin \t# like start but waits for recent commit")
	say("\tmob [n]ext \t# hand over to next typist")
	say("\tmob [d]one \t# finish mob session")
	say("\tmob [r]eset \t# resets any unfinished mob session")
	say("\tmob status \t# show status of mob session")
	say("\tmob --help \t# prints this help")
	say("\tmob --version \t# prints the version")
}

func version() {
	say("BEST version")
}

func silentgit(args ...string) string {
	command := exec.Command("git", args...)
	if debug {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if debug {
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
	if debug {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if debug {
		fmt.Println(output)
	}
	return err == nil
}

func git(args ...string) string {
	command := exec.Command("git", args...)
	if debug {
		fmt.Println(command.Args)
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if debug {
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
