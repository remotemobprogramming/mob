package main

import (
	"encoding/json"
	"errors"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/httpclient"
	"github.com/remotemobprogramming/mob/v4/say"
	"runtime"
	"strconv"
	"time"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(configuration.Timer, configuration); err != nil {
		Exit(1)
	}
}

func startTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No timer configured, not starting timer")
		Exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)
		if err != nil {
			say.Error("remote timer couldn't be started")
			say.Error(err.Error())
			Exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand), getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			Exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". Happy collaborating! :)")
	return nil
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

func StartBreakTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startBreakTimer(configuration.Timer, configuration); err != nil {
		Exit(1)
	}
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	err, timeoutInMinutes := toMinutes(timerInMinutes)
	if err != nil {
		return err
	}

	timeoutInSeconds := timeoutInMinutes * 60
	timeOfTimeout := time.Now().Add(time.Minute * time.Duration(timeoutInMinutes)).Format("15:04")
	say.Debug(fmt.Sprintf("Starting break timer at %s for %d minutes = %d seconds (parsed from user input %s)", timeOfTimeout, timeoutInMinutes, timeoutInSeconds, timerInMinutes))

	room := getMobTimerRoom(configuration)
	startRemoteTimer := room != ""
	startLocalTimer := configuration.TimerLocal

	if !startRemoteTimer && !startLocalTimer {
		say.Error("No break timer configured, not starting break timer")
		Exit(1)
	}

	if startRemoteTimer {
		timerUser := getUserForMobTimer(configuration.TimerUser)
		err := httpPutBreakTimer(timeoutInMinutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure)

		if err != nil {
			say.Error("remote break timer couldn't be started")
			say.Error(err.Error())
			Exit(1)
		}
	}

	if startLocalTimer {
		err := executeCommandsInBackgroundProcess(getSleepCommand(timeoutInSeconds), getVoiceCommand("mob start", configuration.VoiceCommand), getNotifyCommand("mob start", configuration.NotifyCommand), "echo \"mobTimer\"")

		if err != nil {
			say.Error(fmt.Sprintf("break timer couldn't be started on your system (%s)", runtime.GOOS))
			say.Error(err.Error())
			Exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". So take a break now! :)")
	return nil
}

func getUserForMobTimer(userOverride string) string {
	if userOverride == "" {
		return gitUserName()
	}
	return userOverride
}

func toMinutes(timerInMinutes string) (error, int) {
	timeoutInMinutes, err := strconv.Atoi(timerInMinutes)
	if err != nil || timeoutInMinutes < 1 {
		say.Error(fmt.Sprintf("The parameter must be an integer number greater then zero"))
		return errors.New("The parameter must be an integer number greater then zero"), 0
	}
	return nil, timeoutInMinutes
}

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	_, err := httpclient.SendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
	return err
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	_, err := httpclient.SendRequest(putBody, "PUT", timerService+room, disableSSLVerification)
	return err
}

func getSleepCommand(timeoutInSeconds int) string {
	return fmt.Sprintf("sleep %d", timeoutInSeconds)
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
