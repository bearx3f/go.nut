package nut_test

import (
	"context"
	"fmt"
	"log"
	"time"

	nut "github.com/bearx3f/go.nut"
)

// ExamplePool demonstrates using connection pool for high-concurrency scenarios
func ExamplePool() {
	// Create a connection pool
	pool, err := nut.NewPool(nut.PoolConfig{
		Hostname: "localhost",
		Port:     3493,
		MaxSize:  10, // Maximum 10 connections
		ClientOptions: []nut.ClientOption{
			nut.WithConnectTimeout(5 * time.Second),
			nut.WithReadTimeout(2 * time.Second),
			nut.WithLogger(log.Default()),
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// Get a client from the pool
	ctx := context.Background()
	client, err := pool.Get(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Use the client
	upsList, err := client.GetUPSList()
	if err != nil {
		pool.Put(client) // Return even on error
		log.Fatal(err)
	}

	for _, ups := range upsList {
		fmt.Printf("UPS: %s\n", ups.Name)
	}

	// Return client to pool for reuse
	if err := pool.Put(client); err != nil {
		log.Printf("Failed to return client: %v", err)
	}

	// Check pool statistics
	idle, active := pool.Stats()
	fmt.Printf("Pool stats - Idle: %d, Active: %d\n", idle, active)
}

// ExampleClientMetrics demonstrates monitoring client metrics
func ExampleClientMetrics() {
	client, err := nut.Connect("localhost")
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Authenticate
	_, err = client.Authenticate("monuser", "secret")
	if err != nil {
		log.Fatal(err)
	}

	// Perform some operations
	upsList, err := client.GetUPSList()
	if err != nil {
		log.Fatal(err)
	}

	for _, ups := range upsList {
		vars, _ := ups.GetVariables()
		fmt.Printf("UPS %s has %d variables\n", ups.Name, len(vars))
	}

	// Get metrics
	metrics := client.GetMetrics()
	fmt.Printf("Commands sent: %d\n", metrics.CommandsSent)
	fmt.Printf("Commands failed: %d\n", metrics.CommandsFailed)
	fmt.Printf("Bytes sent: %d\n", metrics.BytesSent)
	fmt.Printf("Bytes received: %d\n", metrics.BytesReceived)
	fmt.Printf("Reconnects: %d\n", metrics.Reconnects)
}

// ExampleConnectWithOptions demonstrates using custom options
func ExampleConnectWithOptions() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create logger
	logger := log.New(log.Writer(), "[NUT] ", log.LstdFlags)

	// Connect with custom options
	client, err := nut.ConnectWithOptionsAndConfig(ctx, "localhost", []nut.ClientOption{
		nut.WithConnectTimeout(5 * time.Second),
		nut.WithReadTimeout(3 * time.Second),
		nut.WithLogger(logger),
	}, 3493)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Disconnect()

	// Use context for command execution
	resp, err := client.SendCommandWithContext(ctx, "VER")
	if err != nil {
		log.Fatal(err)
	}

	if len(resp) > 0 {
		fmt.Printf("Server version: %s\n", resp[0])
	}
}
