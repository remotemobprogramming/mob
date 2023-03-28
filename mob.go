package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	x509 "crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/help"
	"github.com/remotemobprogramming/mob/v4/say"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	versionNumber     = "4.3.1"
	minimumGitVersion = "2.13.0"
)

var (
	workingDir                 = ""
	GitPassthroughStderrStdout = false // hack to get git hooks to print to stdout/stderr
)

func openCommandFor(c config.Configuration, filepath string) (string, []string) {
	if !c.IsOpenCommandGiven() {
		return "", []string{}
	}
	split := strings.Split(injectCommandWithMessage(c.OpenCommand, filepath), " ")
	return split[0], split[1:]
}

type GitVersion struct {
	Major int
	Minor int
	Patch int
}

func parseGitVersion(version string) GitVersion {
	// The git version string can be customized, so we need a more complex regex, for example: git version 2.38.1.windows.1
	// "git" and "version" are optional, and the version number can be x, x.y or x.y.z
	r := regexp.MustCompile(`(?:git)?(?: version )?(?P<major>\d+)(?:\.(?P<minor>\d+)(?:\.(?P<patch>\d+))?)?`)
	matches := r.FindStringSubmatch(version)
	var v GitVersion
	var err error
	if len(matches) > r.SubexpIndex("major") {
		v.Major, err = strconv.Atoi(matches[r.SubexpIndex("major")])
		if err != nil {
			v.Major = 0
			return v
		}
	}
	if len(matches) > r.SubexpIndex("minor") {
		v.Minor, err = strconv.Atoi(matches[r.SubexpIndex("minor")])
		if err != nil {
			v.Minor = 0
			return v
		}
	}
	if len(matches) > r.SubexpIndex("patch") {
		v.Patch, err = strconv.Atoi(matches[r.SubexpIndex("patch")])
		if err != nil {
			v.Patch = 0
		}
	}
	return v
}

func (v GitVersion) Less(rhs GitVersion) bool {
	return v.Major < rhs.Major ||
		(v.Major == rhs.Major && v.Minor < rhs.Minor) ||
		(v.Major == rhs.Major && v.Minor == rhs.Minor && v.Patch < rhs.Patch)
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
	say.TurnOnDebuggingByArgs(os.Args)
	say.Debug(runtime.Version())

	versionString := gitVersion()
	if versionString == "" {
		say.Error("'git' command was not found in PATH. It may be not installed. " +
			"To learn how to install 'git' refer to https://git-scm.com/book/en/v2/Getting-Started-Installing-Git.")
		exit(1)
	}

	currentVersion := parseGitVersion(versionString)
	if currentVersion.Less(parseGitVersion(minimumGitVersion)) {
		say.Error(fmt.Sprintf("'git' command version '%s' is lower than the required minimum version (%s). "+
			"Please update your 'git' installation!", versionString, minimumGitVersion))
		exit(1)
	}

	projectRootDir := ""
	if isGit() {
		projectRootDir = gitRootDir()
	}

	if !hasCommits() {
		say.Error("Git repository does not have any commits yet. Please create an initial commit.")
		exit(1)
	}

	configuration := config.ReadConfiguration(projectRootDir)
	say.Debug("Args '" + strings.Join(os.Args, " ") + "'")
	currentCliName := currentCliName(os.Args[0])
	if currentCliName != configuration.CliName {
		say.Debug("Updating cli name to " + currentCliName)
		configuration.CliName = currentCliName
	}

	command, parameters, configuration := config.ParseArgs(os.Args, configuration)
	say.Debug("command '" + command + "'")
	say.Debug("parameters '" + strings.Join(parameters, " ") + "'")
	say.Debug("version " + versionNumber)
	say.Debug("workingDir '" + workingDir + "'")

	// workaround until we have a better design
	if configuration.GitHooksEnabled {
		GitPassthroughStderrStdout = true
	}

	execute(command, parameters, configuration)
}

func hasCommits() bool {
	commitCount := silentgit("rev-list", "--all", "--count")
	println(commitCount)
	return commitCount != "0"
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
			return
		}
		if len(parameter) > 0 {
			timer := parameter[0]
			startTimer(timer, configuration)
		} else if configuration.Timer != "" {
			startTimer(configuration.Timer, configuration)
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
			timer := parameter[0]
			startTimer(timer, configuration)
		} else if configuration.Timer != "" {
			startTimer(configuration.Timer, configuration)
		} else {
			help.Help(configuration)
		}
	case "break":
		if len(parameter) > 0 {
			startBreakTimer(parameter[0], configuration)
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
	case "version", "--version", "-v":
		version()
	case "help", "--help", "-h":
		help.Help(configuration)
	default:
		help.Help(configuration)
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

func clean(configuration config.Configuration) {
	git("fetch", configuration.RemoteName)

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

func getSleepCommand(timeoutInSeconds int) string {
	return fmt.Sprintf("sleep %d", timeoutInSeconds)
}

func injectCommandWithMessage(command string, message string) string {
	placeHolders := strings.Count(command, "%s")
	if placeHolders > 1 {
		say.Error(fmt.Sprintf("Too many placeholders (%d) in format command string: %s", placeHolders, command))
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

func startTimer(timerInMinutes string, configuration config.Configuration) {
	timeoutInMinutes := toMinutes(timerInMinutes)

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No timer configured, not starting timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)
		if err != nil {
			say.Error("remote timer couldn't be started")
			say.Error(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		abortRunningTimers()
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand), getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
}

func abortRunningTimers() {
	processIds := findMobTimerProcessIds()
	for _, processId := range processIds {
		killRunningProcess(processId)
	}
}

func killRunningProcess(processId string) {
	var err error
	switch runtime.GOOS {
	case "darwin", "linux":
		err = killRunningProcessLinuxAndDarwin(processId)
		break
	case "windows":
		err = killRunningProcessWindows(processId)
	}
	if err != nil {
		say.Error(fmt.Sprintf("old timer couldn't be aborted on your system (%s)", runtime.GOOS))
		say.Error(err.Error())
	}
	say.Debug("Killed mob timer with PID " + processId)
}

func killRunningProcessLinuxAndDarwin(processId string) error {
	_, _, err := runCommandSilent("kill", processId)
	return err
}

func killRunningProcessWindows(processId string) error {
	_, _, err := runCommandSilent("powershell", "-command", "Stop-Process", "-Id", processId)
	return err
}

func findMobTimerProcessIds() []string {
	say.Debug("find all mob timer processes")
	switch runtime.GOOS {
	case "darwin", "linux":
		return findMobTimerProcessIdsLinuxAndDarwin()
	case "windows":
		return findMobTimerProcessIdsWindows()
	}
	return []string{}
}

func findMobTimerProcessIdsWindows() []string {
	_, output, err := runCommandSilent("powershell", "-command", "Get-WmiObject", "Win32_Process", "-Filter", "\"commandLine LIKE '%mobTimer%'\"", "|", "Select-Object", "-Property", "ProcessId,CommandLine", "|", "Out-String", "-Width", "10000")
	if err != nil {
		say.Error(fmt.Sprintf("could not find processes on your system (%s)", runtime.GOOS))
		say.Error(err.Error())
	}
	lines := strings.Split(output, "\r\n")
	var filteredLines []string
	for _, line := range lines {
		if line != "" {
			filteredLines = append(filteredLines, line)
		}
	}
	processInfos := filteredLines[2:]
	var timerProcessIds []string
	for _, processInfo := range processInfos {
		if strings.Contains(processInfo, "echo \"mobTimer\"") {
			timerProcessIds = append(timerProcessIds, strings.Split(strings.TrimSpace(processInfo), " ")[0])
		}
	}
	return timerProcessIds
}

func findMobTimerProcessIdsLinuxAndDarwin() []string {
	_, output, err := runCommandSilent("ps", "-axo", "pid,command")
	lines := strings.Split(output, "\n")
	if err != nil {
		say.Error(fmt.Sprintf("could not find processes on your system (%s)", runtime.GOOS))
		say.Error(err.Error())
	}
	var processIds []string
	for _, line := range lines {
		if strings.Contains(line, "echo \"mobTimer\"") {
			line = strings.TrimSpace(line)
			processId := strings.Split(line, " ")[0]
			processIds = append(processIds, processId)
			say.Debug("Found mob timer with PID " + processId)
		}
	}
	return processIds
}

func getMobTimerRoom(configuration config.Configuration) string {
	if !isGit() {
		say.Debug("timer not in git repository, using MOB_TIMER_ROOM for room name")
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
		say.Info("Using wip branch qualifier for room name")
		return currentWipBranchQualifier
	}

	return configuration.TimerRoom
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) {
	timeoutInMinutes := toMinutes(timerInMinutes)

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No break timer configured, not starting break timer")
		exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutBreakTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)

		if err != nil {
			say.Error("remote break timer couldn't be started")
			say.Error(err.Error())
			exit(1)
		}
	}

	if startLocalTimer {
		abortRunningTimers()
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob start", configuration.VoiceCommand), getNotifyCommand("mob start", configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("break timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
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
	say.Info(requestMethod + " " + requestUrl + " " + string(requestBody))

	responseBody := bytes.NewBuffer(requestBody)
	request, requestCreationError := http.NewRequest(requestMethod, requestUrl, responseBody)

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
			say.Error("The timer.mob.sh SSL certificate is signed by an unknown authority!")
			say.Fix("HINT: You can ignore that by adding MOB_TIMER_INSECURE=true to your configuration or environment.",
				"echo MOB_TIMER_INSECURE=true >> ~/.mob")
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
		say.Info(string(body))
	}
	return nil
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
	if hasLocalBranch(currentWipBranch.String()) {
		git("branch", "--delete", "--force", currentWipBranch.String())
	}
	if currentWipBranch.hasRemoteBranch(configuration) {
		gitWithoutEmptyStrings("push", gitHooksOption(configuration), configuration.RemoteName, "--delete", currentWipBranch.String())
	}
	say.Info("Branches " + currentWipBranch.String() + " and " + currentWipBranch.remote(configuration).String() + " deleted")
}

func start(configuration config.Configuration) error {
	uncommittedChanges := hasUncommittedChanges()
	if uncommittedChanges && !configuration.StartIncludeUncommittedChanges {
		say.Info("cannot start; clean working tree required")
		sayUnstagedChangesInfo()
		sayUntrackedFilesInfo()
		if configuration.StartCreate {
			say.Fix("To start, including uncommitted changes and create the remote branch, use", configuration.Mob("start --create --include-uncommitted-changes"))
		} else {
			say.Fix("To start, including uncommitted changes, use", configuration.Mob("start --include-uncommitted-changes"))
		}
		return errors.New("cannot start; clean working tree required")
	}

	git("fetch", configuration.RemoteName, "--prune")
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	if !currentBaseBranch.hasRemoteBranch(configuration) && !configuration.StartCreate {
		say.Error("Remote branch " + currentBaseBranch.remote(configuration).String() + " is missing")
		say.Fix("To start and and create the remote branch", "mob start --create")
		return errors.New("remote branch is missing")
	}

	createRemoteBranch(configuration, currentBaseBranch)

	if currentBaseBranch.hasUnpushedCommits(configuration) {
		say.Error("cannot start; unpushed changes on base branch must be pushed upstream")
		say.Fix("to fix this, push those commits and try again", "git push "+configuration.RemoteName+" "+currentBaseBranch.String())
		return errors.New("cannot start; unpushed changes on base branch must be pushed upstream")
	}

	if uncommittedChanges && silentgit("ls-tree", "-r", "HEAD", "--full-name", "--name-only", ".") == "" {
		say.Error("cannot start; current working dir is an uncommitted subdir")
		say.Fix("to fix this, go to the parent directory and try again", "cd ..")
		return errors.New("cannot start; current working dir is an uncommitted subdir")
	}

	if uncommittedChanges {
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

	if uncommittedChanges && configuration.StartIncludeUncommittedChanges {
		stashes := silentgit("stash", "list")
		stash := findStashByName(stashes, configuration.StashName)
		git("stash", "pop", stash)
	}

	say.Info("you are on wip branch '" + currentWipBranch.String() + "' (base branch '" + currentBaseBranch.String() + "')")
	sayLastCommitsList(currentBaseBranch.String(), currentWipBranch.String())

	openLastModifiedFileIfPresent(configuration)

	return nil // no error
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
	lastModifiedFile := split[1]
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

func showActiveMobSessions(configuration config.Configuration, currentBaseBranch Branch) {
	existingWipBranches := getWipBranchesForBaseBranch(currentBaseBranch, configuration)
	if len(existingWipBranches) > 0 {
		say.Info("remote wip branches detected:")
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
	if doBranchesDiverge(baseBranch.remote(configuration).Name, currentWipBranch.Name) {
		say.Warning("Careful, your wip branch (" + currentWipBranch.Name + ") diverges from your main branch (" + baseBranch.remote(configuration).Name + ") !")
	}

	git("checkout", "-B", currentWipBranch.Name, currentWipBranch.remote(configuration).Name)
	git("branch", "--set-upstream-to="+currentWipBranch.remote(configuration).Name, currentWipBranch.Name)
}

func startNewMobSession(configuration config.Configuration) {
	currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)

	say.Info("starting new session from " + currentBaseBranch.remote(configuration).String())
	git("checkout", "-B", currentWipBranch.Name, currentBaseBranch.remote(configuration).Name)
	git("commit", "--allow-empty", "-m", configuration.StartCommitMessage)
	gitPush(gitHooksOption(configuration), "--set-upstream", configuration.RemoteName, currentWipBranch.Name)
}

func gitPush(args ...string) {
	gitWithoutEmptyStrings(pushArgs(args)...)
}

func pushArgs(args []string) []string {
	return append([]string{"push"}, deleteEmptyStrings(args)...)
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
	abortRunningTimers()

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
	say.InfoIndented(gitCommitHash())
}

func createWipCommitMessage(configuration config.Configuration) string {
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

	say.Debug("Find last modified file")
	if len(files) == 1 {
		lastModifiedFilePath = files[0]
		say.Debug("Just one modified file: " + lastModifiedFilePath)
		return lastModifiedFilePath
	}

	for _, file := range files {
		absoluteFilepath := rootDir + "/" + file
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

// uses git status --short. To work properly files have to be staged.
func getModifiedFiles(rootDir string) []string {
	say.Debug("Find modified files")
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
		say.Debug(relativeFilepath)
		files = append(files, relativeFilepath)
	}
	return files
}

func gitHooksOption(c config.Configuration) string {
	if c.GitHooksEnabled {
		return ""
	} else {
		return "--no-verify"
	}
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
		err := appendCoauthorsToSquashMsg(gitDir())
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
		say.Info("someone else already ended your session")
	}
	abortRunningTimers()
}

func gitDir() string {
	return silentgit("rev-parse", "--absolute-git-dir")
}

func gitRootDir() string {
	return silentgit("rev-parse", "--show-toplevel")
}

func squashOrCommit(configuration config.Configuration) string {
	if configuration.DoneSquash == config.Squash {
		return "--squash"
	} else {
		return "--commit"
	}
}

func status(configuration config.Configuration) {
	if isMobProgramming(configuration) {
		currentBaseBranch, currentWipBranch := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		say.Info("you are on wip branch " + currentWipBranch.String() + " (base branch " + currentBaseBranch.String() + ")")

		sayLastCommitsList(currentBaseBranch.String(), currentWipBranch.String())
	} else {
		currentBaseBranch, _ := determineBranches(gitCurrentBranch(), gitBranches(), configuration)
		say.Info("you are on base branch '" + currentBaseBranch.String() + "'")
		showActiveMobSessions(configuration, currentBaseBranch)
	}
}

func sayLastCommitsList(currentBaseBranch string, currentWipBranch string) {
	commitsBaseWipBranch := currentBaseBranch + ".." + currentWipBranch
	log := silentgit("--no-pager", "log", commitsBaseWipBranch, "--pretty=format:%h %cr <%an>", "--abbrev-commit")
	lines := strings.Split(log, "\n")
	if len(lines) > 5 {
		say.Info("wip branch '" + currentWipBranch + "' contains " + strconv.Itoa(len(lines)) + " commits. The last 5 were:")
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
	output := silentgit("status", "--short")
	return len(output) == 0
}

func hasUncommittedChanges() bool {
	return !isNothingToCommit()
}

func isMobProgramming(configuration config.Configuration) bool {
	currentBranch := gitCurrentBranch()
	_, currentWipBranch := determineBranches(currentBranch, gitBranches(), configuration)
	say.Debug("current branch " + currentBranch.String() + " and currentWipBranch " + currentWipBranch.String())
	return currentWipBranch == currentBranch
}

func hasLocalBranch(localBranch string) bool {
	localBranches := gitBranches()
	say.Debug("Local Branches: " + strings.Join(localBranches, "\n"))
	say.Debug("Local Branch: " + localBranch)

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
	nextTypist, previousCommitters := findNextTypist(lines, gitUserName)
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
	commandString, output, err := runCommandSilent("git", args...)

	if err != nil {
		if !isGit() {
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			say.Error(commandString)
			say.Error(output)
			say.Error(err.Error())
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
			say.Error("expecting the current working directory to be a git repository.")
		} else {
			say.Error(commandString)
			say.Error(output)
			say.Error(err.Error())
		}
		exit(1)
	} else {
		say.Indented(commandString)
	}
}

func gitIgnoreFailure(args ...string) error {
	commandString, output, err := "", "", error(nil)
	if GitPassthroughStderrStdout {
		commandString, output, err = runCommand("git", args...)
	} else {
		commandString, output, err = runCommandSilent("git", args...)
	}

	if err != nil {
		if !isGit() {
			say.Error("expecting the current working directory to be a git repository.")
			exit(1)
		} else {
			say.Warning(commandString)
			say.Warning(output)
			say.Warning(err.Error())
			return err
		}
	}

	say.Indented(commandString)
	return nil
}

func gitCommitHash() string {
	return silentgitignorefailure("rev-parse", "HEAD")
}

func gitVersion() string {
	_, output, err := runCommandSilent("git", "--version")
	if err != nil {
		say.Debug("gitVersion encountered an error: " + err.Error())
		return ""
	}
	return strings.TrimSpace(output)
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
	say.Debug("Running command <" + commandString + "> in silent mode, capturing combined output")
	outputBytes, err := command.CombinedOutput()
	output := string(outputBytes)
	say.Debug(output)
	return commandString, output, err
}

func runCommand(name string, args ...string) (string, string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Running command <" + commandString + "> passing output through")

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
				say.PrintToConsole("  ")
				lineEnded = false
			}
		}
		say.PrintToConsole(character)
		output += character
	}

	errWait := command.Wait()
	if errWait != nil {
		say.Debug(output)
		return commandString, output, errWait
	}

	say.Debug(output)
	return commandString, output, nil
}

func startCommand(name string, args ...string) (string, error) {
	command := exec.Command(name, args...)
	if len(workingDir) > 0 {
		command.Dir = workingDir
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Starting command " + commandString)
	err := command.Start()
	return commandString, err
}

var exit = func(code int) {
	os.Exit(code)
}
