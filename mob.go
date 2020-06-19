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

const (
	versionNumber   = "0.0.21"
	mobStashName    = "mob-stash-name"
	wipBranchPrefix = "mob/"
)

var (
	workingDir    = ""
	configuration Configuration
)

type Configuration struct {
	RemoteName                        string // override with MOB_REMOTE_NAME environment variable
	WipCommitMessage                  string // override with MOB_WIP_COMMIT_MESSAGE environment variable
	VoiceCommand                      string // override with MOB_VOICE_COMMAND environment variable
	MobNextStay                       bool   // override with MOB_NEXT_STAY environment variable
	MobStartIncludeUncommittedChanges bool   // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
	Debug                             bool   // override with MOB_DEBUG environment variable
	WipBranchQualifier                string
	WipBranchQualifierSet             bool
}

func main() {
	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	sayDebug("Args '" + strings.Join(os.Args, " ") + "'")

	command, parameters := parseArgs(os.Args)
	sayDebug("command '" + command + "'")
	sayDebug("parameters '" + strings.Join(parameters, " ") + "'")

	execute(command, parameters)
}

func getDefaultConfiguration() Configuration {
	return Configuration{
		RemoteName:                        "origin",
		WipCommitMessage:                  "mob next [ci-skip]",
		VoiceCommand:                      "say",
		MobNextStay:                       false,
		MobStartIncludeUncommittedChanges: false,
		Debug:                             false,
		WipBranchQualifier:                "",
		WipBranchQualifierSet:             false,
	}
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	removed("MOB_BASE_BRANCH", "Use 'mob start' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use 'mob start --branch <branch>' instead.")

	deprecated("MOB_DEBUG", "Use the parameter --debug instead.")
	deprecated("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setStringFromEnvVariable(&configuration.VoiceCommand, "MOB_VOICE_COMMAND")

	setBoolFromEnvVariable(&configuration.Debug, "MOB_DEBUG")
	setBoolFromEnvVariable(&configuration.MobNextStay, "MOB_NEXT_STAY")
	setBoolFromEnvVariable(&configuration.MobStartIncludeUncommittedChanges, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES")

	return configuration
}

func setStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set && value != "" {
		*s = value
		sayDebug("overriding " + key + " =" + *s)
	}
}

func setBoolFromEnvVariable(s *bool, key string) {
	value, set := os.LookupEnv(key)
	if set && value == "true"{
		*s = true
		sayDebug("overriding " + key + " =" + strconv.FormatBool(*s))
	}
}

func removed(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say("'mob' no longer supports the configuration option '" + key + "'")
		say(message)
	}
}

func deprecated(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say("'mob' will stop supporting the configuration option '" + key + "' sometime in the future")
		say(message)
	}
}

func config() {
	say("MOB_REMOTE_NAME" + "=" + configuration.RemoteName)
	say("MOB_WIP_COMMIT_MESSAGE" + "=" + configuration.WipCommitMessage)
	say("MOB_VOICE_COMMAND" + "=" + configuration.VoiceCommand)
	say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(configuration.MobNextStay))
	say("MOB_START_INCLUDE_UNCOMMITTED_CHANGES" + "=" + strconv.FormatBool(configuration.MobStartIncludeUncommittedChanges))
	say("MOB_DEBUG" + "=" + strconv.FormatBool(configuration.Debug))
}

func parseArgs(args []string) (command string, parameters []string) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-uncommitted-changes":
			configuration.MobStartIncludeUncommittedChanges = true
		case "--debug":
			configuration.Debug = true
		case "--stay", "-s":
			configuration.MobNextStay = true
		case "--branch", "-b":
			if i+1 != len(args) {
				configuration.WipBranchQualifier = args[i+1]
				configuration.WipBranchQualifierSet = true
			}
			i++
		default:
			if i == 1 {
				command = arg
			} else {
				parameters = append(parameters, arg)
			}
		}
	}

	return
}

func execute(command string, parameter []string) {

	switch command {
	case "s", "start":
		start()
		if !isMobProgramming() {
			return
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		}

		status()
	case "n", "next":
		next()
	case "d", "done":
		done()
	case "reset":
		reset()
	case "config":
		config()
	case "status":
		status()
	case "t", "timer":
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		} else {
			help()
		}
	case "moo":
		moo()
	case "version", "--version", "-v":
		version()
	case "help", "--help", "-h":
		help()
	default:
		help()
	}
}

func determineBranches(branch string, branchQualifier string) (baseBranch string, wipBranch string) {
	suffix := ""

	if branchQualifier != "" {
		suffix = "/" + branchQualifier
	}

	preparedBranch := strings.ReplaceAll(branch, wipBranchPrefix, "")
	index := strings.LastIndex(preparedBranch, "/")
	if index != -1 {
		suffix = preparedBranch[index:]
	}

	if branch == "mob-session" || branch == "master" {
		baseBranch = "master"
		if branchQualifier != "" {
			wipBranch = wipBranchPrefix + baseBranch + suffix
		} else {
			wipBranch = "mob-session"
		}
	} else {
		baseBranch = strings.ReplaceAll(strings.ReplaceAll(branch, wipBranchPrefix, ""), suffix, "")
		wipBranch = wipBranchPrefix + baseBranch + suffix
	}

	sayDebug("on branch " + branch + " => BASE " + baseBranch + " WIP " + wipBranch)
	return
}

func startTimer(timerInMinutes string) {
	sayDebug("Starting timer for " + timerInMinutes + " minutes")
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timerInSeconds := strconv.Itoa(timeoutInSeconds)
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")

	voiceMessage := "mob next"
	textMessage := "mob next"

	var commandString string
	var err error
	sayDebug("Operating System " + runtime.GOOS)
	switch runtime.GOOS {
	case "windows":
		commandString, err = startCommand("powershell", "-command", "start-process powershell -NoNewWindow -ArgumentList '-command \"sleep "+timerInSeconds+"; (New-Object -ComObject SAPI.SPVoice).Speak(\\\""+voiceMessage+"\\\")\"'")
	case "darwin":
		commandString, err = startCommand("sh", "-c", "( sleep "+timerInSeconds+" && "+configuration.VoiceCommand+" \""+voiceMessage+"\" && /usr/bin/osascript -e 'display notification \""+textMessage+"\"')  &")
	case "linux":
		commandString, err = startCommand("sh", "-c", "( sleep "+timerInSeconds+" && "+configuration.VoiceCommand+" \""+voiceMessage+"\" && /usr/bin/notify-send \""+textMessage+"\")  &")
	default:
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

func moo() {
	voiceMessage := "moo"

	var commandString string
	var err error

	switch runtime.GOOS {
	case "windows":
		commandString, err = startCommand("powershell", "-command", "start-process powershell -NoNewWindow -ArgumentList '-command \"(New-Object -ComObject SAPI.SPVoice).Speak(\\\""+voiceMessage+"\\\")\"'")
	case "darwin":
		commandString, err = startCommand("sh", "-c", "( "+configuration.VoiceCommand+" \""+voiceMessage+"\")  &")
	case "linux":
		commandString, err = startCommand("sh", "-c", "( "+configuration.VoiceCommand+" \""+voiceMessage+"\")  &")
	default:
		sayError("Cannot run voice command on your system " + runtime.GOOS)
		return
	}

	if err != nil {
		sayError(commandString)
		sayError(err.Error())
	} else {
		sayInfo(voiceMessage)
	}
}

func reset() {
	git("fetch", configuration.RemoteName)

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	git("checkout", currentBaseBranch)
	if hasLocalBranch(currentWipBranch) {
		git("branch", "--delete", "--force", currentWipBranch)
	}
	if hasRemoteBranch(currentWipBranch) {
		git("push", "--no-verify", configuration.RemoteName, "--delete", currentWipBranch)
	}
	sayInfo("Branches " + currentWipBranch + " and " + configuration.RemoteName + "/" + currentWipBranch + " deleted")
}

func start() {
	stashed := false
	if hasUncommittedChanges() {
		if configuration.MobStartIncludeUncommittedChanges {
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

	git("fetch", configuration.RemoteName, "--prune")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	remoteBranches := gitRemoteBranches()

	if !isMobProgramming() && strings.Contains(remoteBranches, configuration.RemoteName+"/"+wipBranchPrefix+currentBaseBranch+"/") && !configuration.WipBranchQualifierSet {
		sayInfo("qualified mob branches detected")
		sayTodo("fix with 'mob start --branch <branch>' (use \"\" for the default mob branch)")
		return
	}

	if !hasRemoteBranch(currentBaseBranch) {
		sayError("Remote branch " + configuration.RemoteName + "/" + currentBaseBranch + " is missing")
		sayTodo("fix with 'git push " + configuration.RemoteName + " " + currentBaseBranch + " --set-upstream'")
		return
	}

	git("pull", "--ff-only")

	if hasRemoteBranch(currentWipBranch) {
		startJoinMobSession()
	} else {
		startNewMobSession()
	}

	if configuration.MobStartIncludeUncommittedChanges && stashed {
		stashes := silentgit("stash", "list")
		stash := findLatestMobStash(stashes)
		git("stash", "pop", stash)
	}
}

func startJoinMobSession() {
	_, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	sayInfo("joining existing mob session from " + configuration.RemoteName + "/" + currentWipBranch)
	git("checkout", "-B", currentWipBranch, configuration.RemoteName+"/"+currentWipBranch)
	git("branch", "--set-upstream-to="+configuration.RemoteName+"/"+currentWipBranch, currentWipBranch)
}

func startNewMobSession() {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	sayInfo("starting new mob session from " + configuration.RemoteName + "/" + currentBaseBranch)
	git("checkout", "-B", currentWipBranch, configuration.RemoteName+"/"+currentBaseBranch)
	git("push", "--no-verify", "--set-upstream", configuration.RemoteName, currentWipBranch)
}

func getUntrackedFiles() string {
	return silentgit("ls-files", "--others", "--exclude-standard")
}

func getUnstagedChanges() string {
	return silentgit("diff", "--stat")
}

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

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	if isNothingToCommit() {
		sayInfo("nothing was done, so nothing to commit")
	} else {
		git("add", "--all")
		git("commit", "--message", "\""+configuration.WipCommitMessage+"\"", "--no-verify")
		changes := getChangesOfLastCommit()
		git("push", "--no-verify", configuration.RemoteName, currentWipBranch)
		say(changes)
	}
	showNext()

	if !configuration.MobNextStay {
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

	git("fetch", configuration.RemoteName, "--prune")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	if hasRemoteBranch(currentWipBranch) {
		if !isNothingToCommit() {
			git("add", "--all")
			git("commit", "--message", "\""+configuration.WipCommitMessage+"\"", "--no-verify")
		}
		git("push", "--no-verify", configuration.RemoteName, currentWipBranch)

		git("checkout", currentBaseBranch)
		git("merge", configuration.RemoteName+"/"+currentBaseBranch, "--ff-only")
		mergeFailed := gitignorefailure("merge", "--squash", "--ff", currentWipBranch)
		if mergeFailed != nil {
			return
		}

		git("branch", "-D", currentWipBranch)
		git("push", "--no-verify", configuration.RemoteName, "--delete", currentWipBranch)

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

		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

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
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, configuration.WipBranchQualifier)
	sayDebug("current branch " + currentBranch + " and currentWipBranch " + currentWipBranch)
	return currentWipBranch == currentBranch
}

func hasLocalBranch(branch string) bool {
	branches := gitBranches()
	return strings.Contains(branches, "  "+branch) || strings.Contains(branches, "* "+branch)
}

func hasRemoteBranch(branch string) bool {
	return strings.Contains(gitRemoteBranches(), "  "+configuration.RemoteName+"/"+branch)
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
	sayDebug("determining next person based on previous changes")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier)

	changes := strings.TrimSpace(silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%an", "--abbrev-commit"))
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	sayDebug("there have been " + strconv.Itoa(numberOfLines) + " changes")
	gitUserName := gitUserName()
	sayDebug("current git user.name is '" + gitUserName + "'")
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
	say("mob start [<minutes>] [--include-uncommitted-changes] [--branch|-b <branch>]\t# start mob session in wip branch")
	say("mob next [-s|--stay] \t\t# handover to next person and switch back to base branch")
	say("mob done \t\t\t# finish mob session by squashing all changes in wip branch to index in base branch")
	say("mob reset [--branch|-b <branch>]# removes local and remote wip branch")
	say("mob status \t\t\t# show status")
	say("mob timer <minutes>\t\t# start a <minutes> timer")
	say("mob config \t\t\t# print configuration")
	say("mob moo \t\t\t# moo!")
	say("mob version \t\t\t# print version number")
	say("mob help \t\t\t# print usage")
	say("")
	say("Add --debug to any option to enable verbose logging")
	say("")
	say("EXAMPLES")
	say("mob start 10 \t\t\t# start 10 min session in wip branch 'mob-session'")
	say("mob start --branch green \t# start session in wip branch 'mob/<base-branch>/green'")
	say("mob next --stay\t\t\t# handover code and stay on wip branch")
	say("mob done \t\t\t# get changes back to base branch")
	say("mob moo \t\t\t# be amazed")
}

func version() {
	say("v" + versionNumber)
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
	sayDebug(commandString)
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	sayDebug(output)
	return commandString, output, err
}

func startCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	sayDebug(commandString)
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
	if configuration.Debug {
		sayWithPrefix(s, " DEBUG ")
	}
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
