package async

import (
	"context"
	"fmt"
	"time"
)

type errorTimeout error

type ErrorReporter interface {
	Error(ctx context.Context, err error)
}

type Carrier interface {
	Set(key, value string)
}

type Injector interface {
	Inject(ctx context.Context, carrier interface{ Carrier })
}

const (
	defaultMaxGoRoutines       = 100
	defaultTimeoutForGuard     = time.Second * 5
	defaultTimeoutForGoRoutine = time.Second * 5
)

type Config struct {
	reporter            ErrorReporter
	maxGoRoutines       uint
	timeoutForGuard     time.Duration
	timeoutForGoRoutine time.Duration
	contextInjectors    []Injector

	// only for pool
	poolSize        uint
	numberOfWorkers int
}

type Async struct {
	traceServiceName    string
	guard               chan struct{}
	reporter            ErrorReporter
	timeoutForGuard     time.Duration
	timeoutForGoRoutine time.Duration
	contextInjectors    []Injector
}

type AsyncOption func(*Config)

func New(options ...AsyncOption) *Async {
	conf := Config{
		reporter:            noopReporter{},
		maxGoRoutines:       defaultMaxGoRoutines,
		timeoutForGuard:     defaultTimeoutForGuard,
		timeoutForGoRoutine: defaultTimeoutForGoRoutine,
	}

	for _, op := range options {
		op(&conf)
	}

	return &Async{
		guard:               make(chan struct{}, conf.maxGoRoutines),
		reporter:            conf.reporter,
		timeoutForGuard:     conf.timeoutForGuard,
		timeoutForGoRoutine: conf.timeoutForGoRoutine,
		contextInjectors:    conf.contextInjectors,
	}
}

func WithContextInjector(injector Injector) AsyncOption {
	return func(conf *Config) {
		conf.contextInjectors = append(conf.contextInjectors, injector)
	}
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

func WithNumberOfWorkers(n int) AsyncOption {
	return func(conf *Config) {
		conf.numberOfWorkers = n
	}
}

func WithPoolSize(n uint) AsyncOption {
	return func(conf *Config) {
		conf.poolSize = n
	}
}

func (a *Async) RunAsync(ctx context.Context, fn func(ctx context.Context) error) {
	ctx = a.asyncContext(ctx)

	select {
	case a.guard <- struct{}{}:
		go func() {
			ctx, cacnelFunc := context.WithTimeout(ctx, a.timeoutForGoRoutine)

			var err error
			defer func() {
				cacnelFunc()
				<-a.guard
			}()

			defer a.recoverPanic(ctx)

			if err = fn(ctx); err != nil {
				a.reporter.Error(ctx, fmt.Errorf("async func failed: %w", err))
			}
		}()

	case <-time.After(a.timeoutForGuard):
		a.reporter.Error(ctx, errorTimeout(fmt.Errorf("async timeout while waiting to guard")))
	}
}

func (a *Async) asyncContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	carrier := ctxCarrier{newCtx}
	for _, inj := range a.contextInjectors {
		inj.Inject(ctx, &carrier)
	}

	return carrier.ctx
}

func (a *Async) recoverPanic(ctx context.Context) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		a.reporter.Error(ctx, fmt.Errorf("mmasyc recoverPanic: %w", err))
	}
}

type noopReporter struct {
}

func (_ noopReporter) Error(ctx context.Context, err error) {

}

type ctxCarrier struct {
	ctx context.Context
}

func (c *ctxCarrier) Set(key, value string) {
	c.ctx = context.WithValue(c.ctx, key, value)
}
