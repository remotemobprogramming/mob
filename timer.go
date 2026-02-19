package main

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/say"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
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

	timers := getTimers(configuration)
	if len(timers) == 0 {
		say.Error("No timer configured, not starting timer")
		exit.Exit(1)
	}

	for _, timer := range timers {
		if err := timer.StartTimer(timeoutInMinutes, configuration); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
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
	if err := startBreakTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
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

	timers := getTimers(configuration)
	if len(timers) == 0 {
		say.Error("No break timer configured, not starting break timer")
		exit.Exit(1)
	}

	for _, timer := range timers {
		if err := timer.StartBreakTimer(timeoutInMinutes, configuration); err != nil {
			say.Error(err.Error())
			exit.Exit(1)
		}
	}

	say.Info("It's now " + currentTime() + ". " + fmt.Sprintf("%d min break timer ends at approx. %s", timeoutInMinutes, timeOfTimeout) + ". So take a break now! :)")
	return nil
}

func getTimers(configuration config.Configuration) []Timer {
	var timers []Timer
	if getMobTimerRoom(configuration) != "" {
		timers = append(timers, WebTimer{})
	}
	if configuration.TimerLocal {
		timers = append(timers, ProcessLocalTimer{})
	}
	return timers
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
