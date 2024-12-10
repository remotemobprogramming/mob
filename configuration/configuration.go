package configuration

import (
	"bufio"
	"github.com/remotemobprogramming/mob/v5/say"
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

const (
	IncludeChanges = "include-changes"
	DiscardChanges = "discard-changes"
	FailWithError  = "fail-with-error"
)

type Configuration struct {
	CliName                        string // override with MOB_CLI_NAME
	RemoteName                     string // override with MOB_REMOTE_NAME
	WipCommitMessage               string // override with MOB_WIP_COMMIT_MESSAGE
	StartCommitMessage             string // override with MOB_START_COMMIT_MESSAGE
	SkipCiPushOptionEnabled        bool   // override with MOB_SKIP_CI_PUSH_OPTION_ENABLED
	GitHooksEnabled                bool   // override with MOB_GIT_HOOKS_ENABLED
	RequireCommitMessage           bool   // override with MOB_REQUIRE_COMMIT_MESSAGE
	VoiceCommand                   string // override with MOB_VOICE_COMMAND
	VoiceMessage                   string // override with MOB_VOICE_MESSAGE
	NotifyCommand                  string // override with MOB_NOTIFY_COMMAND
	NotifyMessage                  string // override with MOB_NOTIFY_MESSAGE
	NextStay                       bool   // override with MOB_NEXT_STAY
	HandleUncommittedChanges       string
	StartCreate                    bool // override with MOB_START_CREATE variable
	StartJoin                      bool
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
	ResetDeleteRemoteWipBranch     bool   // override with MOB_RESET_DELETE_REMOTE_WIP_BRANCH
}

func (c Configuration) Mob(command string) string {
	return c.CliName + " " + command
}

func (c Configuration) WipBranchQualifierSuffix() string {
	return c.WipBranchQualifierSeparator + c.WipBranchQualifier
}

func (c Configuration) CustomWipBranchQualifierConfigured() bool {
	return c.WipBranchQualifier != ""
}

func (c Configuration) HasCustomCommitMessage() bool {
	return GetDefaultConfiguration().WipCommitMessage != c.WipCommitMessage
}

func (c Configuration) IsWipCommitMessage(line string) bool {
	return strings.HasPrefix(line, c.WipCommitMessage)
}

func (c Configuration) IsOpenCommandGiven() bool {
	return strings.TrimSpace(c.OpenCommand) != ""
}

func Config(c Configuration) {
	say.Say("MOB_CLI_NAME" + "=" + quote(c.CliName))
	say.Say("MOB_DONE_SQUASH" + "=" + string(c.DoneSquash))
	say.Say("MOB_GIT_HOOKS_ENABLED" + "=" + strconv.FormatBool(c.GitHooksEnabled))
	say.Say("MOB_NEXT_STAY" + "=" + strconv.FormatBool(c.NextStay))
	say.Say("MOB_NOTIFY_COMMAND" + "=" + quote(c.NotifyCommand))
	say.Say("MOB_NOTIFY_MESSAGE" + "=" + quote(c.NotifyMessage))
	say.Say("MOB_OPEN_COMMAND" + "=" + quote(c.OpenCommand))
	say.Say("MOB_REMOTE_NAME" + "=" + quote(c.RemoteName))
	say.Say("MOB_REQUIRE_COMMIT_MESSAGE" + "=" + strconv.FormatBool(c.RequireCommitMessage))
	say.Say("MOB_SKIP_CI_PUSH_OPTION_ENABLED" + "=" + strconv.FormatBool(c.SkipCiPushOptionEnabled))
	say.Say("MOB_START_COMMIT_MESSAGE" + "=" + quote(c.StartCommitMessage))
	say.Say("MOB_STASH_NAME" + "=" + quote(c.StashName))
	say.Say("MOB_TIMER_INSECURE" + "=" + strconv.FormatBool(c.TimerInsecure))
	say.Say("MOB_TIMER_LOCAL" + "=" + strconv.FormatBool(c.TimerLocal))
	say.Say("MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER" + "=" + strconv.FormatBool(c.TimerRoomUseWipBranchQualifier))
	say.Say("MOB_TIMER_ROOM" + "=" + quote(c.TimerRoom))
	say.Say("MOB_TIMER_URL" + "=" + quote(c.TimerUrl))
	say.Say("MOB_TIMER_USER" + "=" + quote(c.TimerUser))
	say.Say("MOB_TIMER" + "=" + quote(c.Timer))
	say.Say("MOB_VOICE_COMMAND" + "=" + quote(c.VoiceCommand))
	say.Say("MOB_VOICE_MESSAGE" + "=" + quote(c.VoiceMessage))
	say.Say("MOB_WIP_BRANCH_PREFIX" + "=" + quote(c.WipBranchPrefix))
	say.Say("MOB_WIP_BRANCH_QUALIFIER_SEPARATOR" + "=" + quote(c.WipBranchQualifierSeparator))
	say.Say("MOB_WIP_BRANCH_QUALIFIER" + "=" + quote(c.WipBranchQualifier))
	say.Say("MOB_WIP_COMMIT_MESSAGE" + "=" + quote(c.WipCommitMessage))
}

func ReadConfiguration(gitRootDir string) Configuration {
	configuration := GetDefaultConfiguration()
	configuration = parseEnvironmentVariables(configuration)

	userHomeDir, _ := os.UserHomeDir()
	userConfigurationPath := userHomeDir + "/.mob"
	configuration = parseUserConfiguration(configuration, userConfigurationPath)
	if gitRootDir != "" {
		configuration = parseProjectConfiguration(configuration, gitRootDir+"/.mob")
	}
	return configuration
}

func ParseArgs(args []string, configuration Configuration) (command string, parameters []string, newConfiguration Configuration) {
	newConfiguration = configuration

	for i := 1; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--discard-uncommitted-changes", "-d":
			newConfiguration.HandleUncommittedChanges = DiscardChanges
		case "--include-uncommitted-changes", "-i":
			newConfiguration.HandleUncommittedChanges = IncludeChanges
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
		case "--create":
			newConfiguration.StartCreate = true
		case "--join", "-j":
			newConfiguration.StartJoin = true
		case "--room":
			if i+1 != len(args) {
				newConfiguration.TimerRoom = args[i+1]
			}
			i++ // skip consumed parameter

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

func GetDefaultConfiguration() Configuration {
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
		CliName:                     "mob",
		RemoteName:                  "origin",
		WipCommitMessage:            "mob next [ci-skip] [ci skip] [skip ci]",
		StartCommitMessage:          "mob start [ci-skip] [ci skip] [skip ci]",
		SkipCiPushOptionEnabled:     true,
		GitHooksEnabled:             false,
		VoiceCommand:                voiceCommand,
		VoiceMessage:                "mob next",
		NotifyCommand:               notifyCommand,
		NotifyMessage:               "mob next",
		NextStay:                    true,
		RequireCommitMessage:        false,
		HandleUncommittedChanges:    FailWithError,
		StartCreate:                 false,
		WipBranchQualifier:          "",
		WipBranchQualifierSeparator: "-",
		DoneSquash:                  Squash,
		OpenCommand:                 "",
		Timer:                       "",
		TimerLocal:                  true,
		TimerRoom:                   "",
		TimerUser:                   "",
		TimerUrl:                    "https://timer.mob.sh/",
		WipBranchPrefix:             "mob/",
		StashName:                   "mob-stash-name",
		ResetDeleteRemoteWipBranch:  false,
	}
}

func parseUserConfiguration(configuration Configuration, path string) Configuration {
	file, err := os.Open(path)

	if err != nil {
		say.Debug("No user configuration file found. (" + path + ") Error: " + err.Error())
		return configuration
	} else {
		say.Debug("Found user configuration file at " + path)
	}

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		say.Debug(line)
		if !strings.Contains(line, "=") {
			say.Debug("Skip line because line contains no =. Line=" + line)
			continue
		}
		key := line[0:strings.Index(line, "=")]
		value := strings.TrimPrefix(line, key+"=")
		say.Debug("Key is " + key)
		say.Debug("Value is " + value)
		switch key {
		case "MOB_CLI_NAME":
			setUnquotedString(&configuration.CliName, key, value)
		case "MOB_REMOTE_NAME":
			setUnquotedString(&configuration.RemoteName, key, value)
		case "MOB_WIP_COMMIT_MESSAGE":
			setUnquotedString(&configuration.WipCommitMessage, key, value)
		case "MOB_START_COMMIT_MESSAGE":
			setUnquotedString(&configuration.StartCommitMessage, key, value)
		case "MOB_SKIP_CI_PUSH_OPTION_ENABLED":
			setBoolean(&configuration.SkipCiPushOptionEnabled, key, value)
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
		case "MOB_START_CREATE":
			setBoolean(&configuration.StartCreate, key, value)
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
		case "MOB_RESET_DELETE_REMOTE_WIP_BRANCH":
			setBoolean(&configuration.ResetDeleteRemoteWipBranch, key, value)

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		say.Warning("User configuration file exists, but could not be read. (" + path + ")")
	}

	return configuration
}

func parseProjectConfiguration(configuration Configuration, path string) Configuration {
	file, err := os.Open(path)

	if err != nil {
		say.Debug("No project configuration file found. (" + path + ") Error: " + err.Error())
		return configuration
	} else {
		say.Debug("Found project configuration file at " + path)
	}

	fileScanner := bufio.NewScanner(file)

	for fileScanner.Scan() {
		line := strings.TrimSpace(fileScanner.Text())
		say.Debug(line)
		if !strings.Contains(line, "=") {
			say.Debug("Skip line because line contains no =. Line=" + line)
			continue
		}
		key := line[0:strings.Index(line, "=")]
		value := strings.TrimPrefix(line, key+"=")
		say.Debug("Key is " + key)
		say.Debug("Value is " + value)
		switch key {
		case "MOB_VOICE_COMMAND", "MOB_VOICE_MESSAGE", "MOB_NOTIFY_COMMAND", "MOB_NOTIFY_MESSAGE", "MOB_OPEN_COMMAND":
			say.Warning("Skipped overwriting key " + key + " from project/.mob file out of security reasons!")
		case "MOB_CLI_NAME":
			setUnquotedString(&configuration.CliName, key, value)
		case "MOB_REMOTE_NAME":
			setUnquotedString(&configuration.RemoteName, key, value)
		case "MOB_WIP_COMMIT_MESSAGE":
			setUnquotedString(&configuration.WipCommitMessage, key, value)
		case "MOB_START_COMMIT_MESSAGE":
			setUnquotedString(&configuration.StartCommitMessage, key, value)
		case "MOB_SKIP_CI_PUSH_OPTION_ENABLED":
			setBoolean(&configuration.SkipCiPushOptionEnabled, key, value)
		case "MOB_GIT_HOOKS_ENABLED":
			setBoolean(&configuration.GitHooksEnabled, key, value)
		case "MOB_REQUIRE_COMMIT_MESSAGE":
			setBoolean(&configuration.RequireCommitMessage, key, value)
		case "MOB_NEXT_STAY":
			setBoolean(&configuration.NextStay, key, value)
		case "MOB_START_CREATE":
			setBoolean(&configuration.StartCreate, key, value)
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
		case "MOB_RESET_DELETE_REMOTE_WIP_BRANCH":
			setBoolean(&configuration.ResetDeleteRemoteWipBranch, key, value)

		default:
			continue
		}
	}

	if err := fileScanner.Err(); err != nil {
		say.Warning("Project configuration file exists, but could not be read. (" + path + ")")
	}

	return configuration
}

func setUnquotedString(s *string, key string, value string) {
	unquotedValue, err := strconv.Unquote(value)
	if err != nil {
		say.Warning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
		return
	}
	*s = unquotedValue
	say.Debug("Overwriting " + key + " =" + unquotedValue)
}

func setBoolean(s *bool, key string, value string) {
	boolValue, err := strconv.ParseBool(value)
	if err != nil {
		say.Warning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
		return
	}
	*s = boolValue
	say.Debug("Overwriting " + key + " =" + strconv.FormatBool(boolValue))
}

func setMobDoneSquash(configuration *Configuration, key string, value string) {
	if strings.HasPrefix(value, "\"") {
		unquotedValue, err := strconv.Unquote(value)
		if err != nil {
			say.Warning("Could not set key from configuration file because value is not parseable (" + key + "=" + value + ")")
			return
		}
		value = unquotedValue
	}
	configuration.DoneSquash = doneSquash(value)
	say.Debug("Overwriting " + key + " =" + configuration.DoneSquash)
}

func parseEnvironmentVariables(configuration Configuration) Configuration {
	setStringFromEnvVariable(&configuration.CliName, "MOB_CLI_NAME")
	if configuration.CliName != GetDefaultConfiguration().CliName {
		configuration.WipCommitMessage = configuration.CliName + " next [ci-skip] [ci skip] [skip ci]"
		configuration.VoiceMessage = configuration.CliName + " next"
		configuration.NotifyMessage = configuration.CliName + " next"
	}

	removed("MOB_BASE_BRANCH", "Use '"+configuration.Mob("start")+"' on your base branch instead.")
	removed("MOB_WIP_BRANCH", "Use '"+configuration.Mob("start --branch <branch>")+"' instead.")
	removed("MOB_START_INCLUDE_UNCOMMITTED_CHANGES", "Use the parameter --include-uncommitted-changes instead.")
	experimental("MOB_WIP_BRANCH_PREFIX")
	deprecated("MOB_START_COMMIT_MESSAGE", "Please check that everybody you work with uses version 5.0.0 or higher. Then this environment variable can be unset, as it will not have an impact anymore.")

	setStringFromEnvVariable(&configuration.RemoteName, "MOB_REMOTE_NAME")
	setStringFromEnvVariable(&configuration.WipCommitMessage, "MOB_WIP_COMMIT_MESSAGE")
	setStringFromEnvVariable(&configuration.StartCommitMessage, "MOB_START_COMMIT_MESSAGE")
	setBoolFromEnvVariable(&configuration.SkipCiPushOptionEnabled, "MOB_SKIP_CI_PUSH_OPTION_ENABLED")
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

	setBoolFromEnvVariable(&configuration.StartCreate, "MOB_START_CREATE")

	setDoneSquashFromEnvVariable(&configuration, "MOB_DONE_SQUASH")

	setStringFromEnvVariable(&configuration.OpenCommand, "MOB_OPEN_COMMAND")

	setStringFromEnvVariable(&configuration.Timer, "MOB_TIMER")
	setStringFromEnvVariable(&configuration.TimerRoom, "MOB_TIMER_ROOM")
	setBoolFromEnvVariable(&configuration.TimerRoomUseWipBranchQualifier, "MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER")
	setBoolFromEnvVariable(&configuration.TimerLocal, "MOB_TIMER_LOCAL")
	setStringFromEnvVariable(&configuration.TimerUser, "MOB_TIMER_USER")
	setStringFromEnvVariable(&configuration.TimerUrl, "MOB_TIMER_URL")
	setBoolFromEnvVariable(&configuration.TimerInsecure, "MOB_TIMER_INSECURE")

	setBoolFromEnvVariable(&configuration.ResetDeleteRemoteWipBranch, "MOB_RESET_DELETE_REMOTE_WIP_BRANCH")

	return configuration
}

func setStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set && value != "" {
		*s = value
		say.Debug("overriding " + key + "=" + *s)
	}
}

func setOptionalStringFromEnvVariable(s *string, key string) {
	value, set := os.LookupEnv(key)
	if set {
		*s = value
		say.Debug("overriding " + key + "=" + *s)
	}
}

func setBoolFromEnvVariable(s *bool, key string) {
	value, set := os.LookupEnv(key)
	if !set {
		return
	}
	if value == "" {
		say.Debug("ignoring " + key + "=" + value + " (empty string)")
	}

	if value == "true" {
		*s = true
		say.Debug("overriding " + key + "=" + strconv.FormatBool(*s))
	} else if value == "false" {
		*s = false
		say.Debug("overriding " + key + "=" + strconv.FormatBool(*s))
	} else {
		say.Warning("ignoring " + key + "=" + value + " (not a boolean)")
	}
}

func setDoneSquashFromEnvVariable(configuration *Configuration, key string) {
	value, set := os.LookupEnv(key)
	if !set {
		return
	}

	configuration.DoneSquash = doneSquash(value)

	if value == "" {
		say.Debug("ignoring " + key + "=" + value + " (empty string)")
		return
	}

	say.Debug("overriding " + key + "=" + configuration.DoneSquash)
}

func removed(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say.Say("Configuration option '" + key + "' is no longer used.")
		say.Say(message)
	}
}

func deprecated(key string, message string) {
	if _, set := os.LookupEnv(key); set {
		say.Say("Configuration option '" + key + "' is deprecated.")
		say.Say(message)
	}
}

func experimental(key string) {
	if _, set := os.LookupEnv(key); set {
		say.Say("Configuration option '" + key + "' is experimental. Be prepared that this option will be removed!")
	}
}

func doneSquash(value string) string {
	switch value {
	case NoSquash:
		return NoSquash
	case SquashWip:
		return SquashWip
	default:
		return Squash
	}
}

func quote(value string) string {
	return strconv.Quote(value)
}
