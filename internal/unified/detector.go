package unified

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Detector handles API type detection for miners
type Detector struct {
	timeout time.Duration
}

// NewDetector creates a new API detector
func NewDetector(timeout time.Duration) *Detector {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	return &Detector{timeout: timeout}
}

// DetectAPI automatically detects the API type of a miner
func (d *Detector) DetectAPI(ctx context.Context, host string) (APIType, APIInfo, error) {
	detectCtx, cancel := context.WithTimeout(ctx, d.timeout)
	defer cancel()

	type result struct {
		apiType APIType
		info    APIInfo
		err     error
	}

	results := make(chan result, 3)
	var wg sync.WaitGroup

	// Try Braiins gRPC (port 50051)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := d.detectBraiins(detectCtx, host); err == nil {
			results <- result{APITypeBraiins, info, nil}
		} else {
			results <- result{APITypeUnknown, APIInfo{}, err}
		}
	}()

	// Try VNish REST API (port 80/443)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := d.detectVNish(detectCtx, host); err == nil {
			results <- result{APITypeVNish, info, nil}
		} else {
			results <- result{APITypeUnknown, APIInfo{}, err}
		}
	}()

	// Try CGMiner API (port 4028)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if info, err := d.detectCGMiner(detectCtx, host); err == nil {
			results <- result{APITypeCGMiner, info, nil}
		} else {
			results <- result{APITypeUnknown, APIInfo{}, err}
		}
	}()

	// Close results channel when all goroutines complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results with priority
	var lastError error
	var detectedAPIs []result

	for r := range results {
		if r.apiType != APITypeUnknown {
			detectedAPIs = append(detectedAPIs, r)
		} else {
			lastError = r.err
		}
	}

	// Priority order: Braiins > VNish > CGMiner
	for _, priority := range []APIType{APITypeBraiins, APITypeVNish, APITypeCGMiner} {
		for _, detected := range detectedAPIs {
			if detected.apiType == priority {
				// Check for additional API support
				if detected.apiType == APITypeBraiins || detected.apiType == APITypeVNish {
					detected.info.AlsoSupports = d.checkAdditionalAPIs(ctx, host, detected.apiType)
				}
				return detected.apiType, detected.info, nil
			}
		}
	}

	if lastError != nil {
		return APITypeUnknown, APIInfo{}, lastError
	}
	return APITypeUnknown, APIInfo{}, fmt.Errorf("no supported API detected on %s", host)
}

// detectBraiins attempts to detect Braiins gRPC API
func (d *Detector) detectBraiins(ctx context.Context, host string) (APIInfo, error) {
	// Try gRPC connection on port 50051
	address := fmt.Sprintf("%s:50051", host)
	
	conn, err := grpc.DialContext(ctx, address,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return APIInfo{}, fmt.Errorf("failed to connect to Braiins gRPC: %w", err)
	}
	defer conn.Close()

	// Successfully connected to gRPC port
	// In real implementation, we would try to call a version endpoint
	// For now, we assume if gRPC port is open, it's Braiins
	return APIInfo{
		Type:         APITypeBraiins,
		Port:         50051,
		Protocol:     "gRPC",
		RequiresAuth: true,
	}, nil
}

// detectVNish attempts to detect VNish REST API
func (d *Detector) detectVNish(ctx context.Context, host string) (APIInfo, error) {
	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	// Try VNish info endpoint
	url := fmt.Sprintf("http://%s/api/v1/info", host)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return APIInfo{}, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return APIInfo{}, fmt.Errorf("failed to connect to VNish API: %w", err)
	}
	defer resp.Body.Close()

	// VNish API responds with 200 or 401 (if auth required)
	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusUnauthorized {
		info := APIInfo{
			Type:         APITypeVNish,
			Port:         80,
			Protocol:     "HTTP/REST",
			RequiresAuth: resp.StatusCode == http.StatusUnauthorized,
		}

		// Try to parse response if available
		if resp.StatusCode == http.StatusOK {
			var vnishInfo map[string]interface{}
			if err := json.NewDecoder(resp.Body).Decode(&vnishInfo); err == nil {
				if model, ok := vnishInfo["model"].(string); ok {
					info.Model = model
				}
				if version, ok := vnishInfo["version"].(string); ok {
					info.Version = version
				}
			}
		}

		return info, nil
	}

	return APIInfo{}, fmt.Errorf("not VNish API (status: %d)", resp.StatusCode)
}

// detectCGMiner attempts to detect CGMiner API
func (d *Detector) detectCGMiner(ctx context.Context, host string) (APIInfo, error) {
	address := fmt.Sprintf("%s:4028", host)
	
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
	}
	
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return APIInfo{}, fmt.Errorf("failed to connect to CGMiner API: %w", err)
	}
	defer conn.Close()

	// Send version command
	versionCmd := `{"command":"version"}`
	if _, err := conn.Write([]byte(versionCmd)); err != nil {
		return APIInfo{}, fmt.Errorf("failed to send version command: %w", err)
	}

	// Read response
	buffer := make([]byte, 4096)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buffer)
	if err != nil {
		return APIInfo{}, fmt.Errorf("failed to read response: %w", err)
	}

	// Clean response (remove null terminator if present)
	response := string(buffer[:n])
	response = strings.TrimRight(response, "\x00")

	// Try to parse JSON response
	var jsonResp map[string]interface{}
	if err := json.Unmarshal([]byte(response), &jsonResp); err != nil {
		return APIInfo{}, fmt.Errorf("invalid JSON response: %w", err)
	}

	// Check for VERSION field (CGMiner signature)
	if versionData, ok := jsonResp["VERSION"]; ok {
		info := APIInfo{
			Type:     APITypeCGMiner,
			Port:     4028,
			Protocol: "TCP/JSON-RPC",
		}

		// Extract version information
		if versionArray, ok := versionData.([]interface{}); ok && len(versionArray) > 0 {
			if versionMap, ok := versionArray[0].(map[string]interface{}); ok {
				if ver, ok := versionMap["CGMiner"].(string); ok {
					info.Version = ver
				} else if ver, ok := versionMap["BMMiner"].(string); ok {
					info.Version = ver
				} else if ver, ok := versionMap["API"].(string); ok {
					info.Version = fmt.Sprintf("API %s", ver)
				}
			}
		}

		return info, nil
	}

	return APIInfo{}, fmt.Errorf("not CGMiner API (no VERSION field)")
}

// checkAdditionalAPIs checks if a miner supports additional API types
func (d *Detector) checkAdditionalAPIs(ctx context.Context, host string, primaryAPI APIType) []APIType {
	var additional []APIType

	// If primary is Braiins or VNish, check for CGMiner compatibility
	if primaryAPI == APITypeBraiins || primaryAPI == APITypeVNish {
		if _, err := d.detectCGMiner(ctx, host); err == nil {
			additional = append(additional, APITypeCGMiner)
		}
	}

	return additional
}

// QuickDetect performs a quick detection on common ports
func (d *Detector) QuickDetect(host string, port int) APIType {
	switch port {
	case 4028:
		return APITypeCGMiner
	case 80, 443:
		return APITypeVNish
	case 50051:
		return APITypeBraiins
	default:
		return APITypeUnknown
	}
}