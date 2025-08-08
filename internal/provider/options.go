package provider

import "time"

type Option interface {
	ApplyToList(*Options)
}

type Options struct {
	SyncTimeout   time.Duration
	SyncPeriod    time.Duration
	InitSyncDelay time.Duration
	BackendMode   string
}

func (o *Options) ApplyToList(lo *Options) {
	if o.SyncTimeout > 0 {
		lo.SyncTimeout = o.SyncTimeout
	}
	if o.SyncPeriod > 0 {
		lo.SyncPeriod = o.SyncPeriod
	}
	if o.InitSyncDelay > 0 {
		lo.InitSyncDelay = o.InitSyncDelay
	}
	if o.BackendMode != "" {
		lo.BackendMode = o.BackendMode
	}
}

func (o *Options) ApplyOptions(opts []Option) *Options {
	for _, opt := range opts {
		opt.ApplyToList(o)
	}
	return o
}

type backendModeOption string

func (b backendModeOption) ApplyToList(o *Options) {
	o.BackendMode = string(b)
}

func WithBackendMode(mode string) Option {
	return backendModeOption(mode)
}
