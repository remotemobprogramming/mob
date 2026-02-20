package webtimer

import (
	"encoding/json"
	"fmt"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/httpclient"
	"github.com/remotemobprogramming/mob/v5/timer"
)

func init() {
	timer.Register(func(configuration config.Configuration) timer.Timer {
		return NewWebTimer(configuration)
	})
}

// WebTimer is a Timer implementation that notifies a remote timer service via HTTP.
type WebTimer struct {
	room          string
	timerUser     string
	timerUrl      string
	timerInsecure bool
}

func NewWebTimer(configuration config.Configuration) WebTimer {
	return WebTimer{
		room:          configuration.TimerRoom,
		timerUser:     configuration.TimerUser,
		timerUrl:      configuration.TimerUrl,
		timerInsecure: configuration.TimerInsecure,
	}
}

func (t WebTimer) IsActive() bool {
	return t.room != ""
}

func (t WebTimer) StartTimer(minutes int) error {
	if err := httpPutTimer(minutes, t.room, t.timerUser, t.timerUrl, t.timerInsecure); err != nil {
		return fmt.Errorf("remote timer couldn't be started: %w", err)
	}
	return nil
}

func (t WebTimer) StartBreakTimer(minutes int) error {
	if err := httpPutBreakTimer(minutes, t.room, t.timerUser, t.timerUrl, t.timerInsecure); err != nil {
		return fmt.Errorf("remote break timer couldn't be started: %w", err)
	}
	return nil
}

func httpPutTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"timer": timeoutInMinutes,
		"user":  user,
	})
	client := httpclient.CreateHttpClient(disableSSLVerification)
	_, err := client.SendRequest(putBody, "PUT", timerService+room)
	return err
}

func httpPutBreakTimer(timeoutInMinutes int, room string, user string, timerService string, disableSSLVerification bool) error {
	putBody, _ := json.Marshal(map[string]interface{}{
		"breaktimer": timeoutInMinutes,
		"user":       user,
	})
	client := httpclient.CreateHttpClient(disableSSLVerification)
	_, err := client.SendRequest(putBody, "PUT", timerService+room)
	return err
}
