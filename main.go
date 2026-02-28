package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	sch "example.com/m/scheduler"
)

func cleanupHandler(ctx context.Context) error {
	fmt.Println("running cleanup handler")
	return nil
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	scheduler := sch.NewScheduler()
	if err := scheduler.Register(sch.Task{
		ID:       "cleanup",
		Schedule: "@every 5s",
		Handler:  cleanupHandler,
	}); err != nil {
		return
	}

	if err := scheduler.Start(ctx); err != nil {
		return
	}
}
