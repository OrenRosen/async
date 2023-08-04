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

// ContextPropagator is used for moving values from the ctx into the new context.
// This is in order to preserve needed values between the context when initializing a new go routine.
type ContextPropagator interface {
	MoveToContext(from, to context.Context) context.Context
}

type ContextPropagatorFunc func(from, to context.Context) context.Context

func (f ContextPropagatorFunc) MoveToContext(from, to context.Context) context.Context {
	return f(from, to)
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
	contextPropagators  []ContextPropagator
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
		contextPropagators:  conf.contextPropagators,
	}
}

func (a *Async) RunAsync(ctx context.Context, fn HandleFunc) {
	ctx = asyncContext(ctx, a.contextPropagators)

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

func asyncContext(ctx context.Context, contextPropagators []ContextPropagator) context.Context {
	newCtx := context.Background()

	for _, propagator := range contextPropagators {
		newCtx = propagator.MoveToContext(ctx, newCtx)
	}

	return newCtx
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
