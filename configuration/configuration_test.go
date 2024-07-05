package configuration

import (
	"fmt"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/test"
	"os"
	"strings"
	"testing"
)

var (
	tempDir string
)

func TestQuote(t *testing.T) {
	test.Equals(t, "\"mob\"", quote("mob"))
	test.Equals(t, "\"m\\\"ob\"", quote("m\"ob"))
}

func TestParseArgs(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := ParseArgs([]string{"mob", "start", "--branch", "green"}, configuration)

	test.Equals(t, "start", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, "green", configuration.WipBranchQualifier)
}

func TestParseArgsStartCreate(t *testing.T) {
	configuration := GetDefaultConfiguration()

	command, parameters, configuration := ParseArgs([]string{"mob", "start", "--create"}, configuration)

	test.Equals(t, "start", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, true, configuration.StartCreate)
}

func TestParseArgsDoneNoSquash(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, Squash, configuration.DoneSquash)

	command, parameters, configuration := ParseArgs([]string{"mob", "done", "--no-squash"}, configuration)

	test.Equals(t, "done", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, NoSquash, configuration.DoneSquash)
}

func TestParseArgsDoneSquash(t *testing.T) {
	configuration := GetDefaultConfiguration()
	configuration.DoneSquash = NoSquash

	command, parameters, configuration := ParseArgs([]string{"mob", "done", "--squash"}, configuration)

	test.Equals(t, "done", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, Squash, configuration.DoneSquash)
}

func TestParseArgsMessage(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := ParseArgs([]string{"mob", "next", "--message", "ci-skip"}, configuration)

	test.Equals(t, "next", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, "ci-skip", configuration.WipCommitMessage)
}

func TestParseArgsStartRoom(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := ParseArgs([]string{"mob", "start", "--room", "testroom"}, configuration)

	test.Equals(t, "start", command)
	test.Equals(t, "", strings.Join(parameters, ""))
	test.Equals(t, "testroom", configuration.TimerRoom)
}

func TestParseArgsTimerRoom(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := ParseArgs([]string{"mob", "timer", "10", "--room", "testroom"}, configuration)

	test.Equals(t, "timer", command)
	test.Equals(t, "10", strings.Join(parameters, ""))
	test.Equals(t, "testroom", configuration.TimerRoom)
}

func TestParseArgsTimerOpenRoom(t *testing.T) {
	configuration := GetDefaultConfiguration()
	test.Equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := ParseArgs([]string{"mob", "timer", "open", "--room", "testroom"}, configuration)

	test.Equals(t, "timer", command)
	test.Equals(t, "open", strings.Join(parameters, ""))
	test.Equals(t, "testroom", configuration.TimerRoom)
}

func TestMobRemoteNameEnvironmentVariable(t *testing.T) {
	configuration := setEnvVarAndParse("MOB_REMOTE_NAME", "GITHUB")
	test.Equals(t, "GITHUB", configuration.RemoteName)
}

func TestMobRemoteNameEnvironmentVariableEmptyString(t *testing.T) {
	configuration := setEnvVarAndParse("MOB_REMOTE_NAME", "")

	test.Equals(t, "origin", configuration.RemoteName)
}

func TestMobDoneSquashEnvironmentVariable(t *testing.T) {
	assertMobDoneSquashValue(t, "", Squash)
	assertMobDoneSquashValue(t, "garbage", Squash)
	assertMobDoneSquashValue(t, "squash", Squash)
	assertMobDoneSquashValue(t, "no-squash", NoSquash)
	assertMobDoneSquashValue(t, "squash-wip", SquashWip)
}

func assertMobDoneSquashValue(t *testing.T, value string, expected string) {
	configuration := setEnvVarAndParse("MOB_DONE_SQUASH", value)
	test.Equals(t, expected, configuration.DoneSquash)
}

func TestBooleanEnvironmentVariables(t *testing.T) {
	assertBoolEnvVarParsed(t, "MOB_START_CREATE", false, Configuration.GetMobStartCreateRemoteBranch)
	assertBoolEnvVarParsed(t, "MOB_NEXT_STAY", true, Configuration.GetMobNextStay)
	assertBoolEnvVarParsed(t, "MOB_REQUIRE_COMMIT_MESSAGE", false, Configuration.GetRequireCommitMessage)
}

func assertBoolEnvVarParsed(t *testing.T, envVar string, defaultValue bool, actual func(Configuration) bool) {
	t.Run(envVar, func(t *testing.T) {
		assertEnvVarParsed(t, envVar, "", defaultValue, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "true", true, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "false", false, boolToInterface(actual))
		assertEnvVarParsed(t, envVar, "garbage", defaultValue, boolToInterface(actual))
	})
}

func assertEnvVarParsed(t *testing.T, variable string, value string, expected interface{}, actual func(Configuration) interface{}) {
	t.Run(fmt.Sprintf("%s=\"%s\"->(expects:%t)", variable, value, expected), func(t *testing.T) {
		configuration := setEnvVarAndParse(variable, value)
		test.Equals(t, expected, actual(configuration))
	})
}

func setEnvVarAndParse(variable string, value string) Configuration {
	os.Setenv(variable, value)
	defer os.Unsetenv(variable)

	return parseEnvironmentVariables(GetDefaultConfiguration())
}

func boolToInterface(actual func(Configuration) bool) func(c Configuration) interface{} {
	return func(c Configuration) interface{} {
		return actual(c)
	}
}

func (c Configuration) GetMobDoneSquash() string {
	return c.DoneSquash
}

func (c Configuration) GetMobStartIncludeUncommittedChanges() bool {
	return c.StartIncludeUncommittedChanges
}

func (c Configuration) GetMobStartCreateRemoteBranch() bool {
	return c.StartCreate
}

func (c Configuration) GetMobNextStay() bool {
	return c.NextStay
}

func (c Configuration) GetRequireCommitMessage() bool {
	return c.RequireCommitMessage
}

func TestParseRequireCommitMessageEnvVariables(t *testing.T) {
	os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")
	defer os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")

	configuration := parseEnvironmentVariables(GetDefaultConfiguration())
	test.Equals(t, false, configuration.RequireCommitMessage)

	os.Setenv("MOB_REQUIRE_COMMIT_MESSAGE", "false")
	configuration = parseEnvironmentVariables(GetDefaultConfiguration())
	test.Equals(t, false, configuration.RequireCommitMessage)

	os.Setenv("MOB_REQUIRE_COMMIT_MESSAGE", "true")
	configuration = parseEnvironmentVariables(GetDefaultConfiguration())
	test.Equals(t, true, configuration.RequireCommitMessage)
}

func TestReadUserConfigurationFromFileOverrideEverything(t *testing.T) {
	tempDir = t.TempDir()
	test.SetWorkingDir(tempDir)

	test.CreateFile(t, ".mob", `
		MOB_CLI_NAME="team"
		MOB_REMOTE_NAME="gitlab"
		MOB_WIP_COMMIT_MESSAGE="team next"
		MOB_START_COMMIT_MESSAGE="mob: start"
		MOB_SKIP_CI_PUSH_OPTION_ENABLED=false
		MOB_REQUIRE_COMMIT_MESSAGE=true
		MOB_VOICE_COMMAND="whisper \"%s\""
		MOB_VOICE_MESSAGE="team next"
		MOB_NOTIFY_COMMAND="/usr/bin/osascript -e 'display notification \"%s!!!\"'"
		MOB_NOTIFY_MESSAGE="team next"
		MOB_NEXT_STAY=false
		MOB_START_CREATE=true
		MOB_WIP_BRANCH_QUALIFIER="green"
		MOB_WIP_BRANCH_QUALIFIER_SEPARATOR="---"
		MOB_WIP_BRANCH_PREFIX="ensemble/"
		MOB_DONE_SQUASH=no-squash
		MOB_OPEN_COMMAND="idea %s"
		MOB_TIMER="123"
		MOB_TIMER_ROOM="Room_42"
		MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER=true
		MOB_TIMER_LOCAL=false
		MOB_TIMER_USER="Mona"
		MOB_TIMER_URL="https://timer.innoq.io/"
		MOB_STASH_NAME="team-stash-name"
	`)
	actualConfiguration := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, "team", actualConfiguration.CliName)
	test.Equals(t, "gitlab", actualConfiguration.RemoteName)
	test.Equals(t, "team next", actualConfiguration.WipCommitMessage)
	test.Equals(t, "mob: start", actualConfiguration.StartCommitMessage)
	test.Equals(t, false, actualConfiguration.SkipCiPushOptionEnabled)
	test.Equals(t, true, actualConfiguration.RequireCommitMessage)
	test.Equals(t, "whisper \"%s\"", actualConfiguration.VoiceCommand)
	test.Equals(t, "team next", actualConfiguration.VoiceMessage)
	test.Equals(t, "/usr/bin/osascript -e 'display notification \"%s!!!\"'", actualConfiguration.NotifyCommand)
	test.Equals(t, "team next", actualConfiguration.NotifyMessage)
	test.Equals(t, false, actualConfiguration.NextStay)
	test.Equals(t, true, actualConfiguration.StartCreate)
	test.Equals(t, "green", actualConfiguration.WipBranchQualifier)
	test.Equals(t, "---", actualConfiguration.WipBranchQualifierSeparator)
	test.Equals(t, "ensemble/", actualConfiguration.WipBranchPrefix)
	test.Equals(t, NoSquash, actualConfiguration.DoneSquash)
	test.Equals(t, "idea %s", actualConfiguration.OpenCommand)
	test.Equals(t, "123", actualConfiguration.Timer)
	test.Equals(t, "Room_42", actualConfiguration.TimerRoom)
	test.Equals(t, true, actualConfiguration.TimerRoomUseWipBranchQualifier)
	test.Equals(t, false, actualConfiguration.TimerLocal)
	test.Equals(t, "Mona", actualConfiguration.TimerUser)
	test.Equals(t, "https://timer.innoq.io/", actualConfiguration.TimerUrl)
	test.Equals(t, "team-stash-name", actualConfiguration.StashName)

	test.CreateFile(t, ".mob", "\nMOB_TIMER_ROOM=\"Room\\\"\\\"_42\"\n")
	actualConfiguration1 := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, "Room\"\"_42", actualConfiguration1.TimerRoom)
}

func TestReadProjectConfigurationFromFileOverrideEverything(t *testing.T) {
	output := test.CaptureOutput(t)
	tempDir = t.TempDir()
	test.SetWorkingDir(tempDir)

	test.CreateFile(t, ".mob", `
		MOB_CLI_NAME="team"
		MOB_REMOTE_NAME="gitlab"
		MOB_WIP_COMMIT_MESSAGE="team next"
		MOB_START_COMMIT_MESSAGE="mob: start"
		MOB_SKIP_CI_PUSH_OPTION_ENABLED=false
		MOB_REQUIRE_COMMIT_MESSAGE=true
		MOB_VOICE_COMMAND="whisper \"%s\""
		MOB_VOICE_MESSAGE="team next"
		MOB_NOTIFY_COMMAND="/usr/bin/osascript -e 'display notification \"%s!!!\"'"
		MOB_NOTIFY_MESSAGE="team next"
		MOB_NEXT_STAY=false
		MOB_START_CREATE=true
		MOB_WIP_BRANCH_QUALIFIER="green"
		MOB_WIP_BRANCH_QUALIFIER_SEPARATOR="---"
		MOB_WIP_BRANCH_PREFIX="ensemble/"
		MOB_DONE_SQUASH=no-squash
		MOB_OPEN_COMMAND="idea %s"
		MOB_TIMER="123"
		MOB_TIMER_ROOM="Room_42"
		MOB_TIMER_ROOM_USE_WIP_BRANCH_QUALIFIER=true
		MOB_TIMER_LOCAL=false
		MOB_TIMER_USER="Mona"
		MOB_TIMER_URL="https://timer.innoq.io/"
		MOB_STASH_NAME="team-stash-name"
	`)
	actualConfiguration := parseProjectConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, "team", actualConfiguration.CliName)
	test.Equals(t, "gitlab", actualConfiguration.RemoteName)
	test.Equals(t, "team next", actualConfiguration.WipCommitMessage)
	test.Equals(t, "mob: start", actualConfiguration.StartCommitMessage)
	test.Equals(t, false, actualConfiguration.SkipCiPushOptionEnabled)
	test.Equals(t, true, actualConfiguration.RequireCommitMessage)
	test.NotEquals(t, "whisper \"%s\"", actualConfiguration.VoiceCommand)
	test.NotEquals(t, "team next", actualConfiguration.VoiceMessage)
	test.NotEquals(t, "/usr/bin/osascript -e 'display notification \"%s!!!\"'", actualConfiguration.NotifyCommand)
	test.NotEquals(t, "team next", actualConfiguration.NotifyMessage)
	test.Equals(t, false, actualConfiguration.NextStay)
	test.Equals(t, true, actualConfiguration.StartCreate)
	test.Equals(t, "green", actualConfiguration.WipBranchQualifier)
	test.Equals(t, "---", actualConfiguration.WipBranchQualifierSeparator)
	test.Equals(t, "ensemble/", actualConfiguration.WipBranchPrefix)
	test.Equals(t, NoSquash, actualConfiguration.DoneSquash)
	test.NotEquals(t, "idea %s", actualConfiguration.OpenCommand)
	test.Equals(t, "123", actualConfiguration.Timer)
	test.Equals(t, "Room_42", actualConfiguration.TimerRoom)
	test.Equals(t, true, actualConfiguration.TimerRoomUseWipBranchQualifier)
	test.Equals(t, false, actualConfiguration.TimerLocal)
	test.Equals(t, "Mona", actualConfiguration.TimerUser)
	test.Equals(t, "https://timer.innoq.io/", actualConfiguration.TimerUrl)
	test.Equals(t, "team-stash-name", actualConfiguration.StashName)

	test.CreateFile(t, ".mob", "\nMOB_TIMER_ROOM=\"Room\\\"\\\"_42\"\n")
	actualConfiguration1 := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, "Room\"\"_42", actualConfiguration1.TimerRoom)
	test.AssertOutputContains(t, output, "Skipped overwriting key MOB_VOICE_COMMAND from project/.mob file out of security reasons!")
	test.AssertOutputContains(t, output, "Skipped overwriting key MOB_VOICE_MESSAGE from project/.mob file out of security reasons!")
	test.AssertOutputContains(t, output, "Skipped overwriting key MOB_NOTIFY_COMMAND from project/.mob file out of security reasons!")
	test.AssertOutputContains(t, output, "Skipped overwriting key MOB_NOTIFY_MESSAGE from project/.mob file out of security reasons!")
	test.AssertOutputContains(t, output, "Skipped overwriting key MOB_OPEN_COMMAND from project/.mob file out of security reasons!")
}

func TestReadConfigurationFromFileWithNonBooleanQuotedDoneSquashValue(t *testing.T) {
	say.TurnOnDebugging()
	tempDir = t.TempDir()
	test.SetWorkingDir(tempDir)

	test.CreateFile(t, ".mob", "\nMOB_DONE_SQUASH=\"squash-wip\"")
	actualConfiguration := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, SquashWip, actualConfiguration.DoneSquash)
}

func TestReadConfigurationFromFileAndSkipBrokenLines(t *testing.T) {
	say.TurnOnDebugging()
	tempDir = t.TempDir()
	test.SetWorkingDir(tempDir)

	test.CreateFile(t, ".mob", "\nMOB_TIMER_ROOM=\"Broken\" \"String\"")
	actualConfiguration := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, GetDefaultConfiguration().TimerRoom, actualConfiguration.TimerRoom)
}

func TestSkipIfConfigurationDoesNotExist(t *testing.T) {
	say.TurnOnDebugging()
	tempDir = t.TempDir()
	test.SetWorkingDir(tempDir)

	actualConfiguration := parseUserConfiguration(GetDefaultConfiguration(), tempDir+"/.mob")
	test.Equals(t, GetDefaultConfiguration(), actualConfiguration)
}

func TestSetMobDoneSquash(t *testing.T) {
	configuration := GetDefaultConfiguration()
	configuration.DoneSquash = Squash

	setMobDoneSquash(&configuration, "", "no-squash")
	test.Equals(t, NoSquash, configuration.DoneSquash)

	setMobDoneSquash(&configuration, "", "squash")
	test.Equals(t, Squash, configuration.DoneSquash)

	setMobDoneSquash(&configuration, "", "squash-wip")
	test.Equals(t, SquashWip, configuration.DoneSquash)
}

func TestSetMobDoneSquashGarbageValue(t *testing.T) {
	configuration := GetDefaultConfiguration()
	configuration.DoneSquash = NoSquash

	setMobDoneSquash(&configuration, "", "garbage")
	test.Equals(t, Squash, configuration.DoneSquash)
}

func TestSetMobDoneSquashEmptyStringValue(t *testing.T) {
	configuration := GetDefaultConfiguration()
	configuration.DoneSquash = NoSquash

	setMobDoneSquash(&configuration, "", "")
	test.Equals(t, Squash, configuration.DoneSquash)
}
