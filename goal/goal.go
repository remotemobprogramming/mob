package goal

import (
	"encoding/json"
	"fmt"
	config "github.com/remotemobprogramming/mob/v4/configuration"
	"github.com/remotemobprogramming/mob/v4/httpclient"
	"github.com/remotemobprogramming/mob/v4/say"
	"io"
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
	_, err = httpclient.SendRequest(requestBody, "PUT", getGoalUrl(configuration), configuration.TimerInsecure)
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
	_, err = httpclient.SendRequest(requestBody, "DELETE", timerService+room+"/goal", disableSslVerification)
	return err
}

func showGoal(configuration config.Configuration) {
	goal, err := getGoalHttp(configuration.TimerRoom, configuration.TimerUrl, configuration.TimerInsecure)
	if err != nil {
		say.Error(err.Error())
		return
	}
	if goal == "" {
		say.Fix("No goal set. To set a goal, use", configuration.Mob("goal <your awesome goal>"))
		return
	}
	say.Info(goal)
}
func getGoalHttp(room string, timerService string, disableSslVerification bool) (string, error) {
	response, err := httpclient.GetHttpClient(disableSslVerification).Get(timerService + room + "/goal")
	if err != nil {
		say.Debug(err.Error())
		return "", fmt.Errorf("Could not get goal, got an error while requesting it: %w", err)
	}
	if response.StatusCode == 204 {
		return "", nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		say.Debug(err.Error())
		return "", fmt.Errorf("Could not get goal, got an error while requesting it: %w", err)
	}
	var goalResponse GoalResponse
	if err := json.Unmarshal(body, &goalResponse); err != nil {
		say.Debug(err.Error())
		return "", fmt.Errorf("Could not get goal, got an error while parsing response: %w", err)
	}
	return goalResponse.Goal, nil
}
