package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/sai-aurosy/platform/pkg/edge"
	"github.com/sai-aurosy/platform/pkg/telemetry"
)

func main() {
	cfg := edge.LoadConfig()
	bus, err := telemetry.NewBus(cfg.NATSURL)
	if err != nil {
		log.Fatalf("NATS: %v", err)
	}
	defer bus.Close()

	agent := edge.NewAgent(cfg, bus)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		cancel()
	}()

	log.Printf("[edge] Edge agent %s starting (NATS=%s, Cloud=%s)", cfg.EdgeID, cfg.NATSURL, cfg.CloudURL)
	agent.Run(ctx)
	log.Printf("[edge] Edge agent %s stopped", cfg.EdgeID)
	os.Exit(0)
}
