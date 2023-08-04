package main

import (
	"context"
	"fmt"
	"time"

	"github.com/OrenRosen/async"
)

func main() {
	a := async.New(
		async.WithContextPropagation(async.ContextPropagatorFunc(func(from, to context.Context) context.Context {
			value := from.Value("SomeKey")
			return context.WithValue(to, "SomeKey", value)
		})),
	)

	ctx, cancelFunc := context.WithCancel(context.Background())
	ctx = context.WithValue(ctx, "SomeKey", "SomeValue")

	ch := make(chan string)
	a.RunAsync(ctx, func(ctx context.Context) error {
		fmt.Println("Doing stuff in the background - context.SomeKey =", ctx.Value("SomeKey"))
		select {
		case <-ctx.Done():
			fmt.Println("ERROR: in the background, context is done ", ctx.Err().Error())
		case <-time.After(time.Second):
			fmt.Println("Finished work in the background")
		}
		ch <- "done"
		return nil
	})

	fmt.Println("Doing bunch of work and canceling the original context")
	cancelFunc()
	fmt.Println("Waiting for go routine to finish...")
	_ = <-ch

	fmt.Println("Finished")
}

//
