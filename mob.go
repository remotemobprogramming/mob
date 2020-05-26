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

const versionNumber = "0.0.16-dev"

var wipBranch = "mob-session"                 // override with MOB_WIP_BRANCH environment variable
var baseBranch = "master"                     // override with MOB_BASE_BRANCH environment variable
var remoteName = "origin"                     // override with MOB_REMOTE_NAME environment variable
var wipCommitMessage = "mob next [ci-skip]"   // override with MOB_WIP_COMMIT_MESSAGE environment variable
var voiceCommand = "say"                      // override with MOB_VOICE_COMMAND environment variable
var mobNextStay = false                       // override with MOB_NEXT_STAY environment variable
var mobStartIncludeUncommittedChanges = false // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
var debug = false                             // override with MOB_DEBUG environment variable

var workingDir = ""

func config() {
	say("MOB_BASE_BRANCH" + "=" + baseBranch)
	say("MOB_WIP_BRANCH" + "=" + wipBranch)
	say("MOB_REMOTE_NAME" + "=" + remoteName)
	say("MOB_WIP_COMMIT_MESSAGE" + "=" + wipCommitMessage)
	say("MOB_VOICE_COMMAND" + "=" + voiceCommand)
	say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(mobNextStay))
	say("MOB_START_INCLUDE_UNCOMMITTED_CHANGES" + "=" + strconv.FormatBool(mobStartIncludeUncommittedChanges))
	say("MOB_DEBUG" + "=" + strconv.FormatBool(debug))
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
	_, userMobStartIncludeUncommittedChangesSet := os.LookupEnv("MOB_START_INCLUDE_UNCOMMITTED_CHANGES")
	if userMobStartIncludeUncommittedChangesSet {
		mobStartIncludeUncommittedChanges = true
		say("overriding MOB_START_INCLUDE_UNCOMMITTED_CHANGES=" + strconv.FormatBool(mobStartIncludeUncommittedChanges))
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

func parseIncludeUncommittedChangesFlag(args []string) []string {
	if arrayContains(args, "--include-uncommitted-changes") {
		sayInfo("overriding MOB_START_INCLUDE_UNCOMMITTED_CHANGES=true because of parameter")
		mobStartIncludeUncommittedChanges = true
	}

	return arrayRemove(args, "--include-uncommitted-changes")
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
	args := parseIncludeUncommittedChangesFlag(parseDebugFlag(parseFlagsForCommandNext(os.Args[1:])))
	command := getCommand(args)
	parameter := getParameters(args)
	if debug {
		sayDebug("Args '" + strings.Join(args, " ") + "'")
		sayDebug("command '" + command + "'")
		sayDebug("parameter '" + strings.Join(parameter, " ") + "'")
	}

	if command == "s" || command == "start" {
		start()
		if !isMobProgramming() {
			return
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		}

		if len(parameter) > 1 && parameter[1] == "share" {
			startZoomScreenshare()
		}
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

	commandString, err := startCommand("sh", "-c", "( sleep "+timerInSeconds+" && "+voiceCommand+" \"mob next\" && (/usr/bin/notify-send \"mob next\" || /usr/bin/osascript -e 'display notification \"mob next\"')  & )")
	if err != nil {
		sayError("timer couldn't be started... (timer only works on OSX)")
		sayError(commandString)
		sayError(err.Error())
	} else {
		timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
		sayOkay(timerInMinutes + " minutes timer started (finishes at approx. " + timeOfTimeout + ")")
	}
}

func reset() {
	git("fetch")
	git("checkout", baseBranch)
	if hasMobProgrammingBranch() {
		git("branch", "-D", wipBranch)
	}
	if hasMobProgrammingBranchOrigin() {
		git("push", "--no-verify", remoteName, "--delete", wipBranch)
	}
}

func start() {
	stashed := false
	if hasUncommittedChanges() {
		if mobStartIncludeUncommittedChanges {
			git("stash", "push", "--message", mobStashName)
			stashed = true
		} else {
			sayNote("cannot start; clean working tree required")
			sayInfo(silentgit("diff", "--stat"))
			sayTodo("use 'mob start --include-uncommitted-changes' to pull those changes via 'git stash'")
			return
		}
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
		git("push", "--no-verify", "--set-upstream", remoteName, wipBranch)
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
		git("push", "--no-verify", "--set-upstream", remoteName, wipBranch)
	}

	if mobStartIncludeUncommittedChanges && stashed {
		stashes := silentgit("stash", "list")
		stash := findLatestMobStash(stashes)
		git("stash", "pop", stash)
	}
}

var mobStashName = "mob-stash-name"

func findLatestMobStash(stashes string) string {
	lines := strings.Split(stashes, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, mobStashName) {
			return line[:strings.Index(line, ":")]
		}
	}
	return "unknown"
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
		git("push", "--no-verify", remoteName, wipBranch)
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
		git("push", "--no-verify", remoteName, wipBranch)

		git("checkout", baseBranch)
		git("merge", remoteName+"/"+baseBranch, "--ff-only")
		git("merge", "--squash", "--ff", wipBranch)

		git("branch", "-D", wipBranch)
		git("push", "--no-verify", remoteName, "--delete", wipBranch)
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

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
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
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return strings.TrimSpace(silentgit("rev-parse", "--abbrev-ref", "HEAD"))
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
	say("USAGE")
	say("mob start [<minutes> [share]] [--include-uncommitted-changes]\t# start mob session")
	say("mob next [-s|--stay] \t# handover to next person")
	say("mob done \t\t# finish mob session")
	say("mob reset \t\t# reset any unfinished mob session (local & remote)")
	say("mob status \t\t# show status of mob session")
	say("mob share \t\t# start screen sharing in Zoom (requires Zoom configuration)")
	say("mob timer <minutes>\t# start a <minutes> timer")
	say("mob config \t\t# print configuration")
	say("mob help \t\t# print usage")
	say("mob version \t\t# print version number")
	say("")
	say("EXAMPLES")
	say("mob start 10 \t\t# start 10 min session")
	say("mob start 10 share \t# start 10 min session with zoom screenshare")
	say("mob next --stay\t\t# handover code and stay on mob session branch")
	say("mob done \t\t# get changes back to master branch")

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
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	if debug {
		sayDebug(command.String())
	}
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	if debug {
		sayDebug(output)
	}
	return commandString, output, err
}

func startCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	if debug {
		sayDebug(command.String())
	}
	err := command.Start()
	return commandString, err
}

func silentgit(args ...string) string {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
		exit(1)
	}
	return output
}

func git(args ...string) {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
		exit(1)
	} else {
		sayOkay(commandString)
	}
}

var exit = func(code int) {
	os.Exit(code)
}

var printToConsole = func(message string) {
	fmt.Print(message)
}

func say(s string) {
	if len(s) == 0 {
		return
	}
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
