package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"runtime/debug"
	"strconv"
	"strings"
	"testing"
)

var (
	configuration Configuration
)

func TestParseArgs(t *testing.T) {
	configuration = getDefaultConfiguration()
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := parseArgs([]string{"mob", "start", "--branch", "green"}, configuration)

	equals(t, "start", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "green", configuration.WipBranchQualifier)
}

func TestParseArgsDoneNoSquash(t *testing.T) {
	configuration = getDefaultConfiguration()
	equals(t, true, configuration.MobDoneSquash)

	command, parameters, configuration := parseArgs([]string{"mob", "done", "--no-squash"}, configuration)

	equals(t, "done", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, false, configuration.MobDoneSquash)
}

func TestParseArgsDoneSquash(t *testing.T) {
	configuration = getDefaultConfiguration()
	configuration.MobDoneSquash = false

	command, parameters, configuration := parseArgs([]string{"mob", "done", "--squash"}, configuration)

	equals(t, "done", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, true, configuration.MobDoneSquash)
}

func TestParseArgsMessage(t *testing.T) {
	configuration = getDefaultConfiguration()
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := parseArgs([]string{"mob", "next", "--message", "ci-skip"}, configuration)

	equals(t, "next", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "ci-skip", configuration.WipCommitMessage)
}

func TestDetermineBranches(t *testing.T) {
	configuration = getDefaultConfiguration()
	configuration.WipBranchQualifierSeparator = "-"

	assertDetermineBranches(t, "master", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "", []string{}, "master", "mob-session")
	assertDetermineBranches(t, "mob-session", "green", []string{}, "master", "mob-session")

	assertDetermineBranches(t, "master", "green", []string{}, "master", "mob/master-green")
	assertDetermineBranches(t, "mob/master-green", "", []string{}, "master", "mob/master-green")

	assertDetermineBranches(t, "master", "test-branch", []string{}, "master", "mob/master-test-branch")
	assertDetermineBranches(t, "mob/master-test-branch", "", []string{}, "master", "mob/master-test-branch")

	assertDetermineBranches(t, "feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1", "", []string{}, "feature1", "mob/feature1")
	assertDetermineBranches(t, "mob/feature1-green", "", []string{}, "feature1", "mob/feature1-green")
	assertDetermineBranches(t, "feature1", "green", []string{}, "feature1", "mob/feature1-green")

	assertDetermineBranches(t, "feature/test", "", []string{"feature/test"}, "feature/test", "mob/feature/test")
	assertDetermineBranches(t, "mob/feature/test", "", []string{"feature/test", "mob/feature/test"}, "feature/test", "mob/feature/test")

	assertDetermineBranches(t, "feature/test-ch", "", []string{"DPL-2638-update-apis", "DPL-2814-create-project", "feature/test-ch", "fix/smallChanges", "master", "pipeship/pipelineupdate-pipeship-pipeline.yaml"}, "feature/test-ch", "mob/feature/test-ch")
}

func assertDetermineBranches(t *testing.T, branch string, qualifier string, branches []string, expectedBase string, expectedWip string) {
	configuration := getDefaultConfiguration()
	configuration.WipBranchQualifier = qualifier
	baseBranch, wipBranch := determineBranches(branch, branches, configuration)
	equals(t, expectedBase, baseBranch)
	equals(t, expectedWip, wipBranch)
}

func TestRemoveWipPrefix(t *testing.T) {
	equals(t, "master-green", removeWipPrefix("mob/master-green"))
	equals(t, "master-green-blue", removeWipPrefix("mob/master-green-blue"))
	equals(t, "main-branch", removeWipPrefix("mob/main-branch"))
}

func TestRemoveWipBranchQualifier(t *testing.T) {
	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "green"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", removeWipQualifier("master-green", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "test-branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", removeWipQualifier("master-test-branch", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master-test", removeWipQualifier("master-test-branch", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master-test", removeWipQualifier("master-test-branch", []string{"master-test"}, configuration))

	configuration.WipBranchQualifierSeparator = "/-/"
	configuration.WipBranchQualifier = "branch-qualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", removeWipQualifier("main/-/branch-qualifier", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main/branchqualifier", removeWipQualifier("main/branchqualifier", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = ""
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", removeWipQualifier("mainbranchqualifier", []string{}, configuration))
}

func TestRemoveWipBranchQualifierWithoutBranchQualifierSet(t *testing.T) {
	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	equals(t, "main", removeWipQualifier("main", []string{}, configuration))

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	equals(t, "master", removeWipQualifier("master-test-branch", []string{}, configuration))
}

func TestMobRemoteNameEnvironmentVariable(t *testing.T) {
	configuration = setEnvVarAndParse("MOB_REMOTE_NAME", "GITHUB")

	equals(t, "GITHUB", configuration.RemoteName)
}

func TestMobRemoteNameEnvironmentVariableEmptyString(t *testing.T) {
	configuration = setEnvVarAndParse("MOB_REMOTE_NAME", "")

	equals(t, "origin", configuration.RemoteName)
}

func TestBooleanEnvironmentVariables(t *testing.T) {
	assertBoolEnvVarParsed(t, "MOB_DONE_SQUASH", true, Configuration.GetMobDoneSquash)
	assertBoolEnvVarParsed(t, "MOB_START_INCLUDE_UNCOMMITTED_CHANGES", false, Configuration.GetMobStartIncludeUncommittedChanges)
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
		configuration = setEnvVarAndParse(variable, value)
		equals(t, expected, actual(configuration))
	})
}

func setEnvVarAndParse(variable string, value string) Configuration {
	os.Setenv(variable, value)
	defer os.Unsetenv(variable)

	return parseEnvironmentVariables(getDefaultConfiguration())
}

func boolToInterface(actual func(Configuration) bool) func(c Configuration) interface{} {
	return func(c Configuration) interface{} {
		return actual(c)
	}
}

func (c Configuration) GetMobDoneSquash() bool {
	return c.MobDoneSquash
}

func (c Configuration) GetMobStartIncludeUncommittedChanges() bool {
	return c.MobStartIncludeUncommittedChanges
}

func (c Configuration) GetMobNextStay() bool {
	return c.MobNextStay
}

func (c Configuration) GetRequireCommitMessage() bool {
	return c.RequireCommitMessage
}

func TestVersion(t *testing.T) {
	output := setup(t)

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestStatusNotMobProgramming(t *testing.T) {
	output := setup(t)

	status(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestNextNotMobProgramming(t *testing.T) {
	output := setup(t)

	next(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestRequireCommitMessage(t *testing.T) {
	output := setup(t)

	os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")
	defer os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")

	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	equals(t, false, configuration.RequireCommitMessage)

	os.Setenv("MOB_REQUIRE_COMMIT_MESSAGE", "false")
	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	equals(t, false, configuration.RequireCommitMessage)

	os.Setenv("MOB_REQUIRE_COMMIT_MESSAGE", "true")
	configuration = parseEnvironmentVariables(getDefaultConfiguration())
	equals(t, true, configuration.RequireCommitMessage)

	start(configuration)

	next(configuration)
	// ensure we don't complain if there's nothing to commit
	// https://github.com/remotemobprogramming/mob/pull/107#issuecomment-761298861
	assertOutputContains(t, output, "nothing to commit")

	createFile(t, "example.txt", "content")
	next(configuration)
	// failure message should make sense regardless of whether we
	// provided commit message via `-m` or MOB_WIP_COMMIT_MESSAGE
	// https://github.com/remotemobprogramming/mob/pull/107#issuecomment-761591039
	assertOutputContains(t, output, "commit message required")
}

func TestDoneNotMobProgramming(t *testing.T) {
	output := setup(t)

	done(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestStatusMobProgramming(t *testing.T) {
	output := setup(t)
	start(configuration)

	status(configuration)

	assertOutputContains(t, output, "you are mob programming")
}

func TestStatusWithMoreThan5LinesOfLog(t *testing.T) {
	output := setup(t)
	configuration.MobNextStay = true
	start(configuration)

	for i := 0; i < 6; i++ {
		createFile(t, "test"+strconv.Itoa(i)+".txt", "test")
		next(configuration)
	}

	status(configuration)
	assertOutputContains(t, output, "This mob branch contains 6 commits.")
}

func TestExecuteKicksOffStatus(t *testing.T) {
	output := setup(t)

	execute("status", []string{}, getDefaultConfiguration())

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestExecuteInvalidCommandKicksOffHelp(t *testing.T) {
	output := setup(t)

	execute("whatever", []string{}, getDefaultConfiguration())

	assertOutputContains(t, output, "Basic Commands:")
}

func TestStart(t *testing.T) {
	setup(t)

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, "mob-session")
}

func TestStartWithMultipleExistingBranches(t *testing.T) {
	output := setup(t)

	configuration.WipBranchQualifier = "green"
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "master")
	assertOutputContains(t, output, "qualified mob branches detected")
}

func TestStartWithMultipleExistingBranchesAndEmptyWipBranchQualifier(t *testing.T) {
	output := setup(t)

	configuration.WipBranchQualifier = "green"
	start(configuration)
	next(configuration)

	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = true
	start(configuration)
	assertOnBranch(t, "mob-session")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartWithMultipleExistingBranchesWithStay(t *testing.T) {
	output := setup(t)
	configuration.MobNextStay = true

	configuration.WipBranchQualifier = "green"
	assertOnBranch(t, "master")
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	next(configuration)
	assertOnBranch(t, "mob/master-green")

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartNextWithBranch(t *testing.T) {
	setup(t)
	assertOnBranch(t, "master")
	configuration.WipBranchQualifier = "green"

	start(configuration)
	assertOnBranch(t, "mob/master-green")
	assertMobSessionBranches(t, "mob/master-green")
	configuration.WipBranchQualifier = ""

	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = "green"
	reset(configuration)
	assertNoMobSessionBranches(t, "mob/master-green")
}

func TestStartNextStartWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	configuration.MobNextStay = true
	assertOnBranch(t, "master")

	start(configuration)
	assertOnBranch(t, "mob/master-green")

	next(configuration)
	assertOnBranch(t, "mob/master-green")

	start(configuration)
	assertOnBranch(t, "mob/master-green")
}

func TestStartNextOnFeatureWithBranch(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "green"
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")

	start(configuration)
	assertOnBranch(t, "mob/feature1-green")

	next(configuration)
	assertOnBranch(t, "feature1")
}

// reproduces #117
func TestStartNextWithBranchContainingHyphen(t *testing.T) {
	setup(t)
	configuration.WipBranchQualifier = "test-branch"
	configuration.WipBranchQualifierSet = true
	start(configuration)
	assertOnBranch(t, "mob/master-test-branch")
	assertMobSessionBranches(t, "mob/master-test-branch")

	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	next(configuration)
}

func TestReset(t *testing.T) {
	setup(t)

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestResetCommit(t *testing.T) {
	setup(t)
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)
	assertMobSessionBranches(t, "mob-session")

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartUnstagedChanges(t *testing.T) {
	output := setup(t)
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "test.txt", "content")

	start(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "test.txt", "content")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, "mob-session")
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "example.txt", "content")

	start(configuration)

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	setup(t)
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "example.txt", "content")

	start(configuration)

	assertOnBranch(t, "master")
}

func TestStartNextBackToMaster(t *testing.T) {
	setup(t)
	start(configuration)
	createFile(t, "example.txt", "content")
	assertOnBranch(t, "mob-session")

	next(configuration)

	assertOnBranch(t, "master")
	assertMobSessionBranches(t, "mob-session")
}

func TestStartNextStay(t *testing.T) {
	setup(t)
	configuration.MobNextStay = true
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	assertOnBranch(t, "mob-session")

	next(configuration)

	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage)
	assertOnBranch(t, "mob-session")
}

func TestStartDoneWithMobDoneSquashTrue(t *testing.T) {
	setup(t)
	configuration.MobDoneSquash = true

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestRunOutput(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	output := run(t, "cat", "/tmp/mob/local/file1.txt")
	assertOutputContains(t, output, "asdf")
}

func TestTestbed(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/alice")
	start(configuration)
	createFile(t, "file3.txt", "owqe")
	next(configuration)

	setWorkingDir("/tmp/mob/bob")
	start(configuration)
	createFile(t, "file4.txt", "zcvx")
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)

	output := silentgit("log", "--pretty=format:'%ae'")
	assertOutputContains(t, &output, "local")
	assertOutputContains(t, &output, "localother")
	assertOutputContains(t, &output, "alice")
	assertOutputContains(t, &output, "bob")
}

func TestStartDoneWithMobDoneSquashFalse(t *testing.T) {
	setup(t)
	configuration.MobDoneSquash = false

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDonePublishingOneManualCommit(t *testing.T) {
	setup(t)
	// REFACTOR Replace string with enum value
	configuration.MobDoneSquash = false // default is probably true

	start(configuration)
	assertOnBranch(t, "mob-session")
	// should be 1 commit on mob-session so far

	createFileAndCommitIt(t, "example.txt", "content", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done(configuration) // without squash (configuration)

	assertOnBranch(t, "master")
	assertCommitsOnBranch(t, 2, "master")
	assertCommitLogContainsMessage(t, "master", "[manual-commit-1] publish this commit to master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneSquashTheOneManualCommit(t *testing.T) {
	setup(t)
	// REFACTOR Replace string with enum value
	configuration.MobDoneSquash = true

	start(configuration)
	assertOnBranch(t, "mob-session")
	// should be 1 commit on mob-session so far

	createFileAndCommitIt(t, "example.txt", "content", "[manual-commit-1] publish this commit to master")
	assertCommits(t, 2)

	done(configuration)

	// MAYBE assertUnstagedChanges()
	assertOnBranch(t, "master")
	assertCommitsOnBranch(t, 1, "master")
	assertCommitsOnBranch(t, 1, "origin/master")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneFeatureBranch(t *testing.T) {
	setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	done(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartNextFeatureBranch(t *testing.T) {
	setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	next(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, "mob-session")
}

func TestStartDoneLocalFeatureBranch(t *testing.T) {
	output := setup(t)
	git("checkout", "-b", "feature1")

	start(configuration)

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "git push origin feature1 --set-upstream")
}

func TestBothCreateNonemptyCommitWithNext(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")

	setWorkingDir("/tmp/mob/local")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	// next(configuration) not possible, would fail
	git("pull")
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")
}

func TestNothingToCommitCreatesNoCommits(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/local")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	assertCommits(t, 1)
}

func TestStartNextPushManualCommits(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "content", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestStartNextPushManualCommitsFeatureBranch(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")

	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	createFileAndCommitIt(t, "example.txt", "content", "asdf")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	git("fetch")
	git("checkout", "feature1")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestConflictingMobSessions(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "example2.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
}

func TestConflictingMobSessionsNextStay(t *testing.T) {
	setup(t)
	configuration.MobNextStay = true

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
	next(configuration)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir("/tmp/mob/localother")
	start(configuration)
}

func TestDoneMergeConflict(t *testing.T) {
	output := setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("push")

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "Automatic merge failed; fix conflicts and then commit the result.")
}

func TestDoneMerge(t *testing.T) {
	output := setup(t)

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother")
	createFileAndCommitIt(t, "example2.txt", "asdf", "asdf")
	git("push")

	setWorkingDir("/tmp/mob/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "   git commit")
}

func TestStartAndNextInSubdir(t *testing.T) {
	setup(t)

	setWorkingDir("/tmp/mob/local/subdir")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/localother/subdir")
	start(configuration)
	createFile(t, "example2.txt", "content")
	createFile(t, "../example3.txt", "content")
	next(configuration)

	setWorkingDir("/tmp/mob/local/subdir")
	start(configuration)
	done(configuration)

	setWorkingDir("/tmp/mob/local")
	assertFileExist(t, "subdir/example.txt")
	assertFileExist(t, "subdir/example2.txt")
	assertFileExist(t, "example3.txt")
}

func TestIsGitIdentifiesGitRepo(t *testing.T) {
	setup(t)
	equals(t, true, isGit())
}

func TestIsGitIdentifiesOutsideOfGitRepo(t *testing.T) {
	setWorkingDir("/tmp/git/notgit")
	equals(t, false, isGit())
}

func TestNotAGitRepoMessage(t *testing.T) {
	output := setup(t)
	setWorkingDir("/tmp/git/notgit")
	sayGitError("TEST", "TEST", errors.New("TEST"))
	assertOutputContains(t, output, "mob expects the current working directory to be a git repository.")
}

func setup(t *testing.T) *string {
	configuration = getDefaultConfiguration()
	configuration.MobNextStay = false
	output := captureOutput()
	createTestbed(t)
	assertOnBranch(t, "master")
	equals(t, []string{"master"}, gitBranches())
	equals(t, []string{"origin/master"}, gitRemoteBranches())
	assertNoMobSessionBranches(t, "mob-session")
	return output
}

func captureOutput() *string {
	messages := ""
	printToConsole = func(text string) {
		fmt.Print(text)
		messages += text
	}
	return &messages
}

func run(t *testing.T, name string, args ...string) *string {
	commandString, output, err := runCommand(name, args...)
	if err != nil {
		fmt.Println(commandString)
		fmt.Println(output)
		fmt.Println(err.Error())
		t.Error("command " + commandString + " failed")
	}
	return &output
}

func createTestbed(t *testing.T) {
	workingDir = ""
	run(t, "./create-testbed")

	setWorkingDir("/tmp/mob/local")
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, "mob-session")
}

func setWorkingDir(dir string) {
	workingDir = dir
	say("\nSET WORKING DIR TO " + dir + "\n======================\n")
}

func assertCommits(t *testing.T, commits int) {
	assertCommitsOnBranch(t, commits, "HEAD")
}

func assertCommitsOnBranch(t *testing.T, commits int, branchName string) {
	result := silentgit("rev-list", "--count", branchName)
	number, _ := strconv.Atoi(result)
	if number != commits {
		failWithFailure(t, strconv.Itoa(commits)+" commits in "+workingDir, strconv.Itoa(number)+" commits in "+workingDir)
	}
}

func assertCommitLogContainsMessage(t *testing.T, branchName string, commitMessage string) {
	logMessages := silentgit("log", branchName, "--oneline")
	if !strings.Contains(logMessages, commitMessage) {
		failWithFailure(t, "git log contains '"+commitMessage+"'", logMessages)
	}
}

func assertFileExist(t *testing.T, filename string) {
	path := workingDir + "/" + filename
	if _, err := os.Stat(path); os.IsNotExist(err) {
		failWithFailure(t, "existing file "+path, "no file at "+path)
	}
}

func createFileAndCommitIt(t *testing.T, filename string, content string, commitMessage string) {
	createFile(t, filename, content)
	git("add", filename)
	git("commit", "-m", commitMessage)
}

func createFile(t *testing.T, filename string, content string) {
	d1 := []byte(content)
	err := ioutil.WriteFile(workingDir+"/"+filename, d1, 0644)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
}

func assertOnBranch(t *testing.T, branch string) {
	currentBranch := gitCurrentBranch()
	if currentBranch != branch {
		failWithFailure(t, "on branch "+branch, "on branch "+currentBranch)
	}
}

func assertOutputContains(t *testing.T, output *string, contains string) {
	currentOutput := *output
	if !strings.Contains(currentOutput, contains) {
		failWithFailure(t, "output contains '"+contains+"'", currentOutput)
	}
}

func assertOutputNotContains(t *testing.T, output *string, notContains string) {
	if strings.Contains(*output, notContains) {
		failWithFailure(t, "output not contains "+notContains, output)
	}
}

func assertMobSessionBranches(t *testing.T, branch string) {
	if !hasRemoteBranch(branch, configuration) {
		failWithFailure(t, configuration.RemoteName+"/"+branch, "none")
	}
	if !hasLocalBranch(branch) {
		failWithFailure(t, branch, "none")
	}
}

func assertNoMobSessionBranches(t *testing.T, branch string) {
	if hasRemoteBranch(branch, configuration) {
		failWithFailure(t, "none", configuration.RemoteName+"/"+branch)
	}
	if hasLocalBranch(branch) {
		failWithFailure(t, "none", branch)
	}
}

func equals(t *testing.T, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		t.Log(string(debug.Stack()))
		failWithFailure(t, exp, act)
	}
}

func failWithFailure(t *testing.T, exp interface{}, act interface{}) {
	_, file, line, _ := runtime.Caller(1)
	fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
	t.FailNow()
}
