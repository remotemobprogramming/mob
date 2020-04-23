package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const versionNumber = "0.0.8"

var wipBranch = "mob-session"               // override with MOB_WIP_BRANCH environment variable
var baseBranch = "master"                   // override with MOB_BASE_BRANCH environment variable
var remoteName = "origin"                   // override with MOB_REMOTE_NAME environment variable
var wipCommitMessage = "mob next [ci-skip]" // override with MOB_WIP_COMMIT_MESSAGE environment variable
var mobNextStay = false                     // override with MOB_NEXT_STAY environment variable
var voiceCommand = "say"                    // override with MOB_VOICE_COMMAND environment variable
var debug = false                           // override with MOB_DEBUG environment variable

func config() {
	say("baseBranch" + "=" + baseBranch)
	say("wipBranch" + "=" + wipBranch)
	say("remoteName" + "=" + remoteName)
	say("wipCommitMessage" + "=" + wipCommitMessage)
	say("mobNextStay" + "=" + strconv.FormatBool(mobNextStay))
	say("voiceCommand" + "=" + voiceCommand)
	say("debug" + "=" + strconv.FormatBool(debug))
}

func parseEnvironmentVariables() {
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
}

func parseFlagsForCommandNext(args []string) []string {
	if arrayContains(args, "-s") || arrayContains(args, "--stay") {
		sayInfo("overriding MOB_NEXT_STAY=true because of parameter")
		mobNextStay = true
	}

	return arrayRemove(arrayRemove(args, "-s"), "--stay")
}

func parseDebugFlag(args []string) []string {
	if arrayContains(args, "--debug") {
		sayInfo("overriding MOB_DEBUG=true because of parameter")
		debug = true
	}

	return arrayRemove(args, "--debug")
}

func arrayContains(items []string, item string) bool {
	for _, n := range items {
		if item == n {
			return true
		}
	}
	return false
}

func arrayRemove(items []string, item string) []string {
	newitems := []string{}

	for _, i := range items {
		if i != item {
			newitems = append(newitems, i)
		}
	}

	return newitems
}

func main() {
	parseEnvironmentVariables()
	args := parseDebugFlag(parseFlagsForCommandNext(os.Args[1:]))
	command := getCommand(args)
	parameter := getParameters(args)
	if debug {
		sayDebug("Args '" + strings.Join(args, " ") + "'")
		sayDebug("command '" + command + "'")
		sayDebug("parameter '" + strings.Join(parameter, " ") + "'")
	}

	if command == "s" || command == "start" {
		start(parameter)
		status()
	} else if command == "n" || command == "next" {
		next()
	} else if command == "d" || command == "done" {
		done()
	} else if command == "reset" {
		reset()
	} else if command == "config" {
		config()
	} else if command == "t" || command == "timer" {
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		}
	} else if command == "share" {
		startZoomScreenshare()
	} else if command == "help" || command == "--help" || command == "-h" {
		help()
	} else if command == "version" || command == "--version" || command == "-v" {
		version()
	} else {
		status()
	}
}

func startTimer(timerInMinutes string) {
	if debug {
		sayDebug("Starting timer for " + timerInMinutes + " minutes")
	}
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)

	commandString, ouput, err := runCommand("sh", "-c", "( sleep "+timerInSeconds+" && ("+voiceCommand+" \"mob next\" || (/usr/bin/osascript -e 'display notification \"mob next\"' || /usr/bin/notify-send \"mob next\"))  & )")
	if err != nil {
		sayError("timer couldn't be started... (timer only works on OSX)")
		sayError(commandString)
		sayError(ouput)
		sayError(err.Error())
	} else {
		timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
		sayOkay(timerInMinutes + " minutes timer started (finishes at approx. " + timeOfTimeout + ")")
	}
}

func reset() {
	git("fetch", "--prune")
	git("checkout", baseBranch)
	if hasMobProgrammingBranch() {
		git("branch", "-D", wipBranch)
	}
	if hasMobProgrammingBranchOrigin() {
		git("push", remoteName, "--delete", wipBranch)
	}
}

func start(parameter []string) {
	if !isNothingToCommit() {
		sayNote("cannot start; uncommitted changes present")
		say(silentgit("diff", "--stat"))
		os.Exit(1)
	}

	git("fetch", "--prune")
	git("pull", "--ff-only")

	if hasMobProgrammingBranch() && hasMobProgrammingBranchOrigin() {
		sayInfo("rejoining mob session")
		if !isMobProgramming() {
			git("branch", "-D", wipBranch)
			git("checkout", wipBranch)
			git("branch", "--set-upstream-to="+remoteName+"/"+wipBranch, wipBranch)
		}
	} else if !hasMobProgrammingBranch() && !hasMobProgrammingBranchOrigin() {
		sayInfo("create " + wipBranch + " from " + baseBranch)
		git("checkout", baseBranch)
		git("merge", remoteName+"/"+baseBranch, "--ff-only")
		git("branch", wipBranch)
		git("checkout", wipBranch)
		git("push", "--set-upstream", remoteName, wipBranch)
	} else if !hasMobProgrammingBranch() && hasMobProgrammingBranchOrigin() {
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
	commandStr := ""
	if runtime.GOOS == "linux" {
		commandStr = "(xdotool windowactivate $(xdotool search --name --onlyvisible 'zoom meeting') && xdotool keydown Alt s)"
	} else {
		commandStr = "(osascript -e 'tell application \"System Events\" to keystroke \"S\" using {shift down, command down}')"
	}

	commandString, output, err := runCommand("sh", "-c", commandStr)
	if err != nil {
		sayError("screenshare couldn't be started... (screenshare only works on OSX or Linux with xdotool installed)")
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
	} else {
		if runtime.GOOS == "linux" {
			sayOkay("Sharing screen with zoom (requires the global shortcut ALT+S)")
		} else {
			sayOkay("Sharing screen with zoom (requires the global shortcut SHIFT+COMMAND+S)")
		}
	}
}

func next() {
	if !isMobProgramming() {
		sayError("you aren't mob programming")
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
	if !isMobProgramming() {
		sayError("you aren't mob programming")
		return
	}

	git("fetch", "--prune")

	if hasMobProgrammingBranchOrigin() {
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
	if isMobProgramming() {
		sayInfo("mob programming in progress")

		say(silentgit("--no-pager", "log", baseBranch+".."+wipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit"))
	} else {
		sayInfo("you aren't mob programming right now")
	}

	if !hasVoiceCommand() {
		sayNote("text-to-speech disabled because '" + voiceCommand + "' not found")
	}
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	return len(strings.TrimSpace(output)) == 0
}

func isMobProgramming() bool {
	return gitCurrentBranch() == wipBranch
}

func hasMobProgrammingBranch() bool {
	branches := gitBranches()
	return strings.Contains(branches, "  "+wipBranch) || strings.Contains(branches, "* "+wipBranch)
}

func gitBranches() string {
	return silentgit("branch")
}

func hasMobProgrammingBranchOrigin() bool {
	return strings.Contains(gitRemoteBranches(), "  "+remoteName+"/"+wipBranch)
}

func gitRemoteBranches() string {
	return silentgit("branch", "--remotes")
}

func gitCurrentBranch() string {
	return strings.TrimSpace(silentgit("branch", "--show-current"))
}

func gitUserName() string {
	return strings.TrimSpace(silentgit("config", "--get", "user.name"))
}

func showNext() {
	if debug {
		sayDebug("determining next person based on previous changes")
	}
	changes := strings.TrimSpace(silentgit("--no-pager", "log", baseBranch+".."+wipBranch, "--pretty=format:%an", "--abbrev-commit"))
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	if debug {
		sayDebug("there have been " + strconv.Itoa(numberOfLines) + " changes")
	}
	gitUserName := gitUserName()
	if debug {
		sayDebug("current git user.name is '" + gitUserName + "'")
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
	say("\tmob start [<minutes> [share]]\t# start mob programming as typist")
	say("\tmob next [-s|--stay] \t# hand over to next typist")
	say("\tmob done \t# finish mob session")
	say("\tmob reset \t# resets any unfinished mob session")
	say("\tmob status \t# show status of mob session")
	say("\tmob share \t# start screenshare with zoom")
	say("\tmob timer <minutes>\t# start timer for <minutes>")
	say("\tmob config \t# shows config")
	say("\tmob help \t# prints this help info")
	say("\tmob version \t# prints the version")
	say("")
	say("examples")
	say("\t mob start 10 \t# start 10 min session")
	say("\t mob start 10 share \t# start 10 min session with zoom screenshare")
	say("\t mob next \t# after 10 minutes work ...")
	say("\t mob next --stay\t# after 10 minutes work ...")
	say("\t mob done \t# After the work is done")

}

func version() {
	say("v" + versionNumber)
}

func hasVoiceCommand() bool {
	_, _, err := runCommand("which", voiceCommand)
	return err == nil
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

func runCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	commandString := "[" + strings.Join(command.Args, " ") + "]"
	if debug {
		sayDebug("[" + strings.Join(command.Args, " ") + "]")
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if debug {
		sayDebug(output)
	}
	return commandString, output, err
}

func silentgit(args ...string) string {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
		os.Exit(1)
	}
	return output
}

func git(args ...string) {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
		os.Exit(1)
	} else {
		sayOkay(commandString)
	}
}

var printToConsole = func(message string) {
	fmt.Print(message)
}

func say(s string) {
	printToConsole(strings.TrimRight(s, " \r\n\t\v\f\r") + "\n")
}

func sayError(s string) {
	sayWithPrefix(s, " ERROR ")
}

func sayDebug(s string) {
	sayWithPrefix(s, " DEBUG ")
}

func sayWithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		printToConsole(prefix)
		printToConsole(lines[i])
		printToConsole("\n")
	}
}

func sayOkay(s string) {
	sayWithPrefix(s, " ✓ ")
}

func sayNote(s string) {
	sayWithPrefix(s, " ❗ ")
}

func sayTodo(s string) {
	sayWithPrefix(s, " 👉 ")
}

func sayInfo(s string) {
	sayWithPrefix(s, " > ")
}
