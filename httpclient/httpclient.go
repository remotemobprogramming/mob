package httpclient

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"github.com/remotemobprogramming/mob/v5/say"
	"io/ioutil"
	"net/http"
	"net/url"
)

func GetHttpClient(disableSSLVerification bool) *http.Client {
	if disableSSLVerification {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return &http.Client{Transport: transCfg}
	}
	return http.DefaultClient
}

func SendRequest(requestBody []byte, requestMethod string, requestUrl string, disableSSLVerification bool) (string, error) {
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
	if e, ok := responseErr.(*url.Error); ok {
		switch e.Err.(type) {
		case x509.UnknownAuthorityError:
			say.Error("The timer.mob.sh SSL certificate is signed by an unknown authority!")
			say.Fix("HINT: You can ignore that by adding MOB_TIMER_INSECURE=true to your configuration or environment.",
				"echo MOB_TIMER_INSECURE=true >> ~/.mob")
			return "", fmt.Errorf("failed, to make the http request: %w", responseErr)

		default:
			return "", fmt.Errorf("failed to make the http request: %w", responseErr)

		}
	}

	if responseErr != nil {
		return "", fmt.Errorf("failed to make the http request: %w", responseErr)
	}
	if response.StatusCode >= 300 {
		return "", errors.New("got an error from the server: " + requestUrl + " " + response.Status)
	}
	defer response.Body.Close()
	bodyBytes, responseReadingErr := ioutil.ReadAll(response.Body)
	body := string(bodyBytes)
	if responseReadingErr != nil {
		return "", fmt.Errorf("failed to read the http response: %w", responseReadingErr)
	}
	if string(body) != "" {
		say.Info(body)
	}
	return body, nil
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
