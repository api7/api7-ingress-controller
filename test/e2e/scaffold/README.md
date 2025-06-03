# E2E Scaffold Architecture

This document describes the refactored e2e testing scaffold architecture that supports both API7 enterprise and APISIX standalone modes.

## Architecture Overview

```
┌─────────────────┐    ┌─────────────────┐
│    Scaffold     │    │     Factory     │
│                 │    │                 │
│  - Options      │───▶│  CreateDeployer │
│  - DeployMode   │    │                 │
└─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │    Deployer     │
                       │   Interface     │
                       │                 │
                       │  - Deploy()     │
                       │  - Cleanup()    │
                       │  - GetClient()  │
                       └─────────────────┘
                                │
                    ┌───────────┴───────────┐
                    ▼                       ▼
            ┌─────────────────┐    ┌─────────────────┐
            │  API7 Deployer  │    │ APISIX Deployer │
            │                 │    │                 │
            │ - Uses existing │    │ - Standalone    │
            │   framework     │    │   deployment    │
            │   logic         │    │                 │
            └─────────────────┘    └─────────────────┘
```

## Key Components

### 1. Interfaces

#### TestScaffold
- Main interface that defines common methods for all scaffold implementations
- Includes HTTP client methods, resource management, Kubernetes operations
- Both API7 and APISIX scaffolds implement this interface

#### Deployer
- Interface for deploying data plane components
- Abstracts the differences between API7 and APISIX deployments
- Provides unified methods for getting clients and managing deployments

### 2. Scaffold Implementations

#### API7Scaffold (api7_scaffold.go)
- Implements TestScaffold for API7 enterprise version
- Uses API7Deployer internally
- Supports API7-specific features like gateway groups

#### APISIXScaffold (apisix_scafflod.go)
- Implements TestScaffold for APISIX standalone
- Uses APISIXDeployer internally
- Simplified for standalone mode

#### BaseScaffold (common.go)
- Provides common implementation for shared methods
- Reduces code duplication between scaffold implementations
- Contains resource management utilities

### 3. Deployer Implementations

#### API7Deployer (api7_deployer.go)
- Implements Deployer interface for API7 enterprise
- Deploys API7 dashboard and gateway
- Manages enterprise-specific configurations

#### APISIXDeployer (apisix_deployer.go)
- Implements Deployer interface for APISIX standalone
- Deploys APISIX in standalone mode
- Simpler deployment without dashboard

### 4. Factory Pattern

#### DeployerFactory (deployer.go)
- Creates appropriate deployer based on deployment mode
- Environment variable: `DEPLOY_MODE` (api7|apisix)
- Handles framework type validation

## Usage

### Environment Variables

```bash
# For API7 mode
export DEPLOY_MODE=api7
export API7_EE_LICENSE=your_license_key
export DASHBOARD_VERSION=dev

# For APISIX mode  
export DEPLOY_MODE=apisix
export APISIX_IMAGE=apache/apisix:3.8.0
export APISIX_ADMIN_KEY=edd1c9f034335f136f87ad84b625c8f1
```

### Test Entry Points

#### API7 Tests (test/e2e/e2e_test.go)
```go
scaffold.NewScaffold = func(opts *scaffold.Options) scaffold.TestScaffold {
    return scaffold.NewAPI7Scaffold(opts)
}
```

#### APISIX Tests (test/e2e/apisix/e2e_test.go)
```go
scaffold.NewScaffold = func(opts *scaffold.Options) scaffold.TestScaffold {
    return scaffold.NewAPISIXScaffold(opts)
}
```

## Common Methods vs Specific Methods

### Common Methods (in TestScaffold interface)
- `NewAPISIXClient()` - HTTP client for data plane
- `NewAPISIXHttpsClient(host string)` - HTTPS client for data plane
- `CreateResourceFromString()` - K8s resource management
- `AdminKey()` - Admin authentication
- `GetDeployer()` - Access to underlying deployer

### API7-Specific Methods (TODO: Move to separate interface)
- `CreateAdditionalGatewayGroup()` - Multi-gateway support
- `GetAdditionalGatewayGroup()` - Gateway group management
- `NewAPISIXClientForGatewayGroup()` - Per-gateway clients

### Implementation Status

#### ✅ Completed
- Interface definitions
- DeployerFactory with environment-based selection
- API7Deployer basic structure
- APISIXDeployer basic structure  
- BaseScaffold common utilities

#### 🚧 In Progress (marked with TODO)
- Complete API7Deployer implementation
- Complete APISIXDeployer implementation
- APISIX scaffold method implementations
- Virtual dashboard cluster for APISIX standalone

#### 📋 Future Work
- Separate API7-specific methods to dedicated interface
- Remove deprecated gateway group methods from common interface
- Improve error handling and logging
- Add comprehensive test coverage

## Migration Notes

### For API7 Tests
- No changes required - existing tests continue to work
- New deployer provides same functionality through cleaner interface

### For APISIX Tests
- Use new entry point: `test/e2e/apisix/e2e_test.go`
- Set `DEPLOY_MODE=apisix` environment variable
- Tests run against APISIX standalone instead of API7 dashboard

### Code Organization
- Common logic moved to BaseScaffold
- Deployment logic separated into Deployer implementations
- Clear separation between API7 and APISIX concerns 