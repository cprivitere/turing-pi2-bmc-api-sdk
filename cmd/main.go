package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"os"

	"github.com/cprivitere/turing-pi2-bmc-api-sdk/bmcapi"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: program <username> <password>")
		return
	}
	username := os.Args[1]
	password := os.Args[2]

	// Example usage of NewBMCAPI function with bearer auth
	// Note: The baseURL, authType, username, and password should be replaced with actual values.
	baseURL := "https://turingpi.local"
	authType := "bearer"
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // Skip TLS verification for self-signed certs
		}}

	bmcClient, err := bmcapi.NewBMCAPI(baseURL, authType, username, password, client)
	if err != nil {
		fmt.Println("Error creating BMCAPI:", err)
		return
	}

	otherInfo, err := bmcClient.Other()
	if err != nil {
		fmt.Println("Error getting other info:", err)
		return
	}

	fmt.Println("API:", otherInfo.API)
	fmt.Println("Build Version:", otherInfo.BuildVersion)
	fmt.Println("Buildroot:", otherInfo.Buildroot)
	fmt.Println("Buildtime:", otherInfo.Buildtime)
	fmt.Println("IP:", otherInfo.IP)
	fmt.Println("MAC:", otherInfo.MAC)
	fmt.Println("Version:", otherInfo.Version)

}
