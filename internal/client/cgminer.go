package client

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	cgminer "github.com/x1unix/go-cgminer-api"
)

type Result struct {
	IP       string      `json:"ip"`
	Port     int         `json:"port"`
	Command  string      `json:"command"`
	Response interface{} `json:"response,omitempty"`
	Error    string      `json:"error,omitempty"`
	Duration string      `json:"duration"`
}

type Client struct {
	timeout time.Duration
	workers int
}

func NewClient(timeout time.Duration, workers int) *Client {
	if workers <= 0 {
		workers = 10
	}
	return &Client{
		timeout: timeout,
		workers: workers,
	}
}

func (c *Client) ExecuteCommand(ctx context.Context, ips []string, port int, command string, params map[string]interface{}) []Result {
	jobs := make(chan job, len(ips))
	results := make(chan Result, len(ips))

	var wg sync.WaitGroup

	for i := 0; i < c.workers; i++ {
		wg.Add(1)
		go c.worker(ctx, &wg, jobs, results)
	}

	for _, ip := range ips {
		jobs <- job{
			ip:      ip,
			port:    port,
			command: command,
			params:  params,
		}
	}
	close(jobs)

	wg.Wait()
	close(results)

	var allResults []Result
	for r := range results {
		allResults = append(allResults, r)
	}

	return allResults
}

type job struct {
	ip      string
	port    int
	command string
	params  map[string]interface{}
}

func (c *Client) worker(ctx context.Context, wg *sync.WaitGroup, jobs <-chan job, results chan<- Result) {
	defer wg.Done()

	for j := range jobs {
		select {
		case <-ctx.Done():
			results <- Result{
				IP:      j.ip,
				Port:    j.port,
				Command: j.command,
				Error:   "context cancelled",
			}
			continue
		default:
		}

		result := c.executeJob(j)
		results <- result
	}
}

func (c *Client) executeJob(j job) Result {
	start := time.Now()

	miner := cgminer.NewCGMiner(j.ip, j.port, c.timeout)

	var response interface{}
	var err error

	switch j.command {
	case "summary":
		response, err = miner.Summary()
	case "devs":
		response, err = miner.Devs()
	case "pools":
		response, err = miner.Pools()
	case "stats":
		response, err = miner.Stats()
	case "version":
		response, err = miner.Version()
	case "switchpool":
		if poolID, ok := j.params["pool"].(int); ok {
			pools, poolErr := miner.Pools()
			if poolErr != nil {
				err = fmt.Errorf("failed to get pools: %w", poolErr)
			} else if poolID < len(pools) {
				err = miner.SwitchPool(&pools[poolID])
				if err == nil {
					response = "Pool switched successfully"
				}
			} else {
				err = fmt.Errorf("pool ID %d not found", poolID)
			}
		} else {
			err = fmt.Errorf("switchpool command requires 'pool' parameter")
		}
	case "enablepool":
		if poolID, ok := j.params["pool"].(int); ok {
			pools, poolErr := miner.Pools()
			if poolErr != nil {
				err = fmt.Errorf("failed to get pools: %w", poolErr)
			} else if poolID < len(pools) {
				err = miner.EnablePool(&pools[poolID])
				if err == nil {
					response = "Pool enabled successfully"
				}
			} else {
				err = fmt.Errorf("pool ID %d not found", poolID)
			}
		} else {
			err = fmt.Errorf("enablepool command requires 'pool' parameter")
		}
	case "disablepool":
		if poolID, ok := j.params["pool"].(int); ok {
			pools, poolErr := miner.Pools()
			if poolErr != nil {
				err = fmt.Errorf("failed to get pools: %w", poolErr)
			} else if poolID < len(pools) {
				err = miner.DisablePool(&pools[poolID])
				if err == nil {
					response = "Pool disabled successfully"
				}
			} else {
				err = fmt.Errorf("pool ID %d not found", poolID)
			}
		} else {
			err = fmt.Errorf("disablepool command requires 'pool' parameter")
		}
	case "addpool":
		url, urlOk := j.params["url"].(string)
		user, userOk := j.params["user"].(string)
		pass, passOk := j.params["pass"].(string)
		if urlOk && userOk && passOk {
			err = miner.AddPool(url, user, pass)
			if err == nil {
				response = "Pool added successfully"
			}
		} else {
			err = fmt.Errorf("addpool requires 'url', 'user', and 'pass' parameters")
		}
	case "removepool":
		if poolID, ok := j.params["pool"].(int); ok {
			pools, poolErr := miner.Pools()
			if poolErr != nil {
				err = fmt.Errorf("failed to get pools: %w", poolErr)
			} else if poolID < len(pools) {
				err = miner.RemovePool(&pools[poolID])
				if err == nil {
					response = "Pool removed successfully"
				}
			} else {
				err = fmt.Errorf("pool ID %d not found", poolID)
			}
		} else {
			err = fmt.Errorf("removepool command requires 'pool' parameter")
		}
	case "restart":
		err = miner.Restart()
		if err == nil {
			response = "Miner restarting"
		}
	case "quit":
		err = miner.Quit()
		if err == nil {
			response = "Miner quitting"
		}
	case "custom":
		if cmd, ok := j.params["cmd"].(string); ok {
			var args interface{}
			if argsParam, exists := j.params["args"]; exists {
				args = argsParam
			}
			response, err = c.customCommand(miner, cmd, args)
		} else {
			err = fmt.Errorf("custom command requires 'cmd' parameter")
		}
	default:
		err = fmt.Errorf("unknown command: %s", j.command)
	}

	duration := time.Since(start)

	result := Result{
		IP:       j.ip,
		Port:     j.port,
		Command:  j.command,
		Duration: duration.String(),
	}

	if err != nil {
		result.Error = err.Error()
	} else {
		result.Response = response
	}

	return result
}

func (c *Client) customCommand(miner *cgminer.CGMiner, command string, args interface{}) (interface{}, error) {
	ctx := context.Background()

	// Create the command based on whether we have parameters
	var cmd cgminer.Command
	if args != nil {
		if strArg, ok := args.(string); ok {
			cmd = cgminer.NewCommand(command, strArg)
		} else {
			// Convert args to JSON string if it's not already a string
			argsBytes, err := json.Marshal(args)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal args: %w", err)
			}
			cmd = cgminer.NewCommand(command, string(argsBytes))
		}
	} else {
		cmd = cgminer.NewCommandWithoutParameter(command)
	}

	respBytes, err := miner.RawCall(ctx, cmd)
	if err != nil {
		return nil, err
	}

	var response interface{}
	if err := json.Unmarshal(respBytes, &response); err != nil {
		return string(respBytes), nil
	}

	return response, nil
}

func GetAvailableCommands() []string {
	return []string{
		"summary",
		"devs",
		"pools",
		"stats",
		"version",
		"switchpool",
		"enablepool",
		"disablepool",
		"addpool",
		"removepool",
		"restart",
		"quit",
		"custom",
	}
}

func GetCommandDescription(cmd string) string {
	descriptions := map[string]string{
		"summary":     "Get mining summary information",
		"devs":        "Get information about all devices",
		"pools":       "Get information about all pools",
		"stats":       "Get detailed statistics",
		"version":     "Get miner version information",
		"switchpool":  "Switch to a different pool (requires --pool)",
		"enablepool":  "Enable a pool (requires --pool)",
		"disablepool": "Disable a pool (requires --pool)",
		"addpool":     "Add a new pool (requires --url, --user, --pass)",
		"removepool":  "Remove a pool (requires --pool)",
		"restart":     "Restart the miner",
		"quit":        "Stop the miner",
		"custom":      "Run a custom command (requires --cmd)",
	}
	return descriptions[cmd]
}
