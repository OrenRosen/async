package async

import "time"

// Async options

type AsyncOption func(*Config)

type Config struct {
	reporter            ErrorReporter
	timeoutForGoRoutine time.Duration
	contextInjectors    []Injector
	maxGoRoutines       uint
	timeoutForGuard     time.Duration
}

func WithTimeoutForGuard(t time.Duration) AsyncOption {
	return func(conf *Config) {
		conf.timeoutForGuard = t
	}
}

func WithTimeoutForGoRoutine(t time.Duration) AsyncOption {
	return func(conf *Config) {
		conf.timeoutForGoRoutine = t
	}
}

func WithErrorReporter(reporter ErrorReporter) AsyncOption {
	return func(conf *Config) {
		conf.reporter = reporter
	}
}

func WithMaxGoRoutines(n uint) AsyncOption {
	return func(conf *Config) {
		conf.maxGoRoutines = n
	}
}

// WithContextInjector add an injector. An Injector is used for injecting values from the ctx into the carrier.
//
// This is in order to preserve needed values between the contexts when initializing a new go routine.
func WithContextInjector(injector Injector) AsyncOption {
	return func(conf *Config) {
		conf.contextInjectors = append(conf.contextInjectors, injector)
	}
}

// pool options

type PoolOption func(*PoolConfig)

type PoolConfig struct {
	reporter               ErrorReporter
	timeoutForGoroutine    time.Duration
	timeoutForInsertToPool time.Duration
	contextInjectors       []Injector
	poolSize               uint
	numberOfWorkers        int
}

func WithPoolTimeoutForGoRoutine(t time.Duration) PoolOption {
	return func(conf *PoolConfig) {
		conf.timeoutForGoroutine = t
	}
}

func WithPoolTimeoutInsertToPool(t time.Duration) PoolOption {
	return func(conf *PoolConfig) {
		conf.timeoutForInsertToPool = t
	}
}

func WithPoolErrorReporter(reporter ErrorReporter) PoolOption {
	return func(conf *PoolConfig) {
		conf.reporter = reporter
	}
}

func WithPoolNumberOfWorkers(n int) PoolOption {
	return func(conf *PoolConfig) {
		conf.numberOfWorkers = n
	}
}

func WithPoolSize(n uint) PoolOption {
	return func(conf *PoolConfig) {
		conf.poolSize = n
	}
}

func WithPoolContextInjector(injector Injector) PoolOption {
	return func(conf *PoolConfig) {
		conf.contextInjectors = append(conf.contextInjectors, injector)
	}
}
