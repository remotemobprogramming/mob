package main

import (
	"bufio"
	"os"
	"runtime"
	"strconv"
	"strings"
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

func (c Configuration) isOpenCommandGiven() bool {
	return strings.TrimSpace(c.OpenCommand) != ""
}

func (c Configuration) gitHooksOption() string {
	if c.GitHooksEnabled {
		return ""
	} else {
		return "--no-verify"
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

	debugInfo("overriding " + key + "=" + configuration.DoneSquash)
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

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		sayWarning("Project configuration file exists, but could not be read. (" + path + ")")
	}

	return configuration
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

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		sayWarning("User configuration file exists, but could not be read. (" + path + ")")
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
