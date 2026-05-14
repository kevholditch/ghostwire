package main

import (
	"log"
	"net/http"
	"time"

	"github.com/kevholditch/ghostwire/internal/controlplane"
)

func main() {
	cfg, err := controlplane.ConfigFromEnv()
	if err != nil {
		log.Fatal(err)
	}
	ipam, err := controlplane.NewIPAM(cfg.NetworkCIDR)
	if err != nil {
		log.Fatal(err)
	}
	registry := controlplane.NewRegistry(cfg.RegistryConfig(), ipam)
	server := controlplane.NewServer(registry, time.Now, cfg.APIToken)

	log.Printf("ghostwire-control listening on %s", cfg.ListenAddr)
	if err := http.ListenAndServe(cfg.ListenAddr, server); err != nil {
		log.Fatal(err)
	}
}
