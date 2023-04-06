package async

import (
	"context"
	"fmt"
	"time"
)

const (
	defaulttimeoutForGoroutine = time.Second * 5
	defaulttimeoutInsertToPool = time.Second * 5

	defaultNumWorkers = 10
	defaultPoolSize   = 100
)

type dataCtx[T any] struct {
	ctx  context.Context
	data T
}

// Pool is a generic type for handling asynchronous calls.
//
// It opens n workers that listen
type Pool[T any] struct {
	dataChannel         chan dataCtx[T]
	reporter            ErrorReporter
	timeoutInsertToPool time.Duration
	contextInjectors    []Injector
}

type PoolHandleFunc[T any] func(ctx context.Context, data T) error

// NewPool creates a new Pool instance. The method initializes n number of workers (10 is the default) that listen for a received data.
//
// When calling Pool.Dispatch with data of type T, it adds the data to a channel which consumed by the workers.
//
// Args:
//	- fn: function to be called when the data T is consumed by a worker.
//	- options: configurable options for the pool
//		- WithTimeoutForGoRoutine: the max time to wait for fn to be finished.
//		- WithErrorReporter: add a custom reporter that will be triggered in case of an error.
//		- WithContextInjector
//		- WithNumberOfWorkers: The amount of workers.
//		- WithPoolSize: The size of the pool. When calling Pool.Dispatch when the pool is fool, it will wait until the timeout had reached.
// Note - can't close the pool.
//
// TODO - add Close functionality.
func NewPool[T any](fn PoolHandleFunc[T], options ...PoolOption) *Pool[T] {
	conf := PoolConfig{
		reporter:               noopReporter{},
		timeoutForInsertToPool: defaulttimeoutInsertToPool,
		timeoutForGoroutine:    defaultTimeoutForGoRoutine,
		numberOfWorkers:        defaultNumWorkers,
		poolSize:               defaultPoolSize,
	}

	for _, op := range options {
		op(&conf)
	}

	p := &Pool[T]{
		dataChannel:         make(chan dataCtx[T], conf.poolSize),
		reporter:            conf.reporter,
		timeoutInsertToPool: conf.timeoutForInsertToPool,
		contextInjectors:    conf.contextInjectors,
	}

	for i := 0; i < conf.numberOfWorkers; i++ {
		w := worker[T]{
			id:          i,
			dataChannel: p.dataChannel,
			fn:          fn,
			reporter:    conf.reporter,
			timeout:     conf.timeoutForGoroutine,
		}
		w.startReceivingData()
	}

	return p
}

// Dispatch adds the data into the channel which will be received by a worker.
func (p *Pool[T]) Dispatch(ctx context.Context, data T) {
	ctx = p.asyncContext(ctx)

	go func() {
		select {
		case p.dataChannel <- dataCtx[T]{
			ctx:  ctx,
			data: data,
		}:
		case <-time.After(p.timeoutInsertToPool):
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

	defer recoverPanic(ctx, w.reporter)

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
