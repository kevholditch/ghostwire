package agent

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	AgentID                    string
	ControlURL                 string
	EnrollmentToken            string
	StateDir                   string
	Hostname                   string
	Endpoint                   string
	InterfaceName              string
	ListenPort                 int
	PersistentKeepaliveSeconds int
	HeartbeatInterval          time.Duration
	PollInterval               time.Duration
}

func ConfigFromEnv() (Config, error) {
	hostname := os.Getenv("GHOSTWIRE_HOSTNAME")
	if hostname == "" {
		name, err := os.Hostname()
		if err != nil {
			return Config{}, fmt.Errorf("hostname: %w", err)
		}
		hostname = name
	}
	cfg := Config{
		ControlURL:                 os.Getenv("GHOSTWIRE_CONTROL_URL"),
		AgentID:                    os.Getenv("GHOSTWIRE_AGENT_ID"),
		EnrollmentToken:            os.Getenv("GHOSTWIRE_ENROLLMENT_TOKEN"),
		StateDir:                   envOrDefault("GHOSTWIRE_STATE_DIR", "/var/lib/ghostwire"),
		Hostname:                   hostname,
		Endpoint:                   os.Getenv("GHOSTWIRE_ENDPOINT"),
		InterfaceName:              envOrDefault("GHOSTWIRE_INTERFACE", "gw0"),
		ListenPort:                 intEnvOrDefault("GHOSTWIRE_LISTEN_PORT", 51820),
		PersistentKeepaliveSeconds: intEnvOrDefault("GHOSTWIRE_PERSISTENT_KEEPALIVE", 25),
		HeartbeatInterval:          durationEnvOrDefault("GHOSTWIRE_HEARTBEAT_INTERVAL", 5*time.Second),
		PollInterval:               durationEnvOrDefault("GHOSTWIRE_POLL_INTERVAL", 5*time.Second),
	}
	if cfg.ControlURL == "" {
		return Config{}, fmt.Errorf("GHOSTWIRE_CONTROL_URL is required")
	}
	if cfg.EnrollmentToken == "" {
		return Config{}, fmt.Errorf("GHOSTWIRE_ENROLLMENT_TOKEN is required")
	}
	return cfg, nil
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func intEnvOrDefault(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
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
