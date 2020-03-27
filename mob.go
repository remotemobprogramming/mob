package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var wipBranch = "mob-session"               // override with MOB_WIP_BRANCH environment variable
var baseBranch = "master"                   // override with MOB_BASE_BRANCH environment variable
var remoteName = "origin"                   // override with MOB_REMOTE_NAME environment variable
var wipCommitMessage = "mob next [ci-skip]" // override with MOB_WIP_COMMIT_MESSAGE environment variable
var mobNextStay = false                     // override with MOB_NEXT_STAY environment variable
var voiceCommand = "say"                    // override with MOB_VOICE_COMMAND environment variable
var debug = false                           // override with MOB_DEBUG environment variable

func parseEnvironmentVariables() []string {
	userBaseBranch, userBaseBranchSet := os.LookupEnv("MOB_BASE_BRANCH")
	if userBaseBranchSet {
		baseBranch = userBaseBranch
		say("overriding MOB_BASE_BRANCH=" + baseBranch)
	}
	userWipBranch, userWipBranchSet := os.LookupEnv("MOB_WIP_BRANCH")
	if userWipBranchSet {
		wipBranch = userWipBranch
		say("overriding MOB_WIP_BRANCH=" + wipBranch)
	}
	userRemoteName, userRemoteNameSet := os.LookupEnv("MOB_REMOTE_NAME")
	if userRemoteNameSet {
		remoteName = userRemoteName
		say("overriding MOB_REMOTE_NAME=" + remoteName)
	}
	userWipCommitMessage, userWipCommitMessageSet := os.LookupEnv("MOB_WIP_COMMIT_MESSAGE")
	if userWipCommitMessageSet {
		wipCommitMessage = userWipCommitMessage
		say("overriding MOB_WIP_COMMIT_MESSAGE=" + wipCommitMessage)
	}
	userMobVoiceCommand, userMobVoiceCommandSet := os.LookupEnv("MOB_VOICE_COMMAND")
	if userMobVoiceCommandSet {
		voiceCommand = userMobVoiceCommand
		say("overriding MOB_VOICE_COMMAND=" + voiceCommand)
	}
	_, userMobDebugSet := os.LookupEnv("MOB_DEBUG")
	if userMobDebugSet {
		debug = true
		say("overriding MOB_DEBUG=" + strconv.FormatBool(debug))
	}
	_, userMobNextStaySet := os.LookupEnv("MOB_NEXT_STAY")
	if userMobNextStaySet {
		mobNextStay = true
		say("overriding MOB_NEXT_STAY=" + strconv.FormatBool(mobNextStay))
	}

	flagMobNextStaySet := flag.Bool("stay", false, "don't change back")
	flagMobNextSSet := flag.Bool("s", false, "(shorthand)")

	flag.Parse()

	if *flagMobNextStaySet {
		mobNextStay = true
	}
	if *flagMobNextSSet {
		mobNextStay = true
	}

	return flag.Args()
}

func main() {
	args := parseEnvironmentVariables()
	command := getCommand(args)
	parameter := getParameters(args)
	if debug {
		fmt.Println("Args '" + strings.Join(args, " ") + "'")
		fmt.Println("command '" + command + "'")
		fmt.Println("parameter '" + strings.Join(parameter, " ") + "'")
	}

	if command == "s" || command == "start" {
		start(parameter)
		status()
	} else if command == "n" || command == "next" {
		next()
	} else if command == "d" || command == "done" || command == "e" || command == "end" {
		done()
	} else if command == "r" || command == "reset" {
		reset()
	} else if command == "t" || command == "timer" {
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		}
	} else if command == "share" {
		startZoomScreenshare()
	} else if command == "h" || command == "help" || command == "--help" || command == "-h" {
		help()
	} else if command == "v" || command == "version" || command == "--version" || command == "-v" {
		version()
	} else {
		status()
	}
}

func startTimer(timerInMinutes string) {
	if debug {
		fmt.Println("Starting timer for " + timerInMinutes + " minutes")
	}
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)

	command := exec.Command("sh", "-c", "( sleep "+timerInSeconds+" && "+voiceCommand+" \"mob next\" && (/usr/bin/osascript -e 'display notification \"mob next\"' || /usr/bin/notify-send \"mob next\")  & )")
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
	git("checkout", baseBranch)
	if hasMobbingBranch() {
		git("branch", "-D", wipBranch)
	}
	if hasMobbingBranchOrigin() {
		git("push", remoteName, "--delete", wipBranch)
	}
}

func start(parameter []string) {
	if !isNothingToCommit() {
		sayNote("uncommitted changes")
		return
	}

	git("fetch", "--prune")
	git("pull", "--ff-only")

	if hasMobbingBranch() && hasMobbingBranchOrigin() {
		sayInfo("rejoining mob session")
		if !isMobbing() {
			git("branch", "-D", wipBranch)
			git("checkout", wipBranch)
			git("branch", "--set-upstream-to="+remoteName+"/"+wipBranch, wipBranch)
		}
	} else if !hasMobbingBranch() && !hasMobbingBranchOrigin() {
		sayInfo("create " + wipBranch + " from " + baseBranch)
		git("checkout", baseBranch)
		git("merge", remoteName+"/"+baseBranch, "--ff-only")
		git("branch", wipBranch)
		git("checkout", wipBranch)
		git("push", "--set-upstream", remoteName, wipBranch)
	} else if !hasMobbingBranch() && hasMobbingBranchOrigin() {
		sayInfo("joining mob session")
		git("checkout", wipBranch)
		git("branch", "--set-upstream-to="+remoteName+"/"+wipBranch, wipBranch)
	} else {
		sayInfo("purging local branch and start new " + wipBranch + " branch from " + baseBranch)
		git("branch", "-D", wipBranch) // check if unmerged commits

		git("checkout", baseBranch)
		git("merge", remoteName+"/"+baseBranch, "--ff-only")
		git("branch", wipBranch)
		git("checkout", wipBranch)
		git("push", "--set-upstream", remoteName, wipBranch)
	}

	if len(parameter) > 0 {
		timer := parameter[0]
		startTimer(timer)
	}

	if len(parameter) > 1 && parameter[1] == "share" {
		startZoomScreenshare()
	}
}

func startZoomScreenshare() {
	command := exec.Command("sh", "-c", "(osascript -e 'tell application \"System Events\" to keystroke \"S\" using {shift down, command down}')")
	if debug {
		fmt.Println(command.Args)
	}
	err := command.Start()
	if err != nil {
		sayError("screenshare couldn't be started... (screenshare only works on OSX)")
		sayError(err)
	} else {
		sayOkay("Sharing screen with zoom (requires the global shortcut SHIFT+COMMAND+S)")
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
		git("commit", "--message", "\""+wipCommitMessage+"\"", "--no-verify")
		changes := getChangesOfLastCommit()
		git("push", remoteName, wipBranch)
		say(changes)
	}
	showNext()

	if !mobNextStay {
		git("checkout", baseBranch)
	}
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
			git("commit", "--message", "\""+wipCommitMessage+"\"", "--no-verify")
		}
		git("push", remoteName, wipBranch)

		git("checkout", baseBranch)
		git("merge", remoteName+"/"+baseBranch, "--ff-only")
		git("merge", "--squash", "--ff", wipBranch)

		git("branch", "-D", wipBranch)
		git("push", remoteName, "--delete", wipBranch)
		say(getCachedChanges())
		sayTodo("git commit -m 'describe the changes'")
	} else {
		git("checkout", baseBranch)
		git("branch", "-D", wipBranch)
		sayInfo("someone else already ended your mob session")
	}
}

func status() {
	if isMobbing() {
		sayInfo("mobbing in progress")

		output := silentgit("--no-pager", "log", baseBranch+".."+wipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
		say(output)
	} else {
		sayInfo("you aren't mobbing right now")
	}

	if !hasSay() {
		sayNote("text-to-speech disabled because '" + voiceCommand + "' not found")
	}
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	isMobbing := len(strings.TrimSpace(output)) == 0
	return isMobbing
}

func isMobbing() bool {
	output := silentgit("branch")
	return strings.Contains(output, "* "+wipBranch)
}

func hasMobbingBranch() bool {
	output := silentgit("branch")
	return strings.Contains(output, "  "+wipBranch) || strings.Contains(output, "* "+wipBranch)
}

func hasMobbingBranchOrigin() bool {
	output := silentgit("branch", "--remotes")
	return strings.Contains(output, "  "+remoteName+"/"+wipBranch)
}

func getGitUserName() string {
	return strings.TrimSpace(silentgit("config", "--get", "user.name"))
}

func showNext() {
	if debug {
		say("determining next person based on previous changes")
	}
	changes := strings.TrimSpace(silentgit("--no-pager", "log", baseBranch+".."+wipBranch, "--pretty=format:%an", "--abbrev-commit"))
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
	say("\tmob [-s][-stay] [n]ext \t# hand over to next typist")
	say("\tmob [d]one \t# finish mob session")
	say("\tmob [r]eset \t# resets any unfinished mob session")
	say("\tmob status \t# show status of mob session")
	say("\tmob --help \t# prints this help")
	say("\tmob --version \t# prints the version")
	say("")
	say("examples")
	say("\t mob start 10 \t# start session with 10 min.")
	say("\t mob next \t# after 10 minutes work ...")
	say("\t mob done \t# After the work is done.")

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
	command := exec.Command("which", voiceCommand)
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

func getCommand(args []string) string {
	if len(args) < 1 {
		return ""
	}
	return args[0]
}

func getParameters(args []string) []string {
	if len(args) == 0 {
		return args
	}
	return args[1:]
}
