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
	versionNumber   = "1.0.0-dev"
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
	NotifyCommand                     string // override with MOB_NOTIFY_COMMAND environment variable
	MobNextStay                       bool   // override with MOB_NEXT_STAY environment variable
	MobNextStaySet                    bool   // override with MOB_NEXT_STAY environment variable
	MobStartIncludeUncommittedChanges bool   // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
	Debug                             bool   // override with MOB_DEBUG environment variable
	WipBranchQualifier                string // override with MOB_WIP_BRANCH_QUALIFIER environment variable
	WipBranchQualifierSet             bool
	WipBranchQualifierSeparator       string // override with MOB_WIP_BRANCH_QUALIFIER_SEPARATOR environment variable
}

func main() {
	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	debugInfo("Args '" + strings.Join(os.Args, " ") + "'")

	command, parameters := parseArgs(os.Args)
	debugInfo("command '" + command + "'")
	debugInfo("parameters '" + strings.Join(parameters, " ") + "'")
	debugInfo("version " + versionNumber)
	debugInfo("workingDir " + workingDir)

	execute(command, parameters)
}

func getDefaultConfiguration() Configuration {
	voiceCommand := ""
	notifyCommand := ""
	switch runtime.GOOS {
	case "darwin":
		voiceCommand = "say"
		notifyCommand = "/usr/bin/osascript -e 'display notification \"%s\"'"
	case "linux":
		voiceCommand = "say"
		notifyCommand = "notify-send \"%s\""
	case "windows":
		voiceCommand = "(New-Object -ComObject SAPI.SPVoice).Speak(\\\"%s\\\")\""
	}
	return Configuration{
		RemoteName:                        "origin",
		WipCommitMessage:                  "mob next [ci-skip]",
		VoiceCommand:                      voiceCommand,
		NotifyCommand:                     notifyCommand,
		MobNextStay:                       false,
		MobNextStaySet:                    false,
		MobStartIncludeUncommittedChanges: false,
		Debug:                             false,
		WipBranchQualifier:                "",
		WipBranchQualifierSet:             false,
		WipBranchQualifierSeparator:       "/",
	}
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	removed("MOB_BASE_BRANCH", "Use 'mob start' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use 'mob start --branch <branch>' instead.")

	deprecated("MOB_DEBUG", "Use the parameter --debug instead.")
	deprecated("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setOptionalStringFromEnvVariable(&configuration.VoiceCommand, "MOB_VOICE_COMMAND")
	setOptionalStringFromEnvVariable(&configuration.NotifyCommand, "MOB_NOTIFY_COMMAND")
	setStringFromEnvVariable(&configuration.WipBranchQualifierSeparator, "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR")

	setStringFromEnvVariable(&configuration.WipBranchQualifier, "MOB_WIP_BRANCH_QUALIFIER")
	if configuration.WipBranchQualifier != "" {
		configuration.WipBranchQualifierSet = true
	}

	setBoolFromEnvVariable(&configuration.Debug, "MOB_DEBUG")
	setBoolFromEnvVariableSet(&configuration.MobNextStay, &configuration.MobNextStaySet, "MOB_NEXT_STAY")

	setBoolFromEnvVariable(&configuration.MobStartIncludeUncommittedChanges, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES")

	return configuration
}

func setStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set && value != "" {
		*s = value
		debugInfo("overriding " + key + " =" + *s)
	}
}

func setOptionalStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set {
		*s = value
		debugInfo("overriding " + key + " =" + *s)
	}
}

func setBoolFromEnvVariable(s *bool, key string) {
	value, set := os.LookupEnv(key)
	if set && value == "true" {
		*s = true
		debugInfo("overriding " + key + " =" + strconv.FormatBool(*s))
	}
}

func setBoolFromEnvVariableSet(s *bool, changed *bool, key string) {
	value, set := os.LookupEnv(key)
	if set && value == "true" {
		*s = true
		*changed = true
		debugInfo("overriding " + key + " =" + strconv.FormatBool(*s))
	} else if set && value == "false" {
		*s = false
		*changed = true
		debugInfo("overriding " + key + " =" + strconv.FormatBool(*s))
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
	say("MOB_NOTIFY_COMMAND" + "=" + configuration.NotifyCommand)
	say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(configuration.MobNextStay))
	say("MOB_START_INCLUDE_UNCOMMITTED_CHANGES" + "=" + strconv.FormatBool(configuration.MobStartIncludeUncommittedChanges))
	say("MOB_DEBUG" + "=" + strconv.FormatBool(configuration.Debug))
	say("MOB_WIP_BRANCH_QUALIFIER" + "=" + configuration.WipBranchQualifier)
	say("MOB_WIP_BRANCH_QUALIFIER_SEPARATOR" + "=" + configuration.WipBranchQualifierSeparator)
}

func parseArgs(args []string) (command string, parameters []string) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-uncommitted-changes", "-i":
			configuration.MobStartIncludeUncommittedChanges = true
		case "--debug":
			configuration.Debug = true
		case "--stay", "-s":
			configuration.MobNextStay = true
			configuration.MobNextStaySet = true
		case "--return-to-base-branch", "-r":
			configuration.MobNextStay = false
			configuration.MobNextStaySet = true
		case "--branch", "-b":
			if i+1 != len(args) {
				configuration.WipBranchQualifier = args[i+1]
				configuration.WipBranchQualifierSet = true
			}
			i++
		case "--message", "-m":
			if i+1 != len(args) {
				configuration.WipCommitMessage = args[i+1]
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

func determineBranches(branch string, branchQualifier string, branches string) (baseBranch string, wipBranch string) {
	localBranches := strings.Split(branches, "\n")

	suffix := ""

	if branchQualifier != "" {
		suffix = configuration.WipBranchQualifierSeparator + branchQualifier
	}

	preparedBranch := strings.ReplaceAll(branch, wipBranchPrefix, "")

	branchExists := false
	for i := 0; i < len(localBranches); i++ {
		if localBranches[i] == preparedBranch {
			branchExists = true
		}
	}

	index := strings.LastIndex(preparedBranch, configuration.WipBranchQualifierSeparator)
	if index != -1 && !branchExists {
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

	debugInfo("on branch " + branch + " => BASE " + baseBranch + " WIP " + wipBranch + " with branches " + strings.Join(localBranches, ","))
	return
}

func getSleepCommand(timeoutInSeconds int) string {
	return fmt.Sprintf("sleep %d", timeoutInSeconds)
}

func injectCommandWithMessage(command string, message string) string {
	placeHolders := strings.Count(command, "%s")
	if placeHolders > 1 {
		sayError(fmt.Sprintf("Too many placeholders (%d) in format command string: %s", placeHolders, command))
		exit(1)
	}
	if placeHolders == 0 {
		return fmt.Sprintf("%s %s", command, message)
	}
	return fmt.Sprintf(command, message)
}

func getVoiceCommand(message string) string {
	if len(configuration.VoiceCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(configuration.VoiceCommand, message)
}

func getNotifyCommand(message string) string {
	if len(configuration.NotifyCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(configuration.NotifyCommand, message)
}

func executeCommandsInBackgroundProcess(commands ...string) (err error) {
	cmds := make([]string, 0)
	for _, c := range commands {
		if len(c) > 0 {
			cmds = append(cmds, c)
		}
	}
	debugInfo(fmt.Sprintf("Operating System %s", runtime.GOOS))
	switch runtime.GOOS {
	case "windows":
		_, err = startCommand("powershell", "-command", fmt.Sprintf("start-process powershell -NoNewWindow -ArgumentList '-command \"%s\"'", strings.Join(cmds, ";")))
	case "darwin", "linux":
		_, err = startCommand("sh", "-c", fmt.Sprintf("(%s) &", strings.Join(cmds, ";")))
	default:
		sayError(fmt.Sprintf("Cannot execute background commands on your os: %s", runtime.GOOS))
	}
	return err
}

func startTimer(timerInMinutes string) {
	debugInfo(fmt.Sprintf("Starting timer for %s minutes", timerInMinutes))
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")

	err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob next"), getNotifyCommand("mob next"))

	if err != nil {
		sayError(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
		sayError(err.Error())
	} else {
		sayInfo(fmt.Sprintf("%s minutes timer started (finishes at approx. %s)", timerInMinutes, timeOfTimeout))
	}
}

func moo() {
	voiceMessage := "moo"
	err := executeCommandsInBackgroundProcess(getVoiceCommand(voiceMessage))

	if err != nil {
		sayError(fmt.Sprintf("can't run voice command on your system (%s)", runtime.GOOS))
		sayError(err.Error())
	} else {
		sayInfo(voiceMessage)
	}
}

func reset() {
	git("fetch", configuration.RemoteName)

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

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

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

	hasWipBranchesWithQualifier := hasQualifiedBranches(currentBaseBranch, gitRemoteBranches())

	if !isMobProgramming() && hasWipBranchesWithQualifier && !configuration.WipBranchQualifierSet {
		sayInfo("qualified mob branches detected")
		sayTodo("fix with 'mob start --branch <branch>' (use \"\" for the default mob branch)")
		return
	}

	if !hasRemoteBranch(currentBaseBranch) {
		sayError("Remote branch " + configuration.RemoteName + "/" + currentBaseBranch + " is missing")
		sayTodo("fix with 'git push " + configuration.RemoteName + " " + currentBaseBranch + " --set-upstream'")
		return
	}

	if !isMobProgramming() {
		git("merge", "FETCH_HEAD", "--ff-only")
	}

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

func hasQualifiedBranches(currentBaseBranch string, remoteBranches string) bool {
	debugInfo("check on current base branch " + currentBaseBranch + " with remote branches " + strings.Join(strings.Split(remoteBranches, "\n"), ","))
	hasWipBranchesWithQualifier := strings.Contains(remoteBranches, configuration.RemoteName+"/"+wipBranchPrefix+currentBaseBranch+configuration.WipBranchQualifierSeparator)
	return hasWipBranchesWithQualifier
}

func startJoinMobSession() {
	_, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

	sayInfo("joining existing mob session from " + configuration.RemoteName + "/" + currentWipBranch)
	git("checkout", "-B", currentWipBranch, configuration.RemoteName+"/"+currentWipBranch)
	git("branch", "--set-upstream-to="+configuration.RemoteName+"/"+currentWipBranch, currentWipBranch)
}

func startNewMobSession() {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

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

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

	if isNothingToCommit() {
		if hasLocalCommits(currentWipBranch) {
			git("push", "--no-verify", configuration.RemoteName, currentWipBranch)
		} else {
			sayInfo("nothing was done, so nothing to commit")
		}
	} else {
		git("add", "--all")
		git("commit", "--message", configuration.WipCommitMessage, "--no-verify")
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

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

	if hasRemoteBranch(currentWipBranch) {
		if !isNothingToCommit() {
			git("add", "--all")
			git("commit", "--message", configuration.WipCommitMessage, "--no-verify")
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

		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

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

func hasLocalCommits(branch string) bool {
	local := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/heads/"+branch)
	remote := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/remotes/"+configuration.RemoteName+"/"+branch)
	return strings.TrimSpace(local) != strings.TrimSpace(remote)
}

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
}

func isMobProgramming() bool {
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, configuration.WipBranchQualifier, gitBranches())
	debugInfo("current branch " + currentBranch + " and currentWipBranch " + currentWipBranch)
	return currentWipBranch == currentBranch
}

func hasLocalBranch(localBranch string) bool {
	localBranchesRaw := gitBranches()
	debugInfo("Local Branches: " + localBranchesRaw)
	debugInfo("Local Branch: " + localBranch)

	localBranches := strings.Split(localBranchesRaw, "\n")
	for i := 0; i < len(localBranches); i++ {
		if localBranches[i] == localBranch {
			return true
		}
	}

	return false
}

func hasRemoteBranch(branch string) bool {
	remoteBranchesRaw := gitRemoteBranches()
	remoteBranch := configuration.RemoteName + "/" + branch
	debugInfo("Remote Branches: " + remoteBranchesRaw)
	debugInfo("Remote Branch: " + remoteBranch)

	remoteBranches := strings.Split(remoteBranchesRaw, "\n")
	for i := 0; i < len(remoteBranches); i++ {
		if remoteBranches[i] == remoteBranch {
			return true
		}
	}

	return false
}

func gitBranches() string {
	return strings.TrimSpace(silentgit("branch", "--format=%(refname:short)"))
}

func gitRemoteBranches() string {
	return strings.TrimSpace(silentgit("branch", "--remotes", "--format=%(refname:short)"))
}

func gitCurrentBranch() string {
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return strings.TrimSpace(silentgit("rev-parse", "--abbrev-ref", "HEAD"))
}

func gitUserName() string {
	return strings.TrimSpace(silentgit("config", "--get", "user.name"))
}

func showNext() {
	debugInfo("determining next person based on previous changes")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), configuration.WipBranchQualifier, gitBranches())

	changes := strings.TrimSpace(silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%an", "--abbrev-commit"))
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	debugInfo("there have been " + strconv.Itoa(numberOfLines) + " changes")
	gitUserName := gitUserName()
	debugInfo("current git user.name is '" + gitUserName + "'")
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
	say("mob start [<minutes>] [--include-uncommitted-changes|-i] [--branch|-b <branch>]\t# start mob session in wip branch")
	say("mob next [--stay|-s] [--return-to-base-branch|-r] [--message|-m <commit-message>]\t\t# handover to next person and switch back to base branch")
	say("mob done \t\t\t# finish mob session by squashing all changes in wip branch to index in base branch")
	say("mob reset [--branch|-b <branch>]# removes local and remote wip branch")
	say("mob status \t\t\t# show status")
	say("mob timer <minutes>\t\t# start a <minutes> timer")
	say("mob config \t\t\t# print configuration")
	say("mob moo \t\t\t# moo!")
	say("mob version \t\t\t# print version number")
	say("mob help \t\t\t# print usage")
	sayEmptyLine()
	say("Add --debug to any option to enable verbose logging")
	sayEmptyLine()
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
	debugInfo("Running command " + commandString)
	outputBinary, err := command.CombinedOutput()
	output := string(outputBinary)
	debugInfo(output)
	return commandString, output, err
}

func startCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	debugInfo("Starting command " + commandString)
	err := command.Start()
	return commandString, err
}

var exit = func(code int) {
	os.Exit(code)
}

func sayError(s string) {
	sayWithPrefix(s, " ERROR ")
}

func debugInfo(s string) {
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
