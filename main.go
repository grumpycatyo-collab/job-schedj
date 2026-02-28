package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"example.com/m/config"
	sch "example.com/m/scheduler"
)

func cleanupHandler(ctx context.Context) error {
	fmt.Println("running cleanup handler")
	return nil
}

const configPath string = "./config.json"

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.NewConfig(configPath)
	if err != nil {
		return
	}

	store := sch.NewRedisStore(cfg.Redis)
	scheduler := sch.NewScheduler(store)
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
