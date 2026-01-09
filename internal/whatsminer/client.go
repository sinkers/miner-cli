package whatsminer

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client represents a WhatsMiner API v3 client
type Client struct {
	ip       string
	port     int
	account  string
	password string
	salt     string
	conn     net.Conn
	timeout  time.Duration
}

// Response represents the standard API response structure
type Response struct {
	Code int             `json:"code"`
	Msg  json.RawMessage `json:"msg"` // Can be object or string
	When int64           `json:"when,omitempty"`
	Desc string          `json:"desc,omitempty"`
}

// GetMsg returns the msg field as a map (when code is 0)
func (r *Response) GetMsg() (map[string]interface{}, error) {
	var msg map[string]interface{}
	if err := json.Unmarshal(r.Msg, &msg); err != nil {
		return nil, err
	}
	return msg, nil
}

// GetMsgString returns the msg field as a string (typically when code != 0)
func (r *Response) GetMsgString() (string, error) {
	var msg string
	if err := json.Unmarshal(r.Msg, &msg); err != nil {
		return "", err
	}
	return msg, nil
}

// IsSuccess checks if the response code indicates success
func (r *Response) IsSuccess() bool {
	return r.Code == 0
}

// GetMessage retrieves a specific message field as a string
func (r *Response) GetMessage(key string) (string, bool) {
	msg, err := r.GetMsg()
	if err != nil {
		return "", false
	}
	if val, ok := msg[key]; ok {
		if str, ok := val.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetFloat retrieves a specific message field as a float64
func (r *Response) GetFloat(key string) (float64, bool) {
	msg, err := r.GetMsg()
	if err != nil {
		return 0, false
	}
	if val, ok := msg[key]; ok {
		if f, ok := val.(float64); ok {
			return f, true
		}
	}
	return 0, false
}

// GetInt retrieves a specific message field as an int
func (r *Response) GetInt(key string) (int, bool) {
	msg, err := r.GetMsg()
	if err != nil {
		return 0, false
	}
	if val, ok := msg[key]; ok {
		if f, ok := val.(float64); ok {
			return int(f), true
		}
		if i, ok := val.(int); ok {
			return i, true
		}
	}
	return 0, false
}

// NewClient creates a new WhatsMiner API client
func NewClient(ip string, port int, account, password string, timeout time.Duration) *Client {
	return &Client{
		ip:       ip,
		port:     port,
		account:  account,
		password: password,
		timeout:  timeout,
	}
}

// Connect establishes a TCP connection to the miner
func (c *Client) Connect() error {
	address := net.JoinHostPort(c.ip, fmt.Sprintf("%d", c.port))
	conn, err := net.DialTimeout("tcp", address, c.timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	c.conn = conn
	return nil
}

// Close closes the TCP connection
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// SetSalt sets the salt value for authentication (retrieved from get.device.info)
func (c *Client) SetSalt(salt string) {
	c.salt = salt
}

// generateToken generates an authentication token
// Token = first 8 characters of Base64(SHA256(command + password + salt + timestamp))
func (c *Client) generateToken(command string, ts int64) string {
	srcBuff := fmt.Sprintf("%s%s%s%d", command, c.password, c.salt, ts)
	hash := sha256.Sum256([]byte(srcBuff))
	encoded := base64.StdEncoding.EncodeToString(hash[:])
	return encoded[:8]
}

// send sends a message to the miner and receives the response
func (c *Client) send(message string) (*Response, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected")
	}

	// Set write deadline
	if err := c.conn.SetWriteDeadline(time.Now().Add(c.timeout)); err != nil {
		return nil, fmt.Errorf("failed to set write deadline: %w", err)
	}

	// Send message length (4 bytes, little-endian)
	messageLen := len(message)
	if messageLen > 0x7FFFFFFF { // Max int32 to prevent overflow
		return nil, fmt.Errorf("message too large: %d bytes", messageLen)
	}
	length := uint32(messageLen)
	lengthBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(lengthBytes, length)

	if _, err := c.conn.Write(lengthBytes); err != nil {
		return nil, fmt.Errorf("failed to send message length: %w", err)
	}

	// Send message
	if _, err := c.conn.Write([]byte(message)); err != nil {
		return nil, fmt.Errorf("failed to send message: %w", err)
	}

	// Set read deadline
	if err := c.conn.SetReadDeadline(time.Now().Add(c.timeout * 2)); err != nil {
		return nil, fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Receive response length (4 bytes, little-endian)
	lengthBytes = make([]byte, 4)
	if _, err := c.conn.Read(lengthBytes); err != nil {
		return nil, fmt.Errorf("failed to receive response length: %w", err)
	}
	responseLen := binary.LittleEndian.Uint32(lengthBytes)

	if responseLen > 1024*1024 { // 1MB max response size
		return nil, fmt.Errorf("invalid response length: %d", responseLen)
	}

	// Receive response data
	buffer := make([]byte, responseLen)
	totalRead := 0
	for totalRead < int(responseLen) {
		n, err := c.conn.Read(buffer[totalRead:])
		if err != nil {
			return nil, fmt.Errorf("failed to receive response data: %w", err)
		}
		totalRead += n
	}

	// Clear deadlines after successful operation
	c.conn.SetDeadline(time.Time{})

	// Parse JSON response
	var response Response
	if err := json.Unmarshal(buffer, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// GetRequest sends a GET command (no authentication required)
func (c *Client) GetRequest(cmd string, param interface{}) (*Response, error) {
	payload := map[string]interface{}{
		"cmd":   cmd,
		"param": param,
	}

	message, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return c.send(string(message))
}

// SetRequest sends a SET command (requires authentication)
func (c *Client) SetRequest(cmd string, param interface{}) (*Response, error) {
	ts := time.Now().Unix()
	token := c.generateToken(cmd, ts)

	payload := map[string]interface{}{
		"cmd":     cmd,
		"param":   param,
		"ts":      ts,
		"token":   token,
		"account": c.account,
	}

	message, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	return c.send(string(message))
}

// GetDeviceInfo retrieves device information and salt
func (c *Client) GetDeviceInfo() (*Response, error) {
	resp, err := c.GetRequest("get.device.info", nil)
	if err != nil {
		return nil, err
	}

	// Auto-set salt if available
	if resp.Code == 0 {
		if msg, err := resp.GetMsg(); err == nil {
			if salt, ok := msg["salt"].(string); ok {
				c.SetSalt(salt)
			}
		}
	}

	return resp, nil
}

// GetMinerStatus retrieves real-time miner status with detailed metrics
func (c *Client) GetMinerStatus(param string) (*Response, error) {
	return c.GetRequest("get.miner.status", param)
}

// SetMinerService starts or stops the mining service
func (c *Client) SetMinerService(param string) (*Response, error) {
	return c.SetRequest("set.miner.service", param)
}

// SetMinerPowerPercent sets the miner power as a percentage of maximum
func (c *Client) SetMinerPowerPercent(mode, percent string) (*Response, error) {
	param := map[string]string{
		"percent": percent,
		"mode":    mode,
	}
	return c.SetRequest("set.miner.power_percent", param)
}
