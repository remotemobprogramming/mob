package localtimer

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/exit"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/workdir"
)

// ProcessLocalTimer is a Timer implementation that uses background OS processes.
type ProcessLocalTimer struct {
	configuration config.Configuration
}

func NewProcessLocalTimer(configuration config.Configuration) ProcessLocalTimer {
	return ProcessLocalTimer{configuration: configuration}
}

func (t ProcessLocalTimer) IsActive() bool {
	return t.configuration.TimerLocal
}

func (t ProcessLocalTimer) StartTimer(minutes int) error {
	timeoutInSeconds := minutes * 60
	if err := executeCommandsInBackgroundProcess(
		sleepCommand(timeoutInSeconds),
		voiceCommand(t.configuration.VoiceMessage, t.configuration.VoiceCommand),
		notifyCommand(t.configuration.NotifyMessage, t.configuration.NotifyCommand),
		"echo \"mobTimer\"",
	); err != nil {
		return fmt.Errorf("timer couldn't be started on your system (%s): %w", runtime.GOOS, err)
	}
	return nil
}

func (t ProcessLocalTimer) StartBreakTimer(minutes int) error {
	timeoutInSeconds := minutes * 60
	if err := executeCommandsInBackgroundProcess(
		sleepCommand(timeoutInSeconds),
		voiceCommand("mob start", t.configuration.VoiceCommand),
		notifyCommand("mob start", t.configuration.NotifyCommand),
		"echo \"mobTimer\"",
	); err != nil {
		return fmt.Errorf("break timer couldn't be started on your system (%s): %w", runtime.GOOS, err)
	}
	return nil
}

func Moo(configuration config.Configuration) {
	voiceMessage := "moo"
	err := executeCommandsInBackgroundProcess(voiceCommand(voiceMessage, configuration.VoiceCommand))
	if err != nil {
		say.Warning(fmt.Sprintf("can't run voice command on your system (%s)", runtime.GOOS))
		say.Warning(err.Error())
		return
	}
	say.Info(voiceMessage)
}

func sleepCommand(timeoutInSeconds int) string {
	return fmt.Sprintf("sleep %d", timeoutInSeconds)
}

func voiceCommand(message string, voiceCommand string) string {
	if len(voiceCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(voiceCommand, message)
}

func notifyCommand(message string, notifyCommand string) string {
	if len(notifyCommand) == 0 {
		return ""
	}
	return injectCommandWithMessage(notifyCommand, message)
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

func executeCommandsInBackgroundProcess(commands ...string) error {
	cmds := make([]string, 0)
	for _, c := range commands {
		if len(c) > 0 {
			cmds = append(cmds, c)
		}
	}
	say.Debug(fmt.Sprintf("Operating System %s", runtime.GOOS))
	var err error
	switch runtime.GOOS {
	case "windows":
		err = runInBackground("powershell", "-command", fmt.Sprintf("start-process powershell -NoNewWindow -ArgumentList '-command \"%s\"'", strings.Join(cmds, ";")))
	case "darwin", "linux":
		err = runInBackground("sh", "-c", fmt.Sprintf("(%s) &", strings.Join(cmds, ";")))
	default:
		say.Warning(fmt.Sprintf("Cannot execute background commands on your os: %s", runtime.GOOS))
	}
	return err
}

func runInBackground(name string, args ...string) error {
	command := exec.Command(name, args...)
	if len(workdir.Path) > 0 {
		command.Dir = workdir.Path
	}
	commandString := strings.Join(command.Args, " ")
	say.Debug("Starting command " + commandString)
	return command.Start()
}
