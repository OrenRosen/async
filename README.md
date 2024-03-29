# Package gosync

[![Build Status](https://github.com/OrenRosen/go/actions/workflows/merge.yaml/badge.svg?branch=main)](https://github.com/OrenRosen/go/blob/main/.github/workflows/merge.yaml)
![Coverage](https://img.shields.io/badge/Coverage-78.9%25-brightgreen)
[![Go Reference](https://pkg.go.dev/badge/github.com/OrenRosen/async.svg)](https://pkg.go.dev/github.com/OrenRosen/async)


Package `async` provides simple wrapper for running code in a goroutine. - you can read about it more in here: https://orenrose.medium.com/goroutine-wrapper-for-recover-and-context-propagation-54dc571ac0f4

### Main features (or why to use this package instead of just `go func`):
- Limiting the number of opened goroutines.
- Recovering from a panic.
- Prevent cancellation from propagate to the goroutine
- Propagate values between contexts.z
- Configurable timeouts.
- Handling errors from the async function.

# Simple Use-Cases:

### Run function in a new goroutine
The most basic functionality is to open a new goroutine when everytime it is called:

Example:

```go
package main

import (
	"context"
	"fmt"
	"time"
	
	"github.com/OrenRosen/async"
)

func main() {
	// initialize the async helper
	a := async.New()
	
	// call `a.RunAsync` with a context and a closure, which will be run in a new goroutine
	a.RunAsync(context.Background(), func(ctx context.Context) error {
		fmt.Println("Running in async")
		return nil
	})
	
	// for the example, sleeping in order to see the print from the async function
	fmt.Println("Going to sleep...")
	time.Sleep(time.Millisecond * 100)
}
```

### Run a function in a pool of goroutines

This use-case is for cases you know you have a lot of traffic, and you would not like to open a new goroutine for each call. Instead, it initializes a pool of goroutines, which are ready to handle an incoming function:

```go
package main

import (
	"context"
	"fmt"
	"time"

	"github.com/OrenRosen/async"
)

func main() {
	// initialize the pool
	// open 10 goroutine, in each goroutine a worker is listens on a channel for a received function 
	pool := async.NewPool()
	
	// call `pool.RunAsync` with a context and a closure.
	// this will add the passed function to the queue channel to be consumed by an available worker 
	pool.RunAsync(context.Background(), func(ctx context.Context) error {
		fmt.Println("running in async pool")
		return nil
	})
	
	// for the example, sleeping in order to see the print from the async function
	fmt.Println("going to sleep...")
	time.Sleep(time.Second)
}
```








## Initializing
```go
func New(options ...AsyncOption) *Async
```
The function `New` initializes a new `Async` instance which implements `Asyncer`:
```go
type Asyncer interface {
    RunAsync(ctx context.Context, fn func(ctx context.Context) error)
}
```
When `RunAsync` is called, it opens a new goroutine, and executes `fn`.

## AsyncOption
You can pass options to the initializer for custom configuration. The available options are

#### Max opened goroutines
```go
func WithMaxGoRoutines(n uint) AsyncOption
```
Use this option for setting the max goroutine that `Async` will open. If `RunAsync` is called when there are already max goroutines open, it waits for a goroutine to be finished before opening a new goroutine.

There is a default timeout for waiting to a goroutine to be finished of 5 seconds. You can change this time with another option

- The default for this option is 100

#### Timeout to wait for goroutine to be finished
```go
func WithTimeoutForGuard(t time.Duration) AsyncOption
```
Use this option in order to set the timeout which `Async` waits when there are max goroutines open.

- The default for this option is 5 seconds.

#### Max time for a goroutine to run
```go
func WithTimeoutForGoRoutine(t time.Duration) AsyncOption
```
Use this option to set the max time that a goroutine is running. This is basically the timeout for the function that is passed in `RunAsync`.  
- The default for this option is 5 seconds.


#### Error Handler
```go
func WithErrorHandler(errorHandler ErrorHandler) AsyncOption
```
Use this option to set the error handler. The reporter implements `ErrorHandler`:
```go
type ErrorReporter interface {
	HandleError(ctx context.Context, err error)
}
```
It will be called in case of an error. You can use this option for logging/monitoring.

#### Context Propagator
Use this option to add a context propagator. 
```go
func WithContextPropagator(propagator ContextPropagator) AsyncOption
```
A context propagator is used to propagate values between contexts. Since the passed function runs async, you wouldn't want that cancelling the original context will affect your code in the passed function, so async uses other context to pass into the function.

There are cases though, you want to propagate a value, to be used in the passed function. For example traceID, user details, log id etc...

For more info, you look at the examples and tests. 

# Propagation Example
As been said, when you start a function in the background, in a different goroutine you can't use the same context. This is because the cancellation of the original context shouldn't affect your async function to run properly. The package `async` takes care and passes other context into the goroutine.

Let's say you have a traceID as a value in the context. The traceID is usually should passed all around the flow, so you want to have it also in the function you pass to the `RunAsync` method.
To do that, when starting the async (or the pool), you can pass ContextPropagator to do exactly that:
```go
// ContextPropagator is used for moving values from the ctx into the new context.
// This is in order to preserve needed values between the context when initializing a new goroutine.
type ContextPropagator interface {
    MoveToContext(from, to context.Context) context.Context
}

// The ContextPropagatorFunc type is an adapter to allow the use of ordinary functions as context propagators.
// If f is a function with the appropriate signature, ContextPropagatorFunc(f) is a propagator that calls f.
type ContextPropagatorFunc func(from, to context.Context) context.Context

func (f ContextPropagatorFunc) MoveToContext(from, to context.Context) context.Context {
    return f(from, to)
}
```
When initializing the `async` or `pool`, use the option `WithContextPropagator`: 

```go
	a := async.New(
        async.WithContextPropagation(async.ContextPropagatorFunc(func(from, to context.Context) context.Context {
            return context.WithValue(to, "SomeKey", from.Value("SomeKey"))
        })),
    )
```

Now, every time you will call `a.RunAsync`, you will have this value in the passed context:
```go
	a.RunAsync(ctx, func(ctx context.Context) error {
		// `ctx` contains the value under key `SomeKey`
		
		//...
	}
```

## TODOs:

- Add option to propagate keys (instead of passing a propagator, only pass the key) 
- Dynamic workers count (auto-scale)
- Close the pool








// 