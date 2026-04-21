package localtimer_test

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/test"
	"github.com/remotemobprogramming/mob/v5/timer/localtimer"
)

func TestIsActiveWhenTimerLocalTrue(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = true

	timer := localtimer.NewProcessLocalTimer(cfg)

	test.Equals(t, true, timer.IsActive())
}

func TestIsInactiveWhenTimerLocalFalse(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = false

	timer := localtimer.NewProcessLocalTimer(cfg)

	test.Equals(t, false, timer.IsActive())
}

func TestVoiceCommandReturnsEmptyWhenCommandNotConfigured(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "")

	test.Equals(t, "", result)
}

func TestVoiceCommandInjectsMessageWithPlaceholder(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "say %s")

	test.Equals(t, "say mob next", result)
}

func TestVoiceCommandAppendsMessageWithoutPlaceholder(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "say")

	test.Equals(t, "say mob next", result)
}
