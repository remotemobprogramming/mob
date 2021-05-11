package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	versionNumber = "1.6.0"
	mobStashName  = "mob-stash-name"
)

var (
	workingDir = ""
	Debug      = false // override with --debug parameter
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
	WipBranchQualifier                string // override with MOB_WIP_BRANCH_QUALIFIER environment variable
	WipBranchQualifierSet             bool   // used to enforce a start on the default wip branch with `mob start --branch ""` when other open wip branches had been detected
	WipBranchQualifierSeparator       string // override with MOB_WIP_BRANCH_QUALIFIER_SEPARATOR environment variable
	MobDoneSquash                     bool   // override with MOB_DONE_SQUASH environment variable
	MobTimer                          string // override with MOB_TIMER environment variable
	WipBranchPrefix                   string // override with MOB_WIP_BRANCH_PREFIX environment variable (experimental)
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

func (c Configuration) isWipCommitMessage(line string) bool {
	return line == c.WipCommitMessage
}

func main() {
	parseDebug(os.Args)

	configuration := getDefaultConfiguration()
	configuration = parseEnvironmentVariables(configuration)
	debugInfo("Args '" + strings.Join(os.Args, " ") + "'")

	command, parameters, configuration := parseArgs(os.Args, configuration)
	debugInfo("command '" + command + "'")
	debugInfo("parameters '" + strings.Join(parameters, " ") + "'")
	debugInfo("version " + versionNumber)
	debugInfo("workingDir " + workingDir)

	execute(command, parameters, configuration)
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
		WipBranchQualifier:                "",
		WipBranchQualifierSet:             false,
		WipBranchQualifierSeparator:       "-",
		MobDoneSquash:                     true,
		MobTimer:                          "",
		WipBranchPrefix:                   "mob/",
	}
}

func parseDebug(args []string) {
	// debug needs to be parsed at the beginning to have DEBUG enabled as quickly as possible
	// otherwise, parsing other environment variables or other parameters don't have debug enabled
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			Debug = true
		}
	}
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	removed("MOB_BASE_BRANCH", "Use 'mob start' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use 'mob start --branch <branch>' instead.")
	deprecated("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")
	experimental("MOB_WIP_BRANCH_PREFIX")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setBoolFromEnvVariable(&configuration.RequireCommitMessage, "MOB_REQUIRE_COMMIT_MESSAGE")
	setOptionalStringFromEnvVariable(&configuration.VoiceCommand, "MOB_VOICE_COMMAND")
	setOptionalStringFromEnvVariable(&configuration.NotifyCommand, "MOB_NOTIFY_COMMAND")
	setStringFromEnvVariable(&configuration.WipBranchQualifierSeparator, "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR")

	setStringFromEnvVariable(&configuration.WipBranchQualifier, "MOB_WIP_BRANCH_QUALIFIER")
	if configuration.customWipBranchQualifierConfigured() {
		configuration.WipBranchQualifierSet = true
	}
	setStringFromEnvVariable(&configuration.WipBranchPrefix, "MOB_WIP_BRANCH_PREFIX")

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

func experimental(key string) {
	if _, set := os.LookupEnv(key); set {
		say("configuration option '" + key + "' is experimental. 'mob' might stop supporting this option sometime in the future")
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

func parseArgs(args []string, configuration Configuration) (command string, parameters []string, newConfiguration Configuration) {
	newConfiguration = configuration

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-uncommitted-changes", "-i":
			newConfiguration.MobStartIncludeUncommittedChanges = true
		case "--debug":
			// ignore this, already parsed
		case "--stay", "-s":
			newConfiguration.MobNextStay = true
			newConfiguration.MobNextStaySet = true
		case "--return-to-base-branch", "-r":
			newConfiguration.MobNextStay = false
			newConfiguration.MobNextStaySet = true
		case "--branch", "-b":
			if i+1 != len(args) {
				newConfiguration.WipBranchQualifier = args[i+1]
				newConfiguration.WipBranchQualifierSet = true
			}
			i++ // skip consumed parameter
		case "--message", "-m":
			if i+1 != len(args) {
				newConfiguration.WipCommitMessage = args[i+1]
			}
			i++ // skip consumed parameter
		case "--squash":
			newConfiguration.MobDoneSquash = true
		case "--no-squash":
			newConfiguration.MobDoneSquash = false
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

func execute(command string, parameter []string, configuration Configuration) {

	switch command {
	case "s", "start":
		err := start(configuration)
		if !isMobProgramming(configuration) || err != nil {
			return
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer, configuration)
		} else if configuration.MobTimer != "" {
			startTimer(configuration.MobTimer, configuration)
		}
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
			startTimer(timer, configuration)
		} else if configuration.MobTimer != "" {
			startTimer(configuration.MobTimer, configuration)
		} else {
			help()
		}
	case "moo":
		moo(configuration)
	case "sw", "squash-wip":
		if len(parameter) > 1 && parameter[0] == "--git-editor" {
			squashWipGitEditor(parameter[1], configuration)
		} else if len(parameter) > 1 && parameter[0] == "--git-sequence-editor" {
			squashWipGitSequenceEditor(parameter[1], configuration)
		} else {
			squashWip(configuration)
		}
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
	} else if configuration.isWipBranch(currentBranch) {
		baseBranch = removeWipQualifier(configuration.removeWipPrefix(currentBranch), localBranches, configuration)
		wipBranch = currentBranch
	} else {
		baseBranch = currentBranch
		wipBranch = addWipQualifier(configuration.addWipPrefix(currentBranch), configuration)
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
		if !configuration.customWipBranchQualifierConfigured() { // WipBranchQualifier not configured
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

func (c Configuration) isWipBranch(branch string) bool {
	return strings.Index(branch, c.WipBranchPrefix) == 0
}

func (c Configuration) addWipPrefix(branch string) string {
	return c.WipBranchPrefix + branch
}

func (c Configuration) removeWipPrefix(branch string) string { //TODO improve, add tests
	return branch[len(c.WipBranchPrefix):]
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

func getVoiceCommand(message string, voiceCommand string) string {
	if len(voiceCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(voiceCommand, message)
}

func getNotifyCommand(message string, notifyCommand string) string {
	if len(notifyCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(notifyCommand, message)
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

func startTimer(timerInMinutes string, configuration Configuration) {
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	if timeoutInMinutes < 0 {
		timeoutInMinutes = 0
	}
	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	debugInfo(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob next", configuration.VoiceCommand), getNotifyCommand("mob next", configuration.NotifyCommand))

	if err != nil {
		sayError(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
		sayError(err.Error())
	} else {
		sayInfo(fmt.Sprintf("%d minutes timer started (finishes at approx. %s)", timeoutInMinutes, timeOfTimeout))
	}
}

func moo(configuration Configuration) {
	voiceMessage := "moo"
	err := executeCommandsInBackgroundProcess(getVoiceCommand(voiceMessage, configuration.VoiceCommand))

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

func start(configuration Configuration) error {
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
			return errors.New("cannot start; clean working tree required")
		}
	}

	git("fetch", configuration.RemoteName, "--prune")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	hasWipBranchesWithQualifier := hasQualifiedBranches(currentBaseBranch, gitRemoteBranches(), configuration)

	if !isMobProgramming(configuration) && hasWipBranchesWithQualifier && !configuration.WipBranchQualifierSet {
		sayInfo("qualified mob branches detected")
		sayTodo("To start mob programming, use", "mob start --branch <branch>")
		sayIndented("(use \"\" for the default mob branch)")
		return errors.New("qualified mob branches detected")
	}

	if !hasRemoteBranch(currentBaseBranch, configuration) {
		sayError("Remote branch " + configuration.RemoteName + "/" + currentBaseBranch + " is missing")
		sayTodo("To set the upstream branch, use", "git push "+configuration.RemoteName+" "+currentBaseBranch+" --set-upstream")
		return errors.New("Remote branch is missing")
	}

	if hasUnpushedCommits(currentBaseBranch, configuration) {
		sayError("cannot start; unpushed changes on base branch must be pushed upstream")
		return errors.New("cannot start; unpushed changes on base branch must be pushed upstream")
	}

	if !isMobProgramming(configuration) {
		git("merge", "FETCH_HEAD", "--ff-only")
	}

	if hasRemoteBranch(currentWipBranch, configuration) {
		startJoinMobSession(configuration)
	} else {
		startNewMobSession(configuration)
	}

	if configuration.MobStartIncludeUncommittedChanges && stashed {
		stashes := silentgit("stash", "list")
		stash := findLatestMobStash(stashes)
		git("stash", "pop", stash)
	}

	sayInfo("on wip branch " + currentWipBranch + " (base branch " + currentBaseBranch + ")")
	sayLastCommitsList(currentBaseBranch, currentWipBranch)

	return nil // no error
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

func hasQualifiedBranches(currentBaseBranch string, remoteBranches []string, configuration Configuration) bool {
	debugInfo("check on current base branch " + currentBaseBranch + " with remote branches " + strings.Join(remoteBranches, ","))
	remoteBranchWithQualifier := configuration.RemoteName + "/" + configuration.WipBranchPrefix + currentBaseBranch + configuration.WipBranchQualifierSeparator
	hasWipBranchesWithQualifier := strings.Contains(strings.Join(remoteBranches, "\n"), remoteBranchWithQualifier)
	return hasWipBranchesWithQualifier
}

func startJoinMobSession(configuration Configuration) {
	_, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	sayInfo("joining existing mob session from " + configuration.RemoteName + "/" + currentWipBranch)
	git("checkout", "-B", currentWipBranch, configuration.RemoteName+"/"+currentWipBranch)
	git("branch", "--set-upstream-to="+configuration.RemoteName+"/"+currentWipBranch, currentWipBranch)
}

func startNewMobSession(configuration Configuration) {
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
		makeWipCommit(configuration)

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
	return silentgit("diff", "HEAD^1", "--stat")
}

func getCachedChanges() string {
	return silentgit("diff", "--cached", "--stat")
}

func makeWipCommit(configuration Configuration) {
	git("add", "--all")
	git("commit", "--message", configuration.WipCommitMessage, "--no-verify")
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
			makeWipCommit(configuration)
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
		err := appendCoauthorsToSquashMsg(gitDir())
		if err != nil {
			sayError(err.Error())
		}
		if configuration.MobDoneSquash {
			sayTodo("To finish, use", "git commit")
		}

	} else {
		git("checkout", currentBaseBranch)
		git("branch", "-D", currentWipBranch)
		sayInfo("someone else already ended your mob session")
	}
}

func gitDir() string {
	return silentgit("rev-parse", "--absolute-git-dir")
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
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	lines := strings.Split(log, "\n")
	if len(lines) > 5 {
		sayInfo("This mob branch contains " + strconv.Itoa(len(lines)) + " commits. The last 5 were:")
		lines = lines[:5]
	}
	ReverseSlice(lines)
	output := strings.Join(lines, "\n")
	say(output)
}

func ReverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func isNothingToCommit() bool {
	output := silentgit("status", "--short")
	return len(output) == 0
}

func hasLocalCommits(branch string, configuration Configuration) bool {
	local := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/heads/"+branch)
	remote := silentgit("for-each-ref", "--format=%(objectname)",
		"refs/remotes/"+configuration.RemoteName+"/"+branch)
	return local != remote
}

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
}

func hasUnpushedCommits(branch string, configuration Configuration) bool {
	countOutput := silentgit(
		"rev-list", "--count", "--left-only",
		"refs/heads/"+branch+"..."+"refs/remotes/"+configuration.RemoteName+"/"+branch,
	)
	unpushedCount, err := strconv.Atoi(countOutput)
	if err != nil {
		panic(err)
	}
	unpushedCommits := unpushedCount != 0
	if unpushedCommits {
		sayInfo(fmt.Sprintf("there are %d unpushed commits on local base branch <%s>", unpushedCount, branch))
	}
	return unpushedCommits
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
	return strings.Split(silentgit("branch", "--format=%(refname:short)"), "\n")
}

func gitRemoteBranches() []string {
	return strings.Split(silentgit("branch", "--remotes", "--format=%(refname:short)"), "\n")
}

func gitCurrentBranch() string {
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return silentgit("rev-parse", "--abbrev-ref", "HEAD")
}

func gitUserName() string {
	return silentgit("config", "--get", "user.name")
}

func gitUserEmail() string {
	return silentgit("config", "--get", "user.email")
}

func showNext(configuration Configuration) {
	debugInfo("determining next person based on previous changes")

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch

	changes := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%an", "--abbrev-commit")
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

Experimental Commands:
  squash-wip                             Combines wip commits in wip branch with subsequent manual commits to leave only manual commits.
                                         ! Works only if all wip commits have the same wip commit message !
    [--git-editor]                       Not intended for manual use. Used as a non-interactive editor (GIT_EDITOR) for git.
    [--git-sequence-editor]              Not intended for manual use. Used as a non-interactive sequence editor (GIT_SEQUENCE_EDITOR) for git.

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
	return strings.TrimSpace(output)
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
		sayError("mob expects the current working directory to be a git repository.")
	} else {
		sayError(commandString)
		sayError(output)
		sayError(err.Error())
	}
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

func sayError(text string) {
	sayWithPrefix(text, " ERROR ")
}

func debugInfo(text string) {
	if Debug {
		sayWithPrefix(text, " DEBUG ")
	}
}

func sayIndented(text string) {
	sayWithPrefix(text, "   ")
}

func sayTodo(text string, command string) {
	sayWithPrefix(text, " 👉 ")
	sayEmptyLine()
	sayIndented(command)
	sayEmptyLine()
}

func sayInfo(text string) {
	sayWithPrefix(text, " > ")
}

func sayWithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		printToConsole(prefix + lines[i] + "\n")
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
