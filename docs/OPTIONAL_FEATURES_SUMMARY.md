# Optional Features Summary

## ✅ Implemented Features

### 1. Context Support
- ✅ `ConnectWithOptions(ctx, hostname, port)` - Connect with context
- ✅ `ConnectWithOptionsAndConfig(ctx, hostname, opts, port)` - Full config with context
- ✅ `SendCommandWithContext(ctx, cmd)` - Send command with cancellation support
- ✅ `readResponseWithContext(ctx, endLine, multiLine)` - Context-aware response reading

**Benefits:**
- Timeout control per operation
- Graceful cancellation
- Deadline propagation

### 2. Custom Timeout
- ✅ `WithConnectTimeout(duration)` - Connection timeout option
- ✅ `WithReadTimeout(duration)` - Read timeout option
- ✅ `WithTLSConfig(config)` - Custom TLS config
- ✅ `WithLogger(logger)` - Debug logging option

**Default Values:**
- Connect: 5 seconds
- Read: 2 seconds

### 3. Connection Pool
- ✅ `NewPool(config)` - Create connection pool
- ✅ `Pool.Get(ctx)` - Get client from pool
- ✅ `Pool.Put(client)` - Return client to pool
- ✅ `Pool.Close()` - Close all pool connections
- ✅ `Pool.Stats()` - Get pool statistics (idle/active)

**Features:**
- Automatic connection reuse
- Configurable max size
- Context support for Get()
- Thread-safe operations

### 4. Metrics & Logging
- ✅ `ClientMetrics` struct with atomic counters
- ✅ `GetMetrics()` - Retrieve current metrics
- ✅ Metrics tracked:
  - Commands sent/failed
  - Bytes sent/received
  - Reconnects count
  - Last command time
- ✅ Optional logger for debugging
- ✅ Log output for connections and commands

## 📝 Usage Examples

### Basic with Context
```go
ctx := context.WithTimeout(context.Background(), 10*time.Second)
client, _ := nut.ConnectWithOptions(ctx, "localhost", 3493)
defer client.Disconnect()

resp, _ := client.SendCommandWithContext(ctx, "VER")
```

### With Options
```go
client, _ := nut.ConnectWithOptionsAndConfig(ctx, "localhost", []nut.ClientOption{
    nut.WithConnectTimeout(5 * time.Second),
    nut.WithReadTimeout(3 * time.Second),
    nut.WithLogger(log.Default()),
}, 3493)
```

### Connection Pool
```go
pool, _ := nut.NewPool(nut.PoolConfig{
    Hostname: "localhost",
    MaxSize:  10,
})
defer pool.Close()

client, _ := pool.Get(context.Background())
defer pool.Put(client)

upsList, _ := client.GetUPSList()
```

### Metrics
```go
metrics := client.GetMetrics()
fmt.Printf("Commands: %d (failed: %d)\n", 
    metrics.CommandsSent, metrics.CommandsFailed)
fmt.Printf("Bytes: sent=%d, received=%d\n",
    metrics.BytesSent, metrics.BytesReceived)
```

## 🔄 Backward Compatibility

The original `Connect(hostname, port...)` function still works and returns `*Client`:

```go
client, err := nut.Connect("localhost")  // Still works!
```

All existing code continues to work without changes.

## 📚 Documentation

- `docs/OPTIONAL_FEATURES.md` - Complete feature documentation
- `example_pool_test.go` - Usage examples with pool and metrics

## 🧪 Testing

All new features maintain thread-safety through:
- Mutex protection for shared state
- Atomic operations for metrics
- Channel-based pool management

## 🎯 Use Cases

1. **High Concurrency** → Use Connection Pool
2. **Long Operations** → Use Context with timeout
3. **Production Monitoring** → Enable Metrics
4. **Debugging** → Enable Logger
5. **Custom Timeouts** → Use ClientOption pattern
