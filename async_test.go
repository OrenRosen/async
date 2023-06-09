package async_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/OrenRosen/async"
)

func Test_async_RunAsync(t *testing.T) {
	rep := &reporter{}
	asyncer := async.New(
		async.WithErrorReporter(rep),
		async.WithContextInjector(injector{}),
	)

	ctx := context.WithValue(context.Background(), "someKey", "someValue")
	ctx = context.WithValue(ctx, "someOtherKey", "someOtherValue")
	ch := make(chan struct{})
	asyncer.RunAsync(ctx, func(ctx context.Context) error {
		defer func() {
			ch <- struct{}{}
		}()

		val, ok := ctx.Value("someKey").(string)
		require.True(t, ok, "didn't find someKey")
		require.Equal(t, "someValue", val)

		_, ok = ctx.Value("someOtherKey").(string)
		require.False(t, ok, "someOtherKey shouldn't persist between contexts")

		return nil
	})

	select {
	case <-ch:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout waiting for channel")
	}

	require.False(t, rep.called)
}

func Test_async_RunAsync_Panic(t *testing.T) {
	rep := &reporter{}
	asyncer := async.New(
		async.WithErrorReporter(rep),
	)

	ctx := context.WithValue(context.Background(), "someKey", "someValue")
	ctx = context.WithValue(ctx, "someOtherKey", "someOtherValue")
	ch := make(chan struct{})
	asyncer.RunAsync(ctx, func(ctx context.Context) error {
		defer func() {
			ch <- struct{}{}
		}()
		panic("aaaa")
	})

	select {
	case <-ch:
	case <-time.After(time.Millisecond * 100):
		t.Fatal("timeout waiting for channel")
	}

	require.True(t, rep.called)
}

func Test_async_options(t *testing.T) {
	rep := &reporter{
		errorCh: make(chan struct{}),
	}

	// start asyncer with limit 1 in guard
	// short timeout for waiting to guard
	// do twice RunAsync that takes a long time
	// expect that reporter will be called
	asyncer := async.New(
		async.WithErrorReporter(rep),
		async.WithMaxGoRoutines(1),
		async.WithTimeoutForGuard(time.Millisecond*10),
		async.WithTimeoutForGoRoutine(time.Second*4),
	)

	asyncer.RunAsync(context.Background(), func(ctx context.Context) error {
		time.Sleep(time.Second * 5)
		return nil
	})

	go func() {
		select {
		case <-rep.errorCh:
		case <-time.After(time.Millisecond * 50):
			t.Error("reporter should have been called")
			return
		}
	}()

	asyncer.RunAsync(context.Background(), func(ctx context.Context) error {
		time.Sleep(time.Second * 5)
		return nil
	})
}

type reporter struct {
	called  bool
	errorCh chan struct{}
}

func (r *reporter) Error(ctx context.Context, err error) {
	r.called = true
	if r.errorCh != nil {
		r.errorCh <- struct{}{}
	}
}

type injector struct {
}

func (i injector) Inject(ctx context.Context, carrier interface{ async.Carrier }) {
	val, ok := ctx.Value("someKey").(string)
	if !ok {
		return
	}

	if val != "" {
		carrier.Set("someKey", val)
	}
}
