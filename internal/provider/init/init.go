package init

import (
	"github.com/apache/apisix-ingress-controller/internal/controller/status"
	"github.com/apache/apisix-ingress-controller/internal/manager/readiness"
	"github.com/apache/apisix-ingress-controller/internal/provider"
	"github.com/apache/apisix-ingress-controller/internal/provider/api7ee"
	"github.com/apache/apisix-ingress-controller/internal/provider/apisix"
)

func init() {
	provider.Register("apisix", apisix.New)
	provider.Register("apisix-standalone", func(statusUpdater status.Updater, readinessManager readiness.ReadinessManager, opts ...provider.Option) (provider.Provider, error) {
		opts = append(opts, provider.WithBackendMode("apisix-standalone"))
		return apisix.New(statusUpdater, readinessManager, opts...)
	})
	provider.Register("api7ee", api7ee.New)
}
