# Package async

Package `async` provides simple tool for asyncoronous work.

## Run function in go routine
The most basic functionality in async is to open a new go routine when called.

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
	a := async.New()
	
	a.RunAsync(context.Background(), func(ctx context.Context) error {
		fmt.Println("Running in async")
		return nil
	})
	
	time.Sleep(time.Millisecond * 100)
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

### AsyncOption
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

#### Context injectors
Use this option to add an injector
```go
func WithContextInjector(injector Injector) AsyncOption
```















d