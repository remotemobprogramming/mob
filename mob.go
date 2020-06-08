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

const versionNumber = "0.0.19-dev"

var wipBranch string                       // override with MOB_WIP_BRANCH environment variable
var baseBranch string                      // override with MOB_BASE_BRANCH environment variable
var remoteName string                      // override with MOB_REMOTE_NAME environment variable
var wipCommitMessage string                // override with MOB_WIP_COMMIT_MESSAGE environment variable
var voiceCommand string                    // override with MOB_VOICE_COMMAND environment variable
var mobNextStay bool                       // override with MOB_NEXT_STAY environment variable
var mobStartIncludeUncommittedChanges bool // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
var debug bool                             // override with MOB_DEBUG environment variable

func setDefaults() {
	wipBranch = "mob-session"
	baseBranch = "master"
	remoteName = "origin"
	voiceCommand = "say"
	wipCommitMessage = "mob next [ci-skip]"
	mobNextStay = false
	mobStartIncludeUncommittedChanges = false
	debug = false
}

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
	userMobDebug, userMobDebugSet := os.LookupEnv("MOB_DEBUG")
	if userMobDebugSet && userMobDebug == "true" {
		debug = true
		say("overriding MOB_DEBUG=" + strconv.FormatBool(debug))
	}
	userMobNextStay, userMobNextStaySet := os.LookupEnv("MOB_NEXT_STAY")
	if userMobNextStaySet && userMobNextStay == "true" {
		mobNextStay = true
		say("overriding MOB_NEXT_STAY=" + strconv.FormatBool(mobNextStay))
	}

	key := "MOB_START_INCLUDE_UNCOMMITTED_CHANGES"
	userMobStartIncludeUncommittedChanges, userMobStartIncludeUncommittedChangesSet := os.LookupEnv(key)
	if userMobStartIncludeUncommittedChangesSet && userMobStartIncludeUncommittedChanges == "true" {
		mobStartIncludeUncommittedChanges = true
		say("overriding " + key + "=" + strconv.FormatBool(mobStartIncludeUncommittedChanges))
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
	setDefaults()
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
		} else {
			help()
		}
	} else if command == "help" || command == "--help" || command == "-h" {
		help()
	} else if command == "version" || command == "--version" || command == "-v" {
		version()
	} else {
		help()
	}
}

func startTimer(timerInMinutes string) {
	if debug {
		sayDebug("Starting timer for " + timerInMinutes + " minutes")
	}
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")

	voiceMessage := "mob next"
	textMessage := "mob next"

	var commandString string
	var err error
	if debug {
		sayDebug("Operating System " + runtime.GOOS)
	}
	if runtime.GOOS == "windows" {
		commandString, err = startCommand("powershell", "-command", "start-process powershell -NoNewWindow -ArgumentList '-command \"sleep "+timerInSeconds+"; (New-Object -ComObject SAPI.SPVoice).Speak(\\\""+voiceMessage+"\\\")\"'")
	} else if runtime.GOOS == "darwin" {
		commandString, err = startCommand("sh", "-c", "( sleep "+timerInSeconds+" && "+voiceCommand+" \""+voiceMessage+"\" && /usr/bin/osascript -e 'display notification \""+textMessage+"\"')  &")
	} else if runtime.GOOS == "linux" {
		commandString, err = startCommand("sh", "-c", "( sleep "+timerInSeconds+" && "+voiceCommand+" \""+voiceMessage+"\" && /usr/bin/notify-send \""+textMessage+"\")  &")
	} else {
		sayError("Cannot start timer at " + runtime.GOOS)
		return
	}

	if err != nil {
		sayError("timer couldn't be started... (timer only works on OSX)")
		sayError(commandString)
		sayError(err.Error())
	} else {
		sayInfo(timerInMinutes + " minutes timer started (finishes at approx. " + timeOfTimeout + ")")
	}
}

func reset() {
	git("fetch", remoteName)
	git("checkout", baseBranch)
	if hasMobProgrammingBranch() {
		git("branch", "--delete", "--force", wipBranch)
	}
	if hasMobProgrammingBranchOrigin() {
		git("push", "--no-verify", remoteName, "--delete", wipBranch)
	}
}

func start() {
	stashed := false
	if hasUncommittedChanges() {
		if mobStartIncludeUncommittedChanges {
			git("stash", "push", "--include-untracked", "--message", mobStashName)
			stashed = true
		} else {
			sayInfo("cannot start; clean working tree required")
			unstagedChanges := getUnstagedChanges()
			untrackedFiles := getUntrackedFiles()
			hasUnstagedChanges := len(unstagedChanges) > 0
			hasUntrackedFiles := len(untrackedFiles) > 0
			if hasUnstagedChanges {
				sayInfo("unstaged changes present:")
				sayInfo(unstagedChanges)
			}
			if hasUntrackedFiles {
				sayInfo("untracked files present:")
				sayInfo(untrackedFiles)
			}
			sayEmptyLine()
			sayTodo("fix with 'mob start --include-uncommitted-changes'")
			return
		}
	}

	git("fetch", remoteName, "--prune")
	git("pull", "--ff-only")

	_, currentWipBranch := determineCurrentBranches()

	if hasMobProgrammingBranchOrigin2(currentWipBranch) {
		startJoinMobSession()
	} else {
		startNewMobSession()
	}

	if mobStartIncludeUncommittedChanges && stashed {
		stashes := silentgit("stash", "list")
		stash := findLatestMobStash(stashes)
		git("stash", "pop", stash)
	}
}

func startJoinMobSession() {
	_, currentWipBranch := determineCurrentBranches()

	sayInfo("joining existing mob session from " + remoteName + "/" + currentWipBranch)
	git("checkout", "-B", currentWipBranch, remoteName+"/"+currentWipBranch)
	git("branch", "--set-upstream-to="+remoteName+"/"+currentWipBranch, currentWipBranch)
}

func startNewMobSession() {
	currentBaseBranch, currentWipBranch := determineCurrentBranches()

	sayInfo("starting new mob session from " + remoteName + "/" + currentBaseBranch)
	git("checkout", "-B", currentWipBranch, remoteName+"/"+currentBaseBranch)
	git("push", "--no-verify", "--set-upstream", remoteName, currentWipBranch)
}

func determineCurrentBranches() (string, string) {
	currentBranch := gitCurrentBranch()
	var currentBaseBranch string
	var currentWipBranch string

	if currentBranch == "mob-session" {
		currentBaseBranch = "master"
	} else if strings.HasPrefix(currentBranch, "mob-session") {
		currentBaseBranch = strings.ReplaceAll(currentBranch, "mob-session-", "")
	} else {
		currentBaseBranch = currentBranch
	}

	if currentBaseBranch == "master" {
		currentWipBranch = "mob-session"
	} else {
		currentWipBranch = "mob-session-" + currentBaseBranch
	}

	sayInfo("on branch " + currentBranch + " => BASE " + currentBaseBranch + " WIP " + currentWipBranch)
	return currentBaseBranch, currentWipBranch
}

func getUntrackedFiles() string {
	return silentgit("ls-files", "--others", "--exclude-standard")
}

func getUnstagedChanges() string {
	return silentgit("diff", "--stat")
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

func next() {
	if !isMobProgramming() {
		sayError("you aren't mob programming")
		sayEmptyLine()
		sayTodo("use 'mob start' to start mob programming")
		return
	}

	currentBaseBranch, currentWipBranch := determineCurrentBranches()

	if isNothingToCommit() {
		sayInfo("nothing was done, so nothing to commit")
	} else {
		git("add", "--all")
		git("commit", "--message", "\""+wipCommitMessage+"\"", "--no-verify")
		changes := getChangesOfLastCommit()
		git("push", "--no-verify", remoteName, currentWipBranch)
		say(changes)
	}
	showNext()

	if !mobNextStay {
		git("checkout", currentBaseBranch)
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
		sayEmptyLine()
		sayTodo("use 'mob start' to start mob programming")
		return
	}

	git("fetch", remoteName, "--prune")

	currentBaseBranch, currentWipBranch := determineCurrentBranches()

	if hasMobProgrammingBranchOrigin2(currentWipBranch) {
		if !isNothingToCommit() {
			git("add", "--all")
			git("commit", "--message", "\""+wipCommitMessage+"\"", "--no-verify")
		}
		git("push", "--no-verify", remoteName, currentWipBranch)

		git("checkout", currentBaseBranch)
		git("merge", remoteName+"/"+currentBaseBranch, "--ff-only")
		git("merge", "--squash", "--ff", currentWipBranch)

		git("branch", "-D", currentWipBranch)
		git("push", "--no-verify", remoteName, "--delete", currentWipBranch)
		say(getCachedChanges())
		sayTodo("git commit -m 'describe the changes'")
	} else {
		git("checkout", currentBaseBranch)
		git("branch", "-D", currentWipBranch)
		sayInfo("someone else already ended your mob session")
	}
}

func status() {
	if isMobProgramming() {
		sayInfo("you are mob programming")

		currentBaseBranch, currentWipBranch := determineCurrentBranches()

		say(silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit"))
	} else {
		sayInfo("you aren't mob programming")
		sayEmptyLine()
		sayTodo("use 'mob start' to start mob programming")
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
	return strings.HasPrefix(gitCurrentBranch(), "mob-session")
}

func hasMobProgrammingBranch() bool {
	branches := gitBranches()
	return strings.Contains(branches, "  "+wipBranch) || strings.Contains(branches, "* "+wipBranch)
}

func hasMobProgrammingBranchOrigin() bool {
	return strings.Contains(gitRemoteBranches(), "  "+remoteName+"/"+wipBranch)
}

func hasMobProgrammingBranchOrigin2(currentWipBranch string) bool {
	return strings.Contains(gitRemoteBranches(), "  "+remoteName+"/"+currentWipBranch)
}

func gitBranches() string {
	return silentgit("branch")
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

	currentBaseBranch, currentWipBranch := determineCurrentBranches()

	changes := strings.TrimSpace(silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%an", "--abbrev-commit"))
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
	say("mob start [<minutes>] [--include-uncommitted-changes]\t# start mob session")
	say("mob next [-s|--stay] \t# handover to next person")
	say("mob done \t\t# finish mob session")
	say("mob reset \t\t# reset any unfinished mob session (local & remote)")
	say("mob status \t\t# show status of mob session")
	say("mob timer <minutes>\t# start a <minutes> timer")
	say("mob config \t\t# print configuration")
	say("mob help \t\t# print usage")
	say("mob version \t\t# print version number")
	say("")
	say("EXAMPLES")
	say("mob start 10 \t\t# start 10 min session")
	say("mob next --stay\t\t# handover code and stay on mob session branch")
	say("mob done \t\t# get changes back to master branch")
}

func version() {
	say("v" + versionNumber)
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

func gitignorefailure(args ...string) error {
	commandString, output, err := runCommand("git", args...)

	sayOkay(commandString)
	if err != nil {
		sayError(output)
		sayError(err.Error())
	}
	return err
}
func runCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	if debug {
		sayDebug(commandString)
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
		sayDebug(commandString)
	}
	err := command.Start()
	return commandString, err
}

var exit = func(code int) {
	os.Exit(code)
}

func sayError(s string) {
	sayWithPrefix(s, " ERROR ")
}

func sayDebug(s string) {
	sayWithPrefix(s, " DEBUG ")
}

func sayOkay(s string) {
	sayWithPrefix(s, "   ")
}

func sayTodo(s string) {
	sayWithPrefix(s, " 👉 ")
}

func sayInfo(s string) {
	sayWithPrefix(s, " > ")
}

func sayWithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		printToConsole(prefix)
		printToConsole(lines[i])
		printToConsole("\n")
	}
}

func say(s string) {
	if len(s) == 0 {
		return
	}
	printToConsole(strings.TrimRight(s, " \r\n\t\v\f\r") + "\n")
}

func sayEmptyLine() {
	printToConsole("\n")
}

var printToConsole = func(message string) {
	fmt.Print(message)
}
