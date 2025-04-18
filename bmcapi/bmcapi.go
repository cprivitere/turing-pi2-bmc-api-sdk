package bmcapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	// BMCAPIURL is the default base URL for the Turing PI 2
	tpiDefaultURL = "https://turingpi.local"
)

type bmcApiAuth struct {
	AccessToken string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Username    string `json:"username"`
	Password    string `json:"password"` // Password for basic auth
}

// BMCAPI is a struct that holds the base URL and HTTP client for making API requests.
type BMCAPI struct {
	auth     *bmcApiAuth
	BaseURL  string
	Client   *http.Client
	AuthType string
}

type bmcOther struct {
	API          string `json:"api"`
	BuildVersion string `json:"build_version"`
	Buildroot    string `json:"buildroot"`
	Buildtime    string `json:"buildtime"`
	IP           string `json:"ip"`
	MAC          string `json:"mac"`
	Version      string `json:"version"`
}

// NewBMCAPI creates a new instance of BMCAPI with the given base URL and HTTP client.
func NewBMCAPI(baseURL, authType, username, password string, client *http.Client) (*BMCAPI, error) {

	// Try default Turing Pi 2 URL if baseURL is empty
	if baseURL == "" {
		baseURL = tpiDefaultURL
	}

	var authResponse bmcApiAuth

	if authType != "basic" && authType != "bearer" {
		return nil, errors.New("invalid auth type: " + authType)
	}

	if authType == "bearer" {

		req, err := http.NewRequest("GET", baseURL+"/api/bmc/authenticate", nil)
		if err != nil {
			return nil, fmt.Errorf("Error creating authentication request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")

		req.Body = io.NopCloser(strings.NewReader("{\"username\":\"" + username + "\",\"password\":\"" + password + "\"}"))

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Error making request: %w", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Error Authenticating: %s", resp.Status)
		}

		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %w", err)
		}

		if err := json.Unmarshal(bodyBytes, &authResponse); err != nil {
			return nil, fmt.Errorf("error parsing json in /token response: %+v", err)
		}
		if authResponse.AccessToken == "" {
			return nil, fmt.Errorf("Authentication response does not contain an auth token")
		}

	} else if authType == "basic" {

		req, err := http.NewRequest("GET", baseURL+"/api/bmc?opt=get&type=info", nil)
		if err != nil {
			return nil, fmt.Errorf("Error creating authentication request: %w", err)
		}
		req.SetBasicAuth(username, password)

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("Error making authentication test request: %w", err)
		}

		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("Error from authentication test: %s", resp.Status)
		}

		// Store basic auth credentials in authResponse
		authResponse.Username = username
		authResponse.Password = password
	}

	return &BMCAPI{
		auth:     &authResponse,
		BaseURL:  baseURL,
		Client:   client,
		AuthType: authType,
	}, nil
}

func (b *BMCAPI) Other() (*bmcOther, error) {
	// Create a new http request to the get other endpoint
	req, err := http.NewRequest("GET", b.BaseURL+"/api/bmc?opt=get&type=other", nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating Get Other request: %w", err)
	}

	// Set the authorization headers
	if b.AuthType == "basic" {
		req.SetBasicAuth(b.auth.Username, b.auth.Password)
	} else if b.AuthType == "bearer" {
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+b.auth.AccessToken)
	}

	resp, err := b.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error making Get Other request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error in response: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	type bmcOtherResponse struct {
		Response []struct {
			Result []bmcOther `json:"result"`
		} `json:"response"`
	}

	var parsed bmcOtherResponse

	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing json in /token response: %+v", err)
	}
	if len(parsed.Response) == 0 || len(parsed.Response[0].Result) == 0 {
		return nil, fmt.Errorf("no data in response")
	}
	return &parsed.Response[0].Result[0], nil

}
