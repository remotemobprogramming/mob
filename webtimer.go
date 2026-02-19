package main

import (
	"encoding/json"
	"fmt"

	config "github.com/remotemobprogramming/mob/v5/configuration"
	"github.com/remotemobprogramming/mob/v5/httpclient"
)

// WebTimer is a Timer implementation that notifies a remote timer service via HTTP.
type WebTimer struct{}

func (t WebTimer) StartTimer(minutes int, configuration config.Configuration) error {
	room := getMobTimerRoom(configuration)
	timerUser := getUserForMobTimer(configuration.TimerUser)
	if err := httpPutTimer(minutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure); err != nil {
		return fmt.Errorf("remote timer couldn't be started: %w", err)
	}
	return nil
}

func (t WebTimer) StartBreakTimer(minutes int, configuration config.Configuration) error {
	room := getMobTimerRoom(configuration)
	timerUser := getUserForMobTimer(configuration.TimerUser)
	if err := httpPutBreakTimer(minutes, room, timerUser, configuration.TimerUrl, configuration.TimerInsecure); err != nil {
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
