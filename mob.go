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
	versionNumber   = "1.3.0"
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
	RequireCommitMessage              bool   // override with MOB_REQUIRE_COMMIT_MESSAGE environment variable
	VoiceCommand                      string // override with MOB_VOICE_COMMAND environment variable
	NotifyCommand                     string // override with MOB_NOTIFY_COMMAND environment variable
	MobNextStay                       bool   // override with MOB_NEXT_STAY environment variable
	MobNextStaySet                    bool   // override with MOB_NEXT_STAY environment variable
	MobStartIncludeUncommittedChanges bool   // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
	Debug                             bool   // override with --debug parameter
	WipBranchQualifier                string // override with MOB_WIP_BRANCH_QUALIFIER environment variable
	WipBranchQualifierSet             bool   // used to enforce a start on the default wip branch with `mob start --branch ""` when other open wip branches had been detected
	WipBranchQualifierSeparator       string // override with MOB_WIP_BRANCH_QUALIFIER_SEPARATOR environment variable
	MobDoneSquash                     bool   // override with MOB_DONE_SQUASH environment variable
	MobTimer                          string // override with MOB_TIMER environment variable
}

func (c Configuration) wipBranchQualifierSuffix() string {
	return c.WipBranchQualifierSeparator + c.WipBranchQualifier
}

func (c Configuration) customWipBranchQualifierConfigured() bool {
	return c.WipBranchQualifier != ""
}

func (c Configuration) hasCustomCommitMessage() bool {
	return getDefaultConfiguration().WipCommitMessage != c.WipCommitMessage
}

func main() {
	configuration = getDefaultConfiguration()
	configuration = parseDebug(configuration, os.Args)

	configuration = parseEnvironmentVariables(configuration)
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
		voiceCommand = "say \"%s\""
		notifyCommand = "/usr/bin/osascript -e 'display notification \"%s\"'"
	case "linux":
		voiceCommand = "say \"%s\""
		notifyCommand = "notify-send \"%s\""
	case "windows":
		voiceCommand = "(New-Object -ComObject SAPI.SPVoice).Speak(\\\"%s\\\")"
	}
	return Configuration{
		RemoteName:                        "origin",
		WipCommitMessage:                  "mob next [ci-skip] [ci skip] [skip ci]",
		VoiceCommand:                      voiceCommand,
		NotifyCommand:                     notifyCommand,
		MobNextStay:                       true,
		MobNextStaySet:                    false,
		RequireCommitMessage:              false,
		MobStartIncludeUncommittedChanges: false,
		Debug:                             false,
		WipBranchQualifier:                "",
		WipBranchQualifierSet:             false,
		WipBranchQualifierSeparator:       "-",
		MobDoneSquash:                     true,
		MobTimer:                          "",
	}
}

func parseDebug(configuration Configuration, args []string) Configuration {
	// debug needs to be parsed at the beginning to have DEBUG enabled as quickly as possible
	// otherwise, parsing other environment variables or other parameters don't have debug enabled
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			configuration.Debug = true
		}
	}

	return configuration
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	removed("MOB_BASE_BRANCH", "Use 'mob start' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use 'mob start --branch <branch>' instead.")
	deprecated("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setBoolFromEnvVariable(&configuration.RequireCommitMessage, "MOB_REQUIRE_COMMIT_MESSAGE")
	setOptionalStringFromEnvVariable(&configuration.VoiceCommand, "MOB_VOICE_COMMAND")
	setOptionalStringFromEnvVariable(&configuration.NotifyCommand, "MOB_NOTIFY_COMMAND")
	setStringFromEnvVariable(&configuration.WipBranchQualifierSeparator, "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR")

	setStringFromEnvVariable(&configuration.WipBranchQualifier, "MOB_WIP_BRANCH_QUALIFIER")
	if configuration.WipBranchQualifier != "" {
		configuration.WipBranchQualifierSet = true
	}

	setBoolFromEnvVariableSet(&configuration.MobNextStay, &configuration.MobNextStaySet, "MOB_NEXT_STAY")

	setBoolFromEnvVariable(&configuration.MobStartIncludeUncommittedChanges, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES")

	setBoolFromEnvVariable(&configuration.MobDoneSquash, "MOB_DONE_SQUASH")

	setStringFromEnvVariable(&configuration.MobTimer, "MOB_TIMER")

	return configuration
}

func setStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set && value != "" {
		*s = value
		debugInfo("overriding " + key + "=" + *s)
	}
}

func setOptionalStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set {
		*s = value
		debugInfo("overriding " + key + "=" + *s)
	}
}

func setBoolFromEnvVariable(s *bool, key string) {
	value, set := os.LookupEnv(key)
	if !set {
		return
	}
	if value == "" {
		debugInfo("ignoring " + key + "=" + value + " (empty string)")
	}

	if value == "true" {
		*s = true
		debugInfo("overriding " + key + "=" + strconv.FormatBool(*s))
	} else if value == "false" {
		*s = false
		debugInfo("overriding " + key + "=" + strconv.FormatBool(*s))
	} else {
		sayError("ignoring " + key + "=" + value + " (not a boolean)")
	}
}

func setBoolFromEnvVariableSet(s *bool, overridden *bool, key string) {
	value, set := os.LookupEnv(key)

	if !set {
		debugInfo("key " + key + " is not set")
		return
	}

	debugInfo("found " + key + "=" + value)

	if value == "true" {
		*s = true
		*overridden = true
		debugInfo("overriding " + key + " =" + strconv.FormatBool(*s))
	} else if value == "false" {
		*s = false
		*overridden = true
		debugInfo("overriding " + key + " =" + strconv.FormatBool(*s))
	} else {
		sayError("ignoring " + key + " =" + value + " (not a boolean)")
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

func config(c Configuration) {
	say("MOB_REMOTE_NAME" + "=" + c.RemoteName)
	say("MOB_WIP_COMMIT_MESSAGE" + "=" + c.WipCommitMessage)
	say("MOB_REQUIRE_COMMIT_MESSAGE" + "=" + strconv.FormatBool(c.RequireCommitMessage))
	say("MOB_VOICE_COMMAND" + "=" + c.VoiceCommand)
	say("MOB_NOTIFY_COMMAND" + "=" + c.NotifyCommand)
	say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(c.MobNextStay))
	say("MOB_START_INCLUDE_UNCOMMITTED_CHANGES" + "=" + strconv.FormatBool(c.MobStartIncludeUncommittedChanges))
	say("MOB_WIP_BRANCH_QUALIFIER" + "=" + c.WipBranchQualifier)
	say("MOB_WIP_BRANCH_QUALIFIER_SEPARATOR" + "=" + c.WipBranchQualifierSeparator)
	say("MOB_DONE_SQUASH" + "=" + strconv.FormatBool(c.MobDoneSquash))
	say("MOB_TIMER" + "=" + c.MobTimer)
}

func parseArgs(args []string) (command string, parameters []string) {
	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-uncommitted-changes", "-i":
			configuration.MobStartIncludeUncommittedChanges = true
		case "--debug":
			// ignore this, already parsed
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
			i++ // skip consumed parameter
		case "--message", "-m":
			if i+1 != len(args) {
				configuration.WipCommitMessage = args[i+1]
			}
			i++ // skip consumed parameter
		case "--squash":
			configuration.MobDoneSquash = true
		case "--no-squash":
			configuration.MobDoneSquash = false
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
		start(configuration)
		if !isMobProgramming(configuration) {
			return
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		} else if configuration.MobTimer != "" {
			startTimer(configuration.MobTimer)
		}

		status(configuration)
	case "n", "next":
		next(configuration)
	case "d", "done":
		done(configuration)
	case "reset":
		reset(configuration)
	case "config":
		config(configuration)
	case "status":
		status(configuration)
	case "t", "timer":
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer)
		} else if configuration.MobTimer != "" {
			startTimer(configuration.MobTimer)
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

func determineBranches(currentBranch string, localBranches []string, configuration Configuration) (baseBranch string, wipBranch string) {
	if currentBranch == "mob-session" || (currentBranch == "master" && !configuration.customWipBranchQualifierConfigured()) {
		baseBranch = "master"
		wipBranch = "mob-session"
	} else if isWipBranch(currentBranch) {
		baseBranch = removeWipQualifier(removeWipPrefix(currentBranch), localBranches, configuration)
		wipBranch = currentBranch
	} else {
		baseBranch = currentBranch
		wipBranch = addWipQualifier(addWipPrefix(currentBranch), configuration)
	}

	debugInfo("on currentBranch " + currentBranch + " => BASE " + baseBranch + " WIP " + wipBranch + " with allLocalBranches " + strings.Join(localBranches, ","))
	if currentBranch != baseBranch && currentBranch != wipBranch {
		// this is unreachable code, but we keep it as a backup
		panic("assertion failed! neither on base nor on wip branch")
	}
	return
}

func addWipQualifier(branch string, configuration Configuration) string {
	if configuration.customWipBranchQualifierConfigured() {
		return addSuffix(branch, configuration.wipBranchQualifierSuffix())
	}
	return branch
}

func removeWipQualifier(branch string, localBranches []string, configuration Configuration) string {
	for !branchExists(branch, localBranches) && hasWipBranchQualifierSeparator(branch, configuration) {
		var afterRemoval string
		if configuration.WipBranchQualifier == "" { // WipBranchQualifier not configured
			afterRemoval = removeFromSeparator(branch, configuration.WipBranchQualifierSeparator)
		} else { // WipBranchQualifier not configured
			afterRemoval = removeSuffix(branch, configuration.wipBranchQualifierSuffix())
		}

		if branch == afterRemoval { // avoids infinite loop
			break
		}

		branch = afterRemoval
	}
	return branch
}

func removeSuffix(branch string, suffix string) string {
	if strings.HasSuffix(branch, suffix) {
		return branch[:strings.LastIndex(branch, suffix)]
	}
	return branch
}

func removeFromSeparator(branch string, separator string) string {
	return branch[:strings.LastIndex(branch, separator)]
}

func isWipBranch(branch string) bool {
	return strings.Index(branch, wipBranchPrefix) == 0
}

func addWipPrefix(branch string) string {
	return wipBranchPrefix + branch
}

func removeWipPrefix(branch string) string { //TODO improve, add tests
	return branch[len(wipBranchPrefix):]
}

func addSuffix(branch string, suffix string) string {
	return branch + suffix
}

func hasWipBranchQualifierSeparator(branch string, configuration Configuration) bool { //TODO improve (dont use strings.Contains, add tests)
	return strings.Contains(branch, configuration.WipBranchQualifierSeparator)
}

func branchExists(branchInQuestion string, existingBranches []string) bool {
	return stringContains(existingBranches, branchInQuestion)
}

func stringContains(list []string, element string) bool {
	found := false
	for i := 0; i < len(list); i++ {
		if list[i] == element {
			found = true
		}
	}
	return found
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

func reset(configuration Configuration) {
	git("fetch", configuration.RemoteName)

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	git("checkout", currentBaseBranch)
	if hasLocalBranch(currentWipBranch) {
		git("branch", "--delete", "--force", currentWipBranch)
	}
	if hasRemoteBranch(currentWipBranch, configuration) {
		git("push", "--no-verify", configuration.RemoteName, "--delete", currentWipBranch)
	}
	sayInfo("Branches " + currentWipBranch + " and " + configuration.RemoteName + "/" + currentWipBranch + " deleted")
}

func start(configuration Configuration) {
	stashed := false
	if hasUncommittedChanges() {
		if configuration.MobStartIncludeUncommittedChanges {
			git("stash", "push", "--include-untracked", "--message", mobStashName)
			stashed = true
		} else {
			sayInfo("cannot start; clean working tree required")
			sayUnstagedChangesInfo()
			sayUntrackedFilesInfo()
			sayTodo("To start mob programming including uncommitted changes, use", "mob start --include-uncommitted-changes")
			return
		}
	}

	git("fetch", configuration.RemoteName, "--prune")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	hasWipBranchesWithQualifier := hasQualifiedBranches(currentBaseBranch, gitRemoteBranches())

	if !isMobProgramming(configuration) && hasWipBranchesWithQualifier && !configuration.WipBranchQualifierSet {
		sayInfo("qualified mob branches detected")
		sayTodo("To start mob programming, use", "mob start --branch <branch>")
		sayIndented("(use \"\" for the default mob branch)")
		return
	}

	if !hasRemoteBranch(currentBaseBranch, configuration) {
		sayError("Remote branch " + configuration.RemoteName + "/" + currentBaseBranch + " is missing")
		sayTodo("To set the upstream branch, use", "git push "+configuration.RemoteName+" "+currentBaseBranch+" --set-upstream")
		return
	}

	if !isMobProgramming(configuration) {
		git("merge", "FETCH_HEAD", "--ff-only")
	}

	if hasRemoteBranch(currentWipBranch, configuration) {
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

func sayUntrackedFilesInfo() {
	untrackedFiles := getUntrackedFiles()
	hasUntrackedFiles := len(untrackedFiles) > 0
	if hasUntrackedFiles {
		sayInfo("untracked files present:")
		sayInfo(untrackedFiles)
	}
}

func sayUnstagedChangesInfo() {
	unstagedChanges := getUnstagedChanges()
	hasUnstagedChanges := len(unstagedChanges) > 0
	if hasUnstagedChanges {
		sayInfo("unstaged changes present:")
		sayInfo(unstagedChanges)
	}
}

func hasQualifiedBranches(currentBaseBranch string, remoteBranches []string) bool {
	debugInfo("check on current base branch " + currentBaseBranch + " with remote branches " + strings.Join(remoteBranches, ","))
	hasWipBranchesWithQualifier := strings.Contains(strings.Join(remoteBranches, "\n"), configuration.RemoteName+"/"+wipBranchPrefix+currentBaseBranch+configuration.WipBranchQualifierSeparator)
	return hasWipBranchesWithQualifier
}

func startJoinMobSession() {
	_, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	sayInfo("joining existing mob session from " + configuration.RemoteName + "/" + currentWipBranch)
	git("checkout", "-B", currentWipBranch, configuration.RemoteName+"/"+currentWipBranch)
	git("branch", "--set-upstream-to="+configuration.RemoteName+"/"+currentWipBranch, currentWipBranch)
}

func startNewMobSession() {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

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

func next(configuration Configuration) {
	if !isMobProgramming(configuration) {
		sayError("you aren't mob programming")
		sayTodo("to start mob programming, use", "mob start")
		return
	}

	if !configuration.hasCustomCommitMessage() && configuration.RequireCommitMessage && !isNothingToCommit() {
		sayError("commit message required")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if isNothingToCommit() {
		if hasLocalCommits(currentWipBranch, configuration) {
			git("push", "--no-verify", configuration.RemoteName, currentWipBranch)
		} else {
			sayInfo("nothing was done, so nothing to commit")
		}
	} else {
		makeWipCommit()

		changes := getChangesOfLastCommit()
		git("push", "--no-verify", configuration.RemoteName, currentWipBranch)
		say(changes)
	}
	showNext(configuration)

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

func makeWipCommit() {
	commitMsg := configuration.WipCommitMessage
	git("add", "--all")
	git("commit", "--message", commitMsg, "--no-verify")
}

func done(configuration Configuration) {
	if !isMobProgramming(configuration) {
		sayError("you aren't mob programming")
		sayTodo("to start mob programming, use", "mob start")
		return
	}

	git("fetch", configuration.RemoteName, "--prune")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if hasRemoteBranch(currentWipBranch, configuration) {
		if !isNothingToCommit() {
			makeWipCommit()
		}
		git("push", "--no-verify", configuration.RemoteName, currentWipBranch)

		git("checkout", currentBaseBranch)
		git("merge", configuration.RemoteName+"/"+currentBaseBranch, "--ff-only")
		mergeFailed := gitignorefailure("merge", squashOrNoCommit(configuration), "--ff", currentWipBranch)
		if mergeFailed != nil {
			return
		}

		git("branch", "-D", currentWipBranch)
		git("push", "--no-verify", configuration.RemoteName, "--delete", currentWipBranch)

		say(getCachedChanges())
		err := appendCoauthorsToSquashMsg(workingDir)
		if err != nil {
			sayError(err.Error())
		}
		sayTodo("To finish, use", "git commit")
	} else {
		git("checkout", currentBaseBranch)
		git("branch", "-D", currentWipBranch)
		sayInfo("someone else already ended your mob session")
	}
}

func squashOrNoCommit(configuration Configuration) string {
	if configuration.MobDoneSquash {
		return "--squash"
	} else {
		return "--no-commit"
	}
}

func status(configuration Configuration) {
	if isMobProgramming(configuration) {
		sayInfo("you are mob programming")

		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		sayInfo("on wip branch " + currentWipBranch + " (base branch " + currentBaseBranch + ")")

		sayLastCommitsList(currentBaseBranch, currentWipBranch)
	} else {
		sayInfo("you aren't mob programming")
		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		sayInfo("on base branch " + currentBaseBranch + " (wip branch " + currentWipBranch + ")")

		sayTodo("to start mob programming, use", "mob start")
	}
}

func sayLastCommitsList(currentBaseBranch string, currentWipBranch string) {
	log := silentgit("--no-pager", "log", currentBaseBranch+".."+currentWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	lines := strings.Split(strings.TrimSpace(log), "\n")
	if len(lines) > 5 {
		sayInfo("This mob branch contains " + strconv.Itoa(len(lines)) + " commits. The last 5 were:")
		lines = lines[:5]
	}
	output := strings.Join(lines, "\n")
	say(output)
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	return len(strings.TrimSpace(output)) == 0
}

func hasLocalCommits(branch string, configuration Configuration) bool {
	local := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/heads/"+branch)
	remote := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/remotes/"+configuration.RemoteName+"/"+branch)
	return strings.TrimSpace(local) != strings.TrimSpace(remote)
}

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
}

func isMobProgramming(configuration Configuration) bool {
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, gitBranches(), configuration)
	debugInfo("current branch " + currentBranch + " and currentWipBranch " + currentWipBranch)
	return currentWipBranch == currentBranch
}

func hasLocalBranch(localBranch string) bool {
	localBranches := gitBranches()
	debugInfo("Local Branches: " + strings.Join(localBranches, "\n"))
	debugInfo("Local Branch: " + localBranch)

	for i := 0; i < len(localBranches); i++ {
		if localBranches[i] == localBranch {
			return true
		}
	}

	return false
}

func hasRemoteBranch(branch string, configuration Configuration) bool {
	remoteBranches := gitRemoteBranches()
	remoteBranch := configuration.RemoteName + "/" + branch
	debugInfo("Remote Branches: " + strings.Join(remoteBranches, "\n"))
	debugInfo("Remote Branch: " + remoteBranch)

	for i := 0; i < len(remoteBranches); i++ {
		if remoteBranches[i] == remoteBranch {
			return true
		}
	}

	return false
}

func gitBranches() []string {
	return strings.Split(strings.TrimSpace(silentgit("branch", "--format=%(refname:short)")), "\n")
}

func gitRemoteBranches() []string {
	return strings.Split(strings.TrimSpace(silentgit("branch", "--remotes", "--format=%(refname:short)")), "\n")
}

func gitCurrentBranch() string {
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return strings.TrimSpace(silentgit("rev-parse", "--abbrev-ref", "HEAD"))
}

func gitUserName() string {
	return strings.TrimSpace(silentgit("config", "--get", "user.name"))
}

func gitUserEmail() string {
	return strings.TrimSpace(silentgit("config", "--get", "user.email"))
}

func showNext(configuration Configuration) {
	debugInfo("determining next person based on previous changes")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

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
	output := `mob enables a smooth Git handover

Basic Commands:
  start              start mob session from base branch in wip branch
  next               handover changes in wip branch to next person
  done               squashes all changes in wip branch to index in base branch
  reset              removes local and remote wip branch

Basic Commands(Options):
  start [<minutes>]                      Start a <minutes> timer
    [--include-uncommitted-changes|-i]   Move uncommitted changes to wip branch
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>/<branch-postfix>'
  next 
    [--stay|-s]                          Stay on wip branch (default)
    [--return-to-base-branch|-r]         Return to base branch
    [--message|-m <commit-message>]      Override commit message
  done
    [--no-squash]                        Do not squash commits from wip branch
    [--squash]                           Squash commits from wip branch
  reset 
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>/<branch-postfix>'

Timer Commands:
  timer <minutes>    start a <minutes> timer
  start <minutes>    start mob session in wip branch and a timer

Get more information:
  status             show the status of the current mob session
  config             show all configuration options
  version            show the version of mob
  help               show help

Other
  moo                moo!

Add --debug to any option to enable verbose logging
`
	say(output)
}

func version() {
	say("v" + versionNumber)
}

func silentgit(args ...string) string {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayGitError(commandString, output, err)
		exit(1)
	}
	return output
}

func git(args ...string) {
	commandString, output, err := runCommand("git", args...)

	if err != nil {
		sayGitError(commandString, output, err)
		exit(1)
	} else {
		sayIndented(commandString)
	}
}

func sayGitError(commandString string, output string, err error) {
	if !isGit() {
		path, err := os.Getwd()
		if err == nil {
			cwdMsg := fmt.Sprintf("The current working directory, %s, is not a git repository.", path)

			sayWithPrefix("mob expects the current working directory to be a git repository.", "🤦🏿 ")
			sayIndented(cwdMsg)
			say(" ")
		}

	}
	sayError(commandString)
	sayError(output)
	sayError(err.Error())
}

func isGit() bool {
	_, _, err := runCommand("git", "rev-parse")
	return err == nil
}

func gitignorefailure(args ...string) error {
	commandString, output, err := runCommand("git", args...)

	sayIndented(commandString)
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

func sayIndented(s string) {
	sayWithPrefix(s, "   ")
}

func sayTodo(s string, cmd string) {
	sayWithPrefix(s, " 👉 ")
	sayEmptyLine()
	sayIndented(cmd)
	sayEmptyLine()
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
