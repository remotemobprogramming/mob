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
	tempDir string
)

func TestParseArgs(t *testing.T) {
	configuration := getDefaultConfiguration()
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := parseArgs([]string{"mob", "start", "--branch", "green"}, configuration)

	equals(t, "start", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "green", configuration.WipBranchQualifier)
}

func TestParseArgsDoneNoSquash(t *testing.T) {
	configuration := getDefaultConfiguration()
	equals(t, true, configuration.MobDoneSquash)

	command, parameters, configuration := parseArgs([]string{"mob", "done", "--no-squash"}, configuration)

	equals(t, "done", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, false, configuration.MobDoneSquash)
}

func TestParseArgsDoneSquash(t *testing.T) {
	configuration := getDefaultConfiguration()
	configuration.MobDoneSquash = false

	command, parameters, configuration := parseArgs([]string{"mob", "done", "--squash"}, configuration)

	equals(t, "done", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, true, configuration.MobDoneSquash)
}

func TestParseArgsMessage(t *testing.T) {
	configuration := getDefaultConfiguration()
	equals(t, configuration.WipBranchQualifier, "")

	command, parameters, configuration := parseArgs([]string{"mob", "next", "--message", "ci-skip"}, configuration)

	equals(t, "next", command)
	equals(t, "", strings.Join(parameters, ""))
	equals(t, "ci-skip", configuration.WipCommitMessage)
}

func TestDetermineBranches(t *testing.T) {
	configuration := getDefaultConfiguration()
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
	baseBranch, wipBranch := determineBranches(newBranch(branch), branches, configuration)
	equals(t, newBranch(expectedBase), baseBranch)
	equals(t, newBranch(expectedWip), wipBranch)
}

func TestRemoveWipPrefix(t *testing.T) {
	configuration := getDefaultConfiguration()
	configuration.WipBranchPrefix = "mob/"
	equals(t, "master-green", newBranch("mob/master-green").removeWipPrefix(configuration).Name)
	equals(t, "master-green-blue", newBranch("mob/master-green-blue").removeWipPrefix(configuration).Name)
	equals(t, "main-branch", newBranch("mob/main-branch").removeWipPrefix(configuration).Name)
}

func TestRemoveWipBranchQualifier(t *testing.T) {
	var configuration Configuration

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "green"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", newBranch("master-green").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "test-branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master-test", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branch"
	configuration.WipBranchQualifierSet = true
	equals(t, "master-test", newBranch("master-test-branch").removeWipQualifier([]string{"master-test"}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "/-/"
	configuration.WipBranchQualifier = "branch-qualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", newBranch("main/-/branch-qualifier").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main/branchqualifier", newBranch("main/branchqualifier").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = ""
	configuration.WipBranchQualifier = "branchqualifier"
	configuration.WipBranchQualifierSet = true
	equals(t, "main", newBranch("mainbranchqualifier").removeWipQualifier([]string{}, configuration).Name)
}

func TestRemoveWipBranchQualifierWithoutBranchQualifierSet(t *testing.T) {
	var configuration Configuration

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	equals(t, "main", newBranch("main").removeWipQualifier([]string{}, configuration).Name)

	configuration.WipBranchQualifierSeparator = "-"
	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	equals(t, "master", newBranch("master-test-branch").removeWipQualifier([]string{}, configuration).Name)
}

func TestMobRemoteNameEnvironmentVariable(t *testing.T) {
	configuration := setEnvVarAndParse("MOB_REMOTE_NAME", "GITHUB")

	equals(t, "GITHUB", configuration.RemoteName)
}

func TestMobRemoteNameEnvironmentVariableEmptyString(t *testing.T) {
	configuration := setEnvVarAndParse("MOB_REMOTE_NAME", "")

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
		configuration := setEnvVarAndParse(variable, value)
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
	output, _ := setup(t)

	version()

	assertOutputContains(t, output, versionNumber)
}

func TestNextNotMobProgramming(t *testing.T) {
	output, configuration := setup(t)

	next(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestRequireCommitMessage(t *testing.T) {
	output, _ := setup(t)

	os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")
	defer os.Unsetenv("MOB_REQUIRE_COMMIT_MESSAGE")

	configuration := parseEnvironmentVariables(getDefaultConfiguration())
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
	output, configuration := setup(t)

	done(configuration)

	assertOutputContains(t, output, "you aren't mob programming")
}

func TestStatusMobProgramming(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)

	status(configuration)

	assertOutputContains(t, output, "you are on wip branch mob-session")
}

func TestStatusWithMoreThan5LinesOfLog(t *testing.T) {
	output, configuration := setup(t)
	configuration.MobNextStay = true
	start(configuration)

	for i := 0; i < 6; i++ {
		createFile(t, "test"+strconv.Itoa(i)+".txt", "test")
		next(configuration)
	}

	status(configuration)
	assertOutputContains(t, output, "wip branch 'mob-session' contains 6 commits.")
}

func TestExecuteKicksOffStatus(t *testing.T) {
	output, _ := setup(t)

	execute("status", []string{}, getDefaultConfiguration())

	assertOutputContains(t, output, "you are on base branch 'master'")
}

func TestExecuteInvalidCommandKicksOffHelp(t *testing.T) {
	output, _ := setup(t)

	execute("whatever", []string{}, getDefaultConfiguration())

	assertOutputContains(t, output, "Basic Commands:")
}

func TestStart(t *testing.T) {
	_, configuration := setup(t)

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartWithMultipleExistingBranches(t *testing.T) {
	output, configuration := setup(t)

	configuration.WipBranchQualifier = "green"
	start(configuration)
	assertOnBranch(t, "mob/master-green")
	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = ""
	start(configuration)
	assertOnBranch(t, "mob-session")
	assertOutputContains(t, output, "preexisting wip branches have been detected")
	assertOutputContains(t, output, "mob/master-green")
}

func TestStartWithMultipleExistingBranchesAndEmptyWipBranchQualifier(t *testing.T) {
	output, configuration := setup(t)

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
	output, configuration := setup(t)
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
	_, configuration := setup(t)
	assertOnBranch(t, "master")
	configuration.WipBranchQualifier = "green"

	start(configuration)
	assertOnBranch(t, "mob/master-green")
	assertMobSessionBranches(t, configuration, "mob/master-green")
	configuration.WipBranchQualifier = ""

	next(configuration)
	assertOnBranch(t, "master")

	configuration.WipBranchQualifier = "green"
	reset(configuration)
	assertNoMobSessionBranches(t, configuration, "mob/master-green")
}

func TestStartNextStartWithBranch(t *testing.T) {
	_, configuration := setup(t)
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

func TestStartFromDivergingBranches(t *testing.T) {
	output, configuration := setup(t)
	checkoutBranch("feature-something")
	checkoutBranch("feature-something-2")

	assertOnBranch(t, "feature-something-2")
	start(configuration)
	assertOnBranch(t, "mob/feature-something-2")
	next(configuration)

	git("checkout", "feature-something")
	start(configuration)
	assertOnBranch(t, "mob/feature-something")
	assertOutputContains(t, output, "preexisting wip branches have been detected")
	assertOutputContains(t, output, "mob/feature-something-2")
}

func TestStartFromDivergingBranches_noWarning(t *testing.T) {
	output, configuration := setup(t)
	checkoutBranch("mob/feature-something")
	checkoutBranch("feature-something")
	checkoutBranch("mob/feature-something-2")
	checkoutBranch("feature-something-2")

	assertOnBranch(t, "feature-something-2")
	start(configuration)
	assertOnBranch(t, "mob/feature-something-2")

	assertOutputNotContains(t, output, "qualified mob branches detected")
}

func TestStartNextOnFeatureWithBranch(t *testing.T) {
	_, configuration := setup(t)
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
	_, configuration := setup(t)
	configuration.WipBranchQualifier = "test-branch"
	configuration.WipBranchQualifierSet = true
	start(configuration)
	assertOnBranch(t, "mob/master-test-branch")
	assertMobSessionBranches(t, configuration, "mob/master-test-branch")

	configuration.WipBranchQualifier = ""
	configuration.WipBranchQualifierSet = false
	next(configuration)
}

func TestReset(t *testing.T) {
	_, configuration := setup(t)

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestResetCommit(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)
	assertMobSessionBranches(t, configuration, "mob-session")

	reset(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartUnstagedChanges(t *testing.T) {
	output, configuration := setup(t)
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "test.txt", "content")

	start(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
	assertOutputContains(t, output, "mob start --include-uncommitted-changes")
}

func TestStartIncludeUnstagedChanges(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "test.txt", "content")

	start(configuration)

	assertOnBranch(t, "mob-session")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartHasUnpushedCommits(t *testing.T) {
	output, configuration := setup(t)
	createFileAndCommitIt(t, "test.txt", "content", "unpushed change")

	start(configuration)

	assertOutputContains(t, output, "cannot start; unpushed changes")
	assertOutputContains(t, output, "unpushed commits")
}

func TestBranch(t *testing.T) {
	output, configuration := setup(t)
	start(configuration)

	branch(configuration)

	assertOutputContains(t, output, "\norigin/mob-session\n")
}

func TestStartIncludeUntrackedFiles(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobStartIncludeUncommittedChanges = true
	createFile(t, "example.txt", "content")

	start(configuration)

	assertOnBranch(t, "mob-session")
}

func TestStartUntrackedFiles(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobStartIncludeUncommittedChanges = false
	createFile(t, "example.txt", "content")

	start(configuration)

	assertOnBranch(t, "master")
}

func TestStartNextBackToMaster(t *testing.T) {
	_, configuration := setup(t)
	start(configuration)
	createFile(t, "example.txt", "content")
	assertOnBranch(t, "mob-session")

	next(configuration)

	assertOnBranch(t, "master")
	assertMobSessionBranches(t, configuration, "mob-session")
}

func TestStartNextStay(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobNextStay = true
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	assertOnBranch(t, "mob-session")

	next(configuration)

	equals(t, silentgit("log", "--format=%B", "-n", "1", "HEAD"), configuration.WipCommitMessage)
	assertOnBranch(t, "mob-session")
}

func TestStartDoneWithMobDoneSquashTrue(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobDoneSquash = true

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestRunOutput(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	output := run(t, "cat", tempDir+"/local/file1.txt")
	assertOutputContains(t, output, "asdf")
}

func TestTestbed(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/alice")
	start(configuration)
	createFile(t, "file3.txt", "owqe")
	next(configuration)

	setWorkingDir(tempDir + "/bob")
	start(configuration)
	createFile(t, "file4.txt", "zcvx")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)

	output := silentgit("log", "--pretty=format:'%ae'")
	assertOutputContains(t, &output, "local")
	assertOutputContains(t, &output, "localother")
	assertOutputContains(t, &output, "alice")
	assertOutputContains(t, &output, "bob")
}

func TestStartDoneWithMobDoneSquashFalse(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobDoneSquash = false

	start(configuration)
	assertOnBranch(t, "mob-session")

	done(configuration)

	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDonePublishingOneManualCommit(t *testing.T) {
	_, configuration := setup(t)
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
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneSquashTheOneManualCommit(t *testing.T) {
	_, configuration := setup(t)
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
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneFeatureBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	done(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartNextFeatureBranch(t *testing.T) {
	_, configuration := setup(t)
	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	next(configuration)

	assertOnBranch(t, "feature1")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func TestStartDoneLocalFeatureBranch(t *testing.T) {
	output, configuration := setup(t)
	git("checkout", "-b", "feature1")

	start(configuration)

	assertOnBranch(t, "feature1")
	assertOutputContains(t, output, "git push origin feature1 --set-upstream")
}

func TestBothCreateNonemptyCommitWithNext(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "file1.txt", "asdf")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	createFile(t, "file2.txt", "asdf")

	setWorkingDir(tempDir + "/local")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	// next(configuration) not possible, would fail
	git("pull")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertFileExist(t, "file1.txt")
	assertFileExist(t, "file2.txt")
}

func TestNothingToCommitCreatesNoCommits(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/local")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	assertCommits(t, 1)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertCommits(t, 1)
}

func TestStartNextPushManualCommits(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")

	start(configuration)
	createFileAndCommitIt(t, "example.txt", "content", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestStartNextPushManualCommitsFeatureBranch(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")

	git("checkout", "-b", "feature1")
	git("push", "origin", "feature1", "--set-upstream")
	assertOnBranch(t, "feature1")
	start(configuration)
	assertOnBranch(t, "mob/feature1")

	createFileAndCommitIt(t, "example.txt", "content", "asdf")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	git("fetch")
	git("checkout", "feature1")
	start(configuration)
	assertFileExist(t, "example.txt")
}

func TestConflictingMobSessions(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example2.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
}

func TestConflictingMobSessionsNextStay(t *testing.T) {
	_, configuration := setup(t)
	configuration.MobNextStay = true

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	start(configuration)
	next(configuration)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	git("commit", "-m", "\"finished mob session\"")

	setWorkingDir(tempDir + "/localother")
	start(configuration)
}

func TestDoneMergeConflict(t *testing.T) {
	output, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	createFileAndCommitIt(t, "example.txt", "asdf", "asdf")
	git("push")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "Automatic merge failed; fix conflicts and then commit the result.")
}

func TestDoneMerge(t *testing.T) {
	output, configuration := setup(t)

	setWorkingDir(tempDir + "/local")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother")
	createFileAndCommitIt(t, "example2.txt", "asdf", "asdf")
	git("push")

	setWorkingDir(tempDir + "/local")
	start(configuration)
	done(configuration)
	assertOutputContains(t, output, "  git commit")
}

func TestStartAndNextInSubdir(t *testing.T) {
	_, configuration := setup(t)

	setWorkingDir(tempDir + "/local/subdir")
	start(configuration)
	createFile(t, "example.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/localother/subdir")
	start(configuration)
	createFile(t, "example2.txt", "content")
	createFile(t, "../example3.txt", "content")
	next(configuration)

	setWorkingDir(tempDir + "/local/subdir")
	start(configuration)
	done(configuration)

	setWorkingDir(tempDir + "/local")
	assertFileExist(t, "subdir/example.txt")
	assertFileExist(t, "subdir/example2.txt")
	assertFileExist(t, "example3.txt")
}

func TestIsGitIdentifiesGitRepo(t *testing.T) {
	setup(t)
	equals(t, true, isGit())
}

func TestIsGitIdentifiesOutsideOfGitRepo(t *testing.T) {
	setWorkingDir(tempDir + "/notgit")
	equals(t, false, isGit())
}

func TestNotAGitRepoMessage(t *testing.T) {
	output, _ := setup(t)
	setWorkingDir(tempDir + "/notgit")
	sayGitError("TEST", "TEST", errors.New("TEST"))
	assertOutputContains(t, output, "'mob' expects the current working directory to be a git repository.")
}

func setup(t *testing.T) (output *string, configuration Configuration) {
	configuration = getDefaultConfiguration()
	configuration.MobNextStay = false
	output = captureOutput(t)
	createTestbed(t, configuration)
	assertOnBranch(t, "master")
	equals(t, []string{"master"}, gitBranches())
	equals(t, []string{"origin/master"}, gitRemoteBranches())
	assertNoMobSessionBranches(t, configuration, "mob-session")
	return output, configuration
}

func captureOutput(t *testing.T) *string {
	messages := ""
	printToConsole = func(text string) {
		t.Log(strings.TrimRight(text, "\n"))
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

func createTestbed(t *testing.T, configuration Configuration) {
	workingDir = ""

	tempDir = t.TempDir()
	say("Creating testbed in temporary directory " + tempDir)

	run(t, "./create-testbed", tempDir)

	setWorkingDir(tempDir + "/local")
	assertOnBranch(t, "master")
	assertNoMobSessionBranches(t, configuration, "mob-session")
}

func setWorkingDir(dir string) {
	workingDir = dir
	say("\n===== cd " + dir)
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

func createFile(t *testing.T, filename string, content string) (pathToFile string) {
	contentAsBytes := []byte(content)
	pathToFile = workingDir + "/" + filename
	err := ioutil.WriteFile(pathToFile, contentAsBytes, 0644)
	if err != nil {
		failWithFailure(t, "creating file "+filename+" with content "+content, "error")
	}
	return
}

func assertOnBranch(t *testing.T, branch string) {
	currentBranch := gitCurrentBranch()
	if currentBranch.Name != branch {
		failWithFailure(t, "on branch "+branch, "on branch "+currentBranch.String())
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

func assertMobSessionBranches(t *testing.T, configuration Configuration, branch string) {
	if !newBranch(branch).hasRemoteBranch(configuration) {
		failWithFailure(t, newBranch(branch).remote(configuration).Name, "none")
	}
	if !hasLocalBranch(branch) {
		failWithFailure(t, branch, "none")
	}
}

func assertNoMobSessionBranches(t *testing.T, configuration Configuration, branch string) {
	if newBranch(branch).hasRemoteBranch(configuration) {
		failWithFailure(t, "none", newBranch(branch).remote(configuration).Name)
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

func checkoutBranch(datBranch string) {
	git("checkout", "-b", datBranch)
	git("push", "origin", datBranch, "--set-upstream")
}
