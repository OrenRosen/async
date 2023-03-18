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

type Async struct {
	*async
}

type async struct {
	traceServiceName    string
	guard               chan struct{}
	reporter            ErrorReporter
	maxGoRoutines       uint
	timeoutForGuard     time.Duration
	timeoutForGoRoutine time.Duration
	contextInjectors    []Injector
}

type AsyncOption func(*async)

func New(options ...AsyncOption) *Async {
	a := &async{
		reporter:            noopReporter{},
		maxGoRoutines:       defaultMaxGoRoutines,
		timeoutForGuard:     defaultTimeoutForGuard,
		timeoutForGoRoutine: defaultTimeoutForGoRoutine,
	}

	for _, op := range options {
		op(a)
	}

	a.guard = make(chan struct{}, a.maxGoRoutines)

	return &Async{
		async: a,
	}
}

func WithContextInjector(injector Injector) AsyncOption {
	return func(a *async) {
		a.contextInjectors = append(a.contextInjectors, injector)
	}
}

func WithTimeoutForGuard(t time.Duration) AsyncOption {
	return func(a *async) {
		a.timeoutForGuard = t
	}
}

func WithTimeoutForGoRoutine(t time.Duration) AsyncOption {
	return func(a *async) {
		a.timeoutForGoRoutine = t
	}
}

func WithErrorReporter(reporter ErrorReporter) AsyncOption {
	return func(a *async) {
		a.reporter = reporter
	}
}

func WithMaxGoRoutines(n uint) AsyncOption {
	return func(a *async) {
		a.maxGoRoutines = n
	}
}

func (a *async) RunAsync(ctx context.Context, fn func(ctx context.Context) error) {
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

func (a *async) asyncContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	carrier := ctxCarrier{newCtx}
	for _, inj := range a.contextInjectors {
		inj.Inject(ctx, &carrier)
	}

	return carrier.ctx
}

func (a *async) recoverPanic(ctx context.Context) {
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
