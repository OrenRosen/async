package main

import (
	"context"
	"fmt"

	"github.com/OrenRosen/async"
)

func main() {
	a := async.New()

	ch := make(chan string)
	a.RunAsync(context.Background(), func(ctx context.Context) error {
		ch <- "done"
		return nil
	})

	done := <-ch
	fmt.Println("---------------------- ", done)
}
