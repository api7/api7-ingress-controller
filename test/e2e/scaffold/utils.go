package scaffold

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"
)

// GetKubeconfig returns the kubeconfig file path.
// Order:
// env KUBECONFIG;
// ~/.kube/config;
// "" (in case in-cluster configuration will be used).
func GetKubeconfig() string {
	kubeconfig := os.Getenv("KUBECONFIG")
	if kubeconfig == "" {
		u, err := user.Current()
		if err != nil {
			panic(err)
		}
		kubeconfig = filepath.Join(u.HomeDir, ".kube", "config")
		if _, err := os.Stat(kubeconfig); err != nil && !os.IsNotExist(err) {
			kubeconfig = ""
		}
	}
	return kubeconfig
}

func runWithRecover(f func()) {
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		err, ok := r.(error)
		if ok {
			// just ignore already closed channel
			if strings.Contains(err.Error(), "close of closed channel") {
				return
			}
		}
		panic(r)
	}()
	f()
}

func waitExponentialBackoff(condFunc func() (bool, error)) error {
	backoff := wait.Backoff{
		Duration: 500 * time.Millisecond,
		Factor:   2,
		Steps:    8,
	}
	return wait.ExponentialBackoff(backoff, condFunc)
}
