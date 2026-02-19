package main

// LocalTimer abstracts the local timer functionality so different implementations can be used.
type LocalTimer interface {
	StartTimer(timeoutInMinutes int, voiceMessage string, voiceCommand string, notifyMessage string, notifyCommand string) error
	StartBreakTimer(timeoutInMinutes int, voiceCommand string, notifyCommand string) error
}

// ProcessLocalTimer is the default LocalTimer implementation that uses background OS processes.
type ProcessLocalTimer struct{}

func (t ProcessLocalTimer) StartTimer(timeoutInMinutes int, voiceMessage string, voiceCommand string, notifyMessage string, notifyCommand string) error {
	timeoutInSeconds := timeoutInMinutes * 60
	return executeCommandsInBackgroundProcess(
		getSleepCommand(timeoutInSeconds),
		getVoiceCommand(voiceMessage, voiceCommand),
		getNotifyCommand(notifyMessage, notifyCommand),
		"echo \"mobTimer\"",
	)
}

func (t ProcessLocalTimer) StartBreakTimer(timeoutInMinutes int, voiceCommand string, notifyCommand string) error {
	timeoutInSeconds := timeoutInMinutes * 60
	return executeCommandsInBackgroundProcess(
		getSleepCommand(timeoutInSeconds),
		getVoiceCommand("mob start", voiceCommand),
		getNotifyCommand("mob start", notifyCommand),
		"echo \"mobTimer\"",
	)
}
