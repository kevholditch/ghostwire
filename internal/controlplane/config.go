package controlplane

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	ListenAddr        string
	APIToken          string
	NetworkCIDR       string
	HeartbeatInterval time.Duration
	PollInterval      time.Duration
	AgentTTL          time.Duration
}

func ConfigFromEnv() (Config, error) {
	cfg := Config{
		ListenAddr:        envOrDefault("GHOSTWIRE_CONTROL_LISTEN", ":8080"),
		APIToken:          os.Getenv("GHOSTWIRE_API_TOKEN"),
		NetworkCIDR:       envOrDefault("GHOSTWIRE_NETWORK_CIDR", "10.44.0.0/24"),
		HeartbeatInterval: durationEnvOrDefault("GHOSTWIRE_HEARTBEAT_INTERVAL", 5*time.Second),
		PollInterval:      durationEnvOrDefault("GHOSTWIRE_POLL_INTERVAL", 5*time.Second),
		AgentTTL:          durationEnvOrDefault("GHOSTWIRE_AGENT_TTL", 30*time.Second),
	}
	if cfg.APIToken == "" {
		return Config{}, fmt.Errorf("GHOSTWIRE_API_TOKEN is required")
	}
	return cfg, nil
}

func (c Config) RegistryConfig() RegistryConfig {
	return RegistryConfig{
		NetworkCIDR:       c.NetworkCIDR,
		HeartbeatInterval: c.HeartbeatInterval,
		PollInterval:      c.PollInterval,
		AgentTTL:          c.AgentTTL,
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func durationEnvOrDefault(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
