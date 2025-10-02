//go:build ignore
// +build ignore

package unified

import (
	"context"
	"fmt"
	"time"

	"github.com/sinkers/miner-cli/internal/braiins/client"
	vnishclient "github.com/sinkers/miner-cli/internal/vnish/client"
)

// ClientConfig contains configuration for creating miner clients
type ClientConfig struct {
	Host     string
	Port     int
	Timeout  time.Duration
	Username string
	Password string
	APIKey   string
}

// Factory creates appropriate miner clients based on detected API type
type Factory struct {
	detector *Detector
	config   ClientConfig
}

// NewFactory creates a new miner client factory
func NewFactory(config ClientConfig) *Factory {
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	return &Factory{
		detector: NewDetector(5 * time.Second),
		config:   config,
	}
}

// CreateClient creates a client based on detected API type
func (f *Factory) CreateClient(ctx context.Context, host string) (MinerAPI, error) {
	// Detect API type
	apiType, info, err := f.detector.DetectAPI(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("failed to detect API type: %w", err)
	}

	// Create appropriate client based on detected type
	switch apiType {
	case APITypeCGMiner:
		return f.createCGMinerClient(host, info)
	case APITypeVNish:
		return f.createVNishClient(host, info)
	case APITypeBraiins:
		return f.createBraiinsClient(host, info)
	default:
		return nil, fmt.Errorf("unsupported API type: %s", apiType)
	}
}

// createCGMinerClient creates a CGMiner API client
func (f *Factory) createCGMinerClient(host string, info APIInfo) (MinerAPI, error) {
	port := info.Port
	if port == 0 {
		port = 4028
	}

	return NewCGMinerAdapter(host, port, f.config.Timeout), nil
}

// createVNishClient creates a VNish API client
func (f *Factory) createVNishClient(host string, info APIInfo) (MinerAPI, error) {
	opts := []vnishclient.Option{
		vnishclient.WithTimeout(f.config.Timeout),
	}

	if f.config.APIKey != "" {
		opts = append(opts, vnishclient.WithAPIKey(f.config.APIKey))
	}

	client := vnishclient.NewClient(host, opts...)
	return NewVNishAdapter(client), nil
}

// createBraiinsClient creates a Braiins API client
func (f *Factory) createBraiinsClient(host string, info APIInfo) (MinerAPI, error) {
	port := info.Port
	if port == 0 {
		port = 50051
	}

	opts := client.SimpleClientOptions{
		Host:     host,
		Port:     port,
		Username: f.config.Username,
		Password: f.config.Password,
		Timeout:  f.config.Timeout,
	}

	braiinsClient, err := client.NewSimpleClient(opts)
	if err != nil {
		return nil, fmt.Errorf("failed to create Braiins client: %w", err)
	}

	return NewBraiinsAdapter(braiinsClient), nil
}

// QuickCreate creates a client with a known API type (skips detection)
func (f *Factory) QuickCreate(host string, apiType APIType) (MinerAPI, error) {
	info := APIInfo{Type: apiType}

	switch apiType {
	case APITypeCGMiner:
		info.Port = 4028
		return f.createCGMinerClient(host, info)
	case APITypeVNish:
		info.Port = 80
		return f.createVNishClient(host, info)
	case APITypeBraiins:
		info.Port = 50051
		return f.createBraiinsClient(host, info)
	default:
		return nil, fmt.Errorf("unsupported API type: %s", apiType)
	}
}

// DetectAndCreateBatch creates clients for multiple hosts in parallel
func (f *Factory) DetectAndCreateBatch(ctx context.Context, hosts []string) map[string]MinerAPI {
	clients := make(map[string]MinerAPI)
	clientChan := make(chan struct {
		host   string
		client MinerAPI
		err    error
	}, len(hosts))

	// Create clients in parallel
	for _, host := range hosts {
		go func(h string) {
			client, err := f.CreateClient(ctx, h)
			clientChan <- struct {
				host   string
				client MinerAPI
				err    error
			}{h, client, err}
		}(host)
	}

	// Collect results
	for i := 0; i < len(hosts); i++ {
		result := <-clientChan
		if result.err == nil && result.client != nil {
			clients[result.host] = result.client
		}
	}

	return clients
}
