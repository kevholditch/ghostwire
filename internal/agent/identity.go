package agent

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kevholditch/ghostwire/internal/wireguard"
)

type Identity struct {
	AgentID    string
	PrivateKey string
	PublicKey  string
}

func LoadOrCreateIdentity(ctx context.Context, stateDir, configuredAgentID string) (Identity, error) {
	if err := os.MkdirAll(stateDir, 0o700); err != nil {
		return Identity{}, fmt.Errorf("create state dir: %w", err)
	}
	agentID, err := loadOrCreateAgentID(filepath.Join(stateDir, "agent-id"), configuredAgentID)
	if err != nil {
		return Identity{}, err
	}
	privateKey, err := loadOrCreatePrivateKey(ctx, filepath.Join(stateDir, "wireguard.key"))
	if err != nil {
		return Identity{}, err
	}
	publicKey, err := wireguard.PublicKey(ctx, privateKey)
	if err != nil {
		return Identity{}, err
	}
	return Identity{AgentID: agentID, PrivateKey: privateKey, PublicKey: publicKey}, nil
}

func loadOrCreateAgentID(path, configured string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(data))
		if id == "" {
			return "", fmt.Errorf("empty agent id file: %s", path)
		}
		return id, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read agent id: %w", err)
	}

	id := configured
	if id == "" {
		buf := make([]byte, 16)
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate agent id: %w", err)
		}
		id = "agent-" + hex.EncodeToString(buf)
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write agent id: %w", err)
	}
	return id, nil
}

func loadOrCreatePrivateKey(ctx context.Context, path string) (string, error) {
	if data, err := os.ReadFile(path); err == nil {
		key := strings.TrimSpace(string(data))
		if key == "" {
			return "", fmt.Errorf("empty wireguard key file: %s", path)
		}
		return key, nil
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("read wireguard key: %w", err)
	}

	key, err := wireguard.GeneratePrivateKey(ctx)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(key+"\n"), 0o600); err != nil {
		return "", fmt.Errorf("write wireguard key: %w", err)
	}
	return key, nil
}
