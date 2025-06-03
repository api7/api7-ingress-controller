package scaffold

// Deployer defines the interface for deploying data plane components
type Deployer interface {
	// Deploy deploys components for scaffold
	DeployDataplane()
	DeployIngress()
	ScaleIngress(replicas int)
	BeforeEach()
	AfterEach()
}

var NewDeployer func(*Scaffold) Deployer
