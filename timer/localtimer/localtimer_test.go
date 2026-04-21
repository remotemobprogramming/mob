package localtimer

import (
	"path/filepath"
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/say"
	"github.com/remotemobprogramming/mob/v5/test"
)

func TestIsActiveWhenTimerLocalTrue(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = true

	timer := NewProcessLocalTimer(cfg)

	test.Equals(t, true, timer.IsActive())
}

func TestIsInactiveWhenTimerLocalFalse(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = false

	timer := NewProcessLocalTimer(cfg)

	test.Equals(t, false, timer.IsActive())
}

func TestVoiceCommandReturnsEmptyWhenCommandNotConfigured(t *testing.T) {
	result := voiceCommand("mob next", "")

	test.Equals(t, "", result)
}

func TestVoiceCommandInjectsMessageWithPlaceholder(t *testing.T) {
	result := voiceCommand("mob next", "say %s")

	test.Equals(t, "say mob next", result)
}

func TestVoiceCommandAppendsMessageWithoutPlaceholder(t *testing.T) {
	result := voiceCommand("mob next", "say")

	test.Equals(t, "say mob next", result)
}

func TestStartTimerExecutesBackgroundProcess(t *testing.T) {
	say.TurnOnDebugging()
	output := test.CaptureOutput(t)
	voiceFile := filepath.Join(t.TempDir(), "timer_voice")
	notifyFile := filepath.Join(t.TempDir(), "timer_notify")
	cfg := config.GetDefaultConfiguration()
	cfg.VoiceCommand = "touch " + voiceFile + "; true"
	cfg.NotifyCommand = "touch " + notifyFile + "; true"
	timer := NewProcessLocalTimer(cfg)

	err := timer.StartTimer(0)

	test.Equals(t, nil, err)
	test.AssertOutputContains(t, output, "Starting command")
	test.AwaitFileCreated(t, voiceFile)
	test.AwaitFileCreated(t, notifyFile)
}

func TestStartBreakTimerExecutesBackgroundProcess(t *testing.T) {
	say.TurnOnDebugging()
	output := test.CaptureOutput(t)
	voiceFile := filepath.Join(t.TempDir(), "break_timer_voice")
	notifyFile := filepath.Join(t.TempDir(), "break_timer_notify")
	cfg := config.GetDefaultConfiguration()
	cfg.VoiceCommand = "touch " + voiceFile + "; true"
	cfg.NotifyCommand = "touch " + notifyFile + "; true"
	timer := NewProcessLocalTimer(cfg)

	err := timer.StartBreakTimer(0)

	test.Equals(t, nil, err)
	test.AssertOutputContains(t, output, "Starting command")
	test.AwaitFileCreated(t, voiceFile)
	test.AwaitFileCreated(t, notifyFile)
}

func TestMooLogsInfoMessage(t *testing.T) {
	output := test.CaptureOutput(t)
	cfg := config.GetDefaultConfiguration()
	cfg.VoiceCommand = "echo"

	Moo(cfg)

	test.AssertOutputContains(t, output, "moo")
}
