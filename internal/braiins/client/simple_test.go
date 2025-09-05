package client

import (
	"testing"
	"time"
)

func TestNewSimpleClient(t *testing.T) {
	tests := []struct {
		name    string
		opts    SimpleClientOptions
		wantErr bool
	}{
		{
			name: "valid options",
			opts: SimpleClientOptions{
				Host:     "192.168.1.100",
				Port:     50051,
				Username: "admin",
				Password: "password",
				Timeout:  30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "missing host",
			opts: SimpleClientOptions{
				Port: 50051,
			},
			wantErr: true,
		},
		{
			name: "default port",
			opts: SimpleClientOptions{
				Host: "192.168.1.100",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Just test validation logic (can't connect without real server)
			if tt.opts.Host == "" {
				_, err := NewSimpleClient(tt.opts)
				if (err != nil) != tt.wantErr {
					t.Errorf("NewSimpleClient() error = %v, wantErr %v", err, tt.wantErr)
				}
			}
		})
	}
}

func TestSimpleClientContext(t *testing.T) {
	client := &SimpleBraiinsClient{
		timeout:   5 * time.Second,
		authToken: "test-token",
	}

	ctx, cancel := client.getContext()
	defer cancel()

	// Check that context has timeout
	deadline, ok := ctx.Deadline()
	if !ok {
		t.Error("Expected context to have deadline")
	}

	expectedDeadline := time.Now().Add(5 * time.Second)
	if deadline.After(expectedDeadline.Add(1 * time.Second)) {
		t.Error("Deadline is too far in the future")
	}
}