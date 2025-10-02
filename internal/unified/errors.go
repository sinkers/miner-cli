package unified

import "errors"

var (
	// ErrNotSupported indicates that the operation is not supported by this API
	ErrNotSupported = errors.New("operation not supported by this API")
	
	// ErrNotConnected indicates that the client is not connected
	ErrNotConnected = errors.New("not connected to miner")
	
	// ErrAuthRequired indicates that authentication is required
	ErrAuthRequired = errors.New("authentication required")
	
	// ErrInvalidResponse indicates an invalid response from the miner
	ErrInvalidResponse = errors.New("invalid response from miner")
	
	// ErrTimeout indicates that the operation timed out
	ErrTimeout = errors.New("operation timed out")
)