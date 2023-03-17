package goal

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/say"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type GoalResponse struct {
	Goal string `json:"goal"`
}

type DeleteGoalRequest struct {
	User string `json:"user"`
}

type PutGoalRequest struct {
	Goal string `json:"goal"`
	User string `json:"user"`
}

func Goal(configuration config.Configuration, parameter []string) {
	if len(parameter) > 0 {
		if parameter[0] == "--delete" {
			deleteCurrentGoal(configuration)
		} else {
			setNewGoal(configuration, strings.Join(parameter, " "))
		}
	} else {
		showGoal(configuration)
	}
}

func setNewGoal(configuration config.Configuration, goal string) {
	if err := putGoalHttp(goal, configuration); err != nil {
		say.Error("Could not set new goal. An error happened while sending the request.")
		say.Debug(err.Error())
		return
	}
	say.Info(fmt.Sprintf("Set new goal to \"%s\"", goal))
}

func putGoalHttp(goal string, configuration config.Configuration) error {
	requestBody, err := json.Marshal(PutGoalRequest{Goal: goal, User: configuration.TimerUser})
	if err != nil {
		return err
	}
	_, err = sendRequest(requestBody, "PUT", getGoalUrl(configuration), configuration.TimerInsecure)
	return err
}

func getGoalUrl(configuration config.Configuration) string {
	return configuration.TimerUrl + configuration.TimerRoom + "/goal"
}

func deleteCurrentGoal(configuration config.Configuration) {
	err := deleteGoalHttp(configuration.TimerRoom, configuration.TimerUser, configuration.TimerUrl, configuration.TimerInsecure)
	if err != nil {
		say.Error("Could not delete goal. An error happened while sending the request.")
		say.Debug(err.Error())
		return
	}
	say.Info("Current goal has been deleted!")
}

func deleteGoalHttp(room string, user string, timerService string, disableSslVerification bool) error {
	requestBody, err := json.Marshal(DeleteGoalRequest{User: user})
	if err != nil {
		return err
	}
	_, err = sendRequest(requestBody, "DELETE", timerService+room+"/goal", disableSslVerification)
	return err
}

func showGoal(configuration config.Configuration) {
	goal := getGoalHttp(configuration.TimerRoom, configuration.TimerUrl, configuration.TimerInsecure)
	if goal == "" {
		say.Fix("No goal set. To set a goal, use", configuration.Mob("goal <your awesome goal>"))
		return
	}
	say.Info(goal)
}
func getGoalHttp(room string, timerService string, disableSslVerification bool) string {
	response, err := getHttpClient(disableSslVerification).Get(timerService + room + "/goal")
	if err != nil {
		say.Error("Could not get goal, got an error while requesting it!")
		say.Debug(err.Error())
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		say.Error("Could not get goal, got an error while requesting it!")
		say.Debug(err.Error())
	}
	var goalResponse GoalResponse
	if err := json.Unmarshal(body, &goalResponse); err != nil {
		say.Error("Could not get goal, got an error while parsing response")
		say.Debug(err.Error())
	}
	return goalResponse.Goal
}

func getHttpClient(disableSSLVerification bool) *http.Client {
	if disableSSLVerification {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return &http.Client{Transport: transCfg}
	}
	return http.DefaultClient
}
func sendRequest(requestBody []byte, requestMethod string, requestUrl string, disableSSLVerification bool) (string, error) {
	say.Info(requestMethod + " " + requestUrl + " " + string(requestBody))

	responseBody := bytes.NewBuffer(requestBody)
	request, requestCreationError := http.NewRequest(requestMethod, requestUrl, responseBody)

	httpClient := http.DefaultClient
	if disableSSLVerification {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient = &http.Client{Transport: transCfg}
	}

	if requestCreationError != nil {
		return "", fmt.Errorf("failed to create the http request object: %w", requestCreationError)
	}

	request.Header.Set("Content-Type", "application/json")
	response, responseErr := httpClient.Do(request)
	defer response.Body.Close()
	bodyBytes, responseReadingErr := ioutil.ReadAll(response.Body)
	body := string(bodyBytes)
	if responseReadingErr != nil {
		return "", fmt.Errorf("failed to read the http response: %w", responseReadingErr)
	}

	if e, ok := responseErr.(*url.Error); ok {
		switch e.Err.(type) {
		case x509.UnknownAuthorityError:
			say.Error("The timer.mob.sh SSL certificate is signed by an unknown authority!")
			say.Fix("HINT: You can ignore that by adding MOB_TIMER_INSECURE=true to your configuration or environment.",
				"echo MOB_TIMER_INSECURE=true >> ~/.mob")
			return body, fmt.Errorf("failed, to amke the http request: %w", responseErr)

		default:
			return body, fmt.Errorf("failed to make the http request: %w", responseErr)

		}
	}

	if responseErr != nil {
		return body, fmt.Errorf("failed to make the http request: %w", responseErr)
	}
	if string(body) != "" {
		say.Info(string(body))
	}
	say.Info(response.Status)
	return body, nil
}
