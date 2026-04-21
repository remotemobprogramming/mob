package localtimer_test

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/timer/localtimer"
)

func TestIsActiveWhenTimerLocalTrue(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = true

	timer := localtimer.NewProcessLocalTimer(cfg)

	if !timer.IsActive() {
		t.Error("expected timer to be active when TimerLocal is true")
	}
}

func TestIsInactiveWhenTimerLocalFalse(t *testing.T) {
	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = false

	timer := localtimer.NewProcessLocalTimer(cfg)

	if timer.IsActive() {
		t.Error("expected timer to be inactive when TimerLocal is false")
	}
}

func TestVoiceCommandReturnsEmptyWhenCommandNotConfigured(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "")

	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestVoiceCommandInjectsMessageWithPlaceholder(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "say %s")

	if result != "say mob next" {
		t.Errorf("expected %q, got %q", "say mob next", result)
	}
}

func TestVoiceCommandAppendsMessageWithoutPlaceholder(t *testing.T) {
	result := localtimer.VoiceCommand("mob next", "say")

	if result != "say mob next" {
		t.Errorf("expected %q, got %q", "say mob next", result)
	}
}
