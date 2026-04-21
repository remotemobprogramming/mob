package localtimer

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
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
