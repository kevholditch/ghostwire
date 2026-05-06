package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"

	"github.com/kevholditch/ghostwire/internal/agent"
	"github.com/kevholditch/ghostwire/internal/wireguard"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := agent.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	identity, err := agent.LoadOrCreateIdentity(ctx, cfg.StateDir, cfg.AgentID)
	if err != nil {
		log.Fatal(err)
	}
	client := agent.NewClient(cfg.ControlURL)
	device := wireguard.NewLinuxDevice(cfg.InterfaceName)
	daemon := agent.NewDaemon(cfg, identity, client, device)

	if err := daemon.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatal(err)
	}
}
