# Optional Features Documentation

This document describes the optional features added to go.nut for production use.

## 1. Context Support

All connection and command operations now support context for cancellation and timeout control.

### Usage

```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Connect with context
client, err := nut.ConnectWithOptions(ctx, "localhost", 3493)
if err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Send command with context
resp, err := client.SendCommandWithContext(ctx, "LIST UPS")
if err != nil {
    log.Fatal(err)
}
```

### Benefits

- **Timeout Control**: Set maximum duration for operations
- **Cancellation**: Cancel long-running operations
- **Deadline Propagation**: Pass deadlines through call stack

## 2. Custom Timeout

Configure custom timeouts for both connection and read operations using the option pattern.

### Usage

```go
client, err := nut.ConnectWithOptionsAndConfig(
    context.Background(),
    "localhost",
    []nut.ClientOption{
        nut.WithConnectTimeout(5 * time.Second),
        nut.WithReadTimeout(3 * time.Second),
    },
    3493,
)
```

### Available Options

- `WithConnectTimeout(duration)`: Set connection establishment timeout
- `WithReadTimeout(duration)`: Set response read timeout
- `WithTLSConfig(config)`: Custom TLS configuration
- `WithLogger(logger)`: Enable debug logging

### Default Values

- Connect Timeout: 5 seconds
- Read Timeout: 2 seconds

## 3. Connection Pool

For high-concurrency scenarios, use the connection pool to reuse client connections.

### Creating a Pool

```go
pool, err := nut.NewPool(nut.PoolConfig{
    Hostname:  "localhost",
    Port:      3493,
    MaxSize:   10, // Maximum 10 concurrent connections
    ClientOptions: []nut.ClientOption{
        nut.WithConnectTimeout(5 * time.Second),
        nut.WithReadTimeout(2 * time.Second),
    },
})
if err != nil {
    log.Fatal(err)
}
defer pool.Close()
```

### Using the Pool

```go
// Get a client (creates new or reuses existing)
ctx := context.Background()
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

// Use the client
upsList, err := client.GetUPSList()

// IMPORTANT: Always return the client to the pool
pool.Put(client)
```

### Pool Statistics

```go
idle, active := pool.Stats()
fmt.Printf("Idle connections: %d, Active connections: %d\n", idle, active)
```

### Best Practices

1. **Always return clients**: Use `defer pool.Put(client)` or return in error paths
2. **Set appropriate MaxSize**: Based on expected concurrency and server limits
3. **Use context**: Pass context to Get() for timeout control
4. **Close the pool**: Call `pool.Close()` when shutting down

## 4. Metrics and Logging

### Metrics

Track client operations for monitoring and debugging:

```go
metrics := client.GetMetrics()
fmt.Printf("Commands sent: %d\n", metrics.CommandsSent)
fmt.Printf("Commands failed: %d\n", metrics.CommandsFailed)
fmt.Printf("Bytes sent: %d\n", metrics.BytesSent)
fmt.Printf("Bytes received: %d\n", metrics.BytesReceived)
fmt.Printf("Reconnects: %d\n", metrics.Reconnects)
```

#### Available Metrics

- `CommandsSent`: Total number of commands sent
- `CommandsFailed`: Number of failed commands (errors)
- `BytesSent`: Total bytes sent to server
- `BytesReceived`: Total bytes received from server
- `Reconnects`: Number of reconnection attempts
- `LastCommandTime`: Timestamp of last command (atomic.Value containing time.Time)

### Logging

Enable debug logging to trace operations:

```go
logger := log.New(os.Stdout, "[NUT] ", log.LstdFlags)

client, err := nut.ConnectWithOptionsAndConfig(
    context.Background(),
    "localhost",
    []nut.ClientOption{
        nut.WithLogger(logger),
    },
    3493,
)
```

#### Log Output Examples

```
[NUT] 2025/01/15 10:30:45 Connecting to localhost:3493 (timeout: 5s)
[NUT] 2025/01/15 10:30:45 Sent command: VER
[NUT] 2025/01/15 10:30:45 Sent command: LIST UPS
```

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/bearx3f/go.nut"
)

func main() {
    // Create logger
    logger := log.New(os.Stdout, "[NUT] ", log.LstdFlags)

    // Create connection pool
    pool, err := nut.NewPool(nut.PoolConfig{
        Hostname: "localhost",
        Port:     3493,
        MaxSize:  10,
        ClientOptions: []nut.ClientOption{
            nut.WithConnectTimeout(5 * time.Second),
            nut.WithReadTimeout(3 * time.Second),
            nut.WithLogger(logger),
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Close()

    // Get client with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := pool.Get(ctx)
    if err != nil {
        log.Fatal(err)
    }
    defer pool.Put(client)

    // Authenticate
    if _, err := client.Authenticate("monuser", "secret"); err != nil {
        log.Fatal(err)
    }

    // Get UPS list
    upsList, err := client.GetUPSList()
    if err != nil {
        log.Fatal(err)
    }

    // Process each UPS
    for _, ups := range upsList {
        fmt.Printf("UPS: %s\n", ups.Name)
        
        // Get variables
        vars, _ := ups.GetVariables()
        for name, value := range vars {
            fmt.Printf("  %s = %v\n", name, value)
        }
    }

    // Show metrics
    metrics := client.GetMetrics()
    fmt.Printf("\nMetrics:\n")
    fmt.Printf("  Commands sent: %d\n", metrics.CommandsSent)
    fmt.Printf("  Commands failed: %d\n", metrics.CommandsFailed)
    fmt.Printf("  Bytes sent: %d\n", metrics.BytesSent)
    fmt.Printf("  Bytes received: %d\n", metrics.BytesReceived)

    // Show pool stats
    idle, active := pool.Stats()
    fmt.Printf("\nPool Stats:\n")
    fmt.Printf("  Idle: %d, Active: %d\n", idle, active)
}
```

## Migration Guide

### From Simple Connect

Before:
```go
client, err := nut.Connect("localhost")
```

After (with options):
```go
client, err := nut.ConnectWithOptionsAndConfig(
    context.Background(),
    "localhost",
    []nut.ClientOption{
        nut.WithConnectTimeout(5 * time.Second),
    },
    3493,
)
```

### Adding Context Support

Before:
```go
resp, err := client.SendCommand("VER")
```

After:
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

resp, err := client.SendCommandWithContext(ctx, "VER")
```

### Using Connection Pool

Before (creating connection per request):
```go
for i := 0; i < 100; i++ {
    client, _ := nut.Connect("localhost")
    upsList, _ := client.GetUPSList()
    client.Disconnect()
}
```

After (reusing connections):
```go
pool, _ := nut.NewPool(nut.PoolConfig{
    Hostname: "localhost",
    MaxSize:  10,
})
defer pool.Close()

for i := 0; i < 100; i++ {
    client, _ := pool.Get(context.Background())
    upsList, _ := client.GetUPSList()
    pool.Put(client)
}
```

## Performance Considerations

1. **Connection Pool**: Use for >10 requests/second
2. **Context Timeout**: Set based on network latency (default 5s usually sufficient)
3. **Read Timeout**: Adjust based on server response time
4. **Max Pool Size**: Typically 10-50 connections, depends on:
   - Server capacity
   - Network bandwidth
   - Expected concurrent requests

## Thread Safety

- ✅ `Client` is thread-safe (protected by mutex)
- ✅ `Pool` is thread-safe
- ✅ `ClientMetrics` uses atomic operations
- ⚠️ `UPS` struct shares underlying Client (thread-safe via Client's mutex)

## Error Handling

All new methods properly handle errors and maintain connection state:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

client, err := pool.Get(ctx)
if err != nil {
    if err == context.DeadlineExceeded {
        // Timeout waiting for available connection
    }
    log.Fatal(err)
}
defer pool.Put(client) // Always return to pool

resp, err := client.SendCommandWithContext(ctx, "LIST UPS")
if err != nil {
    if err == context.Canceled {
        // Operation was cancelled
    }
    // Client is still valid, can be returned to pool
    return
}
```
