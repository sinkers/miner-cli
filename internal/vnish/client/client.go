package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourproject/miner-cli/internal/vnish/models"
)

// Client represents a vnish API client
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	debug      bool
}

// Option is a functional option for configuring the client
type Option func(*Client)

// WithAPIKey sets the API key for authentication
func WithAPIKey(apiKey string) Option {
	return func(c *Client) {
		c.apiKey = apiKey
	}
}

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithDebug enables debug mode
func WithDebug(debug bool) Option {
	return func(c *Client) {
		c.debug = debug
	}
}

// NewClient creates a new vnish API client
func NewClient(host string, opts ...Option) *Client {
	c := &Client{
		baseURL: fmt.Sprintf("http://%s/api/v1", host),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// doRequest performs an HTTP request with proper headers and error handling
func (c *Client) doRequest(ctx context.Context, method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	if c.debug {
		fmt.Printf("Request: %s %s\n", method, req.URL.String())
		if body != nil {
			fmt.Printf("Body: %+v\n", body)
		}
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if c.debug {
		fmt.Printf("Response Status: %d\n", resp.StatusCode)
		fmt.Printf("Response Body: %s\n", string(respBody))
	}

	if resp.StatusCode >= 400 {
		var errResp models.ErrDescr
		if err := json.Unmarshal(respBody, &errResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, errResp.Error)
	}

	return respBody, nil
}

// Warranty Management

// ActivateWarranty activates the warranty
func (c *Client) ActivateWarranty(ctx context.Context) (*models.WarrantyStatus, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/activate-warranty", nil)
	if err != nil {
		return nil, err
	}

	var result models.WarrantyStatus
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// CancelWarranty cancels the warranty
func (c *Client) CancelWarranty(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/cancel-warranty", nil)
	return err
}

// API Key Management

// GetAPIKeys retrieves all API keys
func (c *Client) GetAPIKeys(ctx context.Context) ([]models.ApiKeysJsonItem, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/apikeys", nil)
	if err != nil {
		return nil, err
	}

	var result []models.ApiKeysJsonItem
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// AddAPIKey adds a new API key
func (c *Client) AddAPIKey(ctx context.Context, name string) (*models.AddApiKeyRes, error) {
	req := models.AddApikeyQuery{Name: name}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/apikeys", req)
	if err != nil {
		return nil, err
	}

	var result models.AddApiKeyRes
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteAPIKey deletes an API key
func (c *Client) DeleteAPIKey(ctx context.Context, id string) error {
	req := models.DeleteApikeyQuery{ID: id}
	_, err := c.doRequest(ctx, http.MethodPost, "/apikeys/delete", req)
	return err
}

// Authentication

// CheckAuth checks authentication status
func (c *Client) CheckAuth(ctx context.Context) (*models.AuthCheck, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/auth-check", nil)
	if err != nil {
		return nil, err
	}

	var result models.AuthCheck
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// System Information

// GetInfo retrieves system information
func (c *Client) GetInfo(ctx context.Context) (*models.SystemInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/info", nil)
	if err != nil {
		return nil, err
	}

	var result models.SystemInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetModel retrieves model information
func (c *Client) GetModel(ctx context.Context) (*models.ModelInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/model", nil)
	if err != nil {
		return nil, err
	}

	var result models.ModelInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetStatus retrieves current status
func (c *Client) GetStatus(ctx context.Context) (*models.Status, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/status", nil)
	if err != nil {
		return nil, err
	}

	var result models.Status
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetSummary retrieves summary information
func (c *Client) GetSummary(ctx context.Context) (*models.Summary, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/summary", nil)
	if err != nil {
		return nil, err
	}

	var result models.Summary
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetPerfSummary retrieves performance summary
func (c *Client) GetPerfSummary(ctx context.Context) (*models.PerfSummary, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/perf-summary", nil)
	if err != nil {
		return nil, err
	}

	var result models.PerfSummary
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Hardware Information

// GetChains retrieves chain information
func (c *Client) GetChains(ctx context.Context) ([]models.ChainInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/chains", nil)
	if err != nil {
		return nil, err
	}

	var result []models.ChainInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetFactoryInfo retrieves factory information
func (c *Client) GetFactoryInfo(ctx context.Context) (*models.FactoryInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/chains/factory-info", nil)
	if err != nil {
		return nil, err
	}

	var result models.FactoryInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// GetChips retrieves chip information
func (c *Client) GetChips(ctx context.Context) ([]models.ChipInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/chips", nil)
	if err != nil {
		return nil, err
	}

	var result []models.ChipInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetLayout retrieves layout information
func (c *Client) GetLayout(ctx context.Context) (*models.LayoutInfo, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/layout", nil)
	if err != nil {
		return nil, err
	}

	var result models.LayoutInfo
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Mining Operations

// StartMining starts the mining process
func (c *Client) StartMining(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/start", nil)
	return err
}

// StopMining stops the mining process
func (c *Client) StopMining(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/stop", nil)
	return err
}

// RestartMining restarts the mining process
func (c *Client) RestartMining(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/restart", nil)
	return err
}

// PauseMining pauses the mining process
func (c *Client) PauseMining(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/pause", nil)
	return err
}

// ResumeMining resumes the mining process
func (c *Client) ResumeMining(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/resume", nil)
	return err
}

// SwitchPool switches to a different mining pool
func (c *Client) SwitchPool(ctx context.Context, poolID int) error {
	req := models.SwitchPoolRequest{PoolID: poolID}
	_, err := c.doRequest(ctx, http.MethodPost, "/mining/switch-pool", req)
	return err
}

// Autotune Operations

// GetAutotunePresets retrieves available autotune presets
func (c *Client) GetAutotunePresets(ctx context.Context) (*models.AutotunePresets, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/autotune/presets", nil)
	if err != nil {
		return nil, err
	}

	var result models.AutotunePresets
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ResetAutotune resets autotune for a specific chain
func (c *Client) ResetAutotune(ctx context.Context, chainIndex int) error {
	path := fmt.Sprintf("/autotune/reset?chain=%d", chainIndex)
	_, err := c.doRequest(ctx, http.MethodPost, path, nil)
	return err
}

// ResetAllAutotune resets autotune for all chains
func (c *Client) ResetAllAutotune(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/autotune/reset-all", nil)
	return err
}

// Settings Management

// GetSettings retrieves current settings
func (c *Client) GetSettings(ctx context.Context) (*models.Settings, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/settings", nil)
	if err != nil {
		return nil, err
	}

	var result models.Settings
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// UpdateSettings updates settings
func (c *Client) UpdateSettings(ctx context.Context, settings *models.Settings) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/settings", settings)
	return err
}

// BackupSettings backs up current settings
func (c *Client) BackupSettings(ctx context.Context) (*models.BackupResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/settings/backup", nil)
	if err != nil {
		return nil, err
	}

	var result models.BackupResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RestoreSettings restores settings from backup
func (c *Client) RestoreSettings(ctx context.Context, req *models.RestoreRequest) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/settings/restore", req)
	return err
}

// FactoryReset performs a factory reset
func (c *Client) FactoryReset(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/settings/factory-reset", nil)
	return err
}

// Firmware Operations

// UpdateFirmware updates the firmware
func (c *Client) UpdateFirmware(ctx context.Context, req *models.FirmwareUpdateRequest) (*models.FirmwareUpdateResponse, error) {
	respBody, err := c.doRequest(ctx, http.MethodPost, "/firmware/update", req)
	if err != nil {
		return nil, err
	}

	var result models.FirmwareUpdateResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// RemoveFirmware removes custom firmware
func (c *Client) RemoveFirmware(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/firmware/remove", nil)
	return err
}

// Logs and Metrics

// GetLogs retrieves logs of specified type
func (c *Client) GetLogs(ctx context.Context, logType string) ([]models.LogEntry, error) {
	path := fmt.Sprintf("/logs/%s", logType)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result []models.LogEntry
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// ClearLogs clears logs of specified type
func (c *Client) ClearLogs(ctx context.Context, logType string) error {
	path := fmt.Sprintf("/logs/%s/clear", logType)
	_, err := c.doRequest(ctx, http.MethodPost, path, nil)
	return err
}

// GetMetrics retrieves metrics
func (c *Client) GetMetrics(ctx context.Context) (*models.Metrics, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/metrics", nil)
	if err != nil {
		return nil, err
	}

	var result models.Metrics
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// Notes Management

// GetNotes retrieves all notes
func (c *Client) GetNotes(ctx context.Context) ([]models.Note, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/notes", nil)
	if err != nil {
		return nil, err
	}

	var result []models.Note
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return result, nil
}

// GetNote retrieves a specific note
func (c *Client) GetNote(ctx context.Context, noteID string) (*models.Note, error) {
	path := fmt.Sprintf("/notes/%s", noteID)
	respBody, err := c.doRequest(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var result models.Note
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// CreateNote creates a new note
func (c *Client) CreateNote(ctx context.Context, content string) (*models.Note, error) {
	req := models.CreateNoteRequest{Content: content}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/notes", req)
	if err != nil {
		return nil, err
	}

	var result models.Note
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// UpdateNote updates an existing note
func (c *Client) UpdateNote(ctx context.Context, noteID string, content string) (*models.Note, error) {
	path := fmt.Sprintf("/notes/%s", noteID)
	req := models.CreateNoteRequest{Content: content}
	respBody, err := c.doRequest(ctx, http.MethodPut, path, req)
	if err != nil {
		return nil, err
	}

	var result models.Note
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// DeleteNote deletes a note
func (c *Client) DeleteNote(ctx context.Context, noteID string) error {
	path := fmt.Sprintf("/notes/%s", noteID)
	_, err := c.doRequest(ctx, http.MethodDelete, path, nil)
	return err
}

// UI Settings

// GetUISettings retrieves UI settings
func (c *Client) GetUISettings(ctx context.Context) (*models.UISettings, error) {
	respBody, err := c.doRequest(ctx, http.MethodGet, "/ui", nil)
	if err != nil {
		return nil, err
	}

	var result models.UISettings
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// UpdateUISettings updates UI settings
func (c *Client) UpdateUISettings(ctx context.Context, settings *models.UISettings) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/ui", settings)
	return err
}

// Lock/Unlock Operations

// Lock locks the miner
func (c *Client) Lock(ctx context.Context, password string) error {
	req := models.LockRequest{Password: password}
	_, err := c.doRequest(ctx, http.MethodPost, "/lock", req)
	return err
}

// LockOthers locks other miners
func (c *Client) LockOthers(ctx context.Context, password string) error {
	req := models.LockRequest{Password: password}
	_, err := c.doRequest(ctx, http.MethodPost, "/lock/others", req)
	return err
}

// Unlock unlocks the miner
func (c *Client) Unlock(ctx context.Context, password string) error {
	req := models.UnlockRequest{Password: password}
	_, err := c.doRequest(ctx, http.MethodPost, "/unlock", req)
	return err
}

// System Operations

// Reboot reboots the system
func (c *Client) Reboot(ctx context.Context) error {
	_, err := c.doRequest(ctx, http.MethodPost, "/system/reboot", nil)
	return err
}

// FindMiner helps find the physical miner
func (c *Client) FindMiner(ctx context.Context, blink bool) (*models.FindMinerResponse, error) {
	req := models.FindMinerRequest{Blink: blink}
	respBody, err := c.doRequest(ctx, http.MethodPost, "/find-miner", req)
	if err != nil {
		return nil, err
	}

	var result models.FindMinerResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}