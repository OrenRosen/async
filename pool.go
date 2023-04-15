package async

import (
	"context"
	"fmt"
	"time"
)

const (
	defaulttimeoutInsertToPool = time.Second * 5
	
	defaultNumWorkers = 10
	defaultPoolSize   = 100
)

type funcChannelData struct {
	ctx context.Context
	fn  HandleFunc
}

// Pool is a generic type for handling asynchronous calls.
//
// It opens n workers that listen
type Pool struct {
	funcChannel         chan funcChannelData
	reporter            ErrorReporter
	timeoutInsertToPool time.Duration
	contextInjectors    []Injector
}

type HandleFunc func(ctx context.Context) error

// NewPool creates a new Pool instance. The method initializes n number of workers (10 is the default) that listen for a received function.
//
// When calling Pool.RunAsync with HandleFunc, it adds the function to a channel which consumed by the workers.
//
// Options:
//	- WithTimeoutForGoRoutine: the max time to wait for fn to be finished.
//	- WithErrorReporter: add a custom reporter that will be triggered in case of an error.
//	- WithContextInjector
//	- WithNumberOfWorkers: The amount of workers.
//	- WithPoolSize: The size of the pool. When calling Pool.Dispatch when the pool is fool, it will wait until the timeout had reached.
// Note - can't close the pool.
//
// TODO - add Close functionality.
func NewPool(options ...PoolOption) *Pool {
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
	
	p := &Pool{
		funcChannel:         make(chan funcChannelData, conf.poolSize),
		reporter:            conf.reporter,
		timeoutInsertToPool: conf.timeoutForInsertToPool,
		contextInjectors:    conf.contextInjectors,
	}
	
	for i := 0; i < conf.numberOfWorkers; i++ {
		w := worker{
			id:          i,
			funcChannel: p.funcChannel,
			reporter:    conf.reporter,
			timeout:     conf.timeoutForGoroutine,
		}
		w.startReceivingData()
	}
	
	return p
}

// RunAsync adds the function into the channel which will be received by a worker.
func (p *Pool) RunAsync(ctx context.Context, fn HandleFunc) {
	data := funcChannelData{
		ctx: p.asyncContext(ctx),
		fn:  fn,
	}
	
	go func() {
		select {
		case p.funcChannel <- data:
		case <-time.After(p.timeoutInsertToPool):
			err := fmt.Errorf("pool.Dispatch channel is full, timeout waiting for dispatch")
			p.reporter.Error(ctx, err)
		}
	}()
}

type worker struct {
	id          int
	funcChannel chan funcChannelData
	reporter    ErrorReporter
	timeout     time.Duration
}

func (w *worker) startReceivingData() {
	go func() {
		for data := range w.funcChannel {
			w.handleData(data.ctx, data.fn)
		}
	}()
}

func (w *worker) handleData(ctx context.Context, fn HandleFunc) {
	ctx, cacnelFunc := context.WithTimeout(ctx, w.timeout)
	defer cacnelFunc()
	
	defer recoverPanic(ctx, w.reporter)
	
	if err := fn(ctx); err != nil {
		err = fmt.Errorf("async handleData: %w", err)
		w.reporter.Error(ctx, err)
	}
}

func (p *Pool) asyncContext(ctx context.Context) context.Context {
	newCtx := context.Background()
	
	carrier := ctxCarrier{newCtx}
	for _, inj := range p.contextInjectors {
		inj.Inject(ctx, &carrier)
	}
	
	return carrier.ctx
}
