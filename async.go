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

// Injector is used for injecting values from the ctx into the carrier.
//
// This is in order to preserve needed values between the context when initializing a new go routine.
type Injector interface {
	Inject(ctx context.Context, carrier interface{ Carrier })
}

const (
	defaultMaxGoRoutines       = 100
	defaultTimeoutForGuard     = time.Second * 5
	defaultTimeoutForGoRoutine = time.Second * 5
)

type Async struct {
	traceServiceName    string
	guard               chan struct{}
	reporter            ErrorReporter
	timeoutForGuard     time.Duration
	timeoutForGoRoutine time.Duration
	contextInjectors    []Injector
}

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

func (a *Async) RunAsync(ctx context.Context, fn HandleFunc) {
	ctx = asyncContext(ctx, a.contextInjectors)

	select {
	case a.guard <- struct{}{}:
		go func() {
			ctx, cacnelFunc := context.WithTimeout(ctx, a.timeoutForGoRoutine)

			var err error
			defer func() {
				cacnelFunc()
				<-a.guard
			}()

			defer recoverPanic(ctx, a.reporter)

			if err = fn(ctx); err != nil {
				a.reporter.Error(ctx, fmt.Errorf("async func failed: %w", err))
			}
		}()

	case <-time.After(a.timeoutForGuard):
		a.reporter.Error(ctx, errorTimeout(fmt.Errorf("async timeout while waiting to guard")))
	}
}

func asyncContext(ctx context.Context, injectors []Injector) context.Context {
	newCtx := context.Background()

	carrier := ctxCarrier{newCtx}
	for _, inj := range injectors {
		inj.Inject(ctx, &carrier)
	}

	return carrier.ctx
}

func recoverPanic(ctx context.Context, reporter ErrorReporter) {
	if r := recover(); r != nil {
		err, ok := r.(error)
		if !ok {
			err = fmt.Errorf("%v", r)
		}

		reporter.Error(ctx, fmt.Errorf("mmasyc recoverPanic: %w", err))
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
