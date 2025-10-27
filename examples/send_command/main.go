package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	nut "github.com/bearx3f/go.nut"
)

func main() {
	// Create logger for debugging
	logger := log.New(os.Stdout, "[NUT] ", log.LstdFlags)

	// Get credentials from user
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("========================================")
	fmt.Println("NUT UPS Command Test")
	fmt.Println("========================================")

	fmt.Print("Enter NUT username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Enter NUT password: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	if username == "" || password == "" {
		fmt.Println("Error: Username and password are required")
		return
	}

	ctx := context.Background()

	// Connect with logging enabled
	fmt.Println("\nConnecting to NUT server at 127.0.0.1:63493...")
	client, err := nut.ConnectWithOptionsAndConfig(ctx, "127.0.0.1", []nut.ClientOption{
		nut.WithLogger(logger),
		nut.WithConnectTimeout(5 * time.Second),
		nut.WithReadTimeout(3 * time.Second),
	}, 63493)

	if err != nil {
		logger.Fatalf("Failed to connect: %v", err)
	}
	defer client.Disconnect()

	// Authenticate
	logger.Println("Authenticating...")
	authenticated, err := client.Authenticate(username, password)
	if err != nil {
		logger.Fatalf("Authentication error: %v", err)
	}
	if !authenticated {
		logger.Fatalf("Authentication failed: invalid credentials")
	}
	logger.Println("✓ Authentication successful!")

	// List all UPS devices
	logger.Println("\nGetting UPS list...")
	upsList, err := client.GetUPSList()
	if err != nil {
		logger.Fatalf("Failed to get UPS list: %v", err)
	}

	if len(upsList) == 0 {
		logger.Fatalf("No UPS devices found")
	}

	// Use the first UPS
	ups := upsList[0]
	logger.Printf("\n=== Using UPS: %s ===", ups.Name)
	logger.Printf("Description: %s\n", ups.Description)

	// Get available commands
	logger.Println("Fetching available commands...")
	commands, err := ups.GetCommands()
	if err != nil {
		logger.Fatalf("Failed to get commands: %v", err)
	}

	logger.Printf("Available commands (%d):", len(commands))
	for i, cmd := range commands {
		logger.Printf("  [%d] %s - %s", i+1, cmd.Name, cmd.Description)
	}

	// Try to send test.battery.start command
	fmt.Println("\n========================================")
	fmt.Print("Send test.battery.start command? (yes/no): ")
	confirm, _ := reader.ReadString('\n')
	confirm = strings.TrimSpace(strings.ToLower(confirm))

	if confirm != "yes" && confirm != "y" {
		logger.Println("Command cancelled by user")
		return
	}

	logger.Println("\n⚡ Sending command: test.battery.start")
	success, err := ups.SendCommand("test.battery.start")
	if err != nil {
		logger.Printf("❌ Command failed: %v", err)

		// Try to check if we have master privileges
		logger.Println("\nChecking master privileges...")
		isMaster, _ := ups.CheckIfMaster()
		if !isMaster {
			logger.Println("⚠️  You don't have MASTER privileges")
			logger.Println("   To set master, the user needs 'upsmon master' in upsd.users")
			logger.Println("   Or appropriate INSTCMD permissions")
		}
		return
	}

	if success {
		logger.Println("✓ Command executed successfully!")
		logger.Println("\n📊 Monitor the UPS status to see the battery test progress")
		logger.Println("   You can check 'ups.status' variable")

		// Get current status
		logger.Println("\nFetching current UPS status...")
		vars, err := ups.GetVariables()
		if err == nil {
			for _, v := range vars {
				if v.Name == "ups.status" {
					logger.Printf("   Current status: %v", v.Value)
					break
				}
			}
		}
	} else {
		logger.Println("❌ Command was not successful (no confirmation from server)")
	}

	fmt.Println("\n========================================")
	fmt.Println("Test completed!")
	fmt.Println("========================================")
}
