package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	versionNumber = "2.5.0"
)

var (
	workingDir = ""
	Debug      = false // override with --debug parameter
)

type Configuration struct {
	CliName                        string // override with MOB_CLI_NAME
	RemoteName                     string // override with MOB_REMOTE_NAME
	WipCommitMessage               string // override with MOB_WIP_COMMIT_MESSAGE
	GitHooksEnabled                bool   // override with MOB_GIT_HOOKS_ENABLED
	RequireCommitMessage           bool   // override with MOB_REQUIRE_COMMIT_MESSAGE
	VoiceCommand                   string // override with MOB_VOICE_COMMAND
	VoiceMessage                   string // override with MOB_VOICE_MESSAGE
	NotifyCommand                  string // override with MOB_NOTIFY_COMMAND
	NotifyMessage                  string // override with MOB_NOTIFY_MESSAGE
	NextStay                       bool   // override with MOB_NEXT_STAY
	StartIncludeUncommittedChanges bool   // override with MOB_START_INCLUDE_UNCOMMITTED_CHANGES variable
	StashName                      string // override with MOB_STASH_NAME
	WipBranchQualifier             string // override with MOB_WIP_BRANCH_QUALIFIER
	WipBranchQualifierSeparator    string // override with MOB_WIP_BRANCH_QUALIFIER_SEPARATOR
	WipBranchPrefix                string // override with MOB_WIP_BRANCH_PREFIX
	DoneSquash                     bool   // override with MOB_DONE_SQUASH
	Timer                          string // override with MOB_TIMER
	TimerRoom                      string // override with MOB_TIMER_ROOM
	TimerLocal                     bool   // override with MOB_TIMER_LOCAL
	TimerRoomUseWipBranchQualifier bool   // override with MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER
	TimerUser                      string // override with MOB_TIMER_USER
	TimerUrl                       string // override with MOB_TIMER_URL
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

type Branch struct {
	Name string
}

func newBranch(name string) Branch {
	return Branch{
		Name: strings.TrimSpace(name),
	}
}

func (branch Branch) String() string {
	return branch.Name
}

func (branch Branch) Is(branchName string) bool {
	return branch.Name == branchName
}

func (branch Branch) remote(configuration Configuration) Branch {
	return newBranch(configuration.RemoteName + "/" + branch.Name)
}

func (branch Branch) hasRemoteBranch(configuration Configuration) bool {
	remoteBranches := gitRemoteBranches()
	remoteBranch := branch.remote(configuration).Name
	debugInfo("Remote Branches: " + strings.Join(remoteBranches, "\n"))
	debugInfo("Remote Branch: " + remoteBranch)

	for i := 0; i < len(remoteBranches); i++ {
		if remoteBranches[i] == remoteBranch {
			return true
		}
	}

	return false
}

func (branch Branch) IsWipBranch(configuration Configuration) bool {
	return strings.Index(branch.Name, configuration.WipBranchPrefix) == 0
}

func (branch Branch) addWipPrefix(configuration Configuration) Branch {
	return newBranch(configuration.WipBranchPrefix + branch.Name)
}

func (branch Branch) addWipQualifier(configuration Configuration) Branch {
	if configuration.customWipBranchQualifierConfigured() {
		return newBranch(addSuffix(branch.Name, configuration.wipBranchQualifierSuffix()))
	}
	return branch
}

func addSuffix(branch string, suffix string) string {
	return branch + suffix
}

func (branch Branch) removeWipPrefix(configuration Configuration) Branch {
	return newBranch(removePrefix(branch.Name, configuration.WipBranchPrefix))
}

func removePrefix(branch string, prefix string) string {
	if !strings.HasPrefix(branch, prefix) {
		return branch
	}
	return branch[len(prefix):]
}

func (branch Branch) removeWipQualifier(localBranches []string, configuration Configuration) Branch {
	for !branch.exists(localBranches) && branch.hasWipBranchQualifierSeparator(configuration) {
		afterRemoval := branch.removeWipQualifierSuffixOrSeparator(configuration)

		if branch == afterRemoval { // avoids infinite loop
			break
		}

		branch = afterRemoval
	}
	return branch
}

func (branch Branch) removeWipQualifierSuffixOrSeparator(configuration Configuration) Branch {
	if !configuration.customWipBranchQualifierConfigured() { // WipBranchQualifier not configured
		return branch.removeFromSeparator(configuration.WipBranchQualifierSeparator)
	} else { // WipBranchQualifier not configured
		return branch.removeWipQualifierSuffix(configuration)
	}
}

func (branch Branch) removeFromSeparator(separator string) Branch {
	return newBranch(branch.Name[:strings.LastIndex(branch.Name, separator)])
}

func (branch Branch) removeWipQualifierSuffix(configuration Configuration) Branch {
	if strings.HasSuffix(branch.Name, configuration.wipBranchQualifierSuffix()) {
		return newBranch(branch.Name[:strings.LastIndex(branch.Name, configuration.wipBranchQualifierSuffix())])
	}
	return branch
}

func (branch Branch) exists(existingBranches []string) bool {
	return stringContains(existingBranches, branch.Name)
}

func (branch Branch) hasWipBranchQualifierSeparator(configuration Configuration) bool { //TODO improve (dont use strings.Contains, add tests)
	return strings.Contains(branch.Name, configuration.WipBranchQualifierSeparator)
}

func (branch Branch) hasLocalCommits(configuration Configuration) bool {
	local := silentgit("for-each-ref", "--format=%(objectname)", "refs/heads/"+branch.Name)
	remote := silentgit("for-each-ref", "--format=%(objectname)", "refs/remotes/"+branch.remote(configuration).Name)
	return local != remote
}

func (branch Branch) hasUnpushedCommits(configuration Configuration) bool {
	countOutput := silentgit(
		"rev-list", "--count", "--left-only",
		"refs/heads/"+branch.Name+"..."+"refs/remotes/"+branch.remote(configuration).Name,
	)
	unpushedCount, err := strconv.Atoi(countOutput)
	if err != nil {
		panic(err)
	}
	unpushedCommits := unpushedCount != 0
	if unpushedCommits {
		sayInfo(fmt.Sprintf("there are %d unpushed commits on local base branch <%s>", unpushedCount, branch.Name))
	}
	return unpushedCommits
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

func main() {
	parseDebug(os.Args)

	configuration := getDefaultConfiguration()
	configuration = parseEnvironmentVariables(configuration)

	currentUser, _ := user.Current()
	userConfigurationPath := currentUser.HomeDir + "/.mob"
	configuration = parseUserConfiguration(configuration, userConfigurationPath)
	if isGit() {
		configuration = parseProjectConfiguration(configuration, gitRootDir()+"/.mob")
	}
	debugInfo("Args '" + strings.Join(os.Args, " ") + "'")
	currentCliName := currentCliName(os.Args[0])
	if currentCliName != configuration.CliName {
		debugInfo("Updating cli name to " + currentCliName)
		configuration.CliName = currentCliName
	}

	command, parameters, configuration := parseArgs(os.Args, configuration)
	debugInfo("command '" + command + "'")
	debugInfo("parameters '" + strings.Join(parameters, " ") + "'")
	debugInfo("version " + versionNumber)
	debugInfo("workingDir '" + workingDir + "'")

	execute(command, parameters, configuration)
}

func currentCliName(argZero string) string {
	argZero = strings.TrimSuffix(argZero, ".exe")
	if strings.Contains(argZero, "/") {
		argZero = argZero[strings.LastIndex(argZero, "/")+1:]
	}
	return argZero
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
		CliName:                        "mob",
		RemoteName:                     "origin",
		WipCommitMessage:               "mob next [ci-skip] [ci skip] [skip ci]",
		GitHooksEnabled:                false,
		VoiceCommand:                   voiceCommand,
		VoiceMessage:                   "mob next",
		NotifyCommand:                  notifyCommand,
		NotifyMessage:                  "mob next",
		NextStay:                       true,
		RequireCommitMessage:           false,
		StartIncludeUncommittedChanges: false,
		WipBranchQualifier:             "",
		WipBranchQualifierSeparator:    "-",
		DoneSquash:                     true,
		Timer:                          "",
		TimerLocal:                     true,
		TimerRoom:                      "",
		TimerUser:                      "",
		TimerUrl:                       "https://timer.mob.sh/",
		WipBranchPrefix:                "mob/",
		StashName:                      "mob-stash-name",
	}
}

func parseDebug(args []string) {
	// debug needs to be parsed at the beginning to have DEBUG enabled as quickly as possible
	// otherwise, parsing others or other parameters don't have debug enabled
	for i := 0; i < len(args); i++ {
		if args[i] == "--debug" {
			Debug = true
		}
	}
}

func (c Configuration) mob(command string) string {
	return c.CliName + " " + command
}

func parseUserConfiguration(configuration Configuration, path string) Configuration {
	file, err := os.Open(path)

	if err != nil {
		debugInfo("No user configuration file found. (" + path + ") Error: " + err.Error())
		return configuration
	} else {
		debugInfo("Found user configuration file at " + path)
	}

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		debugInfo(line)
		if !strings.Contains(line, "=") {
			debugInfo("Skip line because line contains no =. Line=" + line)
			continue
		}
		key := line[0:strings.Index(line, "=")]
		value := strings.TrimPrefix(line, key+"=")
		debugInfo("Key is " + key)
		debugInfo("Value is " + value)
		switch key {
		case "MOB_CLI_NAME":
			setUnquotedString(&configuration.CliName, key, value)
		case "MOB_REMOTE_NAME":
			setUnquotedString(&configuration.RemoteName, key, value)
		case "MOB_WIP_COMMIT_MESSAGE":
			setUnquotedString(&configuration.WipCommitMessage, key, value)
		case "MOB_GIT_HOOKS_ENABLED":
			setBoolean(&configuration.GitHooksEnabled, key, value)
		case "MOB_REQUIRE_COMMIT_MESSAGE":
			setBoolean(&configuration.RequireCommitMessage, key, value)
		case "MOB_VOICE_COMMAND":
			setUnquotedString(&configuration.VoiceCommand, key, value)
		case "MOB_VOICE_MESSAGE":
			setUnquotedString(&configuration.VoiceMessage, key, value)
		case "MOB_NOTIFY_COMMAND":
			setUnquotedString(&configuration.NotifyCommand, key, value)
		case "MOB_NOTIFY_MESSAGE":
			setUnquotedString(&configuration.NotifyMessage, key, value)
		case "MOB_NEXT_STAY":
			setBoolean(&configuration.NextStay, key, value)
		case "MOB_START_INCLUDE_UNCOMMITTED_CHANGES":
			setBoolean(&configuration.StartIncludeUncommittedChanges, key, value)
		case "MOB_WIP_BRANCH_QUALIFIER":
			setUnquotedString(&configuration.WipBranchQualifier, key, value)
		case "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR":
			setUnquotedString(&configuration.WipBranchQualifierSeparator, key, value)
		case "MOB_DONE_SQUASH":
			setBoolean(&configuration.DoneSquash, key, value)
		case "MOB_TIMER":
			setUnquotedString(&configuration.Timer, key, value)
		case "MOB_TIMER_ROOM":
			setUnquotedString(&configuration.TimerRoom, key, value)
		case "MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER":
			setBoolean(&configuration.TimerRoomUseWipBranchQualifier, key, value)
		case "MOB_TIMER_LOCAL":
			setBoolean(&configuration.TimerLocal, key, value)
		case "MOB_TIMER_USER":
			setUnquotedString(&configuration.TimerUser, key, value)
		case "MOB_TIMER_URL":
			setUnquotedString(&configuration.TimerUrl, key, value)
		case "MOB_STASH_NAME":
			setUnquotedString(&configuration.StashName, key, value)

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		sayWarning("User configuration file exists, but could not be read. (" + path + ")")
	}

	return configuration
}

func parseProjectConfiguration(configuration Configuration, path string) Configuration {
	file, err := os.Open(path)

	if err != nil {
		debugInfo("No project configuration file found. (" + path + ") Error: " + err.Error())
		return configuration
	} else {
		debugInfo("Found project configuration file at " + path)
	}

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		debugInfo(line)
		if !strings.Contains(line, "=") {
			debugInfo("Skip line because line contains no =. Line=" + line)
			continue
		}
		key := line[0:strings.Index(line, "=")]
		value := strings.TrimPrefix(line, key+"=")
		debugInfo("Key is " + key)
		debugInfo("Value is " + value)
		switch key {
		case "MOB_VOICE_COMMAND", "MOB_VOICE_MESSAGE", "MOB_NOTIFY_COMMAND", "MOB_NOTIFY_MESSAGE":
			sayWarning("Skipped overwriting key " + key + " from project/.mob file out of security reasons!")
		case "MOB_CLI_NAME":
			setUnquotedString(&configuration.CliName, key, value)
		case "MOB_REMOTE_NAME":
			setUnquotedString(&configuration.RemoteName, key, value)
		case "MOB_WIP_COMMIT_MESSAGE":
			setUnquotedString(&configuration.WipCommitMessage, key, value)
		case "MOB_GIT_HOOKS_ENABLED":
			setBoolean(&configuration.GitHooksEnabled, key, value)
		case "MOB_REQUIRE_COMMIT_MESSAGE":
			setBoolean(&configuration.RequireCommitMessage, key, value)
		case "MOB_NEXT_STAY":
			setBoolean(&configuration.NextStay, key, value)
		case "MOB_START_INCLUDE_UNCOMMITTED_CHANGES":
			setBoolean(&configuration.StartIncludeUncommittedChanges, key, value)
		case "MOB_WIP_BRANCH_QUALIFIER":
			setUnquotedString(&configuration.WipBranchQualifier, key, value)
		case "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR":
			setUnquotedString(&configuration.WipBranchQualifierSeparator, key, value)
		case "MOB_DONE_SQUASH":
			setBoolean(&configuration.DoneSquash, key, value)
		case "MOB_TIMER":
			setUnquotedString(&configuration.Timer, key, value)
		case "MOB_TIMER_ROOM":
			setUnquotedString(&configuration.TimerRoom, key, value)
		case "MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER":
			setBoolean(&configuration.TimerRoomUseWipBranchQualifier, key, value)
		case "MOB_TIMER_LOCAL":
			setBoolean(&configuration.TimerLocal, key, value)
		case "MOB_TIMER_USER":
			setUnquotedString(&configuration.TimerUser, key, value)
		case "MOB_TIMER_URL":
			setUnquotedString(&configuration.TimerUrl, key, value)
		case "MOB_STASH_NAME":
			setUnquotedString(&configuration.StashName, key, value)

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		sayWarning("Project configuration file exists, but could not be read. (" + path + ")")
	}

	return configuration
}

func setUnquotedString(s *string, key string, value string) {
	unquotedValue, err := strconv.Unquote(value)
	if err != nil {
		sayWarning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
		return
	}
	*s = unquotedValue
	debugInfo("Overwriting " + key + " =" + unquotedValue)
}

func setBoolean(s *bool, key string, value string) {
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		sayWarning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
		return
	}
	*s = boolValue
	debugInfo("Overwriting " + key + " =" + strconv.FormatBool(boolValue))
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	setStringFromEnvVariable(&configuration.CliName, "MOB_CLI_NAME")
	if configuration.CliName != getDefaultConfiguration().CliName {
		configuration.WipCommitMessage = configuration.CliName + " next [ci-skip] [ci skip] [skip ci]"
		configuration.VoiceMessage = configuration.CliName + " next"
		configuration.NotifyMessage = configuration.CliName + " next"
	}

	removed("MOB_BASE_BRANCH", "Use '"+configuration.mob("start")+"' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use '"+configuration.mob("start --branch <branch>")+"' instead.")
	deprecated("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")
	experimental("MOB_WIP_BRANCH_PREFIX")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setBoolFromEnvVariable(&configuration.GitHooksEnabled, "MOB_GIT_HOOKS_ENABLED")
	setBoolFromEnvVariable(&configuration.RequireCommitMessage, "MOB_REQUIRE_COMMIT_MESSAGE")
	setOptionalStringFromEnvVariable(&configuration.VoiceCommand, "MOB_VOICE_COMMAND")
	setStringFromEnvVariable(&configuration.VoiceMessage, "MOB_VOICE_MESSAGE")
	setOptionalStringFromEnvVariable(&configuration.NotifyCommand, "MOB_NOTIFY_COMMAND")
	setStringFromEnvVariable(&configuration.NotifyMessage, "MOB_NOTIFY_MESSAGE")
	setStringFromEnvVariable(&configuration.WipBranchQualifierSeparator, "MOB_WIP_BRANCH_QUALIFIER_SEPARATOR")

	setStringFromEnvVariable(&configuration.WipBranchQualifier, "MOB_WIP_BRANCH_QUALIFIER")
	setStringFromEnvVariable(&configuration.WipBranchPrefix, "MOB_WIP_BRANCH_PREFIX")

	setBoolFromEnvVariable(&configuration.NextStay, "MOB_NEXT_STAY")

	setBoolFromEnvVariable(&configuration.StartIncludeUncommittedChanges, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES")

	setBoolFromEnvVariable(&configuration.DoneSquash, "MOB_DONE_SQUASH")

	setStringFromEnvVariable(&configuration.Timer, "MOB_TIMER")
	setStringFromEnvVariable(&configuration.TimerRoom, "MOB_TIMER_ROOM")
	setBoolFromEnvVariable(&configuration.TimerRoomUseWipBranchQualifier, "MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER")
	setBoolFromEnvVariable(&configuration.TimerLocal, "MOB_TIMER_LOCAL")
	setStringFromEnvVariable(&configuration.TimerUser, "MOB_TIMER_USER")
	setStringFromEnvVariable(&configuration.TimerUrl, "MOB_TIMER_URL")

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

func removed(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say("Configuration option '" + key + "' is no longer used.")
		say(message)
	}
}

func deprecated(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say("Configuration option '" + key + "' is deprecated.")
		say(message)
	}
}

func experimental(key string) {
	if _, set := os.LookupEnv(key); set {
		say("Configuration option '" + key + "' is experimental. Be prepared that this option will be removed!")
	}
}

func config(c Configuration) {
	say("MOB_CLI_NAME" + "=" + quote(c.CliName))
	say("MOB_REMOTE_NAME" + "=" + quote(c.RemoteName))
	say("MOB_WIP_COMMIT_MESSAGE" + "=" + quote(c.WipCommitMessage))
	say("MOB_GIT_HOOKS_ENABLED" + "=" + strconv.FormatBool(c.GitHooksEnabled))
	say("MOB_REQUIRE_COMMIT_MESSAGE" + "=" + strconv.FormatBool(c.RequireCommitMessage))
	say("MOB_VOICE_COMMAND" + "=" + quote(c.VoiceCommand))
	say("MOB_VOICE_MESSAGE" + "=" + quote(c.VoiceMessage))
	say("MOB_NOTIFY_COMMAND" + "=" + quote(c.NotifyCommand))
	say("MOB_NOTIFY_MESSAGE" + "=" + quote(c.NotifyMessage))
	say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(c.NextStay))
	say("MOB_START_INCLUDE_UNCOMMITTED_CHANGES" + "=" + strconv.FormatBool(c.StartIncludeUncommittedChanges))
	say("MOB_STASH_NAME" + "=" + quote(c.StashName))
	say("MOB_WIP_BRANCH_QUALIFIER" + "=" + quote(c.WipBranchQualifier))
	say("MOB_WIP_BRANCH_QUALIFIER_SEPARATOR" + "=" + quote(c.WipBranchQualifierSeparator))
	say("MOB_WIP_BRANCH_PREFIX" + "=" + quote(c.WipBranchPrefix))
	say("MOB_DONE_SQUASH" + "=" + strconv.FormatBool(c.DoneSquash))
	say("MOB_TIMER" + "=" + quote(c.Timer))
	say("MOB_TIMER_ROOM" + "=" + quote(c.TimerRoom))
	say("MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER" + "=" + strconv.FormatBool(c.TimerRoomUseWipBranchQualifier))
	say("MOB_TIMER_LOCAL" + "=" + strconv.FormatBool(c.TimerLocal))
	say("MOB_TIMER_USER" + "=" + quote(c.TimerUser))
	say("MOB_TIMER_URL" + "=" + quote(c.TimerUrl))
}

func quote(value string) string {
	return strconv.Quote(value)
}

func parseArgs(args []string, configuration Configuration) (command string, parameters []string, newConfiguration Configuration) {
	newConfiguration = configuration

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--include-uncommitted-changes", "-i":
			newConfiguration.StartIncludeUncommittedChanges = true
		case "--debug":
			// ignore this, already parsed
		case "--stay", "-s":
			newConfiguration.NextStay = true
		case "--return-to-base-branch", "-r":
			newConfiguration.NextStay = false
		case "--branch", "-b":
			if i+1 != len(args) {
				newConfiguration.WipBranchQualifier = args[i+1]
			}
			i++ // skip consumed parameter
		case "--message", "-m":
			if i+1 != len(args) {
				newConfiguration.WipCommitMessage = args[i+1]
			}
			i++ // skip consumed parameter
		case "--squash":
			newConfiguration.DoneSquash = true
		case "--no-squash":
			newConfiguration.DoneSquash = false
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
		} else if configuration.Timer != "" {
			startTimer(configuration.Timer, configuration)
		} else {
			sayInfo("It's now " + currentTime() + ". Happy collaborating!")
		}
	case "b", "branch":
		branch(configuration)
	case "n", "next":
		next(configuration)
	case "d", "done":
		done(configuration)
	case "fetch":
		fetch(configuration)
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
		} else if configuration.Timer != "" {
			startTimer(configuration.Timer, configuration)
		} else {
			help(configuration)
		}
	case "break":
		if len(parameter) > 0 {
			startBreakTimer(parameter[0], configuration)
		} else {
			help(configuration)
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
		help(configuration)
	default:
		help(configuration)
	}
}

func branch(configuration Configuration) {
	say(silentgit("branch", "--list", "--remote", newBranch("*").addWipPrefix(configuration).remote(configuration).Name))

	// DEPRECATED
	say(silentgit("branch", "--list", "--remote", newBranch("mob-session").remote(configuration).Name))
}

func determineBranches(currentBranch Branch, localBranches []string, configuration Configuration) (baseBranch Branch, wipBranch Branch) {
	if currentBranch.Is("mob-session") || (currentBranch.Is("master") && !configuration.customWipBranchQualifierConfigured()) {
		// DEPRECATED
		baseBranch = newBranch("master")
		wipBranch = newBranch("mob-session")
	} else if currentBranch.IsWipBranch(configuration) {
		baseBranch = currentBranch.removeWipPrefix(configuration).removeWipQualifier(localBranches, configuration)
		wipBranch = currentBranch
	} else {
		baseBranch = currentBranch
		wipBranch = currentBranch.addWipPrefix(configuration).addWipQualifier(configuration)
	}

	debugInfo("on currentBranch " + currentBranch.String() + " => BASE " + baseBranch.String() + " WIP " + wipBranch.String() + " with allLocalBranches " + strings.Join(localBranches, ","))
	if currentBranch != baseBranch && currentBranch != wipBranch {
		// this is unreachable code, but we keep it as a backup
		panic("assertion failed! neither on base nor on wip branch")
	}
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
	timeoutInMinutes := toMinutes(timerInMinutes)

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	debugInfo(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	timerSuccessful := false

	room := getMobTimerRoom(configuration)
	if room != "" {
		user := getUserForMobTimer(configuration.TimerUser)
		err := httpPutTimer(timeoutInMinutes, room, user, configuration.TimerUrl)
		if err != nil {
			sayError("remote timer couldn't be started")
			sayError(err.Error())
		} else {
			timerSuccessful = true
		}
	}

	if configuration.TimerLocal {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand), getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand))

		if err != nil {
			sayError(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
			sayError(err.Error())
		} else {
			timerSuccessful = true
		}
	}

	if timerSuccessful {
		sayInfo("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating!")
	}
}

func getMobTimerRoom(configuration Configuration) string {
	currentWipBranchQualifier := configuration.WipBranchQualifier
	if currentWipBranchQualifier == "" {
		currentBranch := gitCurrentBranch()
		currentBaseBranch, _ := determineBranches(currentBranch, gitBranches(), configuration)

		if currentBranch.IsWipBranch(configuration) {
			wipBranchWithouthWipPrefix := currentBranch.removeWipPrefix(configuration).Name
			currentWipBranchQualifier = removePrefix(removePrefix(wipBranchWithouthWipPrefix, currentBaseBranch.Name), configuration.WipBranchQualifierSeparator)
		}
	}

	if configuration.TimerRoomUseWipBranchQualifier && currentWipBranchQualifier != "" {
		sayInfo("Using wip branch qualifier for room name")
		return currentWipBranchQualifier
	}
	return configuration.TimerRoom
}

func startBreakTimer(timerInMinutes string, configuration Configuration) {
	timeoutInMinutes := toMinutes(timerInMinutes)

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	debugInfo(fmt.Sprintf("Starting break timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	timerSuccessful := false
	room := getMobTimerRoom(configuration)
	if room != "" {
		user := getUserForMobTimer(configuration.TimerUser)
		err := httpPutBreakTimer(timeoutInMinutes, room, user, configuration.TimerUrl)
		if err != nil {
			sayError("remote break timer couldn't be started")
			sayError(err.Error())
		} else {
			timerSuccessful = true
		}
	}

	if configuration.TimerLocal {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob start", configuration.VoiceCommand), getNotifyCommand("mob start", configuration.NotifyCommand))

		if err != nil {
			sayError(fmt.Sprintf("break timer couldn't be started on your system (%s)", runtime.GOOS))
			sayError(err.Error())
		} else {
			timerSuccessful = true
		}
	}

	if timerSuccessful {
		sayInfo("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating!")
	}
}

func getUserForMobTimer(userOverride string) string {
	if userOverride == "" {
		return gitUserName()
	}
	return userOverride
}

func toMinutes(timerInMinutes string) int {
	timeoutInMinutes, _ := strconv.Atoi(timerInMinutes)
	if timeoutInMinutes < 0 {
		timeoutInMinutes = 0
	}
	return timeoutInMinutes
}

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	return sendRequest(putBody, "PUT", timerService+room)
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	return sendRequest(putBody, "PUT", timerService+room)
}

func sendRequest(requestBody []byte, requestMethod string, requestUrl string) error {
	sayInfo(requestMethod + " " + requestUrl + " " + string(requestBody))

	responseBody := bytes.NewBuffer(requestBody)
	request, requestCreationError := http.NewRequest(requestMethod, requestUrl, responseBody)
	if requestCreationError != nil {
		return fmt.Errorf("failed to create the http request object: %w", requestCreationError)
	}

	request.Header.Set("Content-Type", "application/json")
	response, responseErr := http.DefaultClient.Do(request)
	if responseErr != nil {
		return fmt.Errorf("failed to make the http request: %w", responseErr)
	}
	defer response.Body.Close()
	body, responseReadingErr := ioutil.ReadAll(response.Body)
	if responseReadingErr != nil {
		return fmt.Errorf("failed to read the http response: %w", responseReadingErr)
	}
	if string(body) != "" {
		sayInfo(string(body))
	}
	return nil
}

func currentTime() string {
	return time.Now().Format("15:04")
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

	git("checkout", currentBaseBranch.String())
	if hasLocalBranch(currentWipBranch.String()) {
		git("branch", "--delete", "--force", currentWipBranch.String())
	}
	if currentWipBranch.hasRemoteBranch(configuration) {
		gitWithoutEmptyStrings("push", configuration.gitHooksOption(), configuration.RemoteName, "--delete", currentWipBranch.String())
	}
	sayInfo("Branches " + currentWipBranch.String() + " and " + currentWipBranch.remote(configuration).String() + " deleted")
}

func start(configuration Configuration) error {
	uncommittedChanges := hasUncommittedChanges()
	if uncommittedChanges && !configuration.StartIncludeUncommittedChanges {
		sayInfo("cannot start; clean working tree required")
		sayUnstagedChangesInfo()
		sayUntrackedFilesInfo()
		sayTodo("To start, including uncommitted changes, use", configuration.mob("start --include-uncommitted-changes"))
		return errors.New("cannot start; clean working tree required")
	}

	git("fetch", configuration.RemoteName, "--prune")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if !currentBaseBranch.hasRemoteBranch(configuration) {
		sayError("Remote branch " + currentBaseBranch.remote(configuration).String() + " is missing")
		sayTodo("To set the upstream branch, use", "git push "+configuration.RemoteName+" "+currentBaseBranch.String()+" --set-upstream")
		return errors.New("remote branch is missing")
	}

	if currentBaseBranch.hasUnpushedCommits(configuration) {
		sayError("cannot start; unpushed changes on base branch must be pushed upstream")
		sayTodo("to fix this, push those commits and try again", "git push "+configuration.RemoteName+" "+currentBaseBranch.String())
		return errors.New("cannot start; unpushed changes on base branch must be pushed upstream")
	}

	if uncommittedChanges && silentgit("ls-tree", "-r", "HEAD", "--full-name", "--name-only", ".") == "" {
		sayError("cannot start; current working dir is an uncommitted subdir")
		sayTodo("to fix this, go to the parent directory and try again", "cd ..")
		return errors.New("cannot start; current working dir is an uncommitted subdir")
	}

	if uncommittedChanges {
		git("stash", "push", "--include-untracked", "--message", configuration.StashName)
		sayInfo("uncommitted changes were stashed. If an error occurs later on, you can recover them with 'git stash pop'.")
	}

	if !isMobProgramming(configuration) {
		git("merge", "FETCH_HEAD", "--ff-only")
	}

	if currentWipBranch.hasRemoteBranch(configuration) {
		startJoinMobSession(configuration)
	} else {
		warnForActiveWipBranches(configuration, currentBaseBranch)

		startNewMobSession(configuration)
	}

	if uncommittedChanges && configuration.StartIncludeUncommittedChanges {
		stashes := silentgit("stash", "list")
		stash := findStashByName(stashes, configuration.StashName)
		git("stash", "pop", stash)
	}

	sayInfo("you are on wip branch '" + currentWipBranch.String() + "' (base branch '" + currentBaseBranch.String() + "')")
	sayLastCommitsList(currentBaseBranch.String(), currentWipBranch.String())

	return nil // no error
}

func warnForActiveWipBranches(configuration Configuration, currentBaseBranch Branch) {
	if isMobProgramming(configuration) {
		return
	}

	// TODO show all active wip branches, even non-qualified ones
	existingWipBranches := getWipBranchesForBaseBranch(currentBaseBranch, configuration)
	if len(existingWipBranches) > 0 && configuration.WipBranchQualifier == "" {
		sayWarning("Creating a new wip branch even though preexisting wip branches have been detected.")
		for _, wipBranch := range existingWipBranches {
			sayWithPrefix(wipBranch, "  - ")
		}
	}
}

func showActiveMobSessions(configuration Configuration, currentBaseBranch Branch) {
	existingWipBranches := getWipBranchesForBaseBranch(currentBaseBranch, configuration)
	if len(existingWipBranches) > 0 {
		sayInfo("remote wip branches detected:")
		for _, wipBranch := range existingWipBranches {
			sayWithPrefix(wipBranch, "  - ")
		}
	}
}

func sayUntrackedFilesInfo() {
	untrackedFiles := getUntrackedFiles()
	hasUntrackedFiles := len(untrackedFiles) > 0
	if hasUntrackedFiles {
		sayInfo("untracked files present:")
		sayInfoIndented(untrackedFiles)
	}
}

func sayUnstagedChangesInfo() {
	unstagedChanges := getUnstagedChanges()
	hasUnstagedChanges := len(unstagedChanges) > 0
	if hasUnstagedChanges {
		sayInfo("unstaged changes present:")
		sayInfoIndented(unstagedChanges)
	}
}

func getWipBranchesForBaseBranch(currentBaseBranch Branch, configuration Configuration) []string {
	remoteBranches := gitRemoteBranches()
	debugInfo("check on current base branch " + currentBaseBranch.String() + " with remote branches " + strings.Join(remoteBranches, ","))

	remoteBranchWithQualifier := currentBaseBranch.addWipPrefix(configuration).addWipQualifier(configuration).remote(configuration).Name
	remoteBranchNoQualifier := currentBaseBranch.addWipPrefix(configuration).remote(configuration).Name
	if currentBaseBranch.Is("master") {
		// LEGACY
		remoteBranchNoQualifier = "mob-session"
	}

	var result []string
	for _, remoteBranch := range remoteBranches {
		if strings.Contains(remoteBranch, remoteBranchWithQualifier) || strings.Contains(remoteBranch, remoteBranchNoQualifier) {
			result = append(result, remoteBranch)
		}
	}

	return result
}

func startJoinMobSession(configuration Configuration) {
	_, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	sayInfo("joining existing session from " + currentWipBranch.remote(configuration).String())
	git("checkout", "-B", currentWipBranch.Name, currentWipBranch.remote(configuration).Name)
	git("branch", "--set-upstream-to="+currentWipBranch.remote(configuration).Name, currentWipBranch.Name)
}

func startNewMobSession(configuration Configuration) {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	sayInfo("starting new session from " + currentBaseBranch.remote(configuration).String())
	git("checkout", "-B", currentWipBranch.Name, currentBaseBranch.remote(configuration).Name)
	gitWithoutEmptyStrings("push", configuration.gitHooksOption(), "--set-upstream", configuration.RemoteName, currentWipBranch.Name)
}

func getUntrackedFiles() string {
	return silentgit("ls-files", "--others", "--exclude-standard", "--full-name")
}

func getUnstagedChanges() string {
	return silentgit("diff", "--stat")
}

func findStashByName(stashes string, stash string) string {
	lines := strings.Split(stashes, "\n")
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.Contains(line, stash) {
			return line[:strings.Index(line, ":")]
		}
	}
	return "unknown"
}

func next(configuration Configuration) {
	if !isMobProgramming(configuration) {
		sayTodo("to start working together, use", configuration.mob("start"))
		return
	}

	if !configuration.hasCustomCommitMessage() && configuration.RequireCommitMessage && hasUncommittedChanges() {
		sayError("commit message required")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if isNothingToCommit() {
		if currentWipBranch.hasLocalCommits(configuration) {
			gitWithoutEmptyStrings("push", configuration.gitHooksOption(), configuration.RemoteName, currentWipBranch.Name)
		} else {
			sayInfo("nothing was done, so nothing to commit")
		}
	} else {
		makeWipCommit(configuration)
		gitWithoutEmptyStrings("push", configuration.gitHooksOption(), configuration.RemoteName, currentWipBranch.Name)
	}
	showNext(configuration)

	if !configuration.NextStay {
		git("checkout", currentBaseBranch.Name)
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
	gitWithoutEmptyStrings("commit", "--message", configuration.WipCommitMessage, configuration.gitHooksOption())
	sayInfoIndented(getChangesOfLastCommit())
	sayInfoIndented(gitCommitHash())
}

func (c Configuration) gitHooksOption() string {
	if c.GitHooksEnabled {
		return ""
	} else {
		return "--no-verify"
	}
}

func fetch(configuration Configuration) {
	git("fetch", configuration.RemoteName, "--prune")
}

func done(configuration Configuration) {
	if !isMobProgramming(configuration) {
		sayTodo("to start working together, use", configuration.mob("start"))
		return
	}

	git("fetch", configuration.RemoteName, "--prune")

	baseBranch, wipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if wipBranch.hasRemoteBranch(configuration) {
		uncommittedChanges := hasUncommittedChanges()
		if uncommittedChanges {
			makeWipCommit(configuration)
		}
		gitWithoutEmptyStrings("push", configuration.gitHooksOption(), configuration.RemoteName, wipBranch.Name)

		git("checkout", baseBranch.Name)
		git("merge", baseBranch.remote(configuration).Name, "--ff-only")
		mergeFailed := gitignorefailure("merge", squashOrNoCommit(configuration), "--ff", wipBranch.Name)
		if mergeFailed != nil {
			sayWarning("Skipped deleting " + wipBranch.Name + " because of merge conflicts.")
			sayWarning("To fix this, solve the merge conflict manually, commit, push, and afterwards delete " + wipBranch.Name)
			return
		}

		git("branch", "-D", wipBranch.Name)

		if uncommittedChanges && !configuration.DoneSquash { // give the user the chance to name their final commit
			git("reset", "--soft", "HEAD^")
		}

		gitWithoutEmptyStrings("push", configuration.gitHooksOption(), configuration.RemoteName, "--delete", wipBranch.Name)

		cachedChanges := getCachedChanges()
		hasCachedChanges := len(cachedChanges) > 0
		if hasCachedChanges {
			sayInfoIndented(cachedChanges)
		}
		err := appendCoauthorsToSquashMsg(gitDir())
		if err != nil {
			sayError(err.Error())
		}

		if hasUncommittedChanges() {
			sayTodo("To finish, use", "git commit")
		} else if configuration.DoneSquash {
			sayInfo("nothing was done, so nothing to commit")
		}

	} else {
		git("checkout", baseBranch.Name)
		git("branch", "-D", wipBranch.Name)
		sayInfo("someone else already ended your session")
	}
}

func gitDir() string {
	return silentgit("rev-parse", "--absolute-git-dir")
}

func gitRootDir() string {
	return strings.TrimSuffix(gitDir(), "/.git")
}

func squashOrNoCommit(configuration Configuration) string {
	if configuration.DoneSquash {
		return "--squash"
	} else {
		return "--no-commit"
	}
}

func status(configuration Configuration) {
	if isMobProgramming(configuration) {
		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		sayInfo("you are on wip branch " + currentWipBranch.String() + " (base branch " + currentBaseBranch.String() + ")")

		sayLastCommitsList(currentBaseBranch.String(), currentWipBranch.String())
	} else {
		currentBaseBranch, _ := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		sayInfo("you are on base branch '" + currentBaseBranch.String() + "'")
		showActiveMobSessions(configuration, currentBaseBranch)
	}
}

func sayLastCommitsList(currentBaseBranch string, currentWipBranch string) {
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	lines := strings.Split(log, "\n")
	if len(lines) > 5 {
		sayInfo("wip branch '" + currentWipBranch + "' contains " + strconv.Itoa(len(lines)) + " commits. The last 5 were:")
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

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
}

func isMobProgramming(configuration Configuration) bool {
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, gitBranches(), configuration)
	debugInfo("current branch " + currentBranch.String() + " and currentWipBranch " + currentWipBranch.String())
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

func gitBranches() []string {
	return strings.Split(silentgit("branch", "--format=%(refname:short)"), "\n")
}

func gitRemoteBranches() []string {
	return strings.Split(silentgit("branch", "--remotes", "--format=%(refname:short)"), "\n")
}

func gitCurrentBranch() Branch {
	// upgrade to branch --show-current when git v2.21 is more widely spread
	return newBranch(silentgit("rev-parse", "--abbrev-ref", "HEAD"))
}

func gitUserName() string {
	return silentgitignorefailure("config", "--get", "user.name")
}

func gitUserEmail() string {
	return silentgit("config", "--get", "user.email")
}

func showNext(configuration Configuration) {
	debugInfo("determining next person based on previous changes")
	gitUserName := gitUserName()
	if gitUserName == "" {
		sayWarning("failed to detect who's next because you haven't set your git user name")
		sayTodo("To fix, use", "git config --global user.name \"Your Name Here\"")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + currentWipBranch.String()

	changes := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%an", "--abbrev-commit")
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	debugInfo("there have been " + strconv.Itoa(numberOfLines) + " changes")
	debugInfo("current git user.name is '" + gitUserName + "'")
	if numberOfLines < 1 {
		return
	}
	nextTypist, previousCommitters := findNextTypist(lines, gitUserName)
	if nextTypist != "" {
		sayInfo("Committers after your last commit: " + strings.Join(previousCommitters, ", "))
		sayInfo("***" + nextTypist + "*** is (probably) next.")
	}
}

func help(configuration Configuration) {
	output := configuration.CliName + ` enables a smooth Git handover

Basic Commands:
  start              start session from base branch in wip branch
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
  start <minutes>    start mob session in wip branch and a <minutes> timer
  break <minutes>    start a <minutes> break timer

Get more information:
  status             show the status of the current session
  fetch              fetch remote state
  branch             show remote wip branches
  config             show all configuration options
  version            show the version
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

func silentgitignorefailure(args ...string) string {
	_, output, err := runCommand("git", args...)

	if err != nil {
		return ""
	}
	return strings.TrimSpace(output)
}

func deleteEmptyStrings(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

func gitWithoutEmptyStrings(args ...string) {
	argsWithoutEmptyStrings := deleteEmptyStrings(args)
	git(argsWithoutEmptyStrings...)
}

func git(args ...string) {
	argsWithoutEmptyStrings := deleteEmptyStrings(args)
	commandString, output, err := runCommand("git", argsWithoutEmptyStrings...)

	if err != nil {
		sayGitError(commandString, output, err)
		exit(1)
	} else {
		sayIndented(commandString)
	}
}

func gitCommitHash() string {
	return silentgitignorefailure("rev-parse", "HEAD")
}

func sayGitError(commandString string, output string, err error) {
	if !isGit() {
		sayError("expecting the current working directory to be a git repository.")
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
	sayWithPrefix(text, "ERROR ")
}

func debugInfo(text string) {
	if Debug {
		sayWithPrefix(text, "DEBUG ")
	}
}

func sayIndented(text string) {
	sayWithPrefix(text, "  ")
}

func sayTodo(text string, command string) {
	sayWithPrefix(text, "👉 ")
	sayEmptyLine()
	sayIndented(command)
	sayEmptyLine()
}

func sayInfo(text string) {
	sayWithPrefix(text, "> ")
}

func sayInfoIndented(text string) {
	sayWithPrefix(text, "    ")
}

func sayWarning(text string) {
	sayWithPrefix(text, "⚠ ")
}

func sayWithPrefix(s string, prefix string) {
	lines := strings.Split(strings.TrimSpace(s), "\n")
	for i := 0; i < len(lines); i++ {
		printToConsole(prefix + strings.TrimSpace(lines[i]) + "\n")
	}
}

func say(s string) {
	if len(s) == 0 {
		return
	}
	printToConsole(strings.TrimRight(s, " \r\n\t\v\f") + "\n")
}

func sayEmptyLine() {
	printToConsole("\n")
}

var printToConsole = func(message string) {
	fmt.Print(message)
}
