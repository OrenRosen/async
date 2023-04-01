package async

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultNumWorkers = 10
	defaultPoolSize   = 100
)

type dataCtx[T any] struct {
	ctx  context.Context
	data T
}

type Pool[T any] struct {
	dataChannel      chan dataCtx[T]
	reporter         ErrorReporter
	timeout          time.Duration
	contextInjectors []Injector
}

type PoolHandleFunc[T any] func(ctx context.Context, data T) error

// NewPool creates a new pool instance with N number of workers
// the workers are listening on channel for handling the receiving data
// by triggering the received function (fn PoolHandleFunc[T]).
//
// the Pool instance that returned implements Dispatch for adding a data to the channel
// 			func Dispatch(ctx context.Context, data T)
//
// Note - can't close the pool.
// TODO - add Close functionality.
func NewPool[T any](fn PoolHandleFunc[T], options ...AsyncOption) *Pool[T] {
	conf := Config{
		reporter:            noopReporter{},
		maxGoRoutines:       defaultMaxGoRoutines,
		timeoutForGuard:     defaultTimeoutForGuard,
		timeoutForGoRoutine: defaultTimeoutForGoRoutine,
		numberOfWorkers:     defaultNumWorkers,
		poolSize:            defaultPoolSize,
	}

	for _, op := range options {
		op(&conf)
	}

	p := &Pool[T]{
		dataChannel:      make(chan dataCtx[T], conf.poolSize),
		reporter:         conf.reporter,
		timeout:          conf.timeoutForGuard,
		contextInjectors: conf.contextInjectors,
	}

	for i := 0; i < conf.numberOfWorkers; i++ {
		w := worker[T]{
			id:          i,
			dataChannel: p.dataChannel,
			fn:          fn,
			reporter:    conf.reporter,
			timeout:     p.timeout,
		}
		w.startReceivingData()
	}

	return p
}

// Dispatch adds a data to the channel which will be received by a worker.
func (p *Pool[T]) Dispatch(ctx context.Context, data T) {
	ctx = p.asyncContext(ctx)

	go func() {
		select {
		case p.dataChannel <- dataCtx[T]{
			ctx:  ctx,
			data: data,
		}:
		case <-time.After(p.timeout):
			err := fmt.Errorf("pool.Dispatch channel is full, timeout waiting for dispatch")
			p.reporter.Error(ctx, err)
		}
	}()
}

type worker[T any] struct {
	id          int
	dataChannel chan dataCtx[T]
	fn          PoolHandleFunc[T]
	reporter    ErrorReporter
	timeout     time.Duration
}

func (w *worker[T]) startReceivingData() {
	go func() {
		for data := range w.dataChannel {
			w.handleData(data.ctx, data.data)
		}
	}()
}

func (w *worker[T]) handleData(ctx context.Context, data T) {
	ctx, cacnelFunc := context.WithTimeout(ctx, w.timeout)
	defer cacnelFunc()

	if err := w.fn(ctx, data); err != nil {
		err = fmt.Errorf("async handleData: %w", err)
		w.reporter.Error(ctx, err)
	}
}

func (p *Pool[T]) asyncContext(ctx context.Context) context.Context {
	newCtx := context.Background()

	carrier := ctxCarrier{newCtx}
	for _, inj := range p.contextInjectors {
		inj.Inject(ctx, &carrier)
	}

	return carrier.ctx
}
