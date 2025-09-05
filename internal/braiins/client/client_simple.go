package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"time"

	pb "github.com/sinkers/miner-cli/internal/braiins/bos/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// SimpleBraiinsClient provides a simplified interface to Braiins OS+ API
type SimpleBraiinsClient struct {
	conn      *grpc.ClientConn
	authToken string
	host      string
	port      int
	timeout   time.Duration
}

// SimpleClientOptions contains configuration for the simple client
type SimpleClientOptions struct {
	Host               string
	Port               int
	Username           string
	Password           string
	Timeout            time.Duration
	UseTLS             bool
	InsecureSkipVerify bool
}

// NewSimpleClient creates a new simplified Braiins client
func NewSimpleClient(opts SimpleClientOptions) (*SimpleBraiinsClient, error) {
	if opts.Host == "" {
		return nil, fmt.Errorf("host is required")
	}

	if opts.Port == 0 {
		opts.Port = 50051 // Default gRPC port
	}

	if opts.Timeout == 0 {
		opts.Timeout = 30 * time.Second
	}

	// Create gRPC connection
	address := fmt.Sprintf("%s:%d", opts.Host, opts.Port)

	var dialOpts []grpc.DialOption

	if opts.UseTLS {
		tlsConfig := &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		}
		creds := credentials.NewTLS(tlsConfig)
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(creds))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.Dial(address, dialOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	client := &SimpleBraiinsClient{
		conn:    conn,
		host:    opts.Host,
		port:    opts.Port,
		timeout: opts.Timeout,
	}

	// Authenticate if credentials provided
	if opts.Username != "" && opts.Password != "" {
		authClient := pb.NewAuthenticationServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()

		resp, err := authClient.Login(ctx, &pb.LoginRequest{
			Username: opts.Username,
			Password: opts.Password,
		})
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		client.authToken = resp.Token
	}

	return client, nil
}

// Close closes the client connection
func (c *SimpleBraiinsClient) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// getContext creates a context with timeout and auth
func (c *SimpleBraiinsClient) getContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	if c.authToken != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", c.authToken)
	}
	return ctx, cancel
}

// GetMinerDetails retrieves miner information
func (c *SimpleBraiinsClient) GetMinerDetails() (*pb.GetMinerDetailsResponse, error) {
	client := pb.NewMinerServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetMinerDetails(ctx, &pb.GetMinerDetailsRequest{})
}

// GetMinerStats retrieves mining statistics
func (c *SimpleBraiinsClient) GetMinerStats() (*pb.GetMinerStatsResponse, error) {
	client := pb.NewMinerServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetMinerStats(ctx, &pb.GetMinerStatsRequest{})
}

// GetHashboards retrieves hashboard information
func (c *SimpleBraiinsClient) GetHashboards() (*pb.GetHashboardsResponse, error) {
	client := pb.NewMinerServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetHashboards(ctx, &pb.GetHashboardsRequest{})
}

// GetPoolGroups retrieves pool configuration
func (c *SimpleBraiinsClient) GetPoolGroups() (*pb.GetPoolGroupsResponse, error) {
	client := pb.NewPoolServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetPoolGroups(ctx, &pb.GetPoolGroupsRequest{})
}

// GetCoolingState retrieves cooling information
func (c *SimpleBraiinsClient) GetCoolingState() (*pb.GetCoolingStateResponse, error) {
	client := pb.NewCoolingServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetCoolingState(ctx, &pb.GetCoolingStateRequest{})
}

// SetImmersionMode configures immersion cooling
func (c *SimpleBraiinsClient) SetImmersionMode(enabled bool) (*pb.SetImmersionModeResponse, error) {
	client := pb.NewCoolingServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.SetImmersionMode(ctx, &pb.SetImmersionModeRequest{
		SaveAction:          pb.SaveAction_SAVE_ACTION_SAVE,
		EnableImmersionMode: enabled,
	})
}

// GetTunerState retrieves performance tuning state
func (c *SimpleBraiinsClient) GetTunerState() (*pb.GetTunerStateResponse, error) {
	client := pb.NewPerformanceServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetTunerState(ctx, &pb.GetTunerStateRequest{})
}

// SetPowerTarget sets power consumption target
func (c *SimpleBraiinsClient) SetPowerTarget(watts uint64) (*pb.SetPowerTargetResponse, error) {
	client := pb.NewPerformanceServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.SetPowerTarget(ctx, &pb.SetPowerTargetRequest{
		SaveAction: pb.SaveAction_SAVE_ACTION_SAVE,
		PowerTarget: &pb.Power{
			Watt: watts,
		},
	})
}

// SetHashrateTarget sets hashrate target
func (c *SimpleBraiinsClient) SetHashrateTarget(thps float64) (*pb.SetHashrateTargetResponse, error) {
	client := pb.NewPerformanceServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.SetHashrateTarget(ctx, &pb.SetHashrateTargetRequest{
		SaveAction: pb.SaveAction_SAVE_ACTION_SAVE,
		HashrateTarget: &pb.TeraHashrate{
			TerahashPerSecond: thps,
		},
	})
}

// StartMining starts mining operation
func (c *SimpleBraiinsClient) StartMining() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.Start(ctx, &pb.StartRequest{})
	return err
}

// StopMining stops mining operation
func (c *SimpleBraiinsClient) StopMining() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.Stop(ctx, &pb.StopRequest{})
	return err
}

// RestartMining restarts mining operation
func (c *SimpleBraiinsClient) RestartMining() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.Restart(ctx, &pb.RestartRequest{})
	return err
}

// PauseMining pauses mining operation
func (c *SimpleBraiinsClient) PauseMining() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.PauseMining(ctx, &pb.PauseMiningRequest{})
	return err
}

// ResumeMining resumes mining operation
func (c *SimpleBraiinsClient) ResumeMining() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.ResumeMining(ctx, &pb.ResumeMiningRequest{})
	return err
}

// Reboot reboots the miner
func (c *SimpleBraiinsClient) Reboot() error {
	client := pb.NewActionsServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	_, err := client.Reboot(ctx, &pb.RebootRequest{})
	return err
}

// GetLicenseState retrieves license information
func (c *SimpleBraiinsClient) GetLicenseState() (*pb.GetLicenseStateResponse, error) {
	client := pb.NewLicenseServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetLicenseState(ctx, &pb.GetLicenseStateRequest{})
}

// GetMinerConfiguration retrieves miner configuration
func (c *SimpleBraiinsClient) GetMinerConfiguration() (*pb.GetMinerConfigurationResponse, error) {
	client := pb.NewConfigurationServiceClient(c.conn)
	ctx, cancel := c.getContext()
	defer cancel()
	return client.GetMinerConfiguration(ctx, &pb.GetMinerConfigurationRequest{})
}