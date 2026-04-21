package timer

import (
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
)

type mockTimer struct {
	active                bool
	startTimerCalled      bool
	startBreakTimerCalled bool
}

func (m *mockTimer) IsActive() bool { return m.active }
func (m *mockTimer) StartTimer(_ int) error {
	m.startTimerCalled = true
	return nil
}
func (m *mockTimer) StartBreakTimer(_ int) error {
	m.startBreakTimerCalled = true
	return nil
}

func TestGetActiveTimerReturnsFirstActiveTimer(t *testing.T) {
	inactive := &mockTimer{active: false}
	active := &mockTimer{active: true}

	result := getActiveTimer([]Timer{inactive, active})

	if result != active {
		t.Error("expected the first active timer to be returned")
	}
}

func TestGetActiveTimerReturnsNilWhenNoneActive(t *testing.T) {
	result := getActiveTimer([]Timer{&mockTimer{active: false}})

	if result != nil {
		t.Error("expected nil when no timer is active")
	}
}

func TestGetActiveTimerPrefersFirstOverSecond(t *testing.T) {
	first := &mockTimer{active: true}
	second := &mockTimer{active: true}

	result := getActiveTimer([]Timer{first, second})

	if result != first {
		t.Error("expected the first active timer to take priority")
	}
}

func TestRunTimerReturnsErrorForZeroMinutes(t *testing.T) {
	err := RunTimer("0", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunTimerReturnsErrorForNonNumericInput(t *testing.T) {
	err := RunTimer("NotANumber", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestRunBreakTimerReturnsErrorForZeroMinutes(t *testing.T) {
	err := RunBreakTimer("0", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunBreakTimerReturnsErrorForNonNumericInput(t *testing.T) {
	err := RunBreakTimer("NotANumber", config.GetDefaultConfiguration())

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}
