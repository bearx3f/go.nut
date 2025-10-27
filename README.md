[![GoDoc](https://pkg.go.dev/github.com/bearx3f/go.nut?status.svg)](https://pkg.go.dev/github.com/bearx3f/go.nut)

# go.nut
go.nut is a Golang library for interacting with [NUT (Network UPS Tools)](https://networkupstools.org/)

This is a maintained fork with support for modern NUT servers, including TLS/SSL via STARTTLS, improved error handling, thread-safety, context support, connection pooling, and production-ready features.

## Features

### Core Features
- ✅ TCP communication with NUT servers (port 3493)
- ✅ TLS/SSL support via STARTTLS (NUT >= 2.7.0)
- ✅ Authentication support (username/password)
- ✅ UPS listing and management
- ✅ Variable and command querying
- ✅ Thread-safe operations with mutex protection
- ✅ Proper error handling and connection lifecycle management
- ✅ Support for UPS names with spaces/special characters

### Production Features (Optional)
- ✅ **Context Support**: Cancellation and timeout control for all operations
- ✅ **Connection Pool**: Efficient connection reuse for high-concurrency scenarios
- ✅ **Metrics & Monitoring**: Track commands, bytes, errors, and reconnects
- ✅ **Custom Timeouts**: Flexible timeout configuration via options pattern
- ✅ **Debug Logging**: Optional logger for troubleshooting
- ✅ **Go modules support** (go 1.21+)

# Getting started
```go
import "github.com/bearx3f/go.nut"
```

## Basic Connection
```go
// Simple connection (backward compatible)
client, err := nut.Connect("192.168.1.100")
if err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Get list of UPS devices
upsList, err := client.GetUPSList()
if err != nil {
    log.Fatal(err)
}

// Access UPS variables
for _, ups := range upsList {
    vars, _ := ups.GetVariables()
    fmt.Printf("UPS: %s\n", ups.Name)
    for name, value := range vars {
        fmt.Printf("  %s = %v\n", name, value)
    }
}
```

## Using TLS/SSL (STARTTLS)
```go
client, err := nut.Connect("192.168.1.100")
if err != nil {
    log.Fatal(err)
}

// Start TLS connection
err = client.StartTLS()
if err != nil {
    log.Fatal(err)
}

// Now authenticate and use securely
authenticated, err := client.Authenticate("username", "password")
if err != nil || !authenticated {
    log.Fatal("Authentication failed")
}

defer client.Disconnect()
```

## Advanced Features

### Context Support & Custom Options
```go
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// Connect with custom options
client, err := nut.ConnectWithOptionsAndConfig(ctx, "192.168.1.100", 
    []nut.ClientOption{
        nut.WithConnectTimeout(5 * time.Second),
        nut.WithReadTimeout(3 * time.Second),
        nut.WithLogger(log.Default()),
    }, 
    3493,
)
if err != nil {
    log.Fatal(err)
}
defer client.Disconnect()

// Send command with context
resp, err := client.SendCommandWithContext(ctx, "VER")
if err != nil {
    log.Fatal(err)
}
```

### Connection Pool (High-Concurrency)
```go
// Create a connection pool
pool, err := nut.NewPool(nut.PoolConfig{
    Hostname:  "192.168.1.100",
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

// Get a client from the pool
ctx := context.Background()
client, err := pool.Get(ctx)
if err != nil {
    log.Fatal(err)
}

// Use the client
upsList, err := client.GetUPSList()

// Always return the client to the pool
pool.Put(client)

// Check pool statistics
idle, active := pool.Stats()
fmt.Printf("Pool - Idle: %d, Active: %d\n", idle, active)
```

### Metrics & Monitoring
```go
// Track client operations
metrics := client.GetMetrics()
fmt.Printf("Commands sent: %d\n", metrics.CommandsSent)
fmt.Printf("Commands failed: %d\n", metrics.CommandsFailed)
fmt.Printf("Bytes sent: %d\n", metrics.BytesSent)
fmt.Printf("Bytes received: %d\n", metrics.BytesReceived)
fmt.Printf("Reconnects: %d\n", metrics.Reconnects)
```

## Documentation

- **Examples**: See [`example_test.go`](example_test.go) and [`example_pool_test.go`](example_pool_test.go)
- **API Reference**: [Godocs](https://pkg.go.dev/github.com/bearx3f/go.nut)
- **Optional Features**: See [`docs/OPTIONAL_FEATURES.md`](docs/OPTIONAL_FEATURES.md) for detailed documentation on:
  - Context support and cancellation
  - Connection pooling
  - Metrics and monitoring
  - Custom timeouts and logging
- **Modernization Guide**: [`docs/MODERNIZATION.md`](docs/MODERNIZATION.md)
- **Changelog**: [`docs/CHANGELOG_2025.md`](docs/CHANGELOG_2025.md)

## Bug Fixes & Improvements

This fork includes numerous critical bug fixes and improvements:

### Critical Fixes
- ✅ **TLS Implementation**: Fixed STARTTLS to properly use `tls.Client()` instead of `tls.Server()`
- ✅ **Thread Safety**: Added mutex protection for concurrent access to connections
- ✅ **Memory Leaks**: Fixed connection cleanup in error paths
- ✅ **Buffer Management**: Persistent `bufio.Reader` to prevent data loss
- ✅ **Name Escaping**: Proper quoting for UPS/variable names with spaces

### Error Handling
- ✅ Response length validation before array access (prevents panics)
- ✅ Proper error propagation throughout the stack
- ✅ Connection state validation
- ✅ Graceful error recovery

### Type Detection
- ✅ Improved variable type detection (RW/RO flags)
- ✅ Better boolean type inference
- ✅ Enhanced numeric parsing with regex

## Performance Features

- **Connection Pooling**: Reuse connections for high-throughput scenarios (10+ req/s)
- **Atomic Metrics**: Lock-free statistics tracking
- **Context Cancellation**: Cancel long-running operations efficiently
- **Configurable Timeouts**: Fine-tune for your network conditions

# Other resources
* [Network protocol information](http://networkupstools.org/docs/developer-guide.chunked/ar01s09.html)
* [NUT Official Documentation](https://networkupstools.org/)
* [STARTTLS Support](https://networkupstools.org/docs/developer-guide.chunked/ar01s09.html#_starttls)

# Contributing

1. Fork it
2. Create your feature branch (`git checkout -b my-new-feature`)
3. Make sure `golint` and `go vet` run successfully
4. `go fmt` your code
5. Commit your changes (`git commit -am "Add some feature"`)
6. Push to the branch (`git push origin my-new-feature`)
7. Create a new Pull Request

# License
[MIT](LICENSE)

# Compatibility
- **NUT Version**: Tested with NUT 2.7.0+ (supports older versions without STARTTLS)
- **Go Version**: 1.21+
- **OS**: Linux, Windows, macOS
- **Thread-Safe**: Yes, all operations protected by mutex
- **Production-Ready**: Includes metrics, logging, and connection pooling

## Migration from Original go.nut

This fork is **backward compatible**. Existing code will continue to work:

```go
// Old code still works!
client, err := nut.Connect("localhost")
defer client.Disconnect()
```

To use new features, see the [Optional Features Guide](docs/OPTIONAL_FEATURES.md).
