package utils

import "sync"

type ParallelExecutor struct {
	wg sync.WaitGroup

	errorsLock sync.Mutex
	errors     []error
}

func (exec *ParallelExecutor) Add(handler func()) {
	exec.wg.Add(1)
	go func() {
		defer exec.wg.Done()
		handler()
	}()
}

func (exec *ParallelExecutor) AddE(handler func() error) {
	exec.wg.Add(1)
	go func() {
		defer exec.wg.Done()
		err := handler()
		if err != nil {
			exec.errorsLock.Lock()
			defer exec.errorsLock.Unlock()
			exec.errors = append(exec.errors, err)
		}
	}()
}

func (exec *ParallelExecutor) Wait() {
	exec.wg.Wait()
}

func (exec *ParallelExecutor) Errors() []error {
	return exec.errors
}
