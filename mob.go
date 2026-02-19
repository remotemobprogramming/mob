package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/remotemobprogramming/mob/v5/coauthors"
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/findnext"
	mobgit "github.com/remotemobprogramming/mob/v5/git"
	"github.com/remotemobprogramming/mob/v5/goal"
	"github.com/remotemobprogramming/mob/v5/help"
	"github.com/remotemobprogramming/mob/v5/open"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/workdir"
)

const (
	versionNumber     = "5.4.2"
	minimumGitVersion = "2.13.0"
)

var (
	gitClient = &mobgit.Client{}
	args []string
)

func openCommandFor(c config.Configuration, filepath string) (string, []string) {
	if !c.IsOpenCommandGiven() {
		return "", []string{}
	}
	filepathWithoutSpaces := strings.ReplaceAll(filepath, " ", "&spc&")
	split := strings.Split(injectCommandWithMessage(c.OpenCommand, filepathWithoutSpaces), " ")
	for i := 0; i < len(split); i++ {
		split[i] = strings.ReplaceAll(split[i], "&spc&", " ")
	}
	return split[0], split[1:]
}

type GitVersion = mobgit.GitVersion

func parseGitVersion(version string) GitVersion {
	return mobgit.ParseVersion(version)
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

func (branch Branch) remote(configuration config.Configuration) Branch {
	return newBranch(configuration.RemoteName + "/" + branch.Name)
}

func (branch Branch) hasRemoteBranch(configuration config.Configuration) bool {
	remoteBranches := gitRemoteBranches()
	remoteBranch := branch.remote(configuration).Name
	say.Debug("Remote Branches: " + strings.Join(remoteBranches, "\n"))
	say.Debug("Remote Branch: " + remoteBranch)

	for i := 0; i < len(remoteBranches); i++ {
		if remoteBranches[i] == remoteBranch {
			return true
		}
	}

	return false
}

func (branch Branch) hasLocalBranch() bool {
	localBranches := gitBranches()
	say.Debug("Local Branches: " + strings.Join(localBranches, "\n"))
	say.Debug("Local Branch: " + branch.Name)

	for i := 0; i < len(localBranches); i++ {
		if localBranches[i] == branch.Name {
			return true
		}
	}

	return false
}

func (branch Branch) IsWipBranch(configuration config.Configuration) bool {
	if branch.Name == "mob-session" {
		return true
	}

	return strings.Index(branch.Name, configuration.WipBranchPrefix) == 0
}

func (branch Branch) addWipPrefix(configuration config.Configuration) Branch {
	return newBranch(configuration.WipBranchPrefix + branch.Name)
}

func (branch Branch) addWipQualifier(configuration config.Configuration) Branch {
	if configuration.CustomWipBranchQualifierConfigured() {
		return newBranch(addSuffix(branch.Name, configuration.WipBranchQualifierSuffix()))
	}
	return branch
}

func addSuffix(branch string, suffix string) string {
	return branch + suffix
}

func (branch Branch) removeWipPrefix(configuration config.Configuration) Branch {
	return newBranch(removePrefix(branch.Name, configuration.WipBranchPrefix))
}

func removePrefix(branch string, prefix string) string {
	if !strings.HasPrefix(branch, prefix) {
		return branch
	}
	return branch[len(prefix):]
}

func (branch Branch) removeWipQualifier(localBranches []string, configuration config.Configuration) Branch {
	for !branch.exists(localBranches) && branch.hasWipBranchQualifierSeparator(configuration) {
		afterRemoval := branch.removeWipQualifierSuffixOrSeparator(configuration)

		if branch == afterRemoval { // avoids infinite loop
			break
		}

		branch = afterRemoval
	}
	return branch
}

func (branch Branch) removeWipQualifierSuffixOrSeparator(configuration config.Configuration) Branch {
	if !configuration.CustomWipBranchQualifierConfigured() { // WipBranchQualifier not configured
		return branch.removeFromSeparator(configuration.WipBranchQualifierSeparator)
	} else { // WipBranchQualifier not configured
		return branch.removeWipQualifierSuffix(configuration)
	}
}

func (branch Branch) removeFromSeparator(separator string) Branch {
	return newBranch(branch.Name[:strings.LastIndex(branch.Name, separator)])
}

func (branch Branch) removeWipQualifierSuffix(configuration config.Configuration) Branch {
	if strings.HasSuffix(branch.Name, configuration.WipBranchQualifierSuffix()) {
		return newBranch(branch.Name[:strings.LastIndex(branch.Name, configuration.WipBranchQualifierSuffix())])
	}
	return branch
}

func (branch Branch) exists(existingBranches []string) bool {
	return stringContains(existingBranches, branch.Name)
}

func (branch Branch) hasWipBranchQualifierSeparator(configuration config.Configuration) bool { //TODO improve (dont use strings.Contains, add tests)
	return strings.Contains(branch.Name, configuration.WipBranchQualifierSeparator)
}

func (branch Branch) hasLocalCommits(configuration config.Configuration) bool {
	local := silentgit("for-each-ref", "--format=%(objectname)", "refs/heads/"+branch.Name)
	remote := silentgit("for-each-ref", "--format=%(objectname)", "refs/remotes/"+branch.remote(configuration).Name)
	return local != remote
}

func (branch Branch) hasUnpushedCommits(configuration config.Configuration) bool {
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
		say.Info(fmt.Sprintf("there are %d unpushed commits on local base branch <%s>", unpushedCount, branch.Name))
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
	run(os.Args)
}

func run(osArgs []string) {
	args = osArgs
	say.TurnOnDebuggingByArgs(args)
	say.Debug(runtime.Version())

	versionString := gitVersion()
	if versionString == "" {
		say.Error("'git' command was not found in PATH. It may be not installed. " +
			"To learn how to install 'git' refer to https://git-scm.com/book/en/v2/Getting-Started-Installing-Git.")
		exit.Exit(1)
	}

	currentVersion := parseGitVersion(versionString)
	if currentVersion.Less(parseGitVersion(minimumGitVersion)) {
		say.Error(fmt.Sprintf("'git' command version '%s' is lower than the required minimum version (%s). "+
			"Please update your 'git' installation!", versionString, minimumGitVersion))
		exit.Exit(1)
	}

	projectRootDir := ""
	if isGit() {
		projectRootDir = gitRootDir()
		if !hasCommits() {
			say.Error("Git repository does not have any commits yet. Please create an initial commit.")
			exit.Exit(1)
		}
	}

	configuration := config.ReadConfiguration(projectRootDir)
	say.Debug("Args '" + strings.Join(args, " ") + "'")
	currentCliName := currentCliName(args[0])
	if currentCliName != configuration.CliName {
		say.Debug("Updating cli name to " + currentCliName)
		configuration.CliName = currentCliName
	}

	command, parameters, configuration := config.ParseArgs(args, configuration)
	say.Debug("command '" + command + "'")
	say.Debug("parameters '" + strings.Join(parameters, " ") + "'")
	say.Debug("version " + versionNumber)
	say.Debug("workingDir '" + workdir.Path + "'")

	// workaround until we have a better design
	if configuration.GitHooksEnabled {
		gitClient.PassthroughStderrStdout = true
	}

	execute(command, parameters, configuration)
}

func hasCommits() bool {
	return gitClient.HasCommits()
}

func currentCliName(argZero string) string {
	return strings.TrimSuffix(filepath.Base(argZero), ".exe")
}

func execute(command string, parameter []string, configuration config.Configuration) {
	if helpRequested(parameter) {
		help.Help(configuration)
		return
	}

	switch command {
	case "s", "start":
		err := start(configuration)
		if !isMobProgramming(configuration) || err != nil {
			exit.Exit(1)
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			StartTimer(timer, configuration)
		} else if configuration.Timer != "" {
			StartTimer(configuration.Timer, configuration)
		} else {
			say.Info("It's now " + currentTime() + ". Happy collaborating! :)")
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
		config.Config(configuration)
	case "status":
		status(configuration)
	case "t", "timer":
		if len(parameter) > 0 {
			if parameter[0] == "open" || parameter[0] == "o" {
				if err := openTimerInBrowser(configuration); err != nil {
					say.Error(fmt.Sprintf("Could not open webtimer: %s", err.Error()))
				}
			} else {
				timer := parameter[0]
				StartTimer(timer, configuration)
			}
		} else if configuration.Timer != "" {
			StartTimer(configuration.Timer, configuration)
		} else {
			help.Help(configuration)
		}
	case "break":
		if len(parameter) > 0 {
			StartBreakTimer(parameter[0], configuration)
		} else {
			help.Help(configuration)
		}
	case "moo":
		moo(configuration)
	case "sw", "squash-wip":
		if len(parameter) > 1 && parameter[0] == "--git-editor" {
			squashWipGitEditor(parameter[1], configuration)
		} else if len(parameter) > 1 && parameter[0] == "--git-sequence-editor" {
			squashWipGitSequenceEditor(parameter[1], configuration)
		}
	case "g", "goal":
		goal.Goal(configuration, parameter)
	case "version", "--version", "-v":
		version()
	case "help", "--help", "-h":
		help.Help(configuration)
	default:
		help.Help(configuration)
	}
}

func openTimerInBrowser(configuration config.Configuration) error {
	timerurl := configuration.TimerUrl
	if timerurl == "" {
		return fmt.Errorf("Timer url is not configured")
	}
	if configuration.TimerRoom != "" {
		if !strings.HasSuffix(configuration.TimerUrl, "/") {
			timerurl += "/"
		}
		timerurl += configuration.TimerRoom
	} else {
		say.Warning("Timer Room is not configured. To open specific room please configure timer room variable.")
	}
	return open.OpenInBrowser(timerurl)
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

func clean(configuration config.Configuration) {
	git("fetch", configuration.RemoteName, "--prune")

	currentBranch := gitCurrentBranch()
	localBranches := gitBranches()

	if currentBranch.isOrphanWipBranch(configuration) {
		currentBaseBranch, _ := determineBranches(currentBranch, localBranches, configuration)

		say.Info("Current branch " + currentBranch.Name + " is an orphan")
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
			say.Info("Removing orphan wip branch " + b.Name)
			git("branch", "-D", b.Name)
		}
	}

}

func (branch Branch) isOrphanWipBranch(configuration config.Configuration) bool {
	return branch.IsWipBranch(configuration) && !branch.hasRemoteBranch(configuration)
}

func branch(configuration config.Configuration) {
	say.Say(silentgit("branch", "--list", "--remote", newBranch("*").addWipPrefix(configuration).remote(configuration).Name))

	// DEPRECATED
	say.Say(silentgit("branch", "--list", "--remote", newBranch("mob-session").remote(configuration).Name))
}

func determineBranches(currentBranch Branch, localBranches []string, configuration config.Configuration) (baseBranch Branch, wipBranch Branch) {
	if currentBranch.Is("mob-session") || (currentBranch.Is("master") && !configuration.CustomWipBranchQualifierConfigured()) {
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

	say.Debug("on currentBranch " + currentBranch.String() + " => BASE " + baseBranch.String() + " WIP " + wipBranch.String() + " with allLocalBranches " + strings.Join(localBranches, ","))
	if currentBranch != baseBranch && currentBranch != wipBranch {
		// this is unreachable code, but we keep it as a backup
		panic("assertion failed! neither on base nor on wip branch")
	}
	return
}

func injectCommandWithMessage(command string, message string) string {
	placeHolders := strings.Count(command, "%s")
	if placeHolders > 1 {
		say.Error(fmt.Sprintf("Too many placeholders (%d) in format command string: %s", placeHolders, command))
		exit.Exit(1)
	}
	if placeHolders == 0 {
		return fmt.Sprintf("%s %s", command, message)
	}
	return fmt.Sprintf(command, message)
}

func executeCommandsInBackgroundProcess(commands ...string) (err error) {
	cmds := make([]string, 0)
	for _, c := range commands {
		if len(c) > 0 {
			cmds = append(cmds, c)
		}
	}
	say.Debug(fmt.Sprintf("Operating System %s", runtime.GOOS))
	switch runtime.GOOS {
	case "windows":
		_, err = startCommand("powershell", "-command", fmt.Sprintf("start-process powershell -NoNewWindow -ArgumentList '-command \"%s\"'", strings.Join(cmds, ";")))
	case "darwin", "linux":
		_, err = startCommand("sh", "-c", fmt.Sprintf("(%s) &", strings.Join(cmds, ";")))
	default:
		say.Warning(fmt.Sprintf("Cannot execute background commands on your os: %s", runtime.GOOS))
	}
	return err
}

func currentTime() string {
	return time.Now().Format("15:04")
}

func moo(configuration config.Configuration) {
	voiceMessage := "moo"
	err := executeCommandsInBackgroundProcess(getVoiceCommand(voiceMessage, configuration.VoiceCommand))

	if err != nil {
		say.Warning(fmt.Sprintf("can't run voice command on your system (%s)", runtime.GOOS))
		say.Warning(err.Error())
		return
	}

	say.Info(voiceMessage)
}

func reset(configuration config.Configuration) {
	if configuration.ResetDeleteRemoteWipBranch {
		deleteRemoteWipBranch(configuration)
	} else {
		say.Fix("Executing this command deletes the mob branch for everyone. If you're sure you want that, use", configuration.Mob("reset --delete-remote-wip-branch"))
	}
}

func deleteRemoteWipBranch(configuration config.Configuration) {
	git("fetch", configuration.RemoteName)

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	git("checkout", currentBaseBranch.String())
	if currentWipBranch.hasLocalBranch() {
		git("branch", "--delete", "--force", currentWipBranch.String())
	}
	if currentWipBranch.hasRemoteBranch(configuration) {
		gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, "--delete", currentWipBranch.String())
	}
	say.Info("Branches " + currentWipBranch.String() + " and " + currentWipBranch.remote(configuration).String() + " deleted")
}

func start(configuration config.Configuration) error {
	uncommittedChanges := hasUncommittedChanges()
	if uncommittedChanges && configuration.HandleUncommittedChanges == config.FailWithError {
		say.Info("cannot start; clean working tree required")
		sayUnstagedChangesInfo()
		sayUntrackedFilesInfo()
		sayFixUncommittedChanges(configuration)
		return errors.New("cannot start; clean working tree required")
	}

	git("fetch", configuration.RemoteName, "--prune")
	currentBranch := gitCurrentBranch()
	currentBaseBranch, currentWipBranch := determineBranches(currentBranch, gitBranches(), configuration)

	if !currentWipBranch.hasRemoteBranch(configuration) && configuration.StartJoin {
		say.Error("Remote wip branch " + currentWipBranch.remote(configuration).String() + " is missing")
		return errors.New("remote wip branch is missing")
	}

	if !currentBaseBranch.hasRemoteBranch(configuration) && !configuration.StartCreate {
		say.Error("Remote branch " + currentBaseBranch.remote(configuration).String() + " is missing")
		say.Fix("To start and create the remote branch", "mob start --create")
		return errors.New("remote branch is missing")
	}

	createRemoteBranch(configuration, currentBaseBranch)

	if currentBaseBranch.hasLocalBranch() && currentBaseBranch.hasUnpushedCommits(configuration) {
		say.Error("cannot start; unpushed changes on base branch must be pushed upstream")
		say.Fix("to fix this, push those commits and try again", "git push "+configuration.RemoteName+" "+currentBaseBranch.String())
		return errors.New("cannot start; unpushed changes on base branch must be pushed upstream")
	}

	if uncommittedChanges && configuration.HandleUncommittedChanges == config.DiscardChanges {
		git("reset", "--hard")
	}

	if uncommittedChanges && configuration.HandleUncommittedChanges == config.IncludeChanges {
		if silentgit("ls-tree", "-r", "HEAD", "--full-name", "--name-only", ".") == "" {
			say.Error("cannot start; current working dir is an uncommitted subdir")
			say.Fix("to fix this, go to the parent directory and try again", "cd ..")
			return errors.New("cannot start; current working dir is an uncommitted subdir")
		}
		git("stash", "push", "--include-untracked", "--message", configuration.StashName)
		say.Info("uncommitted changes were stashed. If an error occurs later on, you can recover them with 'git stash pop'.")
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

	if uncommittedChanges && configuration.HandleUncommittedChanges == config.IncludeChanges {
		stashes := silentgit("stash", "list")
		stash := findStashByName(stashes, configuration.StashName)
		git("stash", "pop", stash)
	}

	say.Info("you are on wip branch '" + currentWipBranch.String() + "' (base branch '" + currentBaseBranch.String() + "')")
	sayLastCommitsList(currentBaseBranch, currentWipBranch, configuration)

	openLastModifiedFileIfPresent(configuration)

	return nil // no error
}

func sayFixUncommittedChanges(configuration config.Configuration) {
	var instructionInclude string
	var instructionDiscard string
	if configuration.StartCreate {
		instructionInclude = "To start, including uncommitted changes and create the remote branch, use"
		instructionDiscard = "To start, discarding uncommitted changes and create the remote branch, use"
	} else {
		instructionInclude = "To start, including uncommitted changes, use"
		instructionDiscard = "To start, discarding uncommitted changes, use"
	}

	fixCommandStart := configuration.CliName + " start" + createFix(configuration) + branchFix(configuration)
	fixCommandInclude := fixCommandStart + " --include-uncommitted-changes"
	fixCommandDiscard := fixCommandStart + " --discard-uncommitted-changes"

	say.Fix(instructionInclude, fixCommandInclude)
	say.Fix(instructionDiscard, fixCommandDiscard)
}

func createFix(configuration config.Configuration) string {
	if configuration.StartCreate {
		return " --create"
	}
	return ""
}

func branchFix(configuration config.Configuration) string {
	if branchParameter(configuration) {
		return " --branch " + configuration.WipBranchQualifier
	}
	return ""
}

func branchParameter(configuration config.Configuration) bool {
	return containsAny(args, "-b", "--branch") && configuration.WipBranchQualifier != ""
}

func containsAny(list []string, elements ...string) bool {
	for _, value := range list {
		for _, element := range elements {
			if value == element {
				return true
			}
		}
	}
	return false
}

func createRemoteBranch(configuration config.Configuration, currentBaseBranch Branch) {
	if !currentBaseBranch.hasRemoteBranch(configuration) && configuration.StartCreate {
		git("push", configuration.RemoteName, currentBaseBranch.String(), "--set-upstream")
	} else if currentBaseBranch.hasRemoteBranch(configuration) && configuration.StartCreate {
		say.Info("Remote branch " + currentBaseBranch.remote(configuration).String() + " already exists")
	}
}

func openLastModifiedFileIfPresent(configuration config.Configuration) {
	if !configuration.IsOpenCommandGiven() {
		say.Debug("No open command given")
		return
	}

	say.Debug("Try to open last modified file")
	if !lastCommitIsWipCommit(configuration) {
		say.Debug("Last commit isn't a WIP commit.")
		return
	}
	lastCommitMessage := lastCommitMessage()
	split := strings.Split(lastCommitMessage, "lastFile:")
	if len(split) == 1 {
		say.Warning("Couldn't find last modified file in commit message!")
		return
	}
	if len(split) > 2 {
		say.Warning("Could not determine last modified file from commit message, separator was used multiple times!")
		return
	}
	lastModifiedFile := strings.Split(split[1], "\n")[0]
	if strings.HasPrefix(lastModifiedFile, "\"") {
		lastModifiedFile, _ = strconv.Unquote(lastModifiedFile)
	}
	if lastModifiedFile == "" {
		say.Debug("Could not find last modified file in commit message")
		return
	}
	lastModifiedFilePath := gitRootDir() + "/" + lastModifiedFile
	commandname, args := openCommandFor(configuration, lastModifiedFilePath)
	_, err := startCommand(commandname, args...)
	if err != nil {
		say.Warning(fmt.Sprintf("Couldn't open last modified file on your system (%s)", runtime.GOOS))
		say.Warning(err.Error())
		return
	}
	say.Debug("Open last modified file: " + lastModifiedFilePath)
}

func warnForActiveWipBranches(configuration config.Configuration, currentBaseBranch Branch) {
	if isMobProgramming(configuration) {
		return
	}

	// TODO show all active wip branches, even non-qualified ones
	existingWipBranches := getWipBranchesForBaseBranch(currentBaseBranch, configuration)
	if len(existingWipBranches) > 0 && configuration.WipBranchQualifier == "" {
		say.Warning("Creating a new wip branch even though preexisting wip branches have been detected.")
		for _, wipBranch := range existingWipBranches {
			say.WithPrefix(wipBranch, "  - ")
		}
	}
}

func sayUntrackedFilesInfo() {
	untrackedFiles := getUntrackedFiles()
	hasUntrackedFiles := len(untrackedFiles) > 0
	if hasUntrackedFiles {
		say.Info("untracked files present:")
		say.InfoIndented(untrackedFiles)
	}
}

func sayUnstagedChangesInfo() {
	unstagedChanges := getUnstagedChanges()
	hasUnstagedChanges := len(unstagedChanges) > 0
	if hasUnstagedChanges {
		say.Info("unstaged changes present:")
		say.InfoIndented(unstagedChanges)
	}
}

func getWipBranchesForBaseBranch(currentBaseBranch Branch, configuration config.Configuration) []string {
	remoteBranches := gitRemoteBranches()
	say.Debug("check on current base branch " + currentBaseBranch.String() + " with remote branches " + strings.Join(remoteBranches, ","))

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

func startJoinMobSession(configuration config.Configuration) {
	baseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	say.Info("joining existing session from " + currentWipBranch.remote(configuration).String())
	if currentWipBranch.hasLocalBranch() && doBranchesDiverge(baseBranch.remote(configuration).Name, currentWipBranch.Name) {
		say.Warning("Careful, your wip branch (" + currentWipBranch.Name + ") diverges from your main branch (" + baseBranch.remote(configuration).Name + ") !")
	}

	git("checkout", "-B", currentWipBranch.Name, currentWipBranch.remote(configuration).Name)
	git("branch", "--set-upstream-to="+currentWipBranch.remote(configuration).Name, currentWipBranch.Name)
}

func startNewMobSession(configuration config.Configuration) {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	say.Info("starting new session from " + currentBaseBranch.remote(configuration).String())
	git("checkout", "-B", currentWipBranch.Name, currentBaseBranch.remote(configuration).Name)
	gitWithoutEmptyStrings(append(gitPushArgs(configuration), gitHooksOption(configuration), "--set-upstream", configuration.RemoteName, currentWipBranch.Name+":"+currentWipBranch.Name)...)
}

func gitPushArgs(c config.Configuration) []string {
	pushArgs := []string{"push"}
	if !c.SkipCiPushOptionEnabled {
		return pushArgs
	}
	return append(pushArgs, "--push-option", "ci.skip")
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

func next(configuration config.Configuration) {
	if !isMobProgramming(configuration) {
		say.Fix("to start working together, use", configuration.Mob("start"))
		return
	}

	if !configuration.HasCustomCommitMessage() && configuration.RequireCommitMessage && hasUncommittedChanges() {
		say.Error("commit message required")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if isNothingToCommit() {
		if currentWipBranch.hasLocalCommits(configuration) {
			gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, currentWipBranch.Name)
		} else {
			say.Info("nothing was done, so nothing to commit")
		}
	} else {
		makeWipCommit(configuration)
		gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, currentWipBranch.Name)
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

func makeWipCommit(configuration config.Configuration) {
	git("add", "--all")
	commitMessage := createWipCommitMessage(configuration)
	gitWithoutEmptyStrings("commit", "--message", commitMessage, gitHooksOption(configuration))
	say.InfoIndented(getChangesOfLastCommit())
	say.InfoIndented(gitClient.CommitHash())
}

func createWipCommitMessage(configuration config.Configuration) string {
	commitMessage := configuration.WipCommitMessage

	lastModifiedFilePath := getPathOfLastModifiedFile()
	if lastModifiedFilePath != "" {
		commitMessage += "\n\nlastFile:" + lastModifiedFilePath
	}

	return commitMessage
}

// uses git status --porcelain. To work properly files have to be staged.
func getPathOfLastModifiedFile() string {
	rootDir := gitRootDir()
	files := getModifiedFiles(rootDir)
	lastModifiedFilePath := ""
	lastModifiedTime := time.Time{}

	say.Debug("Find last modified file")
	if len(files) == 1 {
		lastModifiedFilePath = files[0]
		say.Debug("Just one modified file: " + lastModifiedFilePath)
		return lastModifiedFilePath
	}

	for _, file := range files {
		unquotedFile := file
		if strings.HasPrefix(file, "\"") {
			var err error
			unquotedFile, err = strconv.Unquote(file)
			if err != nil {
				say.Warning("Could not unquote filename from git: " + file)
				say.Warning(err.Error())
				continue
			}
		}
		absoluteFilepath := rootDir + "/" + unquotedFile
		say.Debug(absoluteFilepath)
		info, err := os.Stat(absoluteFilepath)
		if err != nil {
			say.Warning("Could not get statistics of file: " + absoluteFilepath)
			say.Warning(err.Error())
			continue
		}
		modTime := info.ModTime()
		if modTime.After(lastModifiedTime) {
			lastModifiedTime = modTime
			lastModifiedFilePath = file
		}
		say.Debug(modTime.String())
	}
	return lastModifiedFilePath
}

// uses git status --porcelain. To work properly files have to be staged.
func getModifiedFiles(rootDir string) []string {
	say.Debug("Find modified files")
	oldWorkingDir := workdir.Path
	workdir.Path = rootDir
	gitstatus := silentgit("status", "--porcelain")
	workdir.Path = oldWorkingDir
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
		say.Debug(relativeFilepath)
		files = append(files, relativeFilepath)
	}
	return files
}

func gitHooksOption(c config.Configuration) string {
	return mobgit.HooksOption(c)
}

func fetch(configuration config.Configuration) {
	git("fetch", configuration.RemoteName, "--prune")
}

func done(configuration config.Configuration) {
	if !isMobProgramming(configuration) {
		say.Fix("to start working together, use", configuration.Mob("start"))
		return
	}

	git("fetch", configuration.RemoteName, "--prune")

	baseBranch, wipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if wipBranch.hasRemoteBranch(configuration) {
		if configuration.DoneSquash == config.SquashWip {
			git("merge", "FETCH_HEAD", "--ff-only")
			squashWip(configuration)
		}
		uncommittedChanges := hasUncommittedChanges()
		if uncommittedChanges {
			makeWipCommit(configuration)
		}
		gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, wipBranch.Name)

		git("checkout", baseBranch.Name)
		git("merge", baseBranch.remote(configuration).Name, "--ff-only")
		mergeFailed := gitIgnoreFailure("merge", squashOrCommit(configuration), "--ff", wipBranch.Name)

		if mergeFailed != nil {
			// TODO should this be an error and a fix for that error?
			say.Warning("Skipped deleting " + wipBranch.Name + " because of merge conflicts.")
			say.Warning("To fix this, solve the merge conflict manually, commit, push, and afterwards delete " + wipBranch.Name)
			return
		}

		git("branch", "-D", wipBranch.Name)

		if uncommittedChanges && configuration.DoneSquash != config.Squash { // give the user the chance to name their final commit
			git("reset", "--soft", "HEAD^")
		}

		gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, "--delete", wipBranch.Name)

		cachedChanges := getCachedChanges()
		hasCachedChanges := len(cachedChanges) > 0
		if hasCachedChanges {
			say.InfoIndented(cachedChanges)
		}
		err := coauthors.AppendCoauthorsToSquashMsg(gitDir(), gitUserEmail())
		if err != nil {
			say.Warning(err.Error())
		}

		if hasUncommittedChanges() {
			say.Next("To finish, use", "git commit")
		} else if configuration.DoneSquash == config.Squash {
			say.Info("nothing was done, so nothing to commit")
		}

	} else {
		git("checkout", baseBranch.Name)
		git("branch", "-D", wipBranch.Name)
		git("pull", "--ff-only")
		say.Info("someone else already ended your session")
	}
}

func gitDir() string {
	return gitClient.Dir()
}

func gitRootDir() string {
	return gitClient.RootDir()
}

func squashOrCommit(configuration config.Configuration) string {
	if configuration.DoneSquash == config.Squash {
		return "--squash"
	} else {
		return "--commit"
	}
}

func sayLastCommitsList(currentBaseBranch Branch, currentWipBranch Branch, configuration config.Configuration) {
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + currentWipBranch.String()
	log, err := silentgitignorefailure("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	if err != nil {
		commitsBaseWipBranch = currentBaseBranch.remote(configuration).String() + ".." + currentWipBranch.String()
		log = silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	}
	lines := strings.Split(log, "\n")
	if len(lines) > 5 {
		say.Info("wip branch '" + currentWipBranch.String() + "' contains " + strconv.Itoa(len(lines)) + " commits. The last 5 were:")
		lines = lines[:5]
	}
	ReverseSlice(lines)
	output := strings.Join(lines, "\n")
	say.Say(output)
}

func ReverseSlice(s interface{}) {
	size := reflect.ValueOf(s).Len()
	swap := reflect.Swapper(s)
	for i, j := 0, size-1; i < j; i, j = i+1, j-1 {
		swap(i, j)
	}
}

func isNothingToCommit() bool {
	return gitClient.IsNothingToCommit()
}

func hasUncommittedChanges() bool {
	return gitClient.HasUncommittedChanges()
}

func isMobProgramming(configuration config.Configuration) bool {
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, gitBranches(), configuration)
	say.Debug("current branch " + currentBranch.String() + " and currentWipBranch " + currentWipBranch.String())
	return currentWipBranch == currentBranch
}

func gitBranches() []string {
	return gitClient.Branches()
}

func gitRemoteBranches() []string {
	return gitClient.RemoteBranches()
}

func gitCurrentBranch() Branch {
	return newBranch(gitClient.CurrentBranch())
}

func doBranchesDiverge(ancestor string, successor string) bool {
	return gitClient.DoBranchesDiverge(ancestor, successor)
}

func gitUserName() string {
	return gitClient.UserName()
}

func gitUserEmail() string {
	return gitClient.UserEmail()
}

func showNext(configuration config.Configuration) {
	say.Debug("determining next person based on previous changes")
	gitUserName := gitUserName()
	if gitUserName == "" {
		say.Warning("failed to detect who's next because you haven't set your git user name")
		say.Fix("To fix, use", "git config --global user.name \"Your Name Here\"")
		return
	}

	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
	commitsBaseWipBranch := currentBaseBranch.String() + ".." + currentWipBranch.String()

	changes := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%an", "--abbrev-commit")
	lines := strings.Split(strings.Replace(changes, "\r\n", "\n", -1), "\n")
	numberOfLines := len(lines)
	say.Debug("there have been " + strconv.Itoa(numberOfLines) + " changes")
	say.Debug("current git user.name is '" + gitUserName + "'")
	if numberOfLines < 1 {
		return
	}
	nextTypist, previousCommitters := findnext.FindNextTypist(lines, gitUserName)
	if nextTypist != "" {
		if len(previousCommitters) != 0 {
			say.Info("Committers after your last commit: " + strings.Join(previousCommitters, ", "))
		}
		say.Info("***" + nextTypist + "*** is (probably) next.")
	}
}

func version() {
	say.Say("v" + versionNumber)
}

func silentgit(args ...string) string {
	return gitClient.Silent(args...)
}

func silentgitignorefailure(args ...string) (string, error) {
	return gitClient.SilentIgnoreFailure(args...)
}

func gitWithoutEmptyStrings(args ...string) {
	gitClient.RunWithoutEmptyStrings(args...)
}

func git(args ...string) {
	gitClient.Run(args...)
}

func gitIgnoreFailure(args ...string) error {
	return gitClient.RunIgnoreFailure(args...)
}

func gitVersion() string {
	return gitClient.Version()
}

func isGit() bool {
	return gitClient.IsRepo()
}

func startCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	if len(workdir.Path) > 0 {
		command.Dir = workdir.Path
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Starting command " + commandString)
	err := command.Start()
	return commandString, err
}
