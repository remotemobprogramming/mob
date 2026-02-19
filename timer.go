package main

import (
	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	timerpkg "github.com/remotemobprogramming/mob/v5/timer"
)

func StartTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startTimer(timerInMinutes string, configuration config.Configuration) error {
	room := getMobTimerRoom(configuration)
	timerUser := getUserForMobTimer(configuration.TimerUser)
	return timerpkg.RunTimer(timerInMinutes, room, timerUser, configuration)
}

func StartBreakTimer(timerInMinutes string, configuration config.Configuration) {
	if err := startBreakTimer(timerInMinutes, configuration); err != nil {
		exit.Exit(1)
	}
}

func startBreakTimer(timerInMinutes string, configuration config.Configuration) error {
	room := getMobTimerRoom(configuration)
	timerUser := getUserForMobTimer(configuration.TimerUser)
	return timerpkg.RunBreakTimer(timerInMinutes, room, timerUser, configuration)
}

func getMobTimerRoom(configuration config.Configuration) string {
	if !isGit() {
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
		return currentWipBranchQualifier
	}

	return configuration.TimerRoom
}

func getUserForMobTimer(userOverride string) string {
	if userOverride == "" {
		return gitUserName()
	}
	return userOverride
}

