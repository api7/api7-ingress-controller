package scaffold

import (
	"github.com/gavv/httpexpect/v2"
)

var NewScaffold func(*Options) TestScaffold

// TestScaffold defines the interface for test scaffold implementations
type TestScaffold interface {
	NewAPISIXClient() *httpexpect.Expect
	NewAPISIXHttpsClient(host string) *httpexpect.Expect
}
