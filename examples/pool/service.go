package main

import (
	"context"
	"fmt"
)

type service struct {
}

func (s *service) DoWork(ctx context.Context, i int) error {
	fmt.Printf("service is working on data %d\n", i)
	return nil
}
