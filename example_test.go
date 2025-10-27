package nut

import (
	"fmt"
	"log"
)

// This example demonstrates how to connect to NUT, authenticate, and list UPS devices.
func ExampleConnect() {
	// Connect to NUT server (typically running on port 3493)
	client, err := Connect("127.0.0.1")
	if err != nil {
		log.Fatal(err)
	}

	// Authenticate
	authenticated, err := client.Authenticate("username", "password")
	if err != nil || !authenticated {
		log.Fatal("authentication failed")
	}

	// Get list of UPS devices
	upsList, err := client.GetUPSList()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("Available UPS devices:", len(upsList))
	if len(upsList) > 0 {
		fmt.Println("First UPS:", upsList[0].Name)
	}

	// Clean disconnect
	client.Disconnect()
}
