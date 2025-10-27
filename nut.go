// Package nut provides a Golang interface for interacting with Network UPS Tools (NUT).
//
// It communicates with NUT over the TCP protocol and supports TLS/SSL via STARTTLS
package nut

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Client contains information about the NUT server as well as the connection.
type Client struct {
	Version         string
	ProtocolVersion string
	Hostname        net.Addr
	conn            net.Conn
	reader          *bufio.Reader
	UseTLS          bool
	TLSConfig       *tls.Config
	ConnectTimeout  time.Duration
	ReadTimeout     time.Duration
	Logger          *log.Logger // Optional logger for debugging
	mu              sync.Mutex  // Protects concurrent access to connection
	metrics         *ClientMetrics
}

// ClientMetrics holds statistics for a client connection
type ClientMetrics struct {
	CommandsSent    uint64
	CommandsFailed  uint64
	BytesSent       uint64
	BytesReceived   uint64
	Reconnects      uint64
	LastCommandTime atomic.Value // time.Time
}

// GetMetrics returns a copy of the current metrics
func (c *Client) GetMetrics() ClientMetrics {
	if c.metrics == nil {
		return ClientMetrics{}
	}
	return ClientMetrics{
		CommandsSent:   atomic.LoadUint64(&c.metrics.CommandsSent),
		CommandsFailed: atomic.LoadUint64(&c.metrics.CommandsFailed),
		BytesSent:      atomic.LoadUint64(&c.metrics.BytesSent),
		BytesReceived:  atomic.LoadUint64(&c.metrics.BytesReceived),
		Reconnects:     atomic.LoadUint64(&c.metrics.Reconnects),
	}
}

// ClientOption is a function that configures a Client
type ClientOption func(*Client)

// WithConnectTimeout sets a custom connection timeout
func WithConnectTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.ConnectTimeout = timeout
	}
}

// WithReadTimeout sets a custom read timeout
func WithReadTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.ReadTimeout = timeout
	}
}

// WithTLSConfig sets a custom TLS configuration
func WithTLSConfig(config *tls.Config) ClientOption {
	return func(c *Client) {
		c.TLSConfig = config
	}
}

// WithLogger sets a logger for debugging
func WithLogger(logger *log.Logger) ClientOption {
	return func(c *Client) {
		c.Logger = logger
	}
}

// Connect accepts a hostname/IP string and an optional port, then creates a connection to NUT, returning a Client.
func Connect(hostname string, _port ...int) (*Client, error) {
	return ConnectWithOptions(context.Background(), hostname, _port...)
}

// ConnectWithOptions creates a connection with custom options and context support.
func ConnectWithOptions(ctx context.Context, hostname string, port ...int) (*Client, error) {
	return ConnectWithOptionsAndConfig(ctx, hostname, nil, port...)
}

// ConnectWithOptionsAndConfig creates a connection with full configuration support.
func ConnectWithOptionsAndConfig(ctx context.Context, hostname string, opts []ClientOption, port ...int) (*Client, error) {
	portNum := 3493
	if len(port) > 0 {
		portNum = port[0]
	}

	client := &Client{
		ConnectTimeout: 5 * time.Second,
		ReadTimeout:    2 * time.Second,
		UseTLS:         false,
		metrics:        &ClientMetrics{},
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	// Log connection attempt
	if client.Logger != nil {
		client.Logger.Printf("Connecting to %s:%d (timeout: %v)", hostname, portNum, client.ConnectTimeout)
	}

	// Use net.JoinHostPort to properly handle IPv6 addresses
	address := net.JoinHostPort(hostname, fmt.Sprintf("%d", portNum))

	// Create dialer with timeout and context support
	dialer := &net.Dialer{
		Timeout: client.ConnectTimeout,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		if client.Logger != nil {
			client.Logger.Printf("Connection failed: %v", err)
		}
		return nil, err
	}

	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("failed to convert to TCP connection")
	}

	client.Hostname = tcpConn.RemoteAddr()
	client.conn = tcpConn
	client.reader = bufio.NewReader(tcpConn)

	// Get version info, close connection on error
	_, err = client.GetVersion()
	if err != nil {
		tcpConn.Close()
		if client.Logger != nil {
			client.Logger.Printf("Failed to get version: %v", err)
		}
		return nil, fmt.Errorf("failed to get version: %w", err)
	}

	_, err = client.GetNetworkProtocolVersion()
	if err != nil {
		tcpConn.Close()
		if client.Logger != nil {
			client.Logger.Printf("Failed to get network protocol version: %v", err)
		}
		return nil, fmt.Errorf("failed to get network protocol version: %w", err)
	}

	if client.Logger != nil {
		client.Logger.Printf("Connected successfully. Version: %s, Protocol: %s", client.Version, client.ProtocolVersion)
	}

	return client, nil
}

// StartTLS initiates a TLS/SSL connection with the NUT server using STARTTLS command.
// This requires the NUT server to support STARTTLS (NUT >= 2.7.0).
func (c *Client) StartTLS() error {
	if c.UseTLS {
		return fmt.Errorf("already in TLS mode")
	}

	resp, err := c.SendCommand("STARTTLS")
	if err != nil {
		return fmt.Errorf("STARTTLS command failed: %v", err)
	}

	if len(resp) == 0 || resp[0] != "OK STARTTLS" {
		if len(resp) > 0 {
			return fmt.Errorf("server did not accept STARTTLS: %s", resp[0])
		}
		return fmt.Errorf("server did not accept STARTTLS: empty response")
	}

	// Upgrade connection to TLS
	tlsConfig := c.TLSConfig
	if tlsConfig == nil {
		tlsConfig = &tls.Config{
			InsecureSkipVerify: false,
		}
	}

	// Use tls.Client (not tls.Server) since we are the client
	tlsConn := tls.Client(c.conn, tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return fmt.Errorf("TLS handshake failed: %v", err)
	}

	// Replace the connection with the TLS-wrapped connection
	c.conn = tlsConn
	c.reader = bufio.NewReader(tlsConn) // Reset reader for TLS connection
	c.UseTLS = true
	return nil
}

// Disconnect gracefully disconnects from NUT by sending the LOGOUT command.
func (c *Client) Disconnect() (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if connection is still valid
	if c.conn == nil {
		return false, fmt.Errorf("connection already closed")
	}

	// Try to send LOGOUT, but don't fail if it errors
	logoutResp, _ := c.sendCommandUnsafe("LOGOUT")

	// Always close the connection
	closeErr := c.conn.Close()
	c.conn = nil
	c.reader = nil

	if closeErr != nil {
		return false, closeErr
	}

	// Check if LOGOUT was successful
	if len(logoutResp) > 0 && (logoutResp[0] == "OK Goodbye" || logoutResp[0] == "Goodbye...") {
		return true, nil
	}

	return false, nil
}

// Close closes the connection without sending LOGOUT command.
// Use this if you just want to close the connection immediately.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return fmt.Errorf("connection already closed")
	}

	err := c.conn.Close()
	c.conn = nil
	c.reader = nil
	return err
}

// sendCommandUnsafe is an internal version without mutex lock for use within locked contexts
func (c *Client) sendCommandUnsafe(cmd string) (resp []string, err error) {
	cmdTrimmed := strings.TrimSpace(cmd)
	multiLineResponse := strings.HasPrefix(cmdTrimmed, "LIST ")

	cmdWithNewline := cmd + "\n"
	n, err := fmt.Fprint(c.conn, cmdWithNewline)
	if err != nil {
		if c.metrics != nil {
			atomic.AddUint64(&c.metrics.CommandsFailed, 1)
		}
		return []string{}, fmt.Errorf("failed to send command: %w", err)
	}

	// Track metrics
	if c.metrics != nil {
		atomic.AddUint64(&c.metrics.CommandsSent, 1)
		atomic.AddUint64(&c.metrics.BytesSent, uint64(n))
		c.metrics.LastCommandTime.Store(time.Now())
	}

	// Log command
	if c.Logger != nil {
		c.Logger.Printf("Sent command: %s", cmdTrimmed)
	}

	endLine := "OK\n"
	if multiLineResponse {
		endLine = fmt.Sprintf("END %s\n", cmdTrimmed)
	}

	resp, err = c.ReadResponse(endLine, multiLineResponse)
	if err != nil {
		if c.metrics != nil {
			atomic.AddUint64(&c.metrics.CommandsFailed, 1)
		}
		return []string{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Track bytes received
	if c.metrics != nil {
		for _, line := range resp {
			atomic.AddUint64(&c.metrics.BytesReceived, uint64(len(line)+1)) // +1 for newline
		}
	}

	if len(resp) > 0 && strings.HasPrefix(resp[0], "ERR ") {
		if c.metrics != nil {
			atomic.AddUint64(&c.metrics.CommandsFailed, 1)
		}
		errCode := strings.Split(resp[0], " ")
		if len(errCode) > 1 {
			return []string{}, errorForMessage(errCode[1])
		}
		return []string{}, errorForMessage("UNKNOWN-COMMAND")
	}

	return resp, nil
}

// ReadResponse is a convenience function for reading newline delimited responses.
func (c *Client) ReadResponse(endLine string, multiLineResponse bool) (resp []string, err error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.ReadTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %v", err)
	}

	response := []string{}

	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {
			return nil, fmt.Errorf("error reading response: %v", err)
		}
		if len(line) > 0 {
			cleanLine := strings.TrimSuffix(line, "\n")
			response = append(response, cleanLine)
			if line == endLine || !multiLineResponse {
				break
			}
		}
	}

	return response, nil
}

// SendCommand sends the string cmd to the device, and returns the response.
func (c *Client) SendCommand(cmd string) (resp []string, err error) {
	return c.SendCommandWithContext(context.Background(), cmd)
}

// SendCommandWithContext sends a command with context support for cancellation.
func (c *Client) SendCommandWithContext(ctx context.Context, cmd string) (resp []string, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.Logger != nil {
		c.Logger.Printf("Sending command: %s", cmd)
	}

	// Check context before starting
	select {
	case <-ctx.Done():
		return []string{}, ctx.Err()
	default:
	}

	// Determine if this is a LIST command (multi-line response)
	cmdTrimmed := strings.TrimSpace(cmd)
	multiLineResponse := strings.HasPrefix(cmdTrimmed, "LIST ")

	// Send the command with newline
	cmdWithNewline := cmd + "\n"
	_, err = fmt.Fprint(c.conn, cmdWithNewline)
	if err != nil {
		if c.Logger != nil {
			c.Logger.Printf("Failed to send command: %v", err)
		}
		return []string{}, fmt.Errorf("failed to send command: %w", err)
	}

	// Calculate expected end line
	endLine := "OK\n"
	if multiLineResponse {
		endLine = fmt.Sprintf("END %s\n", cmdTrimmed)
	}

	resp, err = c.readResponseWithContext(ctx, endLine, multiLineResponse)
	if err != nil {
		if c.Logger != nil {
			c.Logger.Printf("Failed to read response: %v", err)
		}
		return []string{}, fmt.Errorf("failed to read response: %w", err)
	}

	if len(resp) > 0 && strings.HasPrefix(resp[0], "ERR ") {
		errCode := strings.Split(resp[0], " ")
		if len(errCode) > 1 {
			if c.Logger != nil {
				c.Logger.Printf("Server error: %s", errCode[1])
			}
			return []string{}, errorForMessage(errCode[1])
		}
		return []string{}, errorForMessage("UNKNOWN-COMMAND")
	}

	if c.Logger != nil {
		c.Logger.Printf("Command successful, received %d lines", len(resp))
	}

	return resp, nil
}

// readResponseWithContext reads response with context support
func (c *Client) readResponseWithContext(ctx context.Context, endLine string, multiLineResponse bool) (resp []string, err error) {
	if err := c.conn.SetReadDeadline(time.Now().Add(c.ReadTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %v", err)
	}

	// Create channel for reading
	type readResult struct {
		lines []string
		err   error
	}
	resultChan := make(chan readResult, 1)

	go func() {
		lines := []string{}
		for {
			line, err := c.reader.ReadString('\n')
			if err != nil {
				resultChan <- readResult{nil, fmt.Errorf("error reading response: %v", err)}
				return
			}
			if len(line) > 0 {
				cleanLine := strings.TrimSuffix(line, "\n")
				lines = append(lines, cleanLine)
				if line == endLine || !multiLineResponse {
					resultChan <- readResult{lines, nil}
					return
				}
			}
		}
	}()

	// Wait for result or context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		if result.err != nil {
			return nil, result.err
		}
		return result.lines, nil
	}
}

// Authenticate accepts a username and passwords and uses them to authenticate the existing NUT session.
func (c *Client) Authenticate(username, password string) (bool, error) {
	usernameResp, err := c.SendCommand(fmt.Sprintf("USERNAME %s", username))
	if err != nil {
		return false, err
	}
	passwordResp, err := c.SendCommand(fmt.Sprintf("PASSWORD %s", password))
	if err != nil {
		return false, err
	}
	if len(usernameResp) > 0 && usernameResp[0] == "OK" && len(passwordResp) > 0 && passwordResp[0] == "OK" {
		return true, nil
	}
	return false, nil
}

// GetUPSList returns a list of all UPSes provided by this NUT instance.
func (c *Client) GetUPSList() ([]UPS, error) {
	upsList := []UPS{}
	resp, err := c.SendCommand("LIST UPS")
	if err != nil {
		return upsList, err
	}
	for _, line := range resp {
		if strings.HasPrefix(line, "UPS ") {
			splitLine := strings.Split(strings.TrimPrefix(line, "UPS "), `"`)
			if len(splitLine) < 1 {
				continue
			}
			newUPS, err := NewUPS(strings.TrimSuffix(splitLine[0], " "), c)
			if err != nil {
				return upsList, err
			}
			upsList = append(upsList, newUPS)
		}
	}
	return upsList, nil
}

// Help returns a list of the commands supported by NUT.
func (c *Client) Help() (string, error) {
	helpResp, err := c.SendCommand("HELP")
	if err != nil {
		return "", err
	}
	if len(helpResp) < 1 {
		return "", fmt.Errorf("empty response from HELP command")
	}
	return helpResp[0], nil
}

// GetVersion returns the the version of the server currently in use.
func (c *Client) GetVersion() (string, error) {
	versionResponse, err := c.SendCommand("VER")
	if err != nil {
		return "", err
	}
	if len(versionResponse) < 1 {
		return "", fmt.Errorf("empty response from VER command")
	}
	c.Version = versionResponse[0]
	return versionResponse[0], nil
}

// GetNetworkProtocolVersion returns the version of the network protocol currently in use.
func (c *Client) GetNetworkProtocolVersion() (string, error) {
	versionResponse, err := c.SendCommand("NETVER")
	if err != nil {
		return "", err
	}
	if len(versionResponse) < 1 {
		return "", fmt.Errorf("empty response from NETVER command")
	}
	c.ProtocolVersion = versionResponse[0]
	return versionResponse[0], nil
}

// Pool manages a pool of Client connections for high-concurrency scenarios.
type Pool struct {
	hostname      string
	port          int
	opts          []ClientOption
	clients       chan *Client
	maxSize       int
	mu            sync.Mutex
	closed        bool
	activeClients int
}

// PoolConfig contains configuration for connection pool
type PoolConfig struct {
	MaxSize       int            // Maximum number of connections in pool
	Hostname      string         // NUT server hostname
	Port          int            // NUT server port (default 3493)
	ClientOptions []ClientOption // Options to apply to each client
}

// NewPool creates a new connection pool with the given configuration.
func NewPool(config PoolConfig) (*Pool, error) {
	if config.MaxSize <= 0 {
		config.MaxSize = 10 // default pool size
	}
	if config.Port == 0 {
		config.Port = 3493
	}
	if config.Hostname == "" {
		return nil, fmt.Errorf("hostname is required")
	}

	pool := &Pool{
		hostname: config.Hostname,
		port:     config.Port,
		opts:     config.ClientOptions,
		clients:  make(chan *Client, config.MaxSize),
		maxSize:  config.MaxSize,
	}

	return pool, nil
}

// Get retrieves a client from the pool, creating a new one if needed.
func (p *Pool) Get(ctx context.Context) (*Client, error) {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil, fmt.Errorf("pool is closed")
	}
	p.mu.Unlock()

	// Try to get an existing client from the pool
	select {
	case client := <-p.clients:
		// Test if connection is still alive
		if client.conn != nil {
			return client, nil
		}
		// Connection is dead, create a new one
		p.mu.Lock()
		p.activeClients--
		p.mu.Unlock()
	default:
		// No idle clients available
	}

	// Create new client if we haven't reached max size
	p.mu.Lock()
	if p.activeClients >= p.maxSize {
		p.mu.Unlock()
		// Wait for an available client
		select {
		case client := <-p.clients:
			return client, nil
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	p.activeClients++
	p.mu.Unlock()

	// Create new connection
	client, err := ConnectWithOptionsAndConfig(ctx, p.hostname, p.opts, p.port)
	if err != nil {
		p.mu.Lock()
		p.activeClients--
		p.mu.Unlock()
		return nil, err
	}

	if client.metrics != nil {
		atomic.AddUint64(&client.metrics.Reconnects, 1)
	}

	return client, nil
}

// Put returns a client to the pool. If the pool is full, the client is closed.
func (p *Pool) Put(client *Client) error {
	if client == nil {
		return nil
	}

	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return client.Close()
	}
	p.mu.Unlock()

	// Try to return to pool
	select {
	case p.clients <- client:
		return nil
	default:
		// Pool is full, close the connection
		p.mu.Lock()
		p.activeClients--
		p.mu.Unlock()
		return client.Close()
	}
}

// Close closes all clients in the pool and prevents new clients from being created.
func (p *Pool) Close() error {
	p.mu.Lock()
	if p.closed {
		p.mu.Unlock()
		return nil
	}
	p.closed = true
	p.mu.Unlock()

	// Close all clients in the pool
	close(p.clients)
	var lastErr error
	for client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}

	return lastErr
}

// Stats returns statistics about the pool
func (p *Pool) Stats() (idle int, active int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.clients), p.activeClients
}
