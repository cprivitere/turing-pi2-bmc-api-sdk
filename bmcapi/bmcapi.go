package bmcapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
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

// bmcResultAPIResponse is a struct that represents the response from the BMC API for a single result.
// It expects the response to be in the format {"response":[{"result":"<result>" }]}
type bmcResultAPIResponse struct {
	Response []struct {
		Result string `json:"result"`
	} `json:"response"`
}

// bmcObjectAPIResponse is a struct that represents the response from the BMC API for an object result.
// It expects the response to be in the format {"response":[{"result":[{<resultobject>}] }]}
type bmcObjectAPIResponse struct {
	Response []struct {
		Result []map[string]string `json:"result"`
	} `json:"response"`
}

type bmcOther struct {
	API          string
	BuildVersion string
	Buildroot    string
	Buildtime    string
	IP           string
	MAC          string
	Version      string
}

// NewBMCAPI creates a new instance of BMCAPI with the given base URL and HTTP client.
// Creates and uses the custom bmcOtherResponse struct to parse the response from the BMC API.
// It returns a bmcOther struct or an error if the authentication fails or if the request cannot be made.
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

	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=get&type=other")
	if err != nil {
		return nil, fmt.Errorf("error during USB Boot API call: %w", err)
	}

	result, err := b.objectAPIParse(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("error parsing response: %w", err)
	}

	bmcOther := bmcOther{
		API:          result["api"],
		BuildVersion: result["build_version"],
		Buildroot:    result["buildroot"],
		Buildtime:    result["buildtime"],
		IP:           result["ip"],
		MAC:          result["mac"],
		Version:      result["version"],
	}

	return &bmcOther, nil

}

// USBBoot sets the USB boot option for the specified node (0-3).
func (b *BMCAPI) USBBoot(node int) (*string, error) {

	// Validate node number
	if node < 0 || node > 3 {
		return nil, fmt.Errorf("node number must be between 0 and 3")
	}

	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=set&type=usb_boot&node=" + strconv.Itoa(node))
	if err != nil {
		return nil, fmt.Errorf("error during USB Boot API call: %w", err)
	}

	return b.resultAPIParse(bodyBytes)

}

// ClearUSBBoot clears the USB boot option for the specified node (0-3).
func (b *BMCAPI) ClearUSBBoot(node int) (*string, error) {

	// Validate node number
	if node < 0 || node > 3 {
		return nil, fmt.Errorf("node number must be between 0 and 3")
	}

	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=set&type=clear_usb_boot&node=" + strconv.Itoa(node))
	if err != nil {
		return nil, fmt.Errorf("error during Clear USB Boot API call: %w", err)
	}

	return b.resultAPIParse(bodyBytes)

}

// ResetNetwork resets the
func (b *BMCAPI) ResetNetwork() (*string, error) {
	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=set&type=network")
	if err != nil {
		return nil, fmt.Errorf("error during Reset Network Switch call: %w", err)
	}

	return b.resultAPIParse(bodyBytes)
}

// NodetoMSD reboots a node into USB Mass Storage Device (MSD) mode.
func (b *BMCAPI) NodetoMSD(node int) (*string, error) {
	// Validate node number
	if node < 0 || node > 3 {
		return nil, fmt.Errorf("node number must be between 0 and 3")
	}

	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=set&type=node_to_msd&node=" + strconv.Itoa(node))
	if err != nil {
		return nil, fmt.Errorf("error during Node to MSD call: %w", err)
	}

	return b.resultAPIParse(bodyBytes)

}

// SetPower sets power status of specified nodes.
// The powerState parameter should be 0 for off and 1 for on.
func (b *BMCAPI) SetPower(node, powerState int) (*string, error) {
	// Validate node number
	if node < 0 || node > 3 {
		return nil, fmt.Errorf("node number must be between 0 and 3")
	}
	// Validate powerState
	if powerState < 0 || powerState > 1 {
		return nil, fmt.Errorf("powerState must be 0 (off) or 1 (on)")
	}

	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=power&type=set&node" + strconv.Itoa(node) + "=" + strconv.Itoa(powerState))
	if err != nil {
		return nil, fmt.Errorf("error during Set Power call: %w", err)
	}

	return b.resultAPIParse(bodyBytes)
}

// GetPower Gets power status of all nodes.
func (b *BMCAPI) GetPower() (map[string]string, error) {
	bodyBytes, err := b.bmcAPICall("/api/bmc?opt=get&type=power")
	if err != nil {
		return nil, fmt.Errorf("error during Get Power call: %w", err)
	}

	// We want to return the whole map here, as it's a map of node numbers to power states
	// e.g. {
	// 	"node1": "1",
	// 	"node2": "0",
	// 	"node3": "1",
	// 	"node4": "0",
	// }
	return b.objectAPIParse(bodyBytes)

}

// bmcAPICall is a helper function that makes a GET request to the BMC API and returns the response body as a byte slice.
func (b *BMCAPI) bmcAPICall(endpoint string) ([]byte, error) {

	// Create a new http request to the get other endpoint
	req, err := http.NewRequest("GET", b.BaseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w", err)
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
		return nil, fmt.Errorf("Error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http error in response: %s", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, fmt.Errorf("error reading response body: %w", err)
	}

	return bodyBytes, nil

}

// resultAPIParse is a helper function that parses the response from the BMC API and returns the result as a map of strings.
// It expects the response to be in the format {"response":[{"result":"<result>" }]}
func (b *BMCAPI) resultAPIParse(bodyBytes []byte) (*string, error) {

	var parsed bmcResultAPIResponse

	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing json in token response: %+v", err)
	}

	result := parsed.Response[0].Result
	if result == "" {
		return nil, fmt.Errorf("result field in API response is empty")
	}

	return &result, nil

}

// objectAPIParse is a helper function that parses the response from the BMC API and returns the result as a map of strings.
// It expects the response to be in the format {"response":[{"result":[{<resultobject>}] }]}
func (b *BMCAPI) objectAPIParse(bodyBytes []byte) (map[string]string, error) {

	var parsed bmcObjectAPIResponse

	if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
		return nil, fmt.Errorf("error parsing json in token response: %+v", err)
	}
	if len(parsed.Response) == 0 || len(parsed.Response[0].Result) == 0 {
		return nil, fmt.Errorf("no data in response")
	}

	return parsed.Response[0].Result[0], nil

}
