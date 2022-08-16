package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	x509 "crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
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
	versionNumber = "3.1.5"
)

var (
	workingDir                 = ""
	Debug                      = false // override with --debug parameter
	GitPassthroughStderrStdout = false // hack to get git hooks to print to stdout/stderr
)

const (
	Squash    = "squash"
	NoSquash  = "no-squash"
	SquashWip = "squash-wip"
)

func doneSquash(value string) string {
	switch value {
	case "false", NoSquash:
		return NoSquash
	case SquashWip:
		return SquashWip
	default:
		return Squash
	}
}

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
	DoneSquash                     string // override with MOB_DONE_SQUASH
	OpenCommand                    string // override with MOB_OPEN_COMMAND
	Timer                          string // override with MOB_TIMER
	TimerRoom                      string // override with MOB_TIMER_ROOM
	TimerLocal                     bool   // override with MOB_TIMER_LOCAL
	TimerRoomUseWipBranchQualifier bool   // override with MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER
	TimerUser                      string // override with MOB_TIMER_USER
	TimerUrl                       string // override with MOB_TIMER_URL
	TimerInsecure                  bool   // override with MOB_TIMER_INSECURE
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
	return strings.HasPrefix(line, c.WipCommitMessage)
}

func (c Configuration) openCommandFor(filepath string) (string, []string) {
	if !c.isOpenCommandGiven() {
		return "", []string{}
	}
	split := strings.Split(injectCommandWithMessage(c.OpenCommand, filepath), " ")
	return split[0], split[1:]
}

func (c Configuration) isOpenCommandGiven() bool {
	return strings.TrimSpace(c.OpenCommand) != ""
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
	if branch.Name == "mob-session" {
		return true
	}

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
	debugInfo(runtime.Version())

	if !isGitInstalled() {
		sayError("'git' command was not found in PATH. It may be not installed. " +
			"To learn how to install 'git' refer to https://git-scm.com/book/en/v2/Getting-Started-Installing-Git.")
		exit(1)
	}

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

	// workaround until we have a better design
	if configuration.GitHooksEnabled {
		GitPassthroughStderrStdout = true
	}

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
		DoneSquash:                     Squash,
		OpenCommand:                    "",
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
		case "MOB_WIP_BRANCH_PREFIX":
			setUnquotedString(&configuration.WipBranchPrefix, key, value)
		case "MOB_DONE_SQUASH":
			setMobDoneSquash(&configuration, key, value)
		case "MOB_OPEN_COMMAND":
			setUnquotedString(&configuration.OpenCommand, key, value)
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
		case "MOB_TIMER_INSECURE":
			setBoolean(&configuration.TimerInsecure, key, value)

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
		case "MOB_VOICE_COMMAND", "MOB_VOICE_MESSAGE", "MOB_NOTIFY_COMMAND", "MOB_NOTIFY_MESSAGE", "MOB_OPEN_COMMAND":
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
		case "MOB_WIP_BRANCH_PREFIX":
			setUnquotedString(&configuration.WipBranchPrefix, key, value)
		case "MOB_DONE_SQUASH":
			setMobDoneSquash(&configuration, key, value)
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
		case "MOB_TIMER_INSECURE":
			setBoolean(&configuration.TimerInsecure, key, value)

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

func setMobDoneSquash(configuration *Configuration, key string, value string) {
	if strings.HasPrefix(value, "\"") {
		unquotedValue, err := strconv.Unquote(value)
		if err != nil {
			sayWarning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
			return
		}
		value = unquotedValue
	}
	printDeprecatedDoneSquashMessage(value)
	configuration.DoneSquash = doneSquash(value)
	debugInfo("Overwriting " + key + " =" + string(configuration.DoneSquash))
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

	setDoneSquashFromEnvVariable(&configuration, "MOB_DONE_SQUASH")

	setStringFromEnvVariable(&configuration.OpenCommand, "MOB_OPEN_COMMAND")

	setStringFromEnvVariable(&configuration.Timer, "MOB_TIMER")
	setStringFromEnvVariable(&configuration.TimerRoom, "MOB_TIMER_ROOM")
	setBoolFromEnvVariable(&configuration.TimerRoomUseWipBranchQualifier, "MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER")
	setBoolFromEnvVariable(&configuration.TimerLocal, "MOB_TIMER_LOCAL")
	setStringFromEnvVariable(&configuration.TimerUser, "MOB_TIMER_USER")
	setStringFromEnvVariable(&configuration.TimerUrl, "MOB_TIMER_URL")
	setBoolFromEnvVariable(&configuration.TimerInsecure, "MOB_TIMER_INSECURE")

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
		sayWarning("ignoring " + key + "=" + value + " (not a boolean)")
	}
}

func setDoneSquashFromEnvVariable(configuration *Configuration, key string) {
	value, set := os.LookupEnv(key)
	if !set {
		return
	}
	printDeprecatedDoneSquashMessage(value)
	configuration.DoneSquash = doneSquash(value)

	if value == "" {
		debugInfo("ignoring " + key + "=" + value + " (empty string)")
		return
	}

	debugInfo("overriding " + key + "=" + string(configuration.DoneSquash))
}

func printDeprecatedDoneSquashMessage(value string) {
	unquotedValue, err := strconv.Unquote(value)
	if err != nil {
		unquotedValue = value
	}

	if unquotedValue == "true" || unquotedValue == "false" {
		newValue := doneSquash(unquotedValue)
		say("MOB_DONE_SQUASH is set to the deprecated value " + value + ". Use the value " + newValue + " instead.")
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
	say("MOB_DONE_SQUASH" + "=" + string(c.DoneSquash))
	say("MOB_OPEN_COMMAND" + "=" + quote(c.OpenCommand))
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
			newConfiguration.DoneSquash = Squash
		case "--no-squash":
			newConfiguration.DoneSquash = NoSquash
		case "--squash-wip":
			newConfiguration.DoneSquash = SquashWip
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
	if helpRequested(parameter) {
		help(configuration)
		return
	}

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
			sayInfo("It's now " + currentTime() + ". Happy collaborating! :)")
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
	case "clean":
		clean(configuration)
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
		}
	case "version", "--version", "-v":
		version()
	case "help", "--help", "-h":
		help(configuration)
	default:
		help(configuration)
	}
}

func helpRequested(parameter []string) bool {
	for i := 0; i < len(parameter); i++ {
		element := parameter[i]
		if element == "help" || element == "--help" || element == "-h" {
			return true
		}
	}
	return false
}

func clean(configuration Configuration) {
	git("fetch", configuration.RemoteName)

	currentBranch := gitCurrentBranch()
	localBranches := gitBranches()

	if currentBranch.isOrphanWipBranch(configuration) {
		currentBaseBranch, _ := determineBranches(currentBranch, localBranches, configuration)

		sayInfo("Current branch " + currentBranch.Name + " is an orphan")
		if currentBaseBranch.exists(localBranches) {
			git("checkout", currentBaseBranch.Name)
		} else if newBranch("main").exists(localBranches) {
			git("checkout", "main")
		} else {
			git("checkout", "master")
		}
	}

	for _, branch := range localBranches {
		b := newBranch(branch)
		if b.isOrphanWipBranch(configuration) {
			sayInfo("Removing orphan wip branch " + b.Name)
			git("branch", "-D", b.Name)
		}
	}

}

func (branch Branch) isOrphanWipBranch(configuration Configuration) bool {
	return branch.IsWipBranch(configuration) && !branch.hasRemoteBranch(configuration)
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
		sayWarning(fmt.Sprintf("Cannot execute background commands on your os: %s", runtime.GOOS))
	}
	return err
}

func startTimer(timerInMinutes string, configuration Configuration) {
	timeoutInMinutes := toMinutes(timerInMinutes)

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	debugInfo(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		sayError("No timer configured, not starting timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)
		if err != nil {
			sayError("remote timer couldn't be started")
			sayError(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand), getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand))

		if err != nil {
			sayError(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
			sayError(err.Error())
			exit(1)
		}
	}

	sayInfo("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
}

func getMobTimerRoom(configuration Configuration) string {
	if !isGit() {
		debugInfo("timer not in git repository, using MOB_TIMER_ROOM for room name")
		return configuration.TimerRoom
	}

	currentWipBranchQualifier := configuration.WipBranchQualifier
	if currentWipBranchQualifier == "" {
		currentBranch := gitCurrentBranch()
		currentBaseBranch, _ := determineBranches(currentBranch, gitBranches(), configuration)

		if currentBranch.IsWipBranch(configuration) {
			wipBranchWithoutWipPrefix := currentBranch.removeWipPrefix(configuration).Name
			currentWipBranchQualifier = removePrefix(removePrefix(wipBranchWithoutWipPrefix, currentBaseBranch.Name), configuration.WipBranchQualifierSeparator)
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

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		sayError("No break timer configured, not starting break timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutBreakTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)

		if err != nil {
			sayError("remote break timer couldn't be started")
			sayError(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob start", configuration.VoiceCommand), getNotifyCommand("mob start", configuration.NotifyCommand))

		if err != nil {
			sayError(fmt.Sprintf("break timer couldn't be started on your system (%s)", runtime.GOOS))
			sayError(err.Error())
			exit(1)
		}
	}

	sayInfo("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
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

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	return sendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	return sendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
}

func sendRequest(requestBody []byte, requestMethod string, requestUrl string, disableSSLVerification bool) error {
	sayInfo(requestMethod + " " + requestUrl + " " + string(requestBody))

	responseBody := bytes.NewBuffer(requestBody)
	request, requestCreationError := http.NewRequest(requestMethod, "https://untrusted-root.badssl.com/", responseBody)

	httpClient := http.DefaultClient
	if disableSSLVerification {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: transCfg}
	}

	if requestCreationError != nil {
		return fmt.Errorf("failed to create the http request object: %w", requestCreationError)
	}

	request.Header.Set("Content-Type", "application/json")
	response, responseErr := httpClient.Do(request)
	if e, ok := responseErr.(*url.Error); ok {
		switch e.Err.(type) {
		case x509.UnknownAuthorityError:
			sayError("The timer.mob.sh SSL certificate is signed by an unknown authority!")
			sayFix("HINT: You can ignore that by adding MOB_TIMER_INSECURE=true to your configuration or environment. Or add is command line parameter:",
				"mob <your command> --timer-insecure")
			return fmt.Errorf("failed, to amke the http request: %w", responseErr)

		default:
			return fmt.Errorf("failed to make the http request: %w", responseErr)

		}
	}

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
		sayWarning(fmt.Sprintf("can't run voice command on your system (%s)", runtime.GOOS))
		sayWarning(err.Error())
		return
	}

	sayInfo(voiceMessage)
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
		sayFix("To start, including uncommitted changes, use", configuration.mob("start --include-uncommitted-changes"))
		return errors.New("cannot start; clean working tree required")
	}

	git("fetch", configuration.RemoteName, "--prune")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if !currentBaseBranch.hasRemoteBranch(configuration) {
		sayError("Remote branch " + currentBaseBranch.remote(configuration).String() + " is missing")
		sayFix("To set the upstream branch, use", "git push "+configuration.RemoteName+" "+currentBaseBranch.String()+" --set-upstream")
		return errors.New("remote branch is missing")
	}

	if currentBaseBranch.hasUnpushedCommits(configuration) {
		sayError("cannot start; unpushed changes on base branch must be pushed upstream")
		sayFix("to fix this, push those commits and try again", "git push "+configuration.RemoteName+" "+currentBaseBranch.String())
		return errors.New("cannot start; unpushed changes on base branch must be pushed upstream")
	}

	if uncommittedChanges && silentgit("ls-tree", "-r", "HEAD", "--full-name", "--name-only", ".") == "" {
		sayError("cannot start; current working dir is an uncommitted subdir")
		sayFix("to fix this, go to the parent directory and try again", "cd ..")
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

	openLastModifiedFileIfPresent(configuration)

	return nil // no error
}

func openLastModifiedFileIfPresent(configuration Configuration) {
	if !configuration.isOpenCommandGiven() {
		debugInfo("No open command given")
		return
	}

	debugInfo("Try to open last modified file")
	if !lastCommitIsWipCommit(configuration) {
		debugInfo("Last commit isn't a WIP commit.")
		return
	}
	lastCommitMessage := lastCommitMessage()
	split := strings.Split(lastCommitMessage, "lastFile:")
	if len(split) == 1 {
		sayWarning("Couldn't find last modified file in commit message!")
		return
	}
	if len(split) > 2 {
		sayWarning("Could not determine last modified file from commit message, separator was used multiple times!")
		return
	}
	lastModifiedFile := split[1]
	if lastModifiedFile == "" {
		debugInfo("Could not find last modified file in commit message")
		return
	}
	lastModifiedFilePath := gitRootDir() + "/" + lastModifiedFile
	commandname, args := configuration.openCommandFor(lastModifiedFilePath)
	_, err := startCommand(commandname, args...)
	if err != nil {
		sayWarning(fmt.Sprintf("Couldn't open last modified file on your system (%s)", runtime.GOOS))
		sayWarning(err.Error())
		return
	}
	debugInfo("Open last modified file: " + lastModifiedFilePath)
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
	baseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	sayInfo("joining existing session from " + currentWipBranch.remote(configuration).String())
	if doBranchesDiverge(baseBranch.remote(configuration).Name, currentWipBranch.Name) {
		sayWarning("Careful, your wip branch (" + currentWipBranch.Name + ") diverges from your main branch (" + baseBranch.remote(configuration).Name + ") !")
	}

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
		sayFix("to start working together, use", configuration.mob("start"))
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
	commitMessage := createWipCommitMessage(configuration)
	gitWithoutEmptyStrings("commit", "--message", commitMessage, configuration.gitHooksOption())
	sayInfoIndented(getChangesOfLastCommit())
	sayInfoIndented(gitCommitHash())
}

func createWipCommitMessage(configuration Configuration) string {
	commitMessage := configuration.WipCommitMessage

	lastModifiedFilePath := getPathOfLastModifiedFile()
	if lastModifiedFilePath != "" {
		commitMessage += "\n\nlastFile:" + lastModifiedFilePath
	}

	return commitMessage
}

// uses git status --short. To work properly files have to be staged.
func getPathOfLastModifiedFile() string {
	rootDir := gitRootDir()
	files := getModifiedFiles(rootDir)
	lastModifiedFilePath := ""
	lastModifiedTime := time.Time{}

	debugInfo("Find last modified file")
	if len(files) == 1 {
		lastModifiedFilePath = files[0]
		debugInfo("Just one modified file: " + lastModifiedFilePath)
		return lastModifiedFilePath
	}

	for _, file := range files {
		absoluteFilepath := rootDir + "/" + file
		debugInfo(absoluteFilepath)
		info, err := os.Stat(absoluteFilepath)
		if err != nil {
			sayWarning("Could not get statistics of file: " + absoluteFilepath)
			sayWarning(err.Error())
			continue
		}
		modTime := info.ModTime()
		if modTime.After(lastModifiedTime) {
			lastModifiedTime = modTime
			lastModifiedFilePath = file
		}
		debugInfo(modTime.String())
	}
	return lastModifiedFilePath
}

// uses git status --short. To work properly files have to be staged.
func getModifiedFiles(rootDir string) []string {
	debugInfo("Find modified files")
	oldWorkingDir := workingDir
	workingDir = rootDir
	gitstatus := silentgit("status", "--short")
	workingDir = oldWorkingDir
	lines := strings.Split(gitstatus, "\n")
	files := []string{}
	for _, line := range lines {
		relativeFilepath := ""
		if strings.HasPrefix(line, "M") {
			relativeFilepath = strings.TrimPrefix(line, "M")
		} else if strings.HasPrefix(line, "A") {
			relativeFilepath = strings.TrimPrefix(line, "A")
		} else {
			continue
		}
		relativeFilepath = strings.TrimSpace(relativeFilepath)
		debugInfo(relativeFilepath)
		files = append(files, relativeFilepath)
	}
	return files
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
		sayFix("to start working together, use", configuration.mob("start"))
		return
	}

	if configuration.DoneSquash == SquashWip {
		squashWip(configuration)
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
		mergeFailed := gitignorefailure("merge", squashOrCommit(configuration), "--ff", wipBranch.Name)

		if mergeFailed != nil {
			// TODO should this be an error and a fix for that error?
			sayWarning("Skipped deleting " + wipBranch.Name + " because of merge conflicts.")
			sayWarning("To fix this, solve the merge conflict manually, commit, push, and afterwards delete " + wipBranch.Name)
			return
		}

		git("branch", "-D", wipBranch.Name)

		if uncommittedChanges && configuration.DoneSquash != Squash { // give the user the chance to name their final commit
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
			sayWarning(err.Error())
		}

		if hasUncommittedChanges() {
			sayNext("To finish, use", "git commit")
		} else if configuration.DoneSquash == Squash {
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

func squashOrCommit(configuration Configuration) string {
	if configuration.DoneSquash == Squash {
		return "--squash"
	} else {
		return "--commit"
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

func doBranchesDiverge(ancestor string, successor string) bool {
	_, _, err := runCommandSilent("git", "merge-base", "--is-ancestor", ancestor, successor)
	if err == nil {
		return false
	}
	return true
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
		sayFix("To fix, use", "git config --global user.name \"Your Name Here\"")
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
		if len(previousCommitters) != 0 {
			sayInfo("Committers after your last commit: " + strings.Join(previousCommitters, ", "))
		}
		sayInfo("***" + nextTypist + "*** is (probably) next.")
	}
}

func help(configuration Configuration) {
	output := configuration.CliName + ` enables a smooth Git handover

Basic Commands:
  start              Start session from base branch in wip branch
  next               Handover changes in wip branch to next person
  done               Squash all changes in wip branch to index in base branch
  reset              Remove local and remote wip branch

Basic Commands with Options:
  start [<minutes>]                      Start <minutes> minutes timer
    [--include-uncommitted-changes|-i]   Move uncommitted changes to wip branch
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>` + configuration.WipBranchQualifierSeparator + `<branch-postfix>'
  next
    [--stay|-s]                          Stay on wip branch (default)
    [--return-to-base-branch|-r]         Return to base branch
    [--message|-m <commit-message>]      Override commit message
  done
    [--no-squash]                        Squash no commits from wip branch, only merge wip branch
    [--squash]                           Squash all commits from wip branch
    [--squash-wip]                       Squash wip commits from wip branch, maintaining manual commits
  reset
    [--branch|-b <branch-postfix>]       Set wip branch to 'mob/<base-branch>` + configuration.WipBranchQualifierSeparator + `<branch-postfix>'
  clean                                  Remove all orphan wip branches

Timer Commands:
  timer <minutes>    Start <minutes> minutes timer
  start <minutes>    Start mob session in wip branch and a <minutes> timer
  break <minutes>    Start <minutes> break timer

Short Commands (Options and descriptions as above):
  s                  Alias for 'start'
  n                  Alias for 'next'
  d                  Alias for 'done'
  b                  Alias for 'branch'
  t                  Alias for 'timer'

Get more information:
  status             Show status of the current session
  fetch              Fetch remote state
  branch             Show remote wip branches
  config             Show all configuration options
  version            Show tool version
  help               Show help

Other
  moo                Moo!

Add '--debug' to any option to enable verbose logging.
`
	say(output)
}

func version() {
	say("v" + versionNumber)
}

func silentgit(args ...string) string {
	commandString, output, err := runCommandSilent("git", args...)

	if err != nil {
		if !isGit() {
			sayError("expecting the current working directory to be a git repository.")
		} else {
			sayError(commandString)
			sayError(output)
			sayError(err.Error())
		}
		exit(1)
	}
	return strings.TrimSpace(output)
}

func silentgitignorefailure(args ...string) string {
	_, output, err := runCommandSilent("git", args...)

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
	commandString, output, err := "", "", error(nil)
	if GitPassthroughStderrStdout {
		commandString, output, err = runCommand("git", args...)
	} else {
		commandString, output, err = runCommandSilent("git", args...)
	}

	if err != nil {
		if !isGit() {
			sayError("expecting the current working directory to be a git repository.")
		} else {
			sayError(commandString)
			sayError(output)
			sayError(err.Error())
		}
		exit(1)
	} else {
		sayIndented(commandString)
	}
}

func gitignorefailure(args ...string) error {
	commandString, output, err := "", "", error(nil)
	if GitPassthroughStderrStdout {
		commandString, output, err = runCommand("git", args...)
	} else {
		commandString, output, err = runCommandSilent("git", args...)
	}

	sayIndented(commandString)

	if err != nil {
		if !isGit() {
			sayError("expecting the current working directory to be a git repository.")
			exit(1)
		} else {
			sayWarning(commandString)
			sayWarning(output)
			sayWarning(err.Error())
			return err
		}
	}

	sayIndented(commandString)
	return nil
}

func gitCommitHash() string {
	return silentgitignorefailure("rev-parse", "HEAD")
}

func isGitInstalled() bool {
	_, _, err := runCommandSilent("git", "--version")
	if err != nil {
		debugInfo("isGitInstalled encountered an error: " + err.Error())
	}
	return err == nil
}

func isGit() bool {
	_, _, err := runCommandSilent("git", "rev-parse")
	return err == nil
}

func runCommandSilent(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	debugInfo("Running command <" + commandString + "> in silent mode, capturing combined output")
	outputBytes, err := command.CombinedOutput()
	output := string(outputBytes)
	debugInfo(output)
	return commandString, output, err
}

func runCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	debugInfo("Running command <" + commandString + "> passing output through")

	stdout, _ := command.StdoutPipe()
	command.Stderr = command.Stdout
	errStart := command.Start()
	if errStart != nil {
		return commandString, "", errStart
	}

	output := ""

	stdoutscanner := bufio.NewScanner(stdout)
	lineEnded := true
	stdoutscanner.Split(bufio.ScanBytes)
	for stdoutscanner.Scan() {
		character := stdoutscanner.Text()
		if character == "\n" {
			lineEnded = true
		} else {
			if lineEnded {
				printToConsole("  ")
				lineEnded = false
			}
		}
		printToConsole(character)
		output += character
	}

	errWait := command.Wait()
	if errWait != nil {
		debugInfo(output)
		return commandString, output, errWait
	}

	debugInfo(output)
	return commandString, output, nil
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

func sayFix(instruction string, command string) {
	sayWithPrefix(instruction, "👉 ")
	sayEmptyLine()
	sayIndented(command)
	sayEmptyLine()
}

func sayNext(instruction string, command string) {
	sayWithPrefix(instruction, "👉 ")
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
