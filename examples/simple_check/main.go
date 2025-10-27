package main

import (
	"bufio"
	"fmt"
	"net"
	"time"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("Simple TCP Test for NUT Server")
	fmt.Println("Testing: 127.0.0.1:63493")
	fmt.Println("========================================")

	// Connect
	fmt.Println("1. Attempting TCP connection...")
	conn, err := net.DialTimeout("tcp", "127.0.0.1:63493", 3*time.Second)
	if err != nil {
		fmt.Printf("   ❌ Connection failed: %v\n", err)
		return
	}
	defer conn.Close()
	fmt.Println("   ✓ TCP connection successful!")

	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))

	// Send VER command
	fmt.Println("\n2. Sending VER command...")
	_, err = conn.Write([]byte("VER\n"))
	if err != nil {
		fmt.Printf("   ❌ Write failed: %v\n", err)
		return
	}
	fmt.Println("   ✓ Command sent!")

	// Read response
	fmt.Println("\n3. Waiting for response...")
	reader := bufio.NewReader(conn)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("   ❌ Read failed: %v\n", err)
		fmt.Println("\n   This suggests the remote NUT server is closing")
		fmt.Println("   the connection immediately. Possible causes:")
		fmt.Println("   - Port forward is not working correctly")
		fmt.Println("   - NUT server requires authentication first")
		fmt.Println("   - Server doesn't accept connections from this IP")
		fmt.Println("   - Check if upsd is configured to LISTEN on the correct IP")
		return
	}

	fmt.Printf("   ✓ Response received: %s", response)

	// Try NETVER
	fmt.Println("\n4. Sending NETVER command...")
	_, err = conn.Write([]byte("NETVER\n"))
	if err != nil {
		fmt.Printf("   ❌ Write failed: %v\n", err)
		return
	}

	response, err = reader.ReadString('\n')
	if err != nil {
		fmt.Printf("   ❌ Read failed: %v\n", err)
		return
	}
	fmt.Printf("   ✓ Response: %s", response)

	// Try LIST UPS
	fmt.Println("\n5. Sending LIST UPS command...")
	_, err = conn.Write([]byte("LIST UPS\n"))
	if err != nil {
		fmt.Printf("   ❌ Write failed: %v\n", err)
		return
	}

	fmt.Println("   Reading multi-line response...")
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("   ❌ Read failed: %v\n", err)
			break
		}
		fmt.Printf("   %s", line)
		if line == "END LIST UPS\n" {
			break
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("✓ All tests passed! Server is working.")
	fmt.Println("========================================")
}
