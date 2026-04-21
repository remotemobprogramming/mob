package timer_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/timer"
)

func TestRunTimerReturnsErrorForZeroMinutes(t *testing.T) {
	cfg := config.GetDefaultConfiguration()

	err := timer.RunTimer("0", cfg)

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunTimerReturnsErrorForNonNumericInput(t *testing.T) {
	cfg := config.GetDefaultConfiguration()

	err := timer.RunTimer("NotANumber", cfg)

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestRunBreakTimerReturnsErrorForZeroMinutes(t *testing.T) {
	cfg := config.GetDefaultConfiguration()

	err := timer.RunBreakTimer("0", cfg)

	if err == nil {
		t.Error("expected error for zero minutes")
	}
}

func TestRunBreakTimerReturnsErrorForNonNumericInput(t *testing.T) {
	cfg := config.GetDefaultConfiguration()

	err := timer.RunBreakTimer("NotANumber", cfg)

	if err == nil {
		t.Error("expected error for non-numeric input")
	}
}

func TestRunTimerSucceedsWithWebTimer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = false
	cfg.TimerRoom = "testroom"
	cfg.TimerUrl = server.URL + "/"

	err := timer.RunTimer("1", cfg)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunBreakTimerSucceedsWithWebTimer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := config.GetDefaultConfiguration()
	cfg.TimerLocal = false
	cfg.TimerRoom = "testroom"
	cfg.TimerUrl = server.URL + "/"

	err := timer.RunBreakTimer("1", cfg)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
