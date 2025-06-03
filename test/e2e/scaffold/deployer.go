package scaffold

import (
	"context"
)

// Deployer defines the interface for deploying data plane components
type Deployer interface {
	// Deploy deploys components for scaffold
	Deploy(ctx context.Context) error
}
