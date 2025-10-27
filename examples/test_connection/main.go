package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	nut "github.com/bearx3f/go.nut"
)

func main() {
	// Create logger for debugging
	logger := log.New(os.Stdout, "[NUT] ", log.LstdFlags)

	// Try different hosts with port forwarded to 127.0.0.1:63493
	hosts := []struct {
		host string
		port int
	}{
		{"127.0.0.1", 63493},
		{"localhost", 63493},
	}

	for _, hostInfo := range hosts {
		logger.Printf("\n========================================")
		logger.Printf("Testing connection to: %s:%d", hostInfo.host, hostInfo.port)
		logger.Printf("========================================")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Connect with logging enabled
		client, err := nut.ConnectWithOptionsAndConfig(ctx, hostInfo.host, []nut.ClientOption{
			nut.WithLogger(logger),
			nut.WithConnectTimeout(5 * time.Second),
			nut.WithReadTimeout(3 * time.Second),
		}, hostInfo.port)

		if err != nil {
			logger.Printf("❌ Failed to connect to %s:%d: %v\n", hostInfo.host, hostInfo.port, err)
			continue
		}

		logger.Printf("✓ Connected successfully to %s:%d!", hostInfo.host, hostInfo.port)

		// List all UPS devices
		logger.Println("Getting UPS list...")
		upsList, err := client.GetUPSList()
		if err != nil {
			logger.Printf("Failed to get UPS list: %v", err)
			client.Disconnect()
			continue
		}

		logger.Printf("Found %d UPS device(s)", len(upsList))

		// For each UPS, try to get variables
		for i, ups := range upsList {
			logger.Printf("\n=== UPS #%d: %s ===", i+1, ups.Name)
			logger.Printf("Description: %s", ups.Description)
			logger.Printf("Number of logins: %d", ups.NumberOfLogins)

			// Now explicitly fetch variables
			logger.Println("\nFetching variables (this will show debug output)...")
			vars, err := ups.GetVariables()
			if err != nil {
				logger.Printf("❌ Error getting variables: %v", err)

				// Try to get just one variable type to see detailed error
				logger.Println("\nTrying to get type of first variable directly...")
				if len(vars) > 0 {
					varType, writeable, maxLen, typeErr := ups.GetVariableType(vars[0].Name)
					if typeErr != nil {
						logger.Printf("❌ GetVariableType error: %v", typeErr)
					} else {
						logger.Printf("✓ Type: %s, Writeable: %v, MaxLen: %d", varType, writeable, maxLen)
					}
				}
				continue
			}

			logger.Printf("✓ Successfully retrieved %d variables", len(vars))

			// Show first 10 variables as sample
			logger.Println("\nSample variables:")
			count := 10
			if len(vars) < count {
				count = len(vars)
			}
			for j := 0; j < count; j++ {
				v := vars[j]
				logger.Printf("  [%d] %s = %v (Type: %s, OrigType: %s, Writeable: %v, MaxLen: %d)",
					j+1, v.Name, v.Value, v.Type, v.OriginalType, v.Writeable, v.MaximumLength)
			}

			// Test GetCommands
			logger.Println("\nFetching commands...")
			cmds, err := ups.GetCommands()
			if err != nil {
				logger.Printf("❌ Error getting commands: %v", err)
			} else {
				logger.Printf("✓ Successfully retrieved %d commands", len(cmds))
				if len(cmds) > 0 {
					logger.Println("Sample commands:")
					for j := 0; j < len(cmds) && j < 5; j++ {
						logger.Printf("  [%d] %s - %s", j+1, cmds[j].Name, cmds[j].Description)
					}
				}
			}
		}

		client.Disconnect()
		logger.Printf("\n✓ Test completed for %s:%d\n", hostInfo.host, hostInfo.port)

		// If we successfully tested one host, we can stop
		break
	}

	fmt.Println("\n========================================")
	fmt.Println("All tests completed!")
	fmt.Println("========================================")
}
