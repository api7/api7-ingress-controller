package scaffold

type APISIXDeployer struct {
	Scaffold *Scaffold
}

func NewAPISIXDeployer(s *Scaffold) *APISIXDeployer {
	return &APISIXDeployer{
		Scaffold: s,
	}
}

func (d *APISIXDeployer) BeforeEach() {

}

func (d *APISIXDeployer) AfterEach() {

}

func (d *APISIXDeployer) DeployDataplane() {

}

func (d *APISIXDeployer) DeployIngress() {

}

func (d *APISIXDeployer) ScaleIngress(replicas int) {

}
