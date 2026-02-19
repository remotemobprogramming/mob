package main

import config "github.com/remotemobprogramming/mob/v5/configuration"

// Timer abstracts the local timer functionality so different implementations can be used.
type Timer interface {
	StartTimer(minutes int, configuration config.Configuration) error
	StartBreakTimer(minutes int, configuration config.Configuration) error
}

// ProcessLocalTimer is the default Timer implementation that uses background OS processes.
type ProcessLocalTimer struct{}

func (t ProcessLocalTimer) StartTimer(minutes int, configuration config.Configuration) error {
	timeoutInSeconds := minutes * 60
	return executeCommandsInBackgroundProcess(
		getSleepCommand(timeoutInSeconds),
		getVoiceCommand(configuration.VoiceMessage, configuration.VoiceCommand),
		getNotifyCommand(configuration.NotifyMessage, configuration.NotifyCommand),
		"echo \"mobTimer\"",
	)
}

func (t ProcessLocalTimer) StartBreakTimer(minutes int, configuration config.Configuration) error {
	timeoutInSeconds := minutes * 60
	return executeCommandsInBackgroundProcess(
		getSleepCommand(timeoutInSeconds),
		getVoiceCommand("mob start", configuration.VoiceCommand),
		getNotifyCommand("mob start", configuration.NotifyCommand),
		"echo \"mobTimer\"",
	)
}
