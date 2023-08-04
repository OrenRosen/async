# Package gosync

Package `gosync` provides simple helpers for running code in a go routine.

### Main features (or why to use this package instead of just `go func`):
- Limiting the number of opened go routines.
- Recovering from a panic.
- Prevent cancellation from propagate to the go routine
- Propagate values between contexts.
- Configurable timeouts.
- Handling errors from the async function.

# Simple Use-Cases:

### Run function in a new go routine
The most basic functionality is to open a new go routine when everytime it is called:

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
	
	// call `a.RunAsync` with a context and a closure, which will be run in a new go routine
	a.RunAsync(context.Background(), func(ctx context.Context) error {
		fmt.Println("Running in async")
		return nil
	})
	
	// for the example, sleeping in order to see the print from the async function
	fmt.Println("Going to sleep...")
	time.Sleep(time.Millisecond * 100)
}
```

### Run a function in a pool of go routines

This use-case is for cases you know you have a lot of traffic, and you would not like to open a new go routine for each call. Instead, it initializes a pool of go routines, which are ready to handle an incoming function:

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
	// open 10 go routine, in each go routine a worker is listens on a channel for a received function 
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
When `RunAsync` is called, it opens a new go routine, and executes `fn`.

## AsyncOption
You can pass options to the initializer for custom configuration. The available options are

#### Max opened go routines
```go
func WithMaxGoRoutines(n uint) AsyncOption
```
Use this option for setting the max go routine that `Async` will open. If `RunAsync` is called when there are already max go routines open, it waits for a go routine to be finished before opening a new go routine.

There is a default timeout for waiting to a go routine to be finished of 5 seconds. You can change this time with another option

- The default for this option is 100

#### Timeout to wait for go routine to be finished
```go
func WithTimeoutForGuard(t time.Duration) AsyncOption
```
Use this option in order to set the timeout which `Async` waits when there are max go routines open.

- The default for this option is 5 seconds.

#### Max time for a go routine to run
```go
func WithTimeoutForGoRoutine(t time.Duration) AsyncOption
```
Use this option to set the max time that a go routine is running. This is basically the timeout for the function that is passed in `RunAsync`.  
- The default for this option is 5 seconds.


#### Error reporter
```go
func WithErrorReporter(reporter ErrorReporter) AsyncOption
```
Use this option to set the reporter. The reporter implements `ErrorReporter`:
```go
type ErrorReporter interface {
	Error(ctx context.Context, err error)
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













