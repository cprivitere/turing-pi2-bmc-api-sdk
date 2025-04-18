package bmcapi

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// mockOther implements http.RoundTripper for testing
// It returns a canned response for the /api/bmc?opt=get&type=other endpoint

type bmcOtherResponse struct {
	Response []struct {
		Result []bmcOther `json:"result"`
	} `json:"response"`
}

type mockOther struct{}

func (m *mockOther) RoundTrip(req *http.Request) (*http.Response, error) {
	if strings.Contains(req.URL.String(), "/api/bmc?opt=get&type=other") {
		jsonResp := `{"response":[{"result":[{"api":"1.1","build_version":"2024.05.1","buildroot":"\"Buildroot 2024.05.1\"","buildtime":"2025-01-17 17:12:52-00:00","ip":"Unknown","mac":"Unknown","version":"2.3.4"}]}]}`
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(jsonResp)),
			Header:     make(http.Header),
		}, nil
	}
	return &http.Response{StatusCode: http.StatusNotFound, Body: io.NopCloser(strings.NewReader("")), Header: make(http.Header)}, nil
}

func TestBMCAPI_Other(t *testing.T) {
	mockClient := &http.Client{Transport: &mockOther{}}
	bmc := &BMCAPI{
		auth:     &bmcApiAuth{Username: "user", Password: "pass"},
		BaseURL:  "http://mock",
		Client:   mockClient,
		AuthType: "basic",
	}
	want := bmcOther{
		API:          "1.1",
		Version:      "2.3.4",
		Buildtime:    "2025-01-17 17:12:52-00:00",
		IP:           "Unknown",
		MAC:          "Unknown",
		BuildVersion: "2024.05.1",
		Buildroot:    "\"Buildroot 2024.05.1\"",
	}
	t.Run("success", func(t *testing.T) {
		// Simulate parsing the nested response
		resp, err := mockClient.Get(bmc.BaseURL + "/api/bmc?opt=get&type=other")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer resp.Body.Close()
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("error reading body: %v", err)
		}
		var parsed bmcOtherResponse
		if err := json.Unmarshal(bodyBytes, &parsed); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if len(parsed.Response) == 0 || len(parsed.Response[0].Result) == 0 {
			t.Fatalf("no data in response")
		}
		got := parsed.Response[0].Result[0]
		if !reflect.DeepEqual(got, want) {
			t.Errorf("BMCAPI.Other() = %v, want %v", got, want)
		}
	})
}
